package rainbow

import (
	"context"
	"fmt"
	"strings"
	"time"

	swrmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/model"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util/errors"
)

func (s *ServerController) CreateImage(ctx context.Context, req *types.CreateImageRequest) error {
	_, err := s.factory.Image().Create(ctx, &model.Image{
		Name:       req.Name,
		RegisterId: req.RegisterId,
		IsPublic:   req.IsPublic,
	})
	if err != nil {
		klog.Errorf("创建镜像失败 %v", err)
	}

	return err
}

func (s *ServerController) UpdateImage(ctx context.Context, req *types.UpdateImageRequest) error {
	updates := make(map[string]interface{})
	updates["logo"] = req.Logo
	updates["description"] = req.Description
	updates["is_public"] = req.IsPublic
	updates["is_locked"] = req.IsLocked
	return s.factory.Image().Update(ctx, req.Id, req.ResourceVersion, updates)
}

// TryUpdateRemotePublic 更新内置标准镜像仓库为 public
func (s *ServerController) TryUpdateRemotePublic(ctx context.Context, req *types.UpdateImageStatusRequest, old *model.Image) error {
	if !s.isDefaultRepo(req.RegistryId) {
		return nil
	}

	if req.Status != types.SyncImageComplete {
		klog.Infof("镜像(%s)状态未推送完成，暂时不需要设置为公开", req.Name)
		return nil
	}

	name := req.Name
	for i := 0; i < 3; i++ {
		if strings.Contains(name, "/") {
			name = strings.ReplaceAll(name, "/", "$")
		}

		klog.Infof("尝试更新镜像 %s 已经为 public", name)
		resp, err := SwrClient.ShowRepository(&swrmodel.ShowRepositoryRequest{Namespace: HuaweiNamespace, Repository: name})
		if err != nil {
			klog.Errorf("获取远端镜像 %s 失败 %v 1s后进行下一次重试", name, err)
			time.Sleep(1 * time.Second)
			continue
		}
		if *resp.IsPublic {
			klog.Infof("镜像%s 已经为 public，无需更新", req.Name)
			return nil
		}

		_, err = SwrClient.UpdateRepo(&swrmodel.UpdateRepoRequest{Namespace: HuaweiNamespace, Repository: name, Body: &swrmodel.UpdateRepoRequestBody{IsPublic: true}})
		if err != nil {
			klog.Errorf("更新远端镜像 %s 失败 %v", name, err)
			time.Sleep(1 * time.Second)
			continue
		} else {
			_ = s.factory.Image().Update(ctx, req.ImageId, old.ResourceVersion, map[string]interface{}{"public_updated": true})
			klog.Infof("镜像 %s 已设置为 public", req.Name)
			return nil
		}
	}

	return fmt.Errorf("更新远端镜像 %s 失败", name)
}

func (s *ServerController) UpdateImageStatus(ctx context.Context, req *types.UpdateImageStatusRequest) error {
	old, err := s.factory.Image().Get(ctx, req.ImageId, false)
	if err != nil {
		klog.Errorf("获取镜像(%d)失败: %v", req.ImageId, err)
		return err
	}

	if !old.PublicUpdated {
		if err := s.TryUpdateRemotePublic(ctx, req, old); err != nil {
			klog.Errorf("尝试设置华为仓库为 public 失败: %v", err)
		}
	} else {
		klog.Infof("镜像(%s)已更新过，跳过远程更新", old.Name)
	}

	parts := strings.Split(req.Target, ":")
	tag := parts[1]
	if err = s.factory.Image().UpdateTag(ctx, req.ImageId, tag, map[string]interface{}{"status": req.Status, "message": req.Message}); err != nil {
		klog.Errorf("更新镜像(%d)的版本(%s)状态失败:%v", req.ImageId, tag, err)
		return err
	}

	// 当状态已经变成完成时，更新镜像的修改时间
	if req.Status == types.SyncImageComplete {
		targetName := old.Name
		if strings.Contains(targetName, "/") {
			targetName = strings.ReplaceAll(targetName, "/", "$")
		}
		// 获取版本大小并同步
		newTag, err := SwrClient.ShowRepoTag(&swrmodel.ShowRepoTagRequest{
			Namespace:  HuaweiNamespace,
			Repository: targetName,
			Tag:        tag,
		})
		if err != nil {
			klog.Warningf("获取远端新版本信息失败 %v", err)
			return nil
		}

		updates := make(map[string]interface{})
		updates["size"] = *newTag.Size
		updates["read_size"] = ByteSizeSimple(*newTag.Size)
		updates["digest"] = newTag.Digest
		updates["manifest"] = newTag.Manifest
		if err = s.factory.Image().UpdateTag(ctx, req.ImageId, tag, updates); err != nil {
			klog.Warningf("更新镜像版本失败 %v", err)
			return nil
		}

		// 重新镜像大小
		if err = s.CalculateImageSize(ctx, req.ImageId); err != nil {
			klog.Warningf("计算镜像大小失败 %v", err)
		}
		// 更新用户限额
		if err = s.CalculateUserQuota(ctx, req.ImageId); err != nil {
			klog.Warningf("更新用户限额失败 %v", err)
		}
	}
	return nil
}

