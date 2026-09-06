package rainbow

import (
	"context"
	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
	"github.com/caoyingjunz/rainbow/pkg/util"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util/encryptutil"
)

func (s *ServerController) CreateAccess(ctx context.Context, req *types.CreateAccessRequest) (*types.AccessResponse, error) {
	ak, err := util.GenerateAK("pixiu")
	if err != nil {
		klog.Errorf("创建ak失败 %v", err)
		return nil, err
	}
	sk, err := util.GenerateSK()
	if err != nil {
		klog.Errorf("创建sk失败 %v", err)
		return nil, err
	}

	// 安全加固：SecretKey 加密存储（密钥来自配置 encrypt_key，环境变量注入）
	encrypted := false
	storeSecret := sk
	if len(s.cfg.Server.EncryptKey) != 0 {
		if enc, err := encryptutil.Encrypt([]byte(sk), encryptutil.DeriveKey(s.cfg.Server.EncryptKey)); err != nil {
			klog.Errorf("加密 sk 失败 %v", err)
			return nil, err
		} else {
			storeSecret = enc
			encrypted = true
		}
	}

	obj := &model.Access{
		UserModel: rainbow.UserModel{
			UserId: req.UserId,
		},
		AccessKey:       ak,
		SecretKey:       storeSecret,
		SecretEncrypted: encrypted,
	}
	if len(req.UserName) == 0 {
		userObj, err := s.factory.Task().GetUser(ctx, req.UserId)
		if err == nil {
			obj.UserName = userObj.Name
		}
	}

	if req.ExpireTime != nil {
		expireTime, err := parseTime(*req.ExpireTime)
		if err != nil {
			klog.Errorf("解析 ak/sk 过期时间失败: %v", err)
			return nil, err
		}
		obj.ExpireTime = &expireTime
	}

	if _, err = s.factory.Access().Create(ctx, obj); err != nil {
		klog.Errorf("创建 ak/sk 失败 %v", err)
		return nil, err
	}

	// 明文 SK 仅在创建时一次性返回，之后列表接口仅返回脱敏值
	return &types.AccessResponse{
		AccessKey: ak,
		SecretKey: sk,
	}, nil
}

func (s *ServerController) DeleteAccess(ctx context.Context, ak string) error {
	return s.factory.Access().Delete(ctx, ak)
}

func (s *ServerController) ListAccesses(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	listOption.SetDefaultPageOption()

	pageResult := types.PageResult{
		PageRequest: types.PageRequest{
			Page:  listOption.Page,
			Limit: listOption.Limit,
		},
	}
	opts := []db.Options{
		db.WithUser(listOption.UserId),
		db.WithAccessKeyLike(listOption.NameSelector),
	}

	var err error
	pageResult.Total, err = s.factory.Access().Count(ctx, opts...)
	if err != nil {
		klog.Errorf("获取 ak/sk 总数失败 %v", err)
		pageResult.Message = err.Error()
	}
	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithCreateOrderByASC(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)
	pageResult.Items, err = s.factory.Access().List(ctx, opts...)
	if err != nil {
		klog.Errorf("获取 ak/sk 列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	// 安全加固：接口返回对 SK 脱敏，不返回明文
	if items, ok := pageResult.Items.([]model.Access); ok {
		for i := range items {
			items[i].SecretKey = maskSecret(items[i].SecretKey)
		}
	}

	return pageResult, nil
}

// maskSecret 脱敏：保留首尾各 4 字符
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
