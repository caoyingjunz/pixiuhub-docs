package rainbow

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
)

type PushMessage struct {
	Text    map[string]string `json:"text"`
	Msgtype string            `json:"msgtype"`
}

// TODO: 根据不同的类型，检查对应配置是否缺失
func (s *ServerController) preCreateNotify(ctx context.Context, req *types.CreateNotificationRequest) error {
	return nil
}

func (s *ServerController) UpdateNotify(ctx context.Context, req *types.UpdateNotificationRequest) error {
	pushCfg, err := req.PushCfg.Marshal()
	if err != nil {
		return err
	}
	return s.factory.Notify().Update(ctx, req.Id, req.ResourceVersion, map[string]interface{}{
		"name":       req.Name,
		"role":       req.Role,
		"enable":     req.Enable,
		"type":       req.Type,
		"short_desc": req.ShortDesc,
		"push_cfg":   pushCfg,
	})
}

func (s *ServerController) EnableNotify(ctx context.Context, req *types.UpdateNotificationRequest) error {
	return s.factory.Notify().Update(ctx, req.Id, req.ResourceVersion, map[string]interface{}{
		"enable": req.Enable,
	})
}

func (s *ServerController) CreateNotify(ctx context.Context, req *types.CreateNotificationRequest) error {
	if err := s.preCreateNotify(ctx, req); err != nil {
		return err
	}

	pushCfg, err := req.PushCfg.Marshal()
	if err != nil {
		return err
	}
	_, err = s.factory.Notify().Create(ctx, &model.Notification{
		Name:   req.Name,
		Role:   req.Role,
		Enable: req.Enable,
		Type:   req.Type,
		UserModel: rainbow.UserModel{
			UserId:   req.UserId,
			UserName: req.UserName,
		},
		PushCfg:   pushCfg,
		ShortDesc: req.ShortDesc,
	})
	if err != nil {
		klog.Errorf("创建推送(%s)记录失败: %v", req.Name, err)
	}

	return err
}

func (s *ServerController) GetNotify(ctx context.Context, notifyId int64) (interface{}, error) {
	no, err := s.factory.Notify().Get(ctx, notifyId)
	if err != nil {
		return nil, err
	}
	var pushCfg types.PushConfig
	if err = pushCfg.Unmarshal(no.PushCfg); err != nil {
		return nil, err
	}

	return types.NotificationResult{
		Id:              no.Id,
		GmtModified:     no.GmtModified,
		GmtCreate:       no.GmtCreate,
		ResourceVersion: no.ResourceVersion,
		CreateNotificationRequest: types.CreateNotificationRequest{
			UserMetaRequest: types.UserMetaRequest{
				UserName: no.UserName,
				UserId:   no.UserId,
			},
			Name:      no.Name,
			Role:      no.Role,
			Enable:    no.Enable,
			Type:      no.Type,
			PushCfg:   &pushCfg,
			ShortDesc: no.ShortDesc,
		},
	}, nil
}

func (s *ServerController) DeleteNotify(ctx context.Context, notifyId int64) error {
	return s.factory.Notify().Delete(ctx, notifyId)
}

func (s *ServerController) makeSendTpl(req *types.SendNotificationRequest) string {
	// 1. 构造推送内容模板
	tpl := ""
	if req.Role == types.SystemNotifyRole {
		tpl = fmt.Sprintf("PixiuHub通知\n同步时间: %v\n%s", time.Now().Format("2006-01-02 15:04:05"), req.Content)
		if len(req.Email) != 0 {
			tpl = fmt.Sprintf("%s\n用户名: %s\n时间: %v\nEmail: %s", req.ShortDesc, req.UserName, time.Now().Format("2006-01-02 15:04:05"), req.Email)
		}
	}
	if req.Role == types.UserNotifyRole {
		tpl = fmt.Sprintf("PixiuHub通知\n完成时间: %v\n%s", time.Now().Format("2006-01-02 15:04:05"), req.Content)
	}
	if req.DryRun {
		tpl = fmt.Sprintf("%s\nHello PixiuHub", "消息测试")
	}

	return tpl
}

