package rainbow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
	"github.com/caoyingjunz/rainbow/pkg/util/errors"
	"github.com/caoyingjunz/rainbow/pkg/util/uuid"
)

const (
	TaskWaitStatus  = "调度中"
	HuaweiNamespace = "pixiu-public" // pixiuHub 内置默认外部命名空间

	DemandPaymentType  = 0 // 按需付费
	PackagePaymentType = 1 // 包年包月
)

const (
	defaultNamespace = "emptyNamespace" // pixiuHub 内置默认空命名空间
	defaultArch      = "linux/amd64"
	defaultDriver    = "skopeo"

	k8sImageCount      = 5   // 默认数量
	defaultRemainCount = 100 // TODO 临时设置，后续缩降到 20
)

func (s *ServerController) preCreateTask(ctx context.Context, req *types.CreateTaskRequest) error {
	// 验证架构
	if err := ValidateArch(req.Architecture); err != nil {
		return err
	}

	// 验证该用户是否还有余额
	if err := s.validateUserQuota(ctx, req); err != nil {
		klog.Errorf("valid user quota failed %v", err)
		return err
	}

	// 验证镜像规格
	if req.Type == 1 {
		if !strings.HasPrefix(req.KubernetesVersion, "v1.") {
			return fmt.Errorf("invaild kubernetes version (%s)", req.KubernetesVersion)
		}
	} else {
		var errs []error
		// TODO: 其他不合规检查
		//for _, image := range req.Images {
		//	if strings.Contains(image, "\"") { // 分割镜像名称和版本
		//		errs = append(errs, fmt.Errorf("invaild image(%s)", image))
		//		parts := strings.Split(image, ":")
		//		if len(parts) != 2 {
		//			errs = append(errs, fmt.Errorf("invalid image format, should be 'name:tag' (%s)", image))
		//		}
		//	}
		//}
		return utilerrors.NewAggregate(errs)
	}

	return nil
}

func (s *ServerController) validateUserQuota(ctx context.Context, req *types.CreateTaskRequest) error {
	userObj, err := s.factory.Task().GetUser(ctx, req.UserId)
	if err != nil {
		klog.Errorf("获取用户失败 %v", err)
		if errors.IsNotFound(err) {
			return fmt.Errorf("用户不存在，请联系管理员")
		}
		return err
	}

	imageCount := k8sImageCount // k8s 镜像数是 5
	if req.Type == 0 {
		imageCount = len(req.Images)
	}

	switch userObj.PaymentType {
	case DemandPaymentType: // 按量付费
		if imageCount > userObj.RemainCount {
			return fmt.Errorf("同步镜像数已超过剩余额度，请联系管理员")
		}
	case PackagePaymentType: // 包年包月
		now := time.Now()
		if now.After(userObj.ExpireTime) {
			return fmt.Errorf("包年包月已过期，请联系管理员")
		}
	}
	return nil
}
func (s *ServerController) GetUserInfoByAccessKey(ctx *gin.Context, listOption types.ListOptions) (*model.User, error) {
	obj, err := s.factory.Access().Get(ctx, listOption.AccessKey)
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, obj.UserId)
}

func (s *ServerController) CreateTaskV2(ctx *gin.Context, req *types.CreateTaskRequest) error {
	return s.CreateTask(ctx, req)
}

