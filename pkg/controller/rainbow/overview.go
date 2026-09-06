package rainbow

import (
	"context"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/model"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	rainbowmodel "github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
)

func (s *ServerController) GetCollection(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	var (
		taskNum   int64
		imageNum  int64
		reviewNum int64
		err       error
	)

	taskNum, err = s.factory.Task().Count(ctx)
	if err != nil {
		klog.Errorf("获取任务数量失败: %v", err)
	}
	imageNum, err = s.factory.Image().Count(ctx)
	if err != nil {
		klog.Errorf("获取镜像数量失败: %v", err)
	}

	reviews, err := s.factory.Task().ListReview(ctx)
	if err != nil {
		klog.Errorf("获取历史浏览数量失败: %v", err)
	}
	for _, review := range reviews {
		reviewNum = +review.Count
	}
	day, err := s.factory.Task().CountDailyReview(ctx)
	if err != nil {
		klog.Errorf("获取当天浏览数量失败: %v", err)
	}
	reviewNum = reviewNum + day

	return map[string]int64{"tasks": taskNum, "images": imageNum, "review": reviewNum}, nil
}

// AddDailyReview 单纯做记录，偶尔的报错可忽略
func (s *ServerController) AddDailyReview(ctx context.Context, page string) error {
	_ = s.factory.Task().AddDailyReview(ctx, &rainbowmodel.Daily{Page: page})
	return nil
}

func (s *ServerController) Overview(ctx context.Context) (interface{}, error) {
	if SwrClient == nil {
		return nil, nil
	}

	return SwrClient.ShowDomainOverview(&model.ShowDomainOverviewRequest{})
}

func (s *ServerController) Downflow(ctx context.Context) (interface{}, error) {
	if SwrClient == nil {
		return nil, nil
	}

	return SwrClient.ShowDomainResourceReports(&model.ShowDomainResourceReportsRequest{
		ResourceType: model.GetShowDomainResourceReportsRequestResourceTypeEnum().DOWNFLOW,
		Frequency:    model.GetShowDomainResourceReportsRequestFrequencyEnum().DAILY,
	})

}
func (s *ServerController) Store(ctx context.Context) (interface{}, error) {
	if SwrClient == nil {
		return nil, nil
	}

	return SwrClient.ShowDomainResourceReports(&model.ShowDomainResourceReportsRequest{
		ResourceType: model.GetShowDomainResourceReportsRequestResourceTypeEnum().STORE,
		Frequency:    model.GetShowDomainResourceReportsRequestFrequencyEnum().DAILY,
	})
}

func (s *ServerController) ImageDownflow(ctx context.Context, downflowMeta types.DownflowMeta) (interface{}, error) {
	_, err := s.factory.Image().List(ctx, db.WithId(downflowMeta.ImageId))
	if err != nil {
		return nil, err
	}

	return nil, nil
}
