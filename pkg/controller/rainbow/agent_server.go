package rainbow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	guuid "github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"

	rainbowconfig "github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
	"github.com/caoyingjunz/rainbow/pkg/util/sshutil"
	"github.com/caoyingjunz/rainbow/pkg/util/uuid"
)

const GitConfig = `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
[remote "origin"]
	url = {{ .URL }}
	fetch = +refs/heads/*:refs/remotes/origin/*
`

type ContainerInfo struct {
	ID           string `json:"ID"`
	Names        string `json:"Names"`
	Image        string `json:"Image"`
	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	Status       string `json:"Status"`
	Ports        string `json:"Ports"`
	Size         string `json:"Size"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Networks     string `json:"Networks"`
}

func (s *ServerController) GetAgent(ctx context.Context, agentId int64) (interface{}, error) {
	return s.factory.Agent().Get(ctx, agentId)
}

func (s *ServerController) ReconcileAgent(ctx context.Context, sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	agentName := agent.Name
	old, err := s.GetAgentContainer(sshConfig, agentName)
	if err != nil {
		klog.Errorf("获取 agent 容器失败 %v", err)
		return err
	}
	if old == nil {
		klog.V(1).Infof("agent 容器(%s)不存在", agentName)
	}

	klog.Infof("agent(%s)正在%s", agentName, agent.Status)

	destStatus := model.RunAgentType
	switch agent.Status {
	case model.RestartingAgentType:
		err = s.RestartAgentContainer(sshConfig, agent)
	case model.StartingAgentType:
		err = s.StartAgentContainer(sshConfig, agent)
	case model.StoppingAgentType:
		err = s.StopAgentContainer(sshConfig, agent)
	case model.UpgradeAgentType, model.RunAgentType:
		if old != nil {
			// 先卸载原有容器，然后刷新配置，重新启动
			if err1 := s.UninstallAgentContainer(sshConfig, agent); err1 != nil {
				return fmt.Errorf("卸载 agent(%s) 失败 %v", agentName, err1)
			}
		}
		err = s.InstallAgentContainer(sshConfig, agent)
	case model.OfflineAgentType, model.DeletingAgentType:
		err = s.UninstallAgentContainer(sshConfig, agent)
		destStatus = model.UnRunAgentType
	case model.UpgradeAgentBinaryType:
		err = s.UpgradeAgentBinaryContainer(sshConfig, agent)
	default:
		klog.Infof("未命中 agent(%s) 状态(%s) 等待下次协同", agent.Name, agent.Status)
		return nil
	}
	if err != nil {
		klog.Errorf("agent(%s) 执行失败 %v", agent.Name, err)
		return err
	}

	if err = s.factory.Agent().Update(context.TODO(), agent.Id, agent.ResourceVersion, map[string]interface{}{"status": destStatus}); err != nil {
		return err
	}
	klog.Infof("agent(%s) 执行完成", agent.Name)
	return nil
}

func (s *ServerController) IsAgentRunningStatus(status string) bool {
	runningStatus := model.GetAgentRunningStatus()
	for _, rs := range runningStatus {
		if rs == status {
			return true
		}
	}

	return false
}

func (s *ServerController) ResetAgentMetadata(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	containerName := agent.Name
	destDir := filepath.Join(s.cfg.Rainbowd.DataDir, containerName)

	result, err := sshClient.RunCommand(fmt.Sprintf("mkdir -p %s", destDir))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("远程命令执行失败: %s", result.Stderr)
	}

	// 配置文件 config.yaml
	data, err := util.ReadFromFile(s.cfg.Rainbowd.TemplateDir + "/config.yaml")
	if err != nil {
		return err
	}
	var cfg rainbowconfig.Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	// 追加差异化配置
	cfg.Agent.Name = containerName
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err = util.WriteIntoFile(string(cfgData), s.cfg.Rainbowd.TemplateDir+fmt.Sprintf("/%s-config.yaml", containerName)); err != nil {
		return err
	}
	// 拷贝配置文件
	if err = sshClient.UploadFile(s.cfg.Rainbowd.TemplateDir+fmt.Sprintf("/%s-config.yaml", containerName), destDir+"/config.yaml", "0644"); err != nil {
		klog.Errorf("传输 yaml 配置文件失败 %v", err)
		return err
	}
	// 拷贝 agent 每次都重置最新
	if err = sshClient.UploadFile(s.cfg.Rainbowd.TemplateDir+"/agent", destDir+"/agent", "0755"); err != nil {
		klog.Errorf("传输 agent 二进制文件失败 %v", err)
		return err
	}
	// 拷贝 plugin 项目
	if err = sshClient.UploadDir(s.cfg.Rainbowd.TemplateDir+"/plugin", destDir+"/plugin", "root:root"); err != nil {
		klog.Errorf("传输 plugin 文件失败 %v", err)
		return err
	}

	return nil
}

func (s *ServerController) UpdateAgentStatus(ctx context.Context, req *types.UpdateAgentStatusRequest) error {
	if req.Status == "强制在线" || req.Status == "强制离线" {
		realStatus := strings.Replace(req.Status, "强制", "", -1)
		return s.factory.Agent().UpdateByName(ctx, req.AgentName, map[string]interface{}{"status": realStatus, "message": fmt.Sprintf("Agent has been set to %s", realStatus)})
	}
	if req.Status == "强制删除" {
		return s.factory.Agent().DeleteBy(ctx, db.WithName(req.AgentName))
	}

	old, err := s.factory.Agent().GetByName(ctx, req.AgentName)
	if err != nil {
		return err
	}
	// 如果是进行中状态，先返回等待
	if s.IsAgentRunningStatus(old.Status) {
		return fmt.Errorf("agent(%s)状态为(%s), 请稍后再试", req.AgentName, old.Status)
	}

	sshConfig, ok := s.sshConfigMap[old.RainbowdName]
	if !ok {
		return fmt.Errorf("未加载 rainbow node(%s)", old.RainbowdName)
	}

	if err := s.factory.Agent().UpdateByName(ctx, req.AgentName, map[string]interface{}{"status": req.Status, "message": fmt.Sprintf("Agent has been set to %s", req.Status)}); err != nil {
		return err
	}
	newAgent, err := s.factory.Agent().GetByName(ctx, req.AgentName)
	if err != nil {
		klog.Errorf("获取最新的agent失败 %v", err)
		return err
	}

	go func() {
		if err = s.ReconcileAgent(ctx, &sshConfig, newAgent); err != nil {
			klog.Errorf("远程更新agent失败 %v", err)
		}
	}()
	return nil
}

func (s *ServerController) UpdateAgent(ctx context.Context, req *types.UpdateAgentRequest) error {
	repo := req.GithubRepository
	if len(repo) == 0 {
		repo = fmt.Sprintf("https://github.com/%s/plugin.git", req.GithubUser)
	}

	updates := make(map[string]interface{})
	updates["github_user"] = req.GithubUser
	updates["github_repository"] = repo
	updates["github_token"] = req.GithubToken
	updates["github_email"] = req.GithubEmail
	updates["rainbowd_name"] = req.RainbowdName
	return s.factory.Agent().UpdateByName(ctx, req.AgentName, updates)
}

func (s *ServerController) ListAgents(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	// 初始化分页属性
	listOption.SetDefaultPageOption()

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}
	opts := []db.Options{
		db.WithNameLike(listOption.NameSelector),
		db.WithStatus(listOption.AgentStatus),
	}

	var err error
	pageResult.Total, err = s.factory.Agent().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取agent总数失败 %v", err)
		pageResult.Message = err.Error()
	}
	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithModifyOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)
	pageResult.Items, err = s.factory.Agent().List(ctx, opts...)
	if err != nil {
		klog.Errorf("获取agent列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	return pageResult, nil
}

func (s *ServerController) GetAgentContainer(sshConfig *sshutil.SSHConfig, containerName string) (*ContainerInfo, error) {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()

	result, err := sshClient.RunCommand(fmt.Sprintf("docker ps -a --filter name=%s --format json", containerName))
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("docker 命令执行失败: %s", result.Stderr)
	}

	containers, err := s.parseDockerJSONOutput(result.Stdout)
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.Names == containerName {
			return &container, nil
		}
	}

	return nil, nil
}

func (s *ServerController) InstallAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	if err := s.ResetAgentMetadata(sshConfig, agent); err != nil {
		return err
	}
	if err := s.BootstrapAgentContainer(sshConfig, agent); err != nil {
		return err
	}

	return s.RunAgentContainer(sshConfig, agent)
}

func (s *ServerController) RunAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	cmd1 := []string{"docker", "run", "-d", "--name", agent.Name,
		"-v", fmt.Sprintf("%s:/data", s.cfg.Rainbowd.DataDir+"/"+agent.Name),
		"-v", "/etc/localtime:/etc/localtime:ro",
		"--network", "host", s.cfg.Rainbowd.AgentImage, "/data/agent", "--configFile", "/data/config.yaml"}
	// 输入 github 的配置
	cmd2 := []string{"docker", "exec", agent.Name, "git", "config", "--global", "user.name", agent.GithubUser}
	cmd3 := []string{"docker", "exec", agent.Name, "git", "config", "--global", "user.email", agent.GithubEmail}

	_, err = sshClient.RunCommands([]string{
		strings.Join(cmd1, " "),
		strings.Join(cmd2, " "),
		strings.Join(cmd3, " "),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *ServerController) BootstrapAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	containerName := agent.Name + uuid.NewRandName("-upgrade-", 8)
	pluginDir := "/data/plugin/"

	cmd1 := []string{"docker", "run", "-d", "--name", containerName, "-v", fmt.Sprintf("%s:/data", s.cfg.Rainbowd.DataDir+"/"+agent.Name), "-v", "/etc/localtime:/etc/localtime:ro", "--network", "host", s.cfg.Rainbowd.AgentImage, "sleep", "infinity"}
	cmd2 := []string{"docker", "exec", containerName, "git", "init", pluginDir}
	cmd3 := []string{"docker", "exec", containerName, "git", "config", "--global", "user.name", agent.GithubUser}
	cmd4 := []string{"docker", "exec", containerName, "git", "config", "--global", "user.email", agent.GithubEmail}

	_, err = sshClient.RunCommands([]string{
		strings.Join(cmd1, " "),
		strings.Join(cmd2, " "),
		strings.Join(cmd3, " "),
		strings.Join(cmd4, " "),
	})
	if err != nil {
		return err
	}
	klog.Infof("agent 初始环境准备完成")

	gc := struct{ URL string }{URL: fmt.Sprintf("https://%s:%s@github.com/%s/plugin.git", agent.GithubUser, agent.GithubToken, agent.GithubUser)}
	tpl := template.New(containerName)
	t := template.Must(tpl.Parse(GitConfig))
	var buf bytes.Buffer
	if err = t.Execute(&buf, gc); err != nil {
		return err
	}
	destDir := filepath.Join(s.cfg.Rainbowd.DataDir, agent.Name)
	if err = ioutil.WriteFile(s.cfg.Rainbowd.TemplateDir+fmt.Sprintf("/%s-git-config", containerName), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("生成 git config 文件失败 %v", err)
	}
	klog.V(0).Infof("传输文件 %s 至 %s", s.cfg.Rainbowd.TemplateDir+fmt.Sprintf("/%s-git-config", containerName), destDir+"/plugin/.git/config")
	if err = sshClient.UploadFile(s.cfg.Rainbowd.TemplateDir+fmt.Sprintf("/%s-git-config", containerName), destDir+"/plugin/.git/config", "0755"); err != nil {
		return fmt.Errorf("传输 /plugin/.git/config 失败 %v", err)
	}
	klog.Infof("元数据传输完成")

	cmd5 := []string{"docker", "exec", "-w", pluginDir, containerName, "git", "add", "."}
	cmd6 := []string{"docker", "exec", "-w", pluginDir, containerName, "git", "commit", "-m", "init"}
	cmd7 := []string{"docker", "exec", "-w", pluginDir, containerName, "git", "push", "--set-upstream", "origin", "master", "--force"}
	cmd8 := []string{"docker", "rm", containerName, "-f"} // 初始化完成之后，清理中间态容器
	_, err = sshClient.RunCommands([]string{
		strings.Join(cmd5, " "),
		strings.Join(cmd6, " "),
		strings.Join(cmd7, " "),
		strings.Join(cmd8, " "),
	})
	if err != nil {
		return err
	}

	klog.Infof("agent 初始化完成")
	return nil
}

// UpgradeAgentBinaryContainer 先停止 agent 容器，然后替换 agent 二进制文件 然后启动 agent 容器
func (s *ServerController) UpgradeAgentBinaryContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	klog.Infof("UpgradeAgentBinaryContainer rainbow %s agent %s", agent.RainbowdName, agent.Name)
	if err := s.StopAgentContainer(sshConfig, agent); err != nil {
		return err
	}

	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	containerName := agent.Name
	destDir := filepath.Join(s.cfg.Rainbowd.DataDir, containerName)
	// 替换 agent 二进制
	if err = sshClient.UploadFile(s.cfg.Rainbowd.TemplateDir+"/agent", destDir+"/agent", "0755"); err != nil {
		klog.Errorf("传输 agent 二进制文件失败 %v", err)
		return err
	}

	return s.StartAgentContainer(sshConfig, agent)
}

func (s *ServerController) UninstallAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	containerName := agent.Name
	destDir := filepath.Join(s.cfg.Rainbowd.DataDir, containerName)
	klog.V(1).Infof("agent(%s) 工作目录(%s) 正在被回收", containerName, destDir)
	cmds := []string{
		fmt.Sprintf("docker rm %s -f", containerName),
		fmt.Sprintf("rm -rf %s", destDir),
	}
	results, err := sshClient.RunCommands(cmds)
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.ExitCode != 0 {
			return fmt.Errorf("远程命令执行失败: %s", result.Stderr)
		}
	}

	return nil
}

func (s *ServerController) StartAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	return s.execAgentContainerByMethod(sshConfig, agent, "start")
}

func (s *ServerController) RemoveAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	return s.execAgentContainerByMethod(sshConfig, agent, "rm", "-f")
}

func (s *ServerController) StopAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	return s.execAgentContainerByMethod(sshConfig, agent, "stop")
}

func (s *ServerController) RestartAgentContainer(sshConfig *sshutil.SSHConfig, agent *model.Agent) error {
	return s.execAgentContainerByMethod(sshConfig, agent, "restart")
}

func (s *ServerController) execAgentContainerByMethod(sshConfig *sshutil.SSHConfig, agent *model.Agent, method string, options ...string) error {
	sshClient, err := sshutil.NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	containerName := agent.Name
	cmd := fmt.Sprintf("docker %s %s", method, containerName)
	if len(options) != 0 {
		cmd = fmt.Sprintf("%s %s", cmd, strings.Join(options, " "))
	}
	klog.V(1).Infof("cmd %s 即将被执行", cmd)
	result, err := sshClient.RunCommand(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("docker 命令执行失败: %s", result.Stderr)
	}

	return nil
}

// parseDockerJSONOutput 解析Docker JSON格式输出
func (s *ServerController) parseDockerJSONOutput(output string) ([]ContainerInfo, error) {
	if strings.TrimSpace(output) == "" {
		return []ContainerInfo{}, nil
	}

	// Docker输出的JSON是每行一个JSON对象，不是JSON数组
	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var container ContainerInfo
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			return nil, fmt.Errorf("解析JSON失败: %w, 原始数据: %s", err, line)
		}

		containers = append(containers, container)
	}

	return containers, nil
}

func (s *ServerController) CreateAgentRepo(ctx context.Context, req *types.CallGithubRequest) (interface{}, error) {
	key := guuid.NewString()
	req.Op = types.OpCreateAction
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallGithubType,
		Uid:               key,
		CallGithubRequest: req,
	})

	_, err = s.Call(ctx, req.ClientId, key, data)
	if err != nil {
		klog.Errorf("创建 agent github repo（%s）失败：%v", req.Repo, err)
		return nil, err
	}
	klog.Infof("创建 agent github repo（%s）成功", req.Repo)
	return nil, nil
}

func (s *ServerController) CreateAgentReposIfNot(ctx context.Context, req *types.CallGithubRequest) error {
	key := guuid.NewString()
	req.Op = types.OpCreateIfNotAction
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallGithubType,
		Uid:               key,
		CallGithubRequest: req,
	})

	_, err = s.Call(ctx, req.ClientId, key, data)
	if err != nil {
		klog.Errorf("创建 agentRepos（%s）失败：%v", req.Repo, err)
		return err
	}
	klog.Infof("创建 agentRepo（%s）成功", req.Repo)
	return nil
}

func (s *ServerController) SyncAgentRepos(ctx context.Context, req *types.CallGithubRequest) error {
	agents, err := s.factory.Agent().List(ctx, db.WithStatus(model.RunAgentType))
	if err != nil {
		klog.Errorf("获取运行中 agent 列表失败 %v", err)
		return err
	}

	for _, agent := range agents {
		err = s.CreateAgentReposIfNot(ctx, &types.CallGithubRequest{
			ClientId: agent.Name,
			Repos:    req.Repos,
		})
		if err != nil {
			klog.Errorf("同步 agent(%s) repo 失败 %v", agent.Name, err)
		}
	}

	return nil
}