func (s *ServerController) SendNotifyByType(ctx context.Context, req *types.SendNotificationRequest) error {
	var err error
	switch req.Type {
	case types.DingtalkNotifyType: // 企微和钉钉的推送格式相同
		err = s.send(ctx, req.PushCfg.Dingtalk.URL, PushMessage{Text: map[string]string{"content": s.makeSendTpl(req)}, Msgtype: "text"})
	case types.QiWeiNotifyType:
		err = s.send(ctx, req.PushCfg.QiWei.URL, PushMessage{Text: map[string]string{"content": s.makeSendTpl(req)}, Msgtype: "text"})
	default:
		return fmt.Errorf("unsupported message type %s", req.Type)
	}
	if err != nil {
		klog.Errorf("notify(%s) 类型(%s) 推送失败 %v", req.Name, req.Type, err)
		return err
	}

	return nil
}

func (s *ServerController) SendNotify(ctx context.Context, req *types.SendNotificationRequest) error {
	if req.DryRun {
		return s.SendNotifyByType(ctx, req)
	}

	opts := []db.Options{db.WithRole(req.Role), db.WithEnable(1)}
	if req.Role == types.UserNotifyRole {
		opts = append(opts, db.WithUser(req.UserId))
	}
	notifies, err := s.factory.Notify().List(ctx, opts...)
	if err != nil {
		klog.Errorf("获取 notify 失败 %v", err)
		return err
	}
	if len(notifies) == 0 {
		klog.Warningf("未发现(%s)已开启的通知通道，忽略本次通知", req.UserName)
		return nil
	}

	for _, notify := range notifies {
		var pushCfg types.PushConfig
		if err = pushCfg.Unmarshal(notify.PushCfg); err != nil {
			return err
		}

		switch notify.Type {
		case types.DingtalkNotifyType:
			err = s.send(ctx, pushCfg.Dingtalk.URL, PushMessage{Text: map[string]string{"content": s.makeSendTpl(req)}, Msgtype: "text"})
		case types.QiWeiNotifyType:
			err = s.send(ctx, pushCfg.QiWei.URL, PushMessage{Text: map[string]string{"content": s.makeSendTpl(req)}, Msgtype: "text"})
		default:
			return fmt.Errorf("unsupported message type %s", notify.Type)
		}
		if err != nil {
			klog.Errorf("notify(%s) 类型(%s) 推送失败 %v", notify.Name, notify.Type, err)
			return err
		}

		klog.Infof("notify(%s) 类型(%s) 推送成功", notify.Name, notify.Type)
	}
	return nil
}

func (s *ServerController) send(ctx context.Context, url string, val interface{}) error {
	http2 := util.NewHttpClient(5*time.Second, url)
	if err := http2.Post(url, nil, val, map[string]string{"Content-Type": "application/json"}); err != nil {
		klog.Errorf("发送请求 %s 失败 %v", url, err)
		return err
	}

	return nil
}

func (s *ServerController) ListNotifies(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	// 设置默认值
	listOption.SetDefaultPageOption()
	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}

	opts := []db.Options{ // 先写条件，再写排序，再偏移，再设置每页数量
		db.WithUser(listOption.UserId),
	}
	var err error
	pageResult.Total, err = s.factory.Notify().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取镜像总数失败 %v", err)
		pageResult.Message = err.Error()
	}

	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithModifyOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)

	nos, err := s.factory.Notify().List(ctx, opts...)
	if err != nil {
		klog.Errorf("获取镜像列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	var convertResults []types.NotificationResult
	for _, no := range nos {
		var pushCfg types.PushConfig
		if err = pushCfg.Unmarshal(no.PushCfg); err != nil {
			pageResult.Message = err.Error()
			return pageResult, err
		}

		convertResults = append(convertResults, types.NotificationResult{
			Id:              no.Id,
			GmtModified:     no.GmtModified,
			GmtCreate:       no.GmtCreate,
			ResourceVersion: no.ResourceVersion,
			CreateNotificationRequest: types.CreateNotificationRequest{
				UserMetaRequest: types.UserMetaRequest{
					UserName: no.UserName,
					UserId:   no.UserId,
				},
				Name:      no.Name,
				Role:      no.Role,
				Enable:    no.Enable,
				Type:      no.Type,
				PushCfg:   &pushCfg,
				ShortDesc: no.ShortDesc,
			},
		})
	}
	pageResult.Items = convertResults

	return pageResult, nil
}

type NotifyType struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *ServerController) GetNotifyTypes(ctx context.Context) (interface{}, error) {
	return []NotifyType{
		{
			Name:  "钉钉",
			Value: "dingtalk",
		},
		{
			Name:  "企微",
			Value: "qiwei",
		},
		//{
		//	Name:  "邮箱",
		//	Value: "email",
		//},
	}, nil
}
