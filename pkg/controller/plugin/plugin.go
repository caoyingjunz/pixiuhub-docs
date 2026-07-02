package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caoyingjunz/pixiulib/exec"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	rainbowtypes "github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
	dockerutil "github.com/caoyingjunz/rainbow/pkg/util/docker"
)

const (
	Kubeadm   = "kubeadm"
	IgnoreKey = "W0508"

	SkopeoDriver = "skopeo"
	DockerDriver = "docker"

	MaxConcurrency = 5
)

type KubeadmVersion struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"clientVersion"`
}

type KubeadmImage struct {
	Images []string `json:"images"`
}

type PluginController struct {
	KubernetesVersion string
	Callback          string
	RegistryId        int64
	TaskId            int64
	Synced            bool

	httpClient util.HttpInterface
	exec       exec.Interface
	docker     *client.Client

	Cfg      config.Config
	Registry config.Registry
	Images   []config.Image

	Runners []Runner
}

type Runner interface {
	GetName() string
	Run() error
}

type login struct {
	name string
	p    *PluginController
}

func (l *login) GetName() string {
	return l.name
}

func (l *login) Run() error {
	cmd := []string{"docker", "login", "-u", l.p.Registry.Username, "-p", l.p.Registry.Password}
	if l.p.Registry.Repository != "" {
		cmd = append(cmd, l.p.Registry.Repository)
	}
	out, err := l.p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to login in image %v %v", string(out), err)
	}
	klog.Infof("镜像仓库登录完成")
	return nil
}

type image struct {
	name string
	p    *PluginController
}

func (i *image) GetName() string {
	return i.name
}

func (i *image) Run() error {
	var images []string
	if i.p.Cfg.Default.PushKubernetes {
		kubeImages, err := i.p.getImages()
		if err != nil {
			klog.Errorf("获取 k8s 镜像失败: %v", err)
			return fmt.Errorf("获取 k8s 镜像失败: %v", err)
		}
		is, err := i.p.CreateImages(kubeImages)
		if err != nil {
			klog.Errorf("回调API创建 kubernetes 镜像失败: %v", err)
			return err
		}

		var tplImages []config.Image
		for _, img := range is {
			for _, tag := range img.Tags {
				tplImages = append(tplImages, config.Image{
					Name: img.Name,
					Path: tag.Path,
					Tags: []string{tag.Name},
					Id:   tag.ImageId,
				})
			}
		}
		klog.Infof("已完成 kubernetes 镜像的回调创建，镜像模板为 %v", tplImages)
		i.p.Images = tplImages
	}

	if i.p.Cfg.Default.PushImages {
		fileImages, err := i.p.getImagesFromFile()
		if err != nil {
			return fmt.Errorf("")
		}
		images = append(images, fileImages...)
	}

	return nil
}

func NewPluginController(cfg config.Config) *PluginController {
	return &PluginController{
		Cfg:        cfg,
		Callback:   cfg.Plugin.Callback,
		TaskId:     cfg.Plugin.TaskId,
		RegistryId: cfg.Plugin.RegistryId,
		Synced:     cfg.Plugin.Synced,
		Images:     cfg.Images,
		httpClient: util.NewHttpClient(5*time.Second, cfg.Plugin.Callback),
	}
}

func (p *PluginController) Validate() error {
	if p.Cfg.Default.PushKubernetes {
		if len(p.KubernetesVersion) == 0 {
			return fmt.Errorf("failed to find kubernetes version")
		}
		// 检查 kubeadm 的版本是否和 k8s 版本一致
		kubeadmVersion, err := p.getKubeadmVersion()
		if err != nil {
			klog.Error("failed to get kubeadm version: %v", err)
			return fmt.Errorf("failed to get kubeadm version: %v", err)
		}
		if kubeadmVersion != p.KubernetesVersion {
			klog.Errorf("kubeadm version %s not match kubernetes version %s", kubeadmVersion, p.KubernetesVersion)
			return fmt.Errorf("kubeadm version %s not match kubernetes version %s", kubeadmVersion, p.KubernetesVersion)
		}
	}

	// 检查 docker 的客户端是否正常
	if _, err := p.docker.Ping(context.Background()); err != nil {
		klog.Errorf("%v", err)
		return err
	}

	klog.Infof("plugin validate completed")
	return nil
}

func (p *PluginController) doComplete() error {
	cli, err := dockerutil.NewClient()
	if err != nil {
		return err
	}
	p.docker = cli
	p.exec = exec.New()
	p.Registry = p.Cfg.Registry

	if p.Cfg.Default.PushKubernetes {
		if len(p.KubernetesVersion) == 0 {
			if len(p.Cfg.Kubernetes.Version) != 0 {
				p.KubernetesVersion = p.Cfg.Kubernetes.Version
			} else {
				p.KubernetesVersion = os.Getenv("KubernetesVersion")
			}
		}
	}

	if p.Cfg.Plugin.Driver == SkopeoDriver {
		cmd := []string{"docker", "pull", "pixiuio/skopeo:1.17.0"}
		klog.Infof("Starting pull skopeo image %s", cmd)
		out, err := p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to pull skopeo image %v %v", string(out), err)
		}
	}

	if p.Cfg.Default.PushKubernetes {
		//cmd := []string{"sudo", "apt-get", "install", "-y", fmt.Sprintf("kubeadm=%s-00", p.Cfg.Kubernetes.Version[1:])}
		cmd := []string{"sudo", "curl", "-LO", fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubeadm", p.Cfg.Kubernetes.Version)}
		klog.Infof("Starting install kubeadm %s", cmd)
		out, err := p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get kubeadm %v %v, 疑似指定的 kubernetes 版本不符合规范", string(out), err)
		}

		cmd2 := []string{"sudo", "install", "-o", "root", "-g", "root", "-m", "0755", "kubeadm", "/usr/local/bin/kubeadm"}
		out, err = p.exec.Command(cmd2[0], cmd2[1:]...).CombinedOutput()
		if err != nil {
			klog.Errorf("安装 kubeadm 失败 %v %v", string(out), err)
			return fmt.Errorf("failed to install kubeadm %v %v", string(out), err)
		}
		p.CreateTaskMessage("kubernetes 镜像推送准备完成")
		klog.Infof("kubeadm 已安装完成")
	}

	p.Runners = []Runner{
		&login{name: "Registry登陆", p: p},
		&image{name: "解析镜像", p: p},
	}
	return p.Validate()
}

func (p *PluginController) Complete() error {
	if len(p.Cfg.Plugin.Driver) == 0 {
		p.Cfg.Plugin.Driver = DockerDriver
	}

	// 执行前校验
	msgResult := "数据校验完成"
	status, msg, process := "初始化成功", "初始化环境结束", 1
	var err error
	if err = p.doComplete(); err != nil {
		msgResult = fmt.Sprintf("数据校验失败，原因：%v", err)
		status, msg, process = "初始化失败", err.Error(), 3
	}
	p.CreateTaskMessage(msgResult)
	p.SyncTaskStatus(status, msg, process)
	return err
}

func (p *PluginController) Close() {
	if p.docker != nil {
		_ = p.docker.Close()
	}
}

func (p *PluginController) getKubeadmVersion() (string, error) {
	if _, err := p.exec.LookPath(Kubeadm); err != nil {
		return "", fmt.Errorf("failed to find %s %v", Kubeadm, err)
	}

	cmd := []string{Kubeadm, "version", "-o", "json"}
	out, err := p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to exec kubeadm version %v %v", string(out), err)
	}

	var kubeadmVersion KubeadmVersion
	if err := json.Unmarshal(out, &kubeadmVersion); err != nil {
		return "", fmt.Errorf("failed to unmarshal kubeadm version %v", err)
	}
	klog.V(2).Infof("kubeadmVersion %+v", kubeadmVersion)

	return kubeadmVersion.ClientVersion.GitVersion, nil
}

func (p *PluginController) cleanImages(in []byte) []byte {
	inStr := string(in)
	if !strings.Contains(inStr, IgnoreKey) {
		return in
	}

	klog.V(2).Infof("cleaning images: %+v", inStr)
	parts := strings.Split(inStr, "\n")
	index := 0
	for _, p := range parts {
		if strings.HasPrefix(p, IgnoreKey) {
			index += 1
		}
	}
	newInStr := strings.Join(parts[index:], "\n")
	klog.V(2).Infof("cleaned images: %+v", newInStr)

	return []byte(newInStr)
}

func (p *PluginController) getImages() ([]string, error) {
	cmd := []string{Kubeadm, "config", "images", "list", "--kubernetes-version", p.KubernetesVersion, "-o", "json"}
	out, err := p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to exec kubeadm config images list %v %v", string(out), err)
	}
	out = p.cleanImages(out)
	klog.V(2).Infof("images is %+v", string(out))

	var kubeadmImage KubeadmImage
	if err := json.Unmarshal(out, &kubeadmImage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kubeadm images %v", err)
	}

	return kubeadmImage.Images, nil
}

func (p *PluginController) sync(imageToPush string, targetImage string, img config.Image) error {
	klog.Infof("preparing to sync image %s to %s", imageToPush, targetImage)
	var cmd []string
	switch p.Cfg.Plugin.Driver {
	case SkopeoDriver:
		klog.Infof("use skopeo to copying image: %s", targetImage)
		cmd1 := []string{"skopeo", "login", "-u", p.Registry.Username, "-p", p.Registry.Password, p.Registry.Repository, ">", "/dev/null", "2>&1", "&&", "skopeo", "copy", "docker://" + imageToPush, "docker://" + targetImage}

		// p.Cfg.Plugin.Arch 解析平台架构配置，格式为: 操作系统/架构/变体 (如: linux/amd64/8)
		// 支持两种格式:
		//   - 完整格式: linux/amd64/8  → os=linux, arch=amd64, variant=8
		//   - 简化格式: linux/amd64    → os=linux, arch=amd64
		parts := strings.Split(p.Cfg.Plugin.Arch, "/")
		if len(parts) >= 2 {
			targetOS := parts[0]
			arch := parts[1]

			cmd1 = append(cmd1, "--override-os", targetOS)
			cmd1 = append(cmd1, "--override-arch", arch)

			if len(parts) >= 3 {
				variant := parts[2]
				cmd1 = append(cmd1, "--override-variant", variant)
			}
		}

		cmd = []string{"docker", "run", "--network", "host", "pixiuio/skopeo:1.17.0", "sh", "-c", strings.Join(cmd1, " ")}
		klog.Infof("即将执行命令(%s)进行同步", cmd)
	case DockerDriver:
		klog.Infof("Pulling image: %s", imageToPush)
		reader, err := p.docker.ImagePull(context.TODO(), imageToPush, dockerimage.PullOptions{})
		if err != nil {
			klog.Errorf("Failed to pull image %s: %v", imageToPush, err)
			return fmt.Errorf("failed to pull image %s: %v", imageToPush, err)
		}
		io.Copy(os.Stdout, reader)

		klog.Infof("Tagging image from %s to %s", imageToPush, targetImage)
		if err := p.docker.ImageTag(context.TODO(), imageToPush, targetImage); err != nil {
			klog.Errorf("Failed to tag image %s to %s: %v", imageToPush, targetImage, err)
			return fmt.Errorf("failed to tag image %s to %s: %v", imageToPush, targetImage, err)
		}

		cmd = []string{"docker", "push", targetImage}
	default:
		return fmt.Errorf("unsupported driver: %s", p.Cfg.Plugin.Driver)
	}

	klog.Infof("syncing image %s to %s", imageToPush, targetImage)
	out, err := p.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		klog.Errorf("Failed to push image %s: %v, output: %s", targetImage, err, string(out))
		return fmt.Errorf("failed to push image %s %v %v", targetImage, string(out), err)
	}

	klog.Infof("Successfully sync image: %s", targetImage)
	return nil
}

func (p *PluginController) doPushImage(img config.Image) error {
	imageMap := img.GetMap(p.Registry.Repository, p.Registry.Namespace)

	for imageToPush, targetImage := range imageMap {
		p.SyncImageStatus(targetImage, rainbowtypes.SyncImageRunning, "", img)
		err := p.sync(imageToPush, targetImage, img)
		if err != nil {
			p.SyncImageStatus(targetImage, rainbowtypes.SyncImageError, err.Error(), img)
			p.CreateTaskMessage(fmt.Sprintf("镜像 %s 同步失败，原因: %v", imageToPush, err))
			continue
		}
		p.SyncImageStatus(targetImage, rainbowtypes.SyncImageComplete, "", img)
		p.CreateTaskMessage(fmt.Sprintf("镜像 %s 同步完成", imageToPush))
	}

	return nil
}

func (p *PluginController) getImagesFromFile() ([]string, error) {
	var imgs []string
	return imgs, nil
}

func (p *PluginController) Run() error {
	if !p.Cfg.Default.PushImages && !p.Cfg.Default.PushKubernetes {
		return nil
	}

	for _, runner := range p.Runners {
		name := runner.GetName()
		msgResult := name + "完成"

		err := runner.Run()
		if err != nil {
			msgResult = fmt.Sprintf("%s失败，原因：%v", name, err)
			p.CreateTaskMessage(msgResult)
			p.SyncTaskStatus(name, name+"失败", 3)
			return err
		} else {
			p.CreateTaskMessage(msgResult)
			p.SyncTaskStatus(name, name+"完成", 1)
		}
	}

	p.SyncTaskStatus("开始同步镜像", "", 1)
	p.CreateTaskMessage("开始同步镜像，请稍等")

	klog.Infof("待推送镜像列表为 %v", p.Images)

	diff := len(p.Images)
	maxCh := make(chan struct{}, MaxConcurrency)
	errCh := make(chan error, diff)
	var wg sync.WaitGroup
	for _, imageToPush := range p.Images {
		wg.Add(1)
		maxCh <- struct{}{} // 获取信号量，控制并发

		go func(image config.Image) {
			defer wg.Done()
			defer func() { <-maxCh }() // 释放信号量

			err := p.doPushImage(image)
			if err != nil {
				errCh <- err
				return
			}
		}(imageToPush)
	}
	wg.Wait()

	select {
	case err := <-errCh:
		if err != nil {
			p.SyncTaskStatus("镜像同步结束", "存在镜像同步异常", 2)
			return err
		}
	default:
	}

	p.SyncTaskStatus("镜像同步完成", "镜像全部同步完成", 2)
	p.CreateTaskMessage("镜像任务执行完成")
	return nil
}

// SyncTaskStatus
// 0 未开始
// 1 执行中
// 2 执行成功
// 3 执行失败
func (p *PluginController) SyncTaskStatus(status string, msg string, process int) {
	if !p.Synced {
		klog.Infof("未启用任务回调同步功能")
		return
	}

	for i := 0; i < 3; i++ {
		err := p.httpClient.Put(
			fmt.Sprintf("%s/rainbow/tasks/%d/status", p.Callback, p.TaskId),
			nil,
			map[string]interface{}{"status": status, "message": msg, "process": process})
		if err == nil {
			klog.Infof("同步任务(%d) 状态(%s) 信息(%s) 完成", p.TaskId, status, msg)
			return
		}

		klog.Errorf("同步任务(%d) 状态(%s) 信息(%s) 失败 %v，尝试重试", p.TaskId, status, msg, err)
		time.Sleep(time.Second)
	}
}

func (p *PluginController) SyncImageStatus(target string, status string, msg string, img config.Image) {
	if !p.Synced {
		klog.Infof("未启用镜像回调同步功能")
		return
	}

	for i := 0; i < 3; i++ {
		err := p.httpClient.Put(
			fmt.Sprintf("%s/rainbow/images/status", p.Callback),
			nil,
			map[string]interface{}{
				"name":        img.Name,
				"image_id":    img.Id,
				"task_id":     p.TaskId,
				"registry_id": p.RegistryId,
				"status":      status,
				"message":     msg,
				"target":      target,
			})
		if err == nil {
			klog.Infof("同步镜像(%d) 状态(%s) 信息(%s) mirror(%s) 成功", p.TaskId, status, msg, target, err)
			return
		}

		klog.Errorf("同步镜像(%d) 状态(%s) 信息(%s) mirror(%s) 失败 %v，尝试重试", p.TaskId, status, msg, target, err)
		time.Sleep(time.Second)
	}
}

func (p *PluginController) CreateImages(names []string) ([]model.Image, error) {
	if !p.Synced {
		return nil, nil
	}

	var resp rainbowtypes.Response
	err := p.httpClient.Post(
		fmt.Sprintf("%s/rainbow/images/batches", p.Callback),
		&resp,
		map[string]interface{}{"task_id": p.TaskId, "names": names}, nil)

	return resp.Result, err
}

func (p *PluginController) CreateTaskMessage(msg string) {
	if !p.Synced {
		return
	}

	if err := p.httpClient.Post(
		fmt.Sprintf("%s/rainbow/tasks/%d/messages", p.Callback, p.TaskId),
		nil,
		map[string]interface{}{"message": msg}, nil); err != nil {
		klog.Errorf("创建 %s 失败 %v", msg, err)
	} else {
		klog.Infof("创建 %s 成功", msg)
	}
}