func (s *ServerController) CreateTask(ctx context.Context, req *types.CreateTaskRequest) error {
	if err := s.preCreateTask(ctx, req); err != nil {
		klog.Errorf("创建任务前置检查未通过 %v", err)
		return err
	}

	// 填充任务名称
	if len(strings.TrimSpace(req.Name)) == 0 {
		req.Name = uuid.NewRandName("", 8)
	}
	req.Namespace = WrapNamespace(req.Namespace, req.UserName)

	// 初始化仓库
	if req.RegisterId == 0 {
		req.RegisterId = *RegistryId
	}

	if len(req.Architecture) == 0 {
		req.Architecture = defaultArch
	}
	if len(req.Driver) == 0 {
		req.Driver = defaultDriver
	}

	// 如果是k8s类型的镜像，则由 plugin 回调创建
	// 0：直接指定镜像列表 1: 指定 kubernetes 版本
	switch req.Type {
	case 0:
		object, err := s.factory.Task().Create(ctx, &model.Task{
			Name:              req.Name,
			UserId:            req.UserId,
			UserName:          req.UserName,
			RegisterId:        req.RegisterId,
			AgentName:         req.AgentName,
			Mode:              req.Mode,
			Status:            TaskWaitStatus,
			Type:              req.Type,
			KubernetesVersion: req.KubernetesVersion,
			Driver:            req.Driver,
			Namespace:         req.Namespace,
			IsPublic:          req.PublicImage,
			Logo:              req.Logo,
			IsOfficial:        req.IsOfficial,
			Architecture:      req.Architecture, // 通用镜像架构，会被镜像自身的架构覆盖
			OwnerRef:          req.OwnerRef,
			SubscribeId:       req.SubscribeId,
		})
		if err != nil {
			return err
		}
		taskId := object.Id
		s.CreateTaskMessages(ctx, taskId, "同步已启动", "数据校验中，预计等待 1 分钟")

		if err = s.CreateImageWithTag(ctx, taskId, req); err != nil {
			s.CreateTaskMessages(ctx, taskId, fmt.Sprintf("创建镜像和版本失败 %v", err))
			_ = s.DeleteTaskWithImages(ctx, taskId)
			return err
		}
	case 1:
		kubernetesVersions := strings.Split(req.KubernetesVersion, ",")
		for _, kv := range kubernetesVersions {
			subName := req.Name + "-" + kv
			object, err := s.factory.Task().Create(ctx, &model.Task{
				Name:              subName,
				UserId:            req.UserId,
				UserName:          req.UserName,
				RegisterId:        req.RegisterId,
				AgentName:         req.AgentName,
				Mode:              req.Mode,
				Status:            TaskWaitStatus,
				Type:              req.Type,
				KubernetesVersion: kv,
				Driver:            req.Driver,
				Namespace:         req.Namespace,
				IsPublic:          req.PublicImage,
				Logo:              req.Logo,
				IsOfficial:        req.IsOfficial,
				Architecture:      req.Architecture,
				OwnerRef:          req.OwnerRef,
				SubscribeId:       req.SubscribeId,
			})
			if err != nil {
				return err
			}

			s.CreateTaskMessages(ctx, object.Id, "同步已启动", "数据校验中，预计等待 1 分钟")
			klog.Infof("(%s) 的子任务(%s)创建成功，其类型为 kubernetes，镜像由 plugin 回调创建", req.Name, subName)
		}
	}

	return nil
}

// CreateTaskMessages 批量创建同步消息
func (s *ServerController) CreateTaskMessages(ctx context.Context, taskId int64, messages ...string) {
	for _, msg := range messages {
		if err := s.factory.Task().CreateTaskMessage(ctx, &model.TaskMessage{TaskId: taskId, Message: msg}); err != nil {
			klog.Errorf("记录 %s 失败 %v", msg, err)
		}
	}
}

func (s *ServerController) parseImageNameFromPath(ctx context.Context, path string, regId int64, namespace string) (string, error) {
	parts2 := strings.Split(path, "/")
	name := parts2[len(parts2)-1]
	if len(name) == 0 {
		return "", fmt.Errorf("不合规镜像名称 %s", path)
	}
	// 如果使用默认内置仓库，则添加租户名称
	if regId == *RegistryId {
		if len(namespace) != 0 {
			name = namespace + "/" + name
		}
	}

	return name, nil
}

