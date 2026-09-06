package rainbow

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/caoyingjunz/pixiulib/exec"
	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/pixiulib/strutil"
	rainbowconfig "github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
	"github.com/caoyingjunz/rainbow/pkg/util/errors"
)

type AgentGetter interface {
	Agent() Interface
}
type Interface interface {
	Run(ctx context.Context, workers int) error

	Subscribe(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)
}

type AgentController struct {
	factory     db.ShareDaoFactory
	cfg         rainbowconfig.Config
	redisClient *redis.Client

	queue workqueue.RateLimitingInterface
	exec  exec.Interface

	name     string
	callback string
	baseDir  string
	token    string
}

func NewAgent(f db.ShareDaoFactory, cfg rainbowconfig.Config, redisClient *redis.Client) *AgentController {
	return &AgentController{
		factory:     f,
		cfg:         cfg,
		redisClient: redisClient,
		name:        cfg.Agent.Name,
		baseDir:     cfg.Agent.DataDir,
		callback:    cfg.Plugin.Callback,
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rainbow-agent"),
		exec:        exec.New(),
	}
}

func (s *AgentController) GetAgentToken(ctx context.Context) (string, error) {
	if len(s.token) != 0 {
		klog.Infof("已从缓存中获取到token，直接返回")
		return s.token, nil
	}

	klog.Infof("缓存中token不存在，尝试从库中获取")
	agent, err := s.factory.Agent().GetByName(ctx, s.name)
	if err != nil {
		klog.Errorf("获取 agent(%s)属性失败 %v", s.name, err)
		return "", err
	}
	if len(agent.GithubToken) == 0 {
		klog.Infof("agent(%s) 的 token 为空", agent.Name)
		return "", fmt.Errorf("agent(%s) 的 token 为空", s.name)
	}

	s.token = agent.GithubToken
	return s.token, nil
}

func (s *AgentController) Subscribe(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		klog.V(0).Infof("收到消息: Topic=%s, MessageID=%s, Body=%s", msg.Topic, msg.MsgId, string(msg.Body))
		if err := s.process(ctx, msg.Body); err != nil {
			klog.Errorf("处理接受失败 %v", err)
		}
	}
	return consumer.ConsumeSuccess, nil
}

func (s *AgentController) process(ctx context.Context, date []byte) error {
	var reqMeta types.CallMetaRequest
	if err := json.Unmarshal(date, &reqMeta); err != nil {
		klog.Errorf("failed to unmarshal remote meta request %v", err)
		return err
	}

	var (
		result []byte
		err    error
	)
	switch reqMeta.Type {
	case types.CallGithubType:
		result, err = s.ProcessGithub(ctx, reqMeta.CallGithubRequest)
	case types.CallKubernetesTagType:
		result, err = s.ProcessKubernetesTags(ctx, reqMeta.CallKubernetesTagRequest)
	case types.CallSearchType:
		result, err = s.ProcessSearch(ctx, reqMeta.CallSearchRequest)
	default:
		return fmt.Errorf("unsupported req call type %d", reqMeta.Type)
	}

	statusCode, errMessage := 0, ""
	if err != nil {
		statusCode, errMessage = 1, err.Error()
		klog.Errorf("远程调用失败 %v", err)
	}
	data, err := json.Marshal(types.SearchResult{Result: result, ErrMessage: errMessage, StatusCode: statusCode})
	if err != nil {
		klog.Errorf("序列化调用结果失败 %v", err)
		return fmt.Errorf("序列化调用结果失败 %v", err)
	}

	// 保存 30s
	if _, err := s.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, reqMeta.Uid, data, 30*time.Second)
		pipe.Publish(ctx, fmt.Sprintf("__keyspace@0__:%s", reqMeta.Uid), "set")
		return nil
	}); err != nil {
		klog.Errorf("临时存储失败 %v", err)
		return err
	}

	klog.Infof("调用结果已暂存, 缓存key(%s)", reqMeta.Uid)
	return nil
}

func (s *AgentController) Run(ctx context.Context, workers int) error {
	// 注册 rainbow 代理
	if err := s.RegisterAgentIfNotExist(ctx); err != nil {
		return err
	}

	go s.startHeartbeat(ctx)
	go s.getNextWorkItems(ctx)
	go s.startSyncActionUsage(ctx)
	go s.startGC(ctx)
	go s.startSubscribe(ctx)

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, s.worker, 1*time.Second)
	}

	return nil
}