func (s *ServerController) CalculateUserQuota(ctx context.Context, imageId int64) error {
	image, err := s.factory.Image().Get(ctx, imageId, false)
	if err != nil {
		return err
	}
	userObj, err := s.factory.Task().GetUser(ctx, image.UserId)
	if userObj.PaymentType == PackagePaymentType {
		// 包年包月无限制
		return nil
	}
	// 按量付费场景，目前仅有镜像个数，每次推送成功后，剩余次数 -1
	newCount := userObj.RemainCount - 1
	return s.factory.Task().UpdateUser(ctx, image.UserId, userObj.ResourceVersion, map[string]interface{}{
		"remain_count": newCount,
	})
}

func (s *ServerController) CalculateImageSize(ctx context.Context, imageId int64) error {
	imageInfo, err := s.factory.Image().Get(ctx, imageId, false)
	if err != nil {
		return err
	}

	var total int64
	for _, tag := range imageInfo.Tags {
		total = total + tag.Size
	}
	return s.factory.Image().Update(ctx, imageId, imageInfo.ResourceVersion, map[string]interface{}{
		"size":      total,
		"read_size": ByteSizeSimple(total),
	})
}

func (s *ServerController) afterDeleteImage(ctx context.Context, delImage *model.Image) {
	klog.Infof("开始延迟删除远端仓库镜像(%s)", delImage.Name)
	if !s.isDefaultRepo(delImage.RegisterId) {
		return
	}

	name := delImage.Name
	if strings.Contains(name, "/") {
		name = strings.ReplaceAll(name, "/", "$")
	}
	_, err := SwrClient.DeleteRepo(&swrmodel.DeleteRepoRequest{
		Namespace:  HuaweiNamespace,
		Repository: name,
	})
	if err != nil {
		klog.Warningf("删除远端镜像失败 %v", err)
	}

	klog.Infof("完成延迟删除远端仓库镜像(%s)", delImage.Name)
}

// DeleteImage 删除镜像和对应的tags
func (s *ServerController) DeleteImage(ctx context.Context, imageId int64) error {
	image, err := s.factory.Image().Get(ctx, imageId, false)
	if err != nil {
		klog.Errorf("获取镜像(%d)失败: %v", imageId, err)
	}
	// 检查 Lock 字段，如果为 true 则不允许删除
	if image.IsLocked {
		return fmt.Errorf("镜像被锁定，不允许删除")
	}

	if err = s.factory.Image().Delete(ctx, imageId); err != nil {
		return fmt.Errorf("删除镜像 %d 失败 %v", imageId, err)
	}
	// 软删除关联的tags
	if err = s.factory.Image().DeleteTagBy(ctx, db.WithImage(imageId)); err != nil {
		klog.Errorf("删除镜像(%d)关联版本失败，非主流程，忽略", imageId)
	}

	go s.afterDeleteImage(ctx, image)
	return nil
}