func (s *ServerController) CreateImageWithTag(ctx context.Context, taskId int64, req *types.CreateTaskRequest) error {
	klog.Infof("使用镜像仓库(%d)", req.RegisterId)
	reg, err := s.factory.Registry().Get(ctx, req.RegisterId)
	if err != nil {
		return fmt.Errorf("获取仓库(%d)失败 %v", req.RegisterId, err)
	}

	imageMap := make(map[string][]string)
	for _, i := range util.TrimAndFilter(req.Images) {
		path, tag, err := ParseImageItem(i)
		if err != nil {
			klog.Errorf("无法解析镜像 %s %v", i, err)
			return fmt.Errorf("failed to parse image %v", err)
		}

		old, ok := imageMap[path]
		if ok {
			old = append(old, tag)
			imageMap[path] = old
		} else {
			imageMap[path] = []string{tag}
		}
	}

	namespace := req.Namespace
	for path, tags := range imageMap {
		var imageId int64
		parts2 := strings.Split(path, "/")
		name := parts2[len(parts2)-1]
		if len(name) == 0 {
			return fmt.Errorf("不合规镜像名称 %s", path)
		}
		if req.RegisterId == *RegistryId {
			if len(namespace) != 0 {
				name = namespace + "/" + name
			}
		}

		mirror := reg.Repository + "/" + reg.Namespace + "/" + name

		oldImage, err := s.factory.Image().GetBy(ctx, db.WithName(name), db.WithUser(req.UserId))
		if err != nil {
			// 镜像不存在，则先创建镜像
			if errors.IsNotFound(err) {
				newImage, err := s.factory.Image().Create(ctx, &model.Image{
					UserId:       req.UserId,
					UserName:     req.UserName,
					RegisterId:   req.RegisterId,
					Namespace:    namespace,
					Logo:         req.Logo,
					Name:         name,
					Mirror:       mirror,
					IsPublic:     req.PublicImage,
					IsOfficial:   req.IsOfficial,
					LastSyncTime: time.Now(),
					IsLocked:     true,
				})
				if err != nil {
					klog.Errorf("创建镜像(%s)失败: %v", path, err)
					return err
				}
				imageId = newImage.Id
			} else {
				return err
			}
		} else {
			klog.Infof("镜像(%s)已存在，复用", path)
			imageId = oldImage.Id
		}

		// 尝试从从远端获取镜像信息，并同步到 pixiuHub
		go s.tryToUpdateImageInfo(ctx, imageId, path)

		// 版本需要和任务关联
		for _, tag := range tags {
			oldTag, tagErr := s.factory.Image().GetTagWithArch(ctx, imageId, tag, req.Architecture, false)
			if tagErr != nil {
				// 非不存在报错，则直接返回异常
				if !errors.IsNotFound(tagErr) {
					klog.Errorf("获取镜像(%d)的版本(%s)失败: %v", imageId, tag, tagErr)
					return tagErr
				}

				// tag 不存在则创建
				if _, err = s.factory.Image().CreateTag(ctx, &model.Tag{
					Path:         path,
					Mirror:       mirror,
					ImageId:      imageId,
					TaskIds:      fmt.Sprintf("%d", taskId),
					Name:         tag,
					Status:       types.SyncImageInitializing,
					Architecture: req.Architecture,
				}); err != nil {
					klog.Errorf("创建镜像(%s)的版本(%s)失败 %v", path, tag, err)
					return err
				}
			} else {
				// 已经存在则写入新关联的 taskId
				newTaskIds := strings.Join([]string{oldTag.TaskIds, fmt.Sprintf("%d", taskId)}, ",")
				update := map[string]interface{}{"task_ids": newTaskIds, "status": types.SyncImageInitializing}
				// 必要时更新 mirror
				// 早期创建的 tag 不存在 mirror 字段，会在其他人任务推送时展示 null
				if mirror != oldTag.Mirror {
					update["mirror"] = mirror
				}
				// 来源修改时，同步调整
				if path != oldTag.Path {
					update["path"] = path
				}
				if err = s.factory.Image().UpdateTag(ctx, imageId, tag, update); err != nil {
					klog.Errorf("更新镜像(%s)的版本(%s)任务Id失败 %v", path, tag, err)
					return err
				}
			}
		}
	}

	return nil
}

func (s *ServerController) tryToUpdateImageInfo(ctx context.Context, imageId int64, path string) {
	image, err := s.factory.Image().GetBy(ctx, db.WithId(imageId))
	if err != nil {
		klog.Errorf("tryToUpdateImageInfo 尝试获取镜像(%d)失败 %v", imageId, err)
		return
	}
	if len(image.Description) != 0 {
		return
	}

	labels, err := s.factory.Label().ListImageLabelNames(ctx, imageId)
	if err != nil {
		klog.Errorf("tryToUpdateImageInfo 尝试获取镜像(%d)关联标签失败 %v", imageId, err)
		return
	}
	if len(labels) != 0 {
		return
	}

	var (
		namespace string
		name      string
	)
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		namespace, name = parts[len(parts)-2], parts[len(parts)-1]
	} else {
		namespace, name = "library", parts[len(parts)-1]
	}

	remoteRepo, err := s.GetRepository(ctx, types.CallSearchRequest{
		Namespace:  namespace,
		Repository: name,
	})
	if err != nil {
		klog.Errorf("获取远端镜像(%s/%s)失败 %v", namespace, name, err)
		return
	}

	desc := remoteRepo.Description
	if len(desc) != 0 {
		_ = s.factory.Image().UpdateWithoutLock(ctx, imageId, map[string]interface{}{"description": desc})
	}

	categoriesMap := make(map[string]bool)
	for _, category := range remoteRepo.Categories {
		categoriesMap[strings.ToLower(category.Name)] = true
	}
	if len(categoriesMap) == 0 {
		return
	}

	allLabels, err := s.factory.Label().List(ctx)
	if err != nil {
		return
	}
	var newBindLabels []int64
	for _, label := range allLabels {
		if categoriesMap[strings.ToLower(label.Name)] {
			newBindLabels = append(newBindLabels, label.Id)
		}
	}
	if err = s.BindImageLabels(ctx, imageId, types.BindImageLabels{OP: 0, LabelIds: newBindLabels}); err != nil {
		klog.Errorf("绑定镜像(%d)标签失败 %v", imageId, err)
	}
}