func (s *AgentController) startSubscribe(ctx context.Context) {
	klog.Infof("Starting redis subscribe controller")

	channel := fmt.Sprintf("__nodekey@0__:%s", s.name) // Redis 通知频道格式
	pubSub := s.redisClient.Subscribe(ctx, channel)
	defer pubSub.Close()

	// 等待订阅确认
	if _, err := pubSub.Receive(ctx); err != nil {
		klog.Errorf("subscribe failed: %v", err)
		return
	}

	ch := pubSub.Channel()
	for {
		select {
		case msg := <-ch:
			// msg.Payload 是生产者 Publish 发送的消息内容（建议为数据键）
			key := msg.Payload

			// 根据数据键获取实际数据
			data, err := s.redisClient.Get(ctx, key).Bytes()
			if err == redis.Nil {
				klog.Warningf("data not found for key: %s", key)
				continue
			} else if err != nil {
				klog.Errorf("get data error: %v", err)
				continue
			}
			klog.Infof("received key=%s, data=%s", key, string(data))
			if err = s.process(ctx, data); err != nil {
				klog.Errorf("处理接受失败 %v", err)
			}
		case <-ctx.Done():
			klog.Info("subscriber stopped")
			return
		}
	}
}

func (s *AgentController) startGC(ctx context.Context) {
	// 1小时尝试回收一次
	ticker := time.NewTicker(900 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.GarbageCollect(ctx); err != nil {
			klog.Errorf("GarbageCollect 失败: %v", err)
			continue
		}
		klog.Infof("GarbageCollect 完成")
	}
}

func (s *AgentController) GarbageCollect(ctx context.Context) error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "plugin" {
			continue
		}

		fileInfo, err := entry.Info()
		if err != nil {
			klog.Errorf("获取文件夹(%s)信息失败 %v, 忽略", entry.Name(), err)
			continue
		}

		// 回收指定时间的文件
		now := time.Now()
		if now.Sub(fileInfo.ModTime()) > 30*time.Minute {
			removeDir := filepath.Join(s.baseDir, fileInfo.Name())
			util.RemoveFile(removeDir)
			klog.Infof("任务文件 %s 已被回收", removeDir)
		} else {
			klog.Infof("任务文件 %s 还在有效期内，暂不回收", fileInfo.Name())
		}
	}

	return nil
}

func (s *AgentController) startSyncActionUsage(ctx context.Context) {
	rand.Seed(time.Now().UnixNano())

	// 15分钟同步一次
	ticker := time.NewTicker(1800 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		agent, err := s.factory.Agent().GetByName(ctx, s.name)
		if err != nil {
			klog.Errorf("获取 agent 失败 %v 等待下次同步", err)
			continue
		}
		if len(agent.GithubUser) == 0 || len(agent.GithubRepository) == 0 || len(agent.GithubToken) == 0 {
			klog.Infof("agent(%s) 的 github 属性存在空值，忽略", agent.Name)
			continue
		}
		if agent.Status != model.RunAgentType {
			klog.Warningf("agent 处于未运行状态，忽略")
			continue
		}

		// TODO: 随机等待一段时间
		klog.Infof("开始同步 agent(%s) 的 usage", agent.Name)
		if err = s.syncActionUsage(ctx, *agent); err != nil {
			klog.Errorf("agent(%s) 同步 usage 失败 %v", agent.Name, err)
			continue
		}
		//klog.Infof("完成同步 agent(%s) 的 usage", agent.Name)
	}
}

