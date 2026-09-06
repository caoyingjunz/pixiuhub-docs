package rainbow

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/caoyingjunz/rainbow/pkg/util/sshutil"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	v2client "github.com/goharbor/go-client/pkg/sdk/v2.0/client"
	swr "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2"
	swrmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/model"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	rainbowconfig "github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util/huaweicloud"
)

type ServerGetter interface {
	Server() ServerInterface
}

type ServerInterface interface {
	CreateRegistry(ctx context.Context, req *types.CreateRegistryRequest) error
	UpdateRegistry(ctx context.Context, req *types.UpdateRegistryRequest) error
	DeleteRegistry(ctx context.Context, registryId int64) error
	GetRegistry(ctx context.Context, registryId int64) (interface{}, error)
	ListRegistries(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	LoginRegistry(ctx context.Context, req *types.CreateRegistryRequest) error

	// 账号密码登录，签发 JWT
	Login(ctx context.Context, req *types.LoginRequest) (*types.LoginResponse, error)

	// RBAC：当前用户可见资源（用户 → 角色 → 权限 → 资源）
	GetUserResources(ctx context.Context, userId string, role int) ([]model.Resource, error)

	// 角色管理
	CreateRole(ctx context.Context, req *types.CreateRoleRequest) (*model.Role, error)
	UpdateRole(ctx context.Context, req *types.UpdateRoleRequest) error
	DeleteRole(ctx context.Context, roleId int64) error
	ListRoles(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	// 用户-角色绑定
	AssignUserRole(ctx context.Context, userId string, roleId int64) error
	RemoveUserRole(ctx context.Context, userId string, roleId int64) error
	// 权限/资源管理
	ListPermissions(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	ListResources(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	CreateTask(ctx context.Context, req *types.CreateTaskRequest) error
	UpdateTask(ctx context.Context, req *types.UpdateTaskRequest) error
	ListTasks(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	DeleteTask(ctx context.Context, taskId int64) error
	GetTask(ctx context.Context, taskId int64) (interface{}, error)
	UpdateTaskStatus(ctx context.Context, req *types.UpdateTaskStatusRequest) error

	CreateTaskV2(ctx *gin.Context, req *types.CreateTaskRequest) error

	CreateSubscribe(ctx context.Context, req *types.CreateSubscribeRequest) error
	ListSubscribes(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	UpdateSubscribe(ctx context.Context, req *types.UpdateSubscribeRequest) error
	DeleteSubscribe(ctx context.Context, subId int64) error
	GetSubscribe(ctx context.Context, subId int64) (interface{}, error)

	ListSubscribeMessages(ctx context.Context, subId int64) (interface{}, error)

	RunSubscribe(ctx context.Context, req *types.RunSubscribeRequest) error

	ListTaskImages(ctx context.Context, taskId int64, listOption types.ListOptions) (interface{}, error)
	ReRunTask(ctx context.Context, req *types.UpdateTaskRequest) error

	ListTasksByIds(ctx context.Context, ids []int64) (interface{}, error)
	DeleteTasksByIds(ctx context.Context, ids []int64) error

	CreateAgent(ctx context.Context, req *types.CreateAgentRequest) error
	UpdateAgent(ctx context.Context, req *types.UpdateAgentRequest) error
	DeleteAgent(ctx context.Context, agentId int64) error
	GetAgent(ctx context.Context, agentId int64) (interface{}, error)
	ListAgents(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	UpdateAgentStatus(ctx context.Context, req *types.UpdateAgentStatusRequest) error

	CreateAgentRepo(ctx context.Context, req *types.CallGithubRequest) (interface{}, error)
	SyncAgentRepos(ctx context.Context, req *types.CallGithubRequest) error

	CreateImage(ctx context.Context, req *types.CreateImageRequest) error
	UpdateImage(ctx context.Context, req *types.UpdateImageRequest) error
	DeleteImage(ctx context.Context, imageId int64) error
	GetImage(ctx context.Context, imageId int64) (interface{}, error)
	ListImages(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	ListImagesForClient(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	ListImageTags(ctx context.Context, imageId int64, listOption types.ListOptions) (interface{}, error)

	ListImagesByIds(ctx context.Context, ids []int64) (interface{}, error)
	DeleteImagesByIds(ctx context.Context, ids []int64) error

	SearchImages(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	UpdateImageStatus(ctx context.Context, req *types.UpdateImageStatusRequest) error
	CreateImages(ctx context.Context, req *types.CreateImagesRequest) ([]model.Image, error)
	DeleteImageTag(ctx context.Context, imageId int64, TagId int64) error
	GetImageTag(ctx context.Context, imageId int64, tagId int64) (interface{}, error)

	BindImageLabels(ctx context.Context, imageId int64, req types.BindImageLabels) error
	ListImageLabels(ctx context.Context, imageId int64, listOption types.ListOptions) (interface{}, error)

	GetCollection(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	AddDailyReview(ctx context.Context, page string) error

	CreateLabel(ctx context.Context, req *types.CreateLabelRequest) error
	DeleteLabel(ctx context.Context, labelId int64) error
	UpdateLabel(ctx context.Context, req *types.UpdateLabelRequest) error
	ListLabels(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	ListLabelImages(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	CreateLogo(ctx context.Context, req *types.CreateLogoRequest) error
	UpdateLogo(ctx context.Context, req *types.UpdateLogoRequest) error
	DeleteLogo(ctx context.Context, logoId int64) error
	ListLogos(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	CreateNamespace(ctx context.Context, req *types.CreateNamespaceRequest) error
	UpdateNamespace(ctx context.Context, req *types.UpdateNamespaceRequest) error
	DeleteNamespace(ctx context.Context, objectId int64) error
	ListNamespaces(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	SyncNamespace(ctx context.Context, req *types.SyncNamespaceRequest) error

	Overview(ctx context.Context) (interface{}, error)
	Downflow(ctx context.Context) (interface{}, error)
	Store(ctx context.Context) (interface{}, error)
	ImageDownflow(ctx context.Context, downflowMeta types.DownflowMeta) (interface{}, error)

	SearchRepositories(ctx context.Context, req types.CallSearchRequest) (interface{}, error)
	SearchRepositoryTags(ctx context.Context, req types.CallSearchRequest) (interface{}, error)
	GetRepositoryTagInfo(ctx context.Context, req types.CallSearchRequest) (interface{}, error)

	CreateTaskMessage(ctx context.Context, req types.CreateTaskMessageRequest) error
	ListTaskMessages(ctx context.Context, taskId int64) (interface{}, error)

	ListArchitectures(ctx context.Context, listOption types.ListOptions) ([]string, error)

	CreateUser(ctx context.Context, req *types.CreateUserRequest) error
	UpdateUser(ctx context.Context, req *types.UpdateUserRequest) error
	ListUsers(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	GetUser(ctx context.Context, userId string) (*model.User, error)
	DeleteUser(ctx context.Context, userId string) error

	CreateOrUpdateUsers(ctx context.Context, req *types.CreateUsersRequest) error
	GetUserInfoByAccessKey(ctx *gin.Context, listOption types.ListOptions) (*model.User, error)

	CreateNotify(ctx context.Context, req *types.CreateNotificationRequest) error
	UpdateNotify(ctx context.Context, req *types.UpdateNotificationRequest) error
	GetNotify(ctx context.Context, notifyId int64) (interface{}, error)
	DeleteNotify(ctx context.Context, notifyId int64) error
	ListNotifies(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	SendNotify(ctx context.Context, req *types.SendNotificationRequest) error

	EnableNotify(ctx context.Context, req *types.UpdateNotificationRequest) error
	GetNotifyTypes(ctx context.Context) (interface{}, error)

	ListKubernetesVersions(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	SyncKubernetesTags(ctx context.Context, req *types.CallKubernetesTagRequest) (interface{}, error)

	ListRainbowds(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	Fix(ctx context.Context, req *types.FixRequest) (interface{}, error)

	// EnableChartRepo Chart Repo API
	EnableChartRepo(ctx context.Context, req *types.EnableChartRepoRequest) error
	ListCharts(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	DeleteChart(ctx context.Context, req types.ChartMetaRequest) error
	ListChartTags(ctx context.Context, req types.ChartMetaRequest, listOption types.ListOptions) (interface{}, error)
	GetChartTag(ctx context.Context, req types.ChartMetaRequest) (interface{}, error)
	DeleteChartTag(ctx context.Context, req types.ChartMetaRequest) error
	UploadChart(ctx *gin.Context, chartReq types.ChartMetaRequest) error
	DownloadChart(ctx *gin.Context, chartReq types.ChartMetaRequest) (string, string, error)

	GetChartStatus(ctx context.Context, req *types.ChartMetaRequest) (interface{}, error)

	GetToken(ctx context.Context, req *types.ChartMetaRequest) (interface{}, error)

	// CreateBuild 镜像构建 API
	CreateBuild(ctx context.Context, req *types.CreateBuildRequest) error
	DeleteBuild(ctx context.Context, buildId int64) error
	UpdateBuild(ctx context.Context, req *types.UpdateBuildRequest) error
	ListBuilds(ctx context.Context, listOption types.ListOptions) (interface{}, error)
	GetBuild(ctx context.Context, buildId int64) (interface{}, error)
	CreateBuildMessage(ctx context.Context, req types.CreateBuildMessageRequest) error
	ListBuildMessages(ctx context.Context, buildId int64) (interface{}, error)
	UpdateBuildStatus(ctx context.Context, req *types.UpdateBuildStatusRequest) error

	ListMetrics(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	CreateAccess(ctx context.Context, req *types.CreateAccessRequest) (*types.AccessResponse, error)
	DeleteAccess(ctx context.Context, ak string) error
	ListAccesses(ctx context.Context, listOption types.ListOptions) (interface{}, error)

	SearchRepo(ctx *gin.Context, listOption types.ListOptions) (interface{}, error)

	DownloadPixiuctl(ctx *gin.Context, version, filename string) (string, error)
	ListPixiuctls(ctx *gin.Context) ([]types.PixiuctlDownloadItem, error)

	Run(ctx context.Context, workers int) error
	Stop(ctx context.Context)
}

var (
	SwrClient  *swr.SwrClient
	RegistryId *int64
)

type ServerController struct {
	factory      db.ShareDaoFactory
	cfg          rainbowconfig.Config
	redisClient  *redis.Client
	Producer     rocketmq.Producer
	chartRepoAPI *v2client.HarborAPI
	sshConfigMap map[string]sshutil.SSHConfig

	lock sync.RWMutex
}

func NewServer(f db.ShareDaoFactory, cfg rainbowconfig.Config, redisClient *redis.Client, p rocketmq.Producer, cr *v2client.HarborAPI) *ServerController {
	// 初始化 rainbow 节点配置
	sshCfgMap := make(map[string]sshutil.SSHConfig)
	for _, sshConfig := range cfg.Rainbowd.Nodes {
		sshCfgMap[sshConfig.Name] = sshutil.SSHConfig{
			Host: sshConfig.Host,
			Port: sshConfig.Port,
		}
	}

	sc := &ServerController{
		factory:      f,
		cfg:          cfg,
		redisClient:  redisClient,
		Producer:     p,
		chartRepoAPI: cr,
		sshConfigMap: sshCfgMap,
	}

	if SwrClient == nil || RegistryId == nil {
		reg, err := f.Registry().GetDefaultRegistry(context.TODO())
		if err == nil {
			if len(reg.Ak) == 0 || len(reg.Sk) == 0 || len(reg.RegionId) == 0 {
				// 安全修复：不在日志中打印明文 AK/SK
				klog.Errorf("默认华为仓库未设置必要配置, regionId(%s)", reg.RegionId)
			} else {
				client, err := huaweicloud.NewHuaweiCloudClient(huaweicloud.HuaweiCloudConfig{
					AK:       reg.Ak,
					SK:       reg.Sk,
					RegionId: reg.RegionId,
				})
				if err == nil {
					SwrClient = client
					RegistryId = &reg.Id
					klog.Infof("创建华为仓库客户端成功，仓库名称: %s(%d) ", reg.Name, *RegistryId)
				} else {
					klog.Errorf("创建为仓库客户端失败 %v", err)
				}
			}
		} else {
			klog.Errorf("获取默认华为仓库失败: %v", err)
		}
	}

	return sc
}

func (s *ServerController) RegisterRainbowd(ctx context.Context) error {
	if len(s.cfg.Rainbowd.Nodes) == 0 {
		klog.Warningf("rainbowd not found")
		return nil
	}

	for _, node := range s.cfg.Rainbowd.Nodes {
		klog.Infof("Rainbowd(%s)即将被处理", node.Name)
		var err error
		_, err = s.factory.Rainbowd().GetByName(ctx, node.Name)
		if err == nil {
			klog.Infof("Rainbowd(%s)已存在", node.Name)
			continue
		}
		_, err = s.factory.Rainbowd().Create(ctx, &model.Rainbowd{
			Name:   node.Name,
			Host:   node.Host,
			Status: model.RunAgentType,
		})
		if err != nil {
			klog.Errorf("Rainbowd(%s)创建失败 %s", node.Name, err)
			return err
		}
		klog.Infof("Rainbowd(%s)注册完成", node.Name)
	}

	return nil
}

func (s *ServerController) Run(ctx context.Context, workers int) error {
	go s.schedule(ctx)
	go s.sync(ctx)
	go s.startSyncDailyPulls(ctx)
	go s.startSyncMetrics(ctx)
	go s.startAgentHeartbeat(ctx)
	go s.startSyncKubernetesTags(ctx)
	go s.startSubscribeController(ctx)

	//klog.Infof("starting rocketmq producer")
	//if err := s.Producer.Start(); err != nil {
	//	return err
	//}
	// 初始化 agent 属性
	klog.Infof("starting register rainbowd")
	if err := s.RegisterRainbowd(ctx); err != nil {
		return err
	}
	// 启动 rainbow 检查进程
	//go s.startRainbowdHeartbeat(ctx)

	return nil
}

func (s *ServerController) startRainbowdHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		klog.V(1).Infof("即将进行 rainbowd 的状态检查")
		for nodeName, sshConfig := range s.sshConfigMap {
			_, err := s.factory.Rainbowd().GetByName(ctx, nodeName)
			if err != nil {
				klog.Errorf("获取 rainbowd %s 失败 %v ", nodeName, err)
				klog.Errorf("等待 %s 下一次检查", nodeName)
				continue
			}

			status := model.RunAgentType
			if !s.IsRainbowReachable(&sshConfig, nodeName) {
				status = model.UnRunAgentType
			}
			klog.V(1).Infof("rainbowd %s 的状态为 %s", nodeName, status)

			//updates := map[string]interface{}{"last_transition_time": time.Now()}
			//if old.Status != status {
			//	updates["status"] = status
			//	klog.Infof("rainbowd 的状态发生改变，即将同步")
			//} else {
			//	klog.V(1).Infof("rainbowd(%s)的状态未发生变化，等待下一次更新", nodeName)
			//}
			//if err = s.factory.Rainbowd().Update(ctx, old.Id, updates); err != nil {
			//	klog.Errorf("同步 rainbowd(%s) 状态失败 %v 等待下一次同步", nodeName, err)
			//}
		}
	}
}

func (s *ServerController) IsRainbowReachable(sshConfig *sshutil.SSHConfig, nodeName string) bool {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		klog.Errorf("创建(%s) ssh client 失败 %v", nodeName, err)
		return false
	}
	defer sshClient.Close()

	if err = sshClient.Ping(); err != nil {
		klog.Infof("检查节点 %s 连通性失败 %v", nodeName, err)
		return false
	}
	return true
}

func (s *ServerController) Stop(ctx context.Context) {
	//klog.Infof("rocketmq producer 停止服务!!!")
	//_ = s.Producer.Shutdown()
}

func (s *ServerController) startSubscribeController(ctx context.Context) {
	klog.Infof("starting subscribe controller")

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		subscribes, err := s.factory.Task().ListSubscribes(ctx, db.WithEnable(1), db.WithFailTimes(6))
		if err != nil {
			klog.Errorf("获取全部订阅失败 %v 15分钟后重新执行订阅", err)
			continue
		}

		for _, sub := range subscribes {
			if sub.FailTimes > 5 {
				klog.Warningf("订阅 (%s) 失败超过限制，已终止订阅", sub.Path)
				s.DisableSubscribeWithMessage(ctx, sub, fmt.Sprintf("订阅(%s)失败已超过限制，终止订阅", sub.Path))
				continue
			}
			now := time.Now()
			if now.Sub(sub.LastNotifyTime) < sub.Interval*time.Second {
				klog.V(1).Infof("订阅 (%s) 时间间隔 %v 暂时无需执行", sub.Path, sub.Interval*time.Second)
				continue
			}

			if err = s.RunSubscribe(ctx, &types.RunSubscribeRequest{SubscribeId: sub.Id}); err != nil {
				klog.Errorf("failed to do Subscribe(%s) %v", sub.Path, err)
				s.CreateSubscribeMessageAndFailTimesAdd(ctx, sub, err.Error())
			} else {
				s.CreateSubscribeMessageWithLog(ctx, sub, fmt.Sprintf("%s 在 %v 订阅触发成功", sub.Path, time.Now().Format("2006-01-02 15:04:05")))
			}

			// 仅保留最新的 n 个事件
			_ = s.cleanSubscribeMessages(ctx, sub.Id, 5)
		}
	}
}

func (s *ServerController) startSyncKubernetesTags(ctx context.Context) {
	klog.Infof("starting kubernetes tags syncer")
	ticker := time.NewTicker(3600 * 6 * time.Second)
	defer ticker.Stop()

	opt := types.CallKubernetesTagRequest{SyncAll: false}
	for range ticker.C {
		if _, err := s.SyncKubernetesTags(ctx, &opt); err != nil {
			klog.Errorf("failed kubernetes version syncer %v", err)
		}
	}
}

func (s *ServerController) startSyncDailyPulls(ctx context.Context) {
	c := cron.New()
	_, err := c.AddFunc("0 1 * * *", func() {
		klog.Infof("执行每天凌晨 1 点任务...")
		s.syncPulls(ctx)
	})
	if err != nil {
		klog.Fatal("SyncDailyPulls 定时任务配置错误:", err)
	}
	c.Start()
	klog.Infof("starting pull images syncer")

	// 优雅关闭（可选）
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	c.Stop()
	klog.Infof("定时任务(pull images)已停止")
}

func (s *ServerController) startSyncMetrics(ctx context.Context) {
	c := cron.New()
	_, err := c.AddFunc("0 6 * * *", func() {
		klog.Infof("执行每天凌晨 6 点任务...")
		s.syncMetrics(ctx)
	})
	if err != nil {
		klog.Fatal("Sync metrics 定时任务配置错误:", err)
	}
	c.Start()
	klog.Infof("starting daily metrics syncer")

	// 优雅关闭（可选）
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	c.Stop()
	klog.Infof("定时任务(daily metrics)已停止")
}

func (s *ServerController) syncPulls(ctx context.Context) {
	now := time.Now()

	klog.Infof("全量执行 pull 同步")
	images, err := s.factory.Image().List(ctx, db.WithLastSyncBefore(now.Add(-30*time.Minute)))
	if err != nil {
		klog.Errorf("获取镜像列表失败 %v", err)
		return
	}
	s.AfterListImages(ctx, images, time.Second*1)
}

func (s *ServerController) schedule(ctx context.Context) {
	klog.Infof("starting scheduler controller")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.doSchedule(ctx); err != nil {
			klog.Errorf("failed to do schedule %v", err)
		}
	}
}

func (s *ServerController) doSchedule(ctx context.Context) error {
	item, err := s.factory.Task().GetOneForSchedule(ctx)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	klog.Infof("获取待处理任务 %v", item)

	targetAgent, err := s.assignAgent(ctx)
	if err != nil {
		return err
	}
	if targetAgent == "" {
		return nil
	}
	if err = s.factory.Task().Update(ctx, item.Id, item.ResourceVersion, map[string]interface{}{
		"agent_name": targetAgent,
	}); err != nil {
		return err
	}
	klog.Infof("任务 %s 已被分配给 agent %s，等待处理中", item.Name, targetAgent)

	return nil
}

func (s *ServerController) sync(ctx context.Context) {
	if SwrClient == nil {
		klog.Infof("未设置默认远程仓库，无需镜像同步")
		return
	}

	klog.Infof("starting remote image sync controller")
	ticker := time.NewTicker(300 * time.Second)
	defer ticker.Stop()

	defaultNamespace := HuaweiNamespace
	for range ticker.C {
		//overview, err := SwrClient.ShowDomainOverview(&swrmodel.ShowDomainOverviewRequest{})
		//if err != nil {
		//	klog.Errorf("获取远程仓库概览失败", err)
		//	continue
		//}
		//klog.Infof("获取远程仓库概览成功 %v", overview)

		// TODO: 后续分页查询
		resp, err := SwrClient.ListReposDetails(&swrmodel.ListReposDetailsRequest{Namespace: &defaultNamespace})
		if err != nil {
			klog.Errorf("获取远程镜像列表失败 %v", err)
			continue
		}
		if resp.Body == nil || len(*resp.Body) == 0 {
			klog.Infof("获取远程镜像为空")
			return
		}

		var imageNames []string
		imageMap := make(map[string]int64)
		for _, reRepo := range *resp.Body {
			imageNames = append(imageNames, reRepo.Name)
			imageMap[reRepo.Name] = reRepo.NumDownload
		}

		targetImages, err := s.factory.Image().List(ctx, db.WithNameIn(imageNames...))
		if err != nil {
			klog.Errorf("查询本地镜像列表失败 %v", err)
			continue
		}
		for _, targetImage := range targetImages {
			pull := imageMap[targetImage.Name]
			if targetImage.Pull == pull {
				klog.V(1).Infof("镜像(%s)下载量未发生变量，无需更新", targetImage.Name)
				continue
			}

			klog.Infof("镜像(%s)下载量已发生变量，延迟更新", targetImage.Name)
			err = s.factory.Image().Update(ctx, targetImage.Id, targetImage.ResourceVersion, map[string]interface{}{"pull": pull})
			if err != nil {
				klog.Errorf("更新镜像(%s)的下载量失败 %v", targetImage.Name, err)
			}
		}
	}
}

func (s *ServerController) assignAgent(ctx context.Context) (string, error) {
	agents, err := s.factory.Agent().ListForSchedule(ctx)
	if err != nil {
		return "", err
	}
	if len(agents) == 0 {
		klog.Warningf("不存在可用工作节点，等待下一次调度")
		return "", nil
	}

	var agentNames []string
	agentMap := make(map[string]int)
	for _, agent := range agents {
		agentNames = append(agentNames, agent.Name)
		agentMap[agent.Name] = 0
	}
	agentSet := sets.NewString(agentNames...)

	runningTasks, err := s.factory.Task().GetRunningTask(ctx)
	if err != nil {
		return "", err
	}

	if len(runningTasks) == 0 {
		rand.Seed(time.Now().UnixNano())
		x := rand.Intn(len(agentNames))
		agent := agentNames[x]
		klog.Infof("当前节点均空闲，工作节点 %s 被随机选中", agent)
		return agent, nil
	} else {
		for _, t := range runningTasks {
			if !agentSet.Has(t.AgentName) {
				continue
			}
			old, ok := agentMap[t.AgentName]
			if ok {
				agentMap[t.AgentName] = old + 1
			} else {
				continue
			}
		}

		min := len(runningTasks)
		agent := ""
		for k, v := range agentMap {
			if min >= v {
				min = v
				agent = k
			}
		}
		// 一个 agent 最大并发为 10
		if min > 10 {
			klog.Warningf("工作节点已满负载，等待下一次调度")
			return "", nil
		}
		if agent == "" {
			klog.Warningf("未选中工作节点，等待下一次调度")
			return "", nil
		}

		klog.Infof("工作节点 %s 已选中", agent)
		return agent, nil
	}
}

func (s *ServerController) startAgentHeartbeat(ctx context.Context) {
	klog.Infof("starting agent heartbeat")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		agents, err := s.factory.Agent().List(ctx)
		if err != nil {
			klog.Warningf("获取 agents 列表失败，等待下一次重试 %v", err)
			continue
		}

		for _, agent := range agents {
			// 如果状态已处于非运行状态，则直接忽略检查
			if agent.Status != model.RunAgentType {
				klog.V(1).Infof("agent(%s)非在线状态，忽略", agent.Name)
				continue
			}

			diff := time.Now().Sub(agent.LastTransitionTime)
			if diff > time.Minute*5 {
				// 如果已是未知状态，则无需更新
				if agent.Status == model.UnknownAgentType {
					continue
				}
				err = s.factory.Agent().UpdateByName(ctx, agent.Name, map[string]interface{}{"status": model.UnknownAgentType, "message": "Agent stopped posting status"})
				if err != nil {
					klog.Errorf("failed to sync agent %s status %v", agent.Name, err)
				} else {
					klog.Infof("agent(%s)被设置成未知", agent.Name)
				}
			}
		}
	}
}

func (s *ServerController) ListArchitectures(ctx context.Context, listOption types.ListOptions) ([]string, error) {
	return []string{
		"linux/amd64",
		"linux/arm64",
		"linux/arm",
		"windows/x86",
		"windows/x86-64",
		"自定义",
	}, nil
}