func (s *ServerController) ListImageTags(ctx context.Context, imageId int64, listOption types.ListOptions) (interface{}, error) {
	listOption.SetDefaultPageOption()

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}

	opts := []db.Options{ // 先写条件，再写排序，再偏移，再设置每页数量
		db.WithNameLike(listOption.NameSelector),
		db.WithImage(imageId),
		db.WithStatus(listOption.CustomStatus),
	}
	var err error

	pageResult.Total, err = s.factory.Image().TagCount(ctx, opts...)
	if err != nil {
		klog.Errorf("获取镜像Tag总数失败 %v", err)
		pageResult.Message = err.Error()
	}

	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithModifyOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	pageResult.Items, err = s.factory.Image().ListTags(ctx, opts...) // 改成不带版本列表以加速显示
	if err != nil {
		klog.Errorf("获取镜像 Tag 列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}
	return pageResult, nil
}

func (s *ServerController) SearchPublicImages(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
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
		db.WithPublic(),
	}

	// 判断用户类型
	if listOption.Trusted == 1 {
		opts = append(opts, db.WithOfficial())
	}
	if listOption.Trusted == 2 {
		opts = append(opts, db.WithNotOfficial())
	}

	var err error
	// 先获取总数
	pageResult.Total, err = s.factory.Image().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取镜像总数失败 %v", err)
		pageResult.Message = err.Error()
	}

	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithCreateOrderByASC(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	objects, err := s.factory.Image().ListWithTagsCount(ctx, opts...) // 优化成不带版本列表以加速显示
	if err != nil {
		klog.Errorf("获取镜像列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	// 添加默认 logo
	defaultLogo, _ := s.GetDefaultLogo(ctx)
	for index, obj := range objects {
		if len(obj.Logo) != 0 {
			continue
		}
		obj.Logo = defaultLogo
		objects[index] = obj
	}

	pageResult.Items = objects

	go s.AfterListImages(ctx, objects, 5*time.Second)
	return pageResult, nil
}

func (s *ServerController) ListImages(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	// 初始化分页属性
	listOption.SetDefaultPageOption()
	if listOption.ExtendLimit > 100 {
		listOption.Limit = listOption.ExtendLimit
	}

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}

	opts := []db.Options{ // 先写条件，再写排序，再偏移，再设置每页数量
		db.WithUser(listOption.UserId),
		db.WithNameLike(listOption.NameSelector),
		db.WithNamespace(listOption.Namespace),
	}
	var err error

	// 先获取总数
	pageResult.Total, err = s.factory.Image().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取镜像总数失败 %v", err)
		pageResult.Message = err.Error()
	}

	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithCreateOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	objects, err := s.factory.Image().ListWithTagsCount(ctx, opts...) // 优化成不带版本列表以加速显示
	if err != nil {
		klog.Errorf("获取镜像列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	// 添加默认 logo
	defaultLogo, _ := s.GetDefaultLogo(ctx)
	for index, obj := range objects {
		if len(obj.Logo) != 0 {
			continue
		}
		obj.Logo = defaultLogo
		objects[index] = obj
	}

	pageResult.Items = objects

	if listOption.ExtendLimit == 0 {
		go s.AfterListImages(ctx, objects, 5*time.Second)
	}
	return pageResult, nil
}

func (s *ServerController) ListImagesForClient(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	// 初始化分页属性
	listOption.SetDefaultPageOption()
	if listOption.ExtendLimit > 100 {
		listOption.Limit = listOption.ExtendLimit
	}

	opts := []db.Options{ // 先写条件，再写排序，再偏移，再设置每页数量
		db.WithUser(listOption.UserId),
		db.WithNameLike(listOption.NameSelector),
		db.WithNamespace(listOption.Namespace),
	}

	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithCreateOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	objects, err := s.factory.Image().ListWithTagsCount(ctx, opts...) // 优化成不带版本列表以加速显示
	if err != nil {
		return nil, err
	}
	if listOption.ExtendLimit == 0 {
		go s.AfterListImages(ctx, objects, 5*time.Second)
	}
	return objects, nil
}

func (s *ServerController) AfterListImages(ctx context.Context, objects []model.Image, interval time.Duration) {
	klog.Infof("开启延迟更新镜像列表属性，镜像数 %d", len(objects))
	for _, obj := range objects {
		time.Sleep(interval) // 延迟执行，避免被限速
		s.AfterGetImage(ctx, &obj)
	}
}

func (s *ServerController) SearchImages(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	labelIds, err := s.parseLabelIds(listOption.LabelIds)
	if err != nil {
		return nil, err
	}

	// 无需标签查询，直接获取全部公开镜像列表
	if len(labelIds) == 0 {
		klog.Infof("进行全量镜像搜索")
		return s.SearchPublicImages(ctx, listOption)
	}

	klog.Infof("进行标签镜像搜索")
	return s.SearchPublicImagesWithLabel(ctx, labelIds, listOption)
}

func (s *ServerController) SearchPublicImagesWithLabel(ctx context.Context, labelIds []int64, listOption types.ListOptions) (interface{}, error) {
	listOption.SetDefaultPageOption()

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}

	images, total, err := s.factory.Label().ListLabelPublicImages(ctx, labelIds, listOption.NameSelector, listOption.UserId, listOption.Trusted, listOption.Page, listOption.Limit)
	if err != nil {
		klog.Errorf("获取 label image 失败 %v", err)
		return nil, err
	}

	pageResult.Items = images
	pageResult.Total = total

	go s.AfterListImages(ctx, images, 5*time.Second)
	return pageResult, nil
}

func (s *ServerController) ListImagesByIds(ctx context.Context, ids []int64) (interface{}, error) {
	return s.factory.Image().List(ctx, db.WithIDIn(ids...))
}

func (s *ServerController) DeleteImagesByIds(ctx context.Context, ids []int64) error {
	return s.factory.Image().DeleteInBatch(ctx, ids)
}

func (s *ServerController) isDefaultRepo(regId int64) bool {
	return regId == *RegistryId
}

func (s *ServerController) isDurationExceeded(t time.Time, duration time.Duration) bool {
	now := time.Now()
	return now.Sub(t) > duration
}

func (s *ServerController) GetDefaultLogo(ctx context.Context) (string, error) {
	lg, err := s.factory.Label().GetLogo(ctx, db.WithName("default"))
	if err != nil {
		return "", err
	}
	return lg.Logo, nil
}

func (s *ServerController) GetImage(ctx context.Context, imageId int64) (interface{}, error) {
	object, err := s.factory.Image().GetImageWithTagsCount(ctx, imageId, false)
	if err != nil {
		return nil, err
	}

	// 如果 logo 不存在，获取默认 logo
	if len(object.Logo) == 0 {
		defaultLogo, err := s.GetDefaultLogo(ctx)
		if err == nil {
			object.Logo = defaultLogo
		}
	}

	// 尝试更新属性
	go s.AfterGetImage(ctx, object)
	return object, nil
}

// AfterGetImage 获取镜像后，尝试更新属性
func (s *ServerController) AfterGetImage(ctx context.Context, object *model.Image) {
	// 非官方内置仓库，无需更新
	if !s.isDefaultRepo(object.RegisterId) {
		klog.V(0).Infof("非官方镜像，无需同步")
		return
	}

	klog.Infof("镜像(%s)状态将被同步", object.Name)
	// 如果 30 分钟内已更新，则直接返回
	if !s.isDurationExceeded(object.LastSyncTime, 30*time.Minute) {
		klog.V(0).Infof("镜像(%s) 30分钟内已同步或者正在同步，忽略", object.Name)
		return
	}
	_ = s.factory.Image().UpdateWithoutLock(ctx, object.Id, map[string]interface{}{"last_sync_time": time.Now()})

	klog.Infof("即将开始同步镜像(%s/%s)的下载数", object.Namespace, object.Name)
	if err := s.SyncInfoByRemoteImage(ctx, object); err != nil {
		klog.Errorf("镜像 (%s/%s) SyncInfoByRemoteImage 失败 %v", object.Namespace, object.Name, err)
		return
	}
	klog.Infof("同步镜像(%s/%s)的下载数同步完成", object.Namespace, object.Name)
}

func (s *ServerController) UpdateTagFromRemote(ctx context.Context, newTag swrmodel.ShowReposTagResp, oldTag model.Tag) error {
	updates := make(map[string]interface{})
	if oldTag.Size != newTag.Size {
		updates["size"] = newTag.Size
		updates["read_size"] = ByteSizeSimple(newTag.Size)
	}
	if oldTag.Digest != newTag.Digest {
		updates["digest"] = newTag.Digest
	}
	if oldTag.Manifest != newTag.Manifest {
		updates["manifest"] = newTag.Manifest
	}

	if len(updates) == 0 {
		return nil
	}
	return s.factory.Image().UpdateTag(ctx, oldTag.ImageId, oldTag.Name, updates)
}

func (s *ServerController) UpdateImageInfoFromRemote(ctx context.Context, newImage *swrmodel.ShowRepositoryResponse, old *model.Image) error {
	updates := make(map[string]interface{})
	if old.Pull != *newImage.NumDownload {
		updates["pull"] = *newImage.NumDownload
	}
	if old.Size != *newImage.Size {
		updates["size"] = *newImage.Size
		updates["read_size"] = ByteSizeSimple(*newImage.Size)
	}

	if len(updates) == 0 {
		klog.Infof("无镜像变化，仅更新同步时间，避免频繁调用外部API")
		return nil
	}

	return s.factory.Image().UpdateWithoutLock(ctx, old.Id, updates)
}

// SyncInfoByRemoteImage 获取远端镜像属性且保存
func (s *ServerController) SyncInfoByRemoteImage(ctx context.Context, object *model.Image) error {
	targetName := object.Name
	if strings.Contains(targetName, "/") {
		targetName = strings.ReplaceAll(targetName, "/", "$")
	}
	oldTagMap := make(map[string]model.Tag)
	for _, t := range object.Tags {
		oldTagMap[t.Name] = t
	}

	// 获取并更新版本
	resp, err := SwrClient.ListRepositoryTags(&swrmodel.ListRepositoryTagsRequest{Namespace: HuaweiNamespace, Repository: targetName})
	if err != nil {
		klog.Errorf("获取远端镜像版本失败 %v", err)
		return nil
	}
	tags := *resp.Body
	for _, t := range tags {
		name := t.Tag
		old, ok := oldTagMap[name]
		if !ok {
			klog.V(1).Infof("远端镜像(%s)的版本(%s)未收录，忽略", targetName, name)
			continue
		}
		if err = s.UpdateTagFromRemote(ctx, t, old); err != nil {
			klog.Errorf("更新远端镜像版本至本地失败 %v", err)
			return err
		}
	}

	// 获取远端镜像属性
	newImage, err := SwrClient.ShowRepository(&swrmodel.ShowRepositoryRequest{Namespace: HuaweiNamespace, Repository: targetName})
	if err != nil {
		klog.Errorf("获取远端镜像详情失败 %v", err)
		return nil
	}
	if err = s.UpdateImageInfoFromRemote(ctx, newImage, object); err != nil {
		return err
	}

	return nil
}

func (s *ServerController) CreateImages(ctx context.Context, req *types.CreateImagesRequest) ([]model.Image, error) {
	klog.Infof("CreateImages, req %v", req)
	task, err := s.factory.Task().Get(ctx, req.TaskId)
	if err != nil {
		klog.Errorf("未传任务名，通过任务ID获取任务详情失败 %v", err)
		return nil, err
	}
	taskReq := &types.CreateTaskRequest{
		RegisterId:  task.RegisterId,
		Images:      req.Names,
		UserName:    task.UserName,
		UserId:      task.UserId,
		Namespace:   task.Namespace,
		PublicImage: task.IsPublic,
		IsOfficial:  task.IsOfficial,
		Logo:        task.Logo,
	}
	if err := s.CreateImageWithTag(ctx, req.TaskId, taskReq); err != nil {
		klog.Errorf("创建k8s镜像记录失败 :%v", err)
		return nil, fmt.Errorf("创建k8s镜像记录失败 :%v", err)
	}

	tags, err := s.factory.Image().ListTags(ctx, db.WithTaskLike(req.TaskId))
	if err != nil {
		klog.Errorf("获取k8s镜像tags失败 :%v", err)
		return nil, fmt.Errorf("获取k8s镜像tags失败 :%v", err)
	}
	klog.Infof("已完成k8s镜像创建 %v", tags)

	var imageIds []int64
	for _, tag := range tags {
		imageIds = append(imageIds, tag.ImageId)
	}

	images, err := s.factory.Image().ListImagesWithTag(ctx, db.WithIDIn(imageIds...))
	if err != nil {
		klog.Errorf("获取已创建的k8s镜像列表失败 :%v", err)
		return nil, fmt.Errorf("获取已创建的k8s镜像列表失败 :%v", err)
	}
	klog.Infof("创建 k8s 镜像成功, 镜像列表为 %v", images)

	return images, nil
}

func (s *ServerController) makeRemoteImageName(imageName string) string {
	if strings.Contains(imageName, "/") {
		imageName = strings.ReplaceAll(imageName, "/", "$")
	}
	return imageName
}

func (s *ServerController) afterDeleteImageTag(ctx context.Context, image *model.Image, delTag *model.Tag) {
	klog.Infof("开始延迟清理远程镜像版本%s:%s", image.Name, delTag.Name)
	if !s.isDefaultRepo(image.RegisterId) {
		return
	}

	request := &swrmodel.DeleteRepoTagRequest{
		Namespace:  HuaweiNamespace,
		Repository: s.makeRemoteImageName(image.Name),
		Tag:        delTag.Name,
	}
	_, err := SwrClient.DeleteRepoTag(request)
	if err != nil {
		klog.Errorf("删除远程镜像 %s tag(%s) 失败 %v", image.Name, delTag.Name, err)
		return
	}

	klog.Infof("完成延迟清理远程镜像版本%s:%s", image.Name, delTag.Name)
}

func (s *ServerController) DeleteImageTag(ctx context.Context, imageId int64, tagId int64) error {
	err := s.factory.Image().DeleteTag(ctx, tagId)
	if err != nil {
		return fmt.Errorf("删除镜像(%d) tag %d 失败:%v", imageId, tagId, err)
	}

	delTag, err := s.factory.Image().GetTag(ctx, tagId, true)
	if err != nil {
		klog.Errorf("获取已删除镜像(%d)的tag(%d) 失败: %v", imageId, tagId, err)
		return nil
	}
	image, err := s.factory.Image().Get(ctx, imageId, false)
	if err != nil {
		klog.Errorf("获取镜像(%d)的失败: %v", imageId, err)
		return nil
	}

	go s.afterDeleteImageTag(ctx, image, delTag)

	return s.CalculateImageSize(ctx, imageId)
}

func (s *ServerController) GetImageTag(ctx context.Context, imageId int64, tagId int64) (interface{}, error) {
	return s.factory.Image().GetTag(ctx, tagId, false)
}

func (s *ServerController) CreateNamespace(ctx context.Context, req *types.CreateNamespaceRequest) error {
	// 全局只能有一个命名空间
	_, err := s.factory.Image().GetNamespace(ctx, db.WithName(req.Name))
	if err == nil {
		return fmt.Errorf("命名空间(%s)已存在，无法重复创建", req.Name)
	}

	// 执行创建
	_, err = s.factory.Image().CreateNamespace(ctx, &model.Namespace{
		Name:        req.Name,
		Logo:        req.Logo,
		Label:       req.Label,
		Description: req.Description,
	})
	if err != nil {
		klog.Errorf("创建命名空间失败 %v", err)
		return err
	}
	return nil
}

func (s *ServerController) DeleteNamespace(ctx context.Context, objectId int64) error {
	if err := s.factory.Image().DeleteNamespace(ctx, objectId); err != nil {
		return fmt.Errorf("删除命名空间 %d 失败 %v", objectId, err)
	}

	return nil
}

func (s *ServerController) UpdateNamespace(ctx context.Context, req *types.UpdateNamespaceRequest) error {
	updates := make(map[string]interface{})
	updates["description"] = req.Description
	updates["logo"] = req.Logo
	updates["label"] = req.Label
	return s.factory.Image().UpdateNamespace(ctx, req.Id, req.ResourceVersion, updates)
}

func (s *ServerController) SyncNamespace(ctx context.Context, req *types.SyncNamespaceRequest) error {
	obj, err := s.factory.Image().GetNamespace(ctx, db.WithId(req.Id))
	if err != nil {
		return fmt.Errorf("无法获取命名空间")
	}
	if obj.Name == defaultNamespace {
		return fmt.Errorf("默认空间无法通过命名空间同步")
	}

	switch req.SyncType {
	case types.SyncNamespaceLogoType:
		return s.syncNamespaceLogo(ctx, req, obj)
	case types.SyncNamespaceLabelType:
		return s.syncNamespaceLabel(ctx, req, obj)
	case types.SyncNamespaceDefaultLogoType:
		return s.syncNamespaceDefaultLogo(ctx, req, obj)
	default:
		return fmt.Errorf("unsupported sync type %d", req.SyncType)
	}
}

func (s *ServerController) syncNamespaceLogo(ctx context.Context, req *types.SyncNamespaceRequest, ns *model.Namespace) error {
	opts := []db.Options{db.WithNamespace(ns.Name)}
	if !req.Rewrite {
		opts = append(opts, db.WithEmptyLogo())
	}

	return s.factory.Image().UpdateImagesLogo(ctx, map[string]interface{}{"logo": ns.Logo}, opts...)
}

func (s *ServerController) syncNamespaceDefaultLogo(ctx context.Context, req *types.SyncNamespaceRequest, ns *model.Namespace) error {
	// 如果logo已经存在
	if len(ns.Logo) != 0 {
		// 如果不是覆盖则直接返回
		if !req.Rewrite {
			return nil
		}
	}

	name := ns.Name
	logoObj, err := s.factory.Label().GetLogo(ctx, db.WithName(name))
	if err != nil {
		klog.Errorf("获取Logo %s 失败 %v", name, err)
		return fmt.Errorf("获取Logo(%s)失败", name)
	}
	if ns.Logo != logoObj.Logo {
		updates := make(map[string]interface{})
		updates["logo"] = logoObj.Logo
		return s.factory.Image().UpdateNamespace(ctx, ns.Id, ns.ResourceVersion, updates)
	}

	return nil
}

func (s *ServerController) syncNamespaceLabel(ctx context.Context, req *types.SyncNamespaceRequest, ns *model.Namespace) error {
	return nil
}

func (s *ServerController) ListNamespaces(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
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
	}

	var err error
	pageResult.Total, err = s.factory.Image().GetNamespaceCount(ctx, opts...)
	if err != nil {
		klog.Errorf("获取命名空间总数失败 %v", err)
		pageResult.Message = err.Error()
	}
	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithModifyOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	objects, err := s.factory.Image().ListNamespaces(ctx, opts...)
	if err != nil {
		klog.Errorf("获取命名空间失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	defaultLogo, _ := s.GetDefaultLogo(ctx)
	for index, obj := range objects {
		if len(obj.Logo) != 0 {
			continue
		}
		obj.Logo = defaultLogo
		objects[index] = obj
	}

	pageResult.Items = objects
	return pageResult, nil
}

func (s *ServerController) BindImageLabels(ctx context.Context, imageId int64, req types.BindImageLabels) error {
	switch req.OP {
	case types.AddImageLabelType:
		klog.Infof("开始新增标签 %v", req.LabelIds)
		for _, labelId := range req.LabelIds {
			if err := s.AddImageLabel(ctx, imageId, labelId); err != nil {
				return err
			}
		}
	case types.DeleteImageLabelType:
		klog.Infof("开始移除标签 %v", req.LabelIds)
		for _, labelId := range req.LabelIds {
			if err := s.DeleteImageLabel(ctx, imageId, labelId); err != nil {
				return err
			}
		}
	case types.IdempotentImageLabelType:
		klog.Infof("幂等标签 %v", req.LabelIds)
		// 判断标签是否需要移除，是则先移除，然后全量新增
		olds, err := s.factory.Label().ListImageLabels(ctx, db.WithImage(imageId))
		if err != nil {
			return err
		}

		newMap := make(map[int64]bool)
		for _, newLabel := range req.LabelIds {
			newMap[newLabel] = true
		}
		for _, old := range olds {
			// 移除
			if !newMap[old.LabelID] {
				if err = s.DeleteImageLabel(ctx, imageId, old.LabelID); err != nil {
					return err
				}
			}
		}

		// 全量新增，如果存在自动跳过
		for _, labelId := range req.LabelIds {
			if err = s.AddImageLabel(ctx, imageId, labelId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ServerController) AddImageLabel(ctx context.Context, imageId int64, labelId int64) error {
	_, err := s.factory.Label().GetImageLabel(ctx, db.WithLabel(labelId), db.WithImage(imageId))
	if err == nil {
		klog.Infof("镜像已绑定该标签，无需重复添加")
		return nil
	}

	if !errors.IsNotFound(err) {
		klog.Errorf("查询镜像标签关联失败 %v", err)
		return err
	}
	if _, err = s.factory.Label().CreateImageLabel(ctx, &model.ImageLabel{ImageID: imageId, LabelID: labelId}); err != nil {
		klog.Errorf("添加镜像标签失败 %v", err)
		return err
	}

	return nil
}

func (s *ServerController) DeleteImageLabel(ctx context.Context, imageId int64, labelId int64) error {
	if err := s.factory.Label().DeleteImageLabel(ctx, db.WithLabel(labelId), db.WithImage(imageId)); err != nil {
		klog.Errorf("删除镜像标签关联失败 %v", err)
		return err
	}
	return nil
}

func (s *ServerController) ListImageLabels(ctx context.Context, imageId int64, listOption types.ListOptions) (interface{}, error) {
	return s.factory.Label().ListImageLabelsV2(ctx, imageId)
}