func (s *AgentController) syncActionUsage(ctx context.Context, agent model.Agent) error {
	month := time.Now().Format("1")

	url := fmt.Sprintf("https://api.github.com/users/%s/settings/billing/usage?month=%s", agent.GithubUser, month)
	klog.Infof("当前 %s 月, 将通过请求 %s 获取本月账单", month, url)

	client := &http.Client{Timeout: 30 * time.Second}
	request, err := http.NewRequest("", url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", agent.GithubToken))
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error resp %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var ud UsageData
	if err = json.Unmarshal(data, &ud); err != nil {
		return err
	}

	var grossAmount float64 = 0
	for _, item := range ud.UsageItems {
		grossAmount += item.GrossAmount
	}

	rounded := math.Round(grossAmount*1000) / 1000
	klog.Infof("Agent(%s)当月截止目前已经使用 %v 美金", agent.Name, rounded)
	if agent.GrossAmount == rounded {
		klog.Infof("agent(%s) 的 grossAmount 未发生变化，等待下一次同步", agent.Name)
		return nil
	}

	return s.factory.Agent().UpdateByName(ctx, agent.Name, map[string]interface{}{"gross_amount": rounded})
}

type UsageData struct {
	UsageItems []UsageItem `json:"usageItems"`
}

type UsageItem struct {
	Date           time.Time `json:"date"`
	Product        string    `json:"product"`
	SKU            string    `json:"sku"`
	Quantity       float64   `json:"quantity"`
	UnitType       string    `json:"unitType"`
	PricePerUnit   float64   `json:"pricePerUnit"`
	GrossAmount    float64   `json:"grossAmount"`
	DiscountAmount float64   `json:"discountAmount"`
	NetAmount      float64   `json:"netAmount"`
	RepositoryName string    `json:"repositoryName"`
}

func (s *AgentController) startHeartbeat(ctx context.Context) {
	klog.Infof("Starting agent heartbeat")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		old, err := s.factory.Agent().GetByName(ctx, s.name)
		if err != nil {
			klog.Errorf("failed to get agent status %v", err)
			continue
		}

		updates := map[string]interface{}{"last_transition_time": time.Now()}
		if old.Status != model.UnRunAgentType {
			if old.Status == model.UnknownAgentType {
				updates["status"] = model.RunAgentType
				updates["message"] = "Agent started posting status"
			}
		}

		if err = s.factory.Agent().UpdateByName(ctx, s.name, updates); err != nil {
			klog.Errorf("同步 agent(%s) 心跳失败%v", s.name, err)
		} else {
			klog.V(2).Infof("同步 agent(%s) 心跳成功 %v", s.name, updates)
		}
	}
}

func (s *AgentController) getNextWorkItems(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 获取未处理
		tasks, err := s.factory.Task().ListWithAgent(ctx, s.name, 0)
		if err != nil {
			klog.Errorf("failed to list tasks %v", err)
			continue
		}
		if len(tasks) == 0 {
			continue
		}

		for _, task := range tasks {
			s.queue.Add(fmt.Sprintf("%d/%d", task.Id, task.ResourceVersion))
		}
	}
}

func (s *AgentController) worker(ctx context.Context) {
	for s.processNextWorkItem(ctx) {
	}
}

func (s *AgentController) processNextWorkItem(ctx context.Context) bool {
	key, quit := s.queue.Get()
	if quit {
		return false
	}
	defer s.queue.Done(key)

	klog.Infof("任务(%v)被调度到本节点，即将开始处理", key)
	taskId, resourceVersion, err := KeyFunc(key)
	if err != nil {
		s.handleErr(ctx, err, key)
	} else {
		_ = s.factory.Task().UpdateDirectly(ctx, taskId, map[string]interface{}{"status": "镜像初始化", "message": "初始化环境中", "process": 1})
		if err = s.factory.Task().CreateTaskMessage(ctx, &model.TaskMessage{TaskId: taskId, Message: "节点调度完成"}); err != nil {
			klog.Errorf("记录节点调度失败 %v", err)
		}
		if err = s.sync(ctx, taskId, resourceVersion); err != nil {
			if msgErr := s.factory.Task().CreateTaskMessage(ctx, &model.TaskMessage{TaskId: taskId, Message: fmt.Sprintf("同步失败，原因: %v", err)}); msgErr != nil {
				klog.Errorf("记录同步失败 %v", msgErr)
			}
			s.handleErr(ctx, err, key)
		}
	}
	return true
}

func (s *AgentController) GetOneAdminRegistry(ctx context.Context) (*model.Registry, error) {
	regs, err := s.factory.Registry().GetAdminRegistries(ctx)
	if err != nil {
		klog.Errorf("获取默认镜像仓库失败: %v", err)
		return nil, err
	}
	if len(regs) == 0 {
		klog.Errorf("no admin or default registry found")
		return nil, fmt.Errorf("no admin or default registry found")
	}

	// 随机分，暂时不考虑负载情况，后续优化
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(len(regs))
	t := regs[x]
	return &t, err
}