func (s *ServerController) UpdateTask(ctx context.Context, req *types.UpdateTaskRequest) error {
	if err := s.factory.Task().Update(ctx, req.Id, req.ResourceVersion, map[string]interface{}{
		"register_id": req.RegisterId,
		"mode":        req.Mode,
	}); err != nil {
		return err
	}

	old, err := s.factory.Image().ListWithTask(ctx, req.Id)
	if err != nil {
		return err
	}
	var oldImages []string
	for _, o := range old {
		oldImages = append(oldImages, o.Name)
	}
	oldImageMap := sets.NewString(oldImages...)

	var addImages []string
	for _, n := range req.Images {
		if oldImageMap.Has(n) {
			continue
		}
		addImages = append(addImages, n)
	}
	var images []model.Image
	trimAddImages := util.TrimAndFilter(req.Images)
	for _, i := range trimAddImages {
		images = append(images, model.Image{
			Name: i,
		})
	}
	if err = s.factory.Image().CreateInBatch(ctx, images); err != nil {
		return fmt.Errorf("failed to create tasks images %v", err)
	}

	return nil
}

func (s *ServerController) ListTasks(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	// 初始化分页属性
	listOption.SetDefaultPageOption()

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}
	opts := []db.Options{
		db.WithUser(listOption.UserId),
		db.WithNameLike(listOption.NameSelector),
		db.WithNamespace(listOption.Namespace),
		db.WithAgent(listOption.Agent),
		db.WithRef(listOption.OwnerRef),
		db.WithSubscribe(listOption.SubscribeId),
	}

	var err error
	pageResult.Total, err = s.factory.Task().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取任务总数失败 %v", err)
		pageResult.Message = err.Error()
	}
	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithModifyOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)
	pageResult.Items, err = s.factory.Task().List(ctx, opts...)
	if err != nil {
		klog.Errorf("获取推送任务列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	return pageResult, nil
}

func (s *ServerController) UpdateTaskStatus(ctx context.Context, req *types.UpdateTaskStatusRequest) error {
	if err := s.factory.Task().UpdateDirectly(ctx, req.TaskId, map[string]interface{}{"status": req.Status, "message": req.Message, "process": req.Process}); err != nil {
		klog.Errorf("更新任务状态失败 %v", err)
		return err
	}

	_ = s.AfterUpdateTaskStatus(ctx, req)
	return nil
}

func (s *ServerController) AfterUpdateTaskStatus(ctx context.Context, req *types.UpdateTaskStatusRequest) error {
	if req.Process != 2 {
		klog.V(1).Infof("任务未结束，暂无操作执行")
		return nil
	}

	// 推送给管理员
	if err := s.sendToAdmin(ctx, req); err != nil {
		klog.Errorf("管理推送失败 %v", err)
	}
	// 推送给普通用户
	if err := s.sendToUser(ctx, req); err != nil {
		klog.Errorf("普通用户推送失败 %v", err)
	}
	return nil
}

func (s *ServerController) sendToAdmin(ctx context.Context, req *types.UpdateTaskStatusRequest) error {
	tags, err := s.factory.Image().ListTags(ctx, db.WithTaskLike(req.TaskId))
	if err != nil {
		klog.Errorf("获取本次任务的镜像tag失败 %v", err)
		return err
	}
	num := len(tags)

	task, err := s.factory.Task().Get(ctx, req.TaskId)
	if err != nil {
		klog.Errorf("获取本次任务的镜像tag失败 %v", err)
		return err
	}
	taskType := "镜像组"
	if task.Type == 1 {
		taskType = "Kubernetes"
	}
	notifyContent := fmt.Sprintf("同步类型: %s\n执行用户: %s\n推送数量: %d", taskType, task.UserName, num)

	return s.SendNotify(ctx, &types.SendNotificationRequest{
		Content: notifyContent,
		CreateNotificationRequest: types.CreateNotificationRequest{
			Role: types.SystemNotifyRole,
			UserMetaRequest: types.UserMetaRequest{
				UserId:   task.UserId,
				UserName: task.UserName,
			},
		},
	})
}

func (s *ServerController) sendToUser(ctx context.Context, req *types.UpdateTaskStatusRequest) error {
	tags, err := s.factory.Image().ListTags(ctx, db.WithTaskLike(req.TaskId))
	if err != nil {
		klog.Errorf("获取本次任务的镜像tag失败 %v", err)
		return err
	}

	successImages, failedImages := make([]string, 0), make([]string, 0)
	for _, tag := range tags {
		repo := tag.Path + ":" + tag.Name
		if tag.Status == "Error" {
			failedImages = append(failedImages, repo)
		} else if tag.Status == "Completed" {
			successImages = append(successImages, repo)
		}
	}

	task, err := s.factory.Task().Get(ctx, req.TaskId)
	if err != nil {
		klog.Errorf("获取本次任务的镜像tag失败 %v", err)
		return err
	}

	taskType := "镜像组"
	if task.Type == 1 {
		taskType = "Kubernetes"
	}
	notifyContent := fmt.Sprintf("同步类型: %s\n推送结果:", taskType)

	if len(successImages) != 0 {
		suc := "  成功:"
		for _, si := range successImages {
			suc = suc + "\n    " + si
		}
		notifyContent = fmt.Sprintf("%s\n%s", notifyContent, suc)
	}
	if len(failedImages) != 0 {
		failed := "  失败:"
		for _, si := range failedImages {
			failed = failed + "\n    " + si
		}
		notifyContent = fmt.Sprintf("%s\n%s", notifyContent, failed)
	}
	notifyContent = fmt.Sprintf("%s\n%s", notifyContent, "详情参考: https://hub.pixiuio.com")

	return s.SendNotify(ctx, &types.SendNotificationRequest{
		Content: notifyContent,
		CreateNotificationRequest: types.CreateNotificationRequest{
			UserMetaRequest: types.UserMetaRequest{
				UserId:   task.UserId,
				UserName: task.UserName,
			},
		},
	})
}

func removeTaskID(taskIds string, taskIDToRemove string) string {
	ids := strings.Split(taskIds, ",")
	result := make([]string, 0, len(ids))

	for _, id := range ids {
		if id != taskIDToRemove {
			result = append(result, id)
		}
	}

	return strings.Join(result, ",")
}

func (s *ServerController) DeleteTask(ctx context.Context, taskId int64) error {
	if err := s.factory.Task().Delete(ctx, taskId); err != nil {
		klog.Errorf("删除任务失败 %v", taskId)
		return err
	}

	tags, err := s.factory.Image().ListTags(ctx, db.WithTaskLike(taskId))
	if err != nil {
		klog.Errorf("获取本次任务的镜像tag失败 %v", err)
		return err
	}
	for _, tag := range tags {
		if err = s.factory.Image().UpdateTag(ctx, tag.ImageId, tag.Name, map[string]interface{}{
			"task_ids": removeTaskID(tag.TaskIds, fmt.Sprintf("%d", taskId)),
		}); err != nil {
			klog.Warningf("移除任务(%d)关联的tag(%s)时失败 %v", taskId, tag.Name, err)
		}
	}

	return nil
}

func (s *ServerController) DeleteTaskWithImages(ctx context.Context, taskId int64) error {
	_ = s.factory.Task().Delete(ctx, taskId)
	return nil
}

func (s *ServerController) GetTask(ctx context.Context, taskId int64) (interface{}, error) {
	return s.factory.Task().Get(ctx, taskId)
}

func (s *ServerController) ReRunTask(ctx context.Context, req *types.UpdateTaskRequest) error {
	updates := map[string]interface{}{
		"agent_name":      "",
		"status":          TaskWaitStatus,
		"process":         0,
		"message":         "触发重新执行",
		"only_push_error": req.OnlyPushError,
	}
	if err := s.factory.Task().Update(ctx, req.Id, req.ResourceVersion, updates); err != nil {
		klog.Errorf("重新执行任务 %d 失败 %v", req.Id, err)
		return err
	}

	// 全量重新推送时，重置任务过程信息
	if !req.OnlyPushError {
		if err := s.factory.Task().DeleteTaskMessages(ctx, req.Id); err != nil {
			klog.Errorf("清理任务(%d)过程信息失败 %v", req.Id, err)
		}
	}
	return nil
}

func (s *ServerController) ListTaskImages(ctx context.Context, taskId int64, listOption types.ListOptions) (interface{}, error) {
	return s.factory.Image().ListTags(ctx, db.WithTaskLike(taskId), db.WithNameLike(listOption.NameSelector))
}

func (s *ServerController) CreateTaskMessage(ctx context.Context, req types.CreateTaskMessageRequest) error {
	return s.factory.Task().CreateTaskMessage(ctx, &model.TaskMessage{
		Message: req.Message,
		TaskId:  req.Id,
	})
}

func (s *ServerController) ListTaskMessages(ctx context.Context, taskId int64) (interface{}, error) {
	return s.factory.Task().ListTaskMessages(ctx, db.WithTask(taskId))
}

func (s *ServerController) ListTasksByIds(ctx context.Context, ids []int64) (interface{}, error) {
	return s.factory.Task().List(ctx, db.WithIDIn(ids...))
}

func (s *ServerController) DeleteTasksByIds(ctx context.Context, ids []int64) error {
	for _, taskId := range ids {
		if err := s.DeleteTask(ctx, taskId); err != nil {
			klog.Errorf("%v", err)
			return err
		}
	}

	return nil
}

func (s *ServerController) preCreateAgent(ctx context.Context, req *types.CreateAgentRequest) error {
	if len(req.AgentName) == 0 {
		return fmt.Errorf("agent 名称不能为空")
	}
	if len(req.GithubUser) == 0 {
		return fmt.Errorf("github 用户名不能为空")
	}
	if len(req.GithubToken) == 0 {
		return fmt.Errorf("github token不能为空")
	}
	if len(req.GithubEmail) == 0 {
		return fmt.Errorf("github email不能为空")
	}

	// 检查agent是否存在
	_, err := s.factory.Agent().GetByName(ctx, req.AgentName)
	if err == nil {
		return fmt.Errorf("agent名称 %s 已存在", req.AgentName)
	}

	return nil
}

func (s *ServerController) CreateAgent(ctx context.Context, req *types.CreateAgentRequest) error {
	if err := s.preCreateAgent(ctx, req); err != nil {
		klog.Infof("创建agent前置检查失败: %v", err)
		return err
	}

	if len(req.GithubRepository) == 0 {
		req.GithubRepository = fmt.Sprintf("https://github.com/%s/plugin.git", req.GithubUser)
	}
	// 创建新的agent记录
	agent := &model.Agent{
		Name:             req.AgentName,
		GithubUser:       req.GithubUser,
		GithubToken:      req.GithubToken,
		GithubRepository: req.GithubRepository,
		GithubEmail:      req.GithubEmail,
		Type:             req.Type,
		RainbowdName:     req.RainbowdName,
		Status:           model.UnStartType,
	}
	if _, err := s.factory.Agent().Create(ctx, agent); err != nil {
		return fmt.Errorf("创建agent失败: %v", err)
	}

	return nil
}

// DeleteAgent 删除已注册的agent信息
func (s *ServerController) DeleteAgent(ctx context.Context, agentId int64) error {
	// TODO 检查是否有正在运行的任务关联该agent
	// 执行删除操作
	old, err := s.factory.Agent().Get(ctx, agentId)
	if err != nil {
		return err
	}
	if err = s.factory.Agent().Update(ctx, agentId, old.ResourceVersion, map[string]interface{}{
		"status": model.DeletingAgentType,
	}); err != nil {
		return fmt.Errorf("删除agent失败: %v", err)
	}

	// 清理缓存
	klog.V(0).Infof("清理 agent(%s) 缓存", old.Name)
	s.lock.Lock()
	defer s.lock.Unlock()
	if RpcClients != nil {
		delete(RpcClients, old.Name)
	}

	return nil
}

func (s *ServerController) ListRainbowds(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	return s.factory.Rainbowd().List(ctx)
}
