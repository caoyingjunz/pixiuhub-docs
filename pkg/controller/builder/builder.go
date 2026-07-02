package builder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/caoyingjunz/pixiulib/exec"
	"github.com/docker/docker/client"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/util"
	dockerutil "github.com/caoyingjunz/rainbow/pkg/util/docker"
)

type BuildController struct {
	Callback       string
	BuildId        int64
	DockerfilePath string
	Repo           string
	Arch           string

	httpClient util.HttpInterface
	exec       exec.Interface
	docker     *client.Client

	Cfg      config.Config
	Registry config.Registry
}

func (b *BuildController) Login() error {
	cmd := []string{"docker", "login", "-u", b.Registry.Username, "-p", b.Registry.Password}
	if b.Registry.Repository != "" {
		cmd = append(cmd, b.Registry.Repository)
	}
	out, err := b.exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to login in image %v %v", string(out), err)
	}
	klog.Infof("镜像仓库登录完成")
	return nil
}

func NewBuilderController(cfg config.Config) *BuildController {
	return &BuildController{
		Cfg:            cfg,
		Callback:       cfg.Build.Callback,
		BuildId:        cfg.Build.BuildId,
		Repo:           cfg.Build.Repo,
		Arch:           cfg.Build.Arch,
		DockerfilePath: cfg.Build.DockerfilePath,
		httpClient:     util.NewHttpClient(5*time.Second, cfg.Build.Callback),
	}
}

func (b *BuildController) Complete() error {
	cli, err := dockerutil.NewClient()
	if err != nil {
		return err
	}
	b.docker = cli
	b.exec = exec.New()
	b.Registry = b.Cfg.Registry

	return b.Validate()
}

func (b *BuildController) Close() {
	if b.docker != nil {
		_ = b.docker.Close()
	}
}

func (b *BuildController) BuildAndPushImage() error {
	imageName := fmt.Sprintf("%s/%s/%s", b.Registry.Repository, b.Registry.Namespace, b.Repo)

	b.SyncBuildStatus("初始化构建环境")
	klog.Info("Init build env")

	buildInitCmd := []string{"docker", "buildx", "create", "--name", "multi-builder", "--driver", "docker-container", "--use"}
	out, err := b.exec.Command(buildInitCmd[0], buildInitCmd[1:]...).CombinedOutput()
	b.SyncBuildMessages(string(out))
	if err != nil {
		b.SyncBuildStatus("初始化构建失败")
		return fmt.Errorf("failed to init buildx: %w\n%s", err, string(out))
	}

	b.SyncBuildStatus("开始构建上传镜像")
	klog.Infof("Starting build image %s", imageName)

	buildAndPushCmd := b.exec.Command("docker", "buildx", "build", "--platform", b.Arch, "-f", b.DockerfilePath, "-t", imageName, "--push", ".")
	if err := b.runCmdAndStream(buildAndPushCmd); err != nil {
		b.SyncBuildStatus("镜像构建上传失败")
		return fmt.Errorf("failed to build and push image: %w", err)
	}

	b.SyncBuildStatus("镜像构建上传成功")
	klog.Infof("Image built successfully: %s", imageName)

	return nil
}

func (b *BuildController) Validate() error {
	if _, err := b.docker.Ping(context.Background()); err != nil {
		return err
	}
	klog.Infof("builder validate completed")
	return nil
}

func (b *BuildController) Run() error {
	if err := b.Login(); err != nil {
		return err
	}
	if err := b.BuildAndPushImage(); err != nil {
		return err
	}

	return nil
}

func (b *BuildController) SyncBuildStatus(status string) {
	data, err := util.BuildHttpBody(map[string]interface{}{"Status": status})
	if err != nil {
		klog.Errorf("构造请求体失败 %v", err)
		return
	}

	httpClient := util.HttpClientV2{URL: fmt.Sprintf("%s/rainbow/set/build/%d/status", b.Callback, b.BuildId)}
	if err1 := httpClient.Method(http.MethodPut).
		WithTimeout(30 * time.Second).
		WithBody(data).
		Do(nil); err1 != nil {
		klog.Errorf("同步状态失败 %v", err1)
		return
	}

	klog.Infof("同步构建(%d)状态(%s)完成", b.BuildId, status)
}

func (b *BuildController) SyncBuildMessages(msg string) {
	url := fmt.Sprintf("%s/rainbow/builds/%d/messages", b.Callback, b.BuildId)
	err := b.httpClient.Post(url, nil, map[string]interface{}{"message": msg}, nil)
	if err != nil {
		klog.Errorf("同步 %s 失败 %v", msg, err)
	} else {
		klog.Infof("同步 %s 成功", msg)
	}
}

func (b *BuildController) runCmdAndStream(cmd exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	readPipe := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1024), 1024*1024)
		for scanner.Scan() {
			b.SyncBuildMessages(scanner.Text())
		}
	}
	go readPipe(stdout)
	go readPipe(stderr)

	return cmd.Wait()
}