func (s *AgentController) makePluginConfig(ctx context.Context, task model.Task) (*rainbowconfig.PluginTemplateConfig, error) {
	taskId := task.Id

	var (
		registry *model.Registry
		err      error
	)
	// 未指定自定义参考时，使用默认仓库
	if task.RegisterId == 0 {
		registry, err = s.GetOneAdminRegistry(ctx)
	} else {
		registry, err = s.factory.Registry().Get(ctx, task.RegisterId)
	}
	if err != nil {
		klog.Errorf("failed to get registry %v", err)
		return nil, fmt.Errorf("failed to get registry %v", err)
	}

	pluginTemplateConfig := &rainbowconfig.PluginTemplateConfig{
		Default: rainbowconfig.DefaultOption{
			Time: time.Now().Unix(), // 注入时间戳，确保每次内容都不相同
		},
		Plugin: rainbowconfig.PluginOption{
			Callback:   s.callback,
			TaskId:     taskId,
			RegistryId: registry.Id,
			Synced:     true,
			Driver:     task.Driver,
			Arch:       task.Architecture,
		},
		Registry: rainbowconfig.Registry{
			Repository: registry.Repository,
			Namespace:  registry.Namespace,
			Username:   registry.Username,
			Password:   registry.Password,
		},
	}

	// 根据type判断是镜像列表推送还是k8s镜像组推送
	switch task.Type {
	case 0:
		tags, err := s.factory.Image().ListTags(ctx, db.WithTaskLike(taskId), db.WithErrorTask(task.OnlyPushError))
		if err != nil {
			klog.Errorf("获取任务所属 tags 失败 %v", err)
			return nil, err
		}

		var imageIds []int64
		for _, tag := range tags {
			imageIds = append(imageIds, tag.ImageId)
		}
		images, err := s.factory.Image().List(ctx, db.WithIDIn(imageIds...))
		if err != nil {
			klog.Errorf("获取任务所属镜像失败 %v", err)
			return nil, err
		}

		iNameMap := make(map[int64]string)
		for _, image := range images {
			iNameMap[image.Id] = image.Name
		}
		var img []rainbowconfig.Image
		for _, tag := range tags {
			name, ok := iNameMap[tag.ImageId]
			if !ok {
				klog.Warningf("未能找到镜像(%d)的名称，忽略", tag.ImageId)
				continue
			}
			img = append(img, rainbowconfig.Image{
				Name: name,
				Id:   tag.ImageId,
				Path: tag.Path,
				Tags: []string{tag.Name},
			})
		}

		pluginTemplateConfig.Default.PushImages = true
		pluginTemplateConfig.Images = img
	case 1:
		pluginTemplateConfig.Default.PushKubernetes = true
		pluginTemplateConfig.Kubernetes.Version = task.KubernetesVersion
	}

	return pluginTemplateConfig, err
}

func (s *AgentController) sync(ctx context.Context, taskId int64, resourceVersion int64) error {
	task, err := s.factory.Task().GetOne(ctx, taskId, resourceVersion)
	if err != nil {
		if errors.IsNotUpdated(err) {
			return nil
		}
		return fmt.Errorf("failted to get one task %d %v", taskId, err)
	}
	klog.Infof("开始处理任务(%s),任务ID(%d)", task.Name, taskId)

	tplCfg, err := s.makePluginConfig(ctx, *task)
	cfg, err := yaml.Marshal(tplCfg)
	if err != nil {
		return err
	}

	taskIdStr := fmt.Sprintf("%d", taskId)

	destDir := filepath.Join(s.baseDir, taskIdStr)
	if err = util.EnsureDirectoryExists(destDir); err != nil {
		return err
	}
	if !util.IsDirectoryExists(destDir + "/plugin") {
		if err = util.Copy(s.baseDir+"/plugin", destDir); err != nil {
			return err
		}
	}

	git := util.NewGit(destDir+"/plugin", taskIdStr, taskIdStr+"-"+time.Now().String())
	if err = git.Checkout(); err != nil {
		return err
	}
	if err = util.WriteIntoFile(string(cfg), destDir+"/plugin/config.yaml"); err != nil {
		return err
	}
	if err = git.Push(); err != nil {
		return err
	}
	return nil
}

// TODO
func (s *AgentController) handleErr(ctx context.Context, err error, key interface{}) {
	if err == nil {
		return
	}
	klog.Error(err)
}

func (s *AgentController) RegisterAgentIfNotExist(ctx context.Context) error {
	if len(s.name) == 0 {
		return fmt.Errorf("agent name missing")
	}

	var err error
	_, err = s.factory.Agent().GetByName(ctx, s.name)
	if err == nil {
		return nil
	}
	_, err = s.factory.Agent().Create(ctx, &model.Agent{Name: s.name, Status: model.RunAgentType, Type: model.PublicAgentType, Message: "Agent started posting status"})
	return err
}

func KeyFunc(key interface{}) (int64, int64, error) {
	str, ok := key.(string)
	if !ok {
		return 0, 0, fmt.Errorf("failed to convert %v to string", key)
	}
	parts := strings.Split(str, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("parts length not 2")
	}

	taskId, err := strutil.ParseInt64(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to Parse taskId to Int64 %v", err)
	}
	resourceVersion, err := strutil.ParseInt64(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to Parse resourceVersion to Int64 %v", err)
	}

	return taskId, resourceVersion, nil
}
