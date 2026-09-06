package rainbow

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util"
	"github.com/caoyingjunz/rainbow/pkg/util/errors"
	"github.com/caoyingjunz/rainbow/pkg/util/loginlimit"
	"github.com/caoyingjunz/rainbow/pkg/util/passwordutil"
	"github.com/caoyingjunz/rainbow/pkg/util/tokenutil"
)

func parseTime(t string) (time.Time, error) {
	pt, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return time.Time{}, fmt.Errorf("解析时间(%s)失败: %v", t, err)
	}

	return pt, nil
}

// Login 账号密码登录，签发 JWT
func (s *ServerController) Login(ctx context.Context, req *types.LoginRequest) (*types.LoginResponse, error) {
	// 用户名维度锁定探测限流
	if !loginlimit.AllowUserAttempt(req.Name) {
		return nil, fmt.Errorf("登录尝试过于频繁，请稍后再试")
	}

	// 按用户名查用户
	user, err := s.factory.Task().GetUserBy(ctx, db.WithName(req.Name))
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("用户不存在或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在或密码错误")
	}
	// 禁用态拦截
	if user.Status == model.UserStatusForbidden {
		return nil, fmt.Errorf("用户已被禁用")
	}
	// bcrypt 校验并发闸门（防爆破打满 CPU）
	if !loginlimit.AcquireVerify() {
		return nil, fmt.Errorf("登录尝试过于频繁，请稍后再试")
	}
	defer loginlimit.ReleaseVerify()

	// 校验密码；失败记录，成功清除
	if err := passwordutil.ValidatePassword(user.Password, req.Password); err != nil {
		loginlimit.RecordUserFailure(req.Name)
		return nil, fmt.Errorf("用户不存在或密码错误")
	}
	loginlimit.ClearUserFailures(req.Name)

	// 签发登录 JWT
	jwtKey := []byte(s.cfg.Server.JWTKey)
	if len(jwtKey) == 0 {
		return nil, fmt.Errorf("服务端未配置 jwt_key，无法登录")
	}
	token, err := tokenutil.GenerateLoginToken(user.UserId, user.Name, user.Role, jwtKey)
	if err != nil {
		return nil, fmt.Errorf("生成登录 token 失败: %v", err)
	}

	return &types.LoginResponse{
		Token:    token,
		UserId:   user.UserId,
		UserName: user.Name,
		Role:     user.Role,
	}, nil
}

// GetUserResources 当前用户可见资源（用户 → 角色 → 权限 → 资源）
// 管理员角色返回全部资源；普通用户按 RBAC 映射链查询
func (s *ServerController) GetUserResources(ctx context.Context, userId string, role int) ([]model.Resource, error) {
	if role == model.RoleAdmin {
		return s.factory.Rbac().ListResources(ctx)
	}
	return s.factory.Rbac().ListResourcesByUser(ctx, userId)
}

func (s *ServerController) isUserExist(ctx context.Context, userId string) (bool, error) {
	_, err := s.factory.Task().GetUser(ctx, userId)
	if err == nil {
		return true, nil
	}
	if errors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (s *ServerController) preCreateOrUpdateUser(ctx context.Context, user *types.CreateUserRequest) error {
	if user.PaymentType == PackagePaymentType {
		if user.ExpireTime == nil {
			return fmt.Errorf("包年包月类型时，结束时间为必填项")
		}
	}
	return nil
}

func (s *ServerController) CreateOrUpdateUser(ctx context.Context, user *types.CreateUserRequest) error {
	if err := s.preCreateOrUpdateUser(ctx, user); err != nil {
		return err
	}

	old, err := s.factory.Task().GetUser(ctx, user.UserId)
	if err == nil {
		// 同步用户时，忽略已存在用户
		if user.Sync {
			return nil
		}

		// 用户已存在，则更新
		updates := make(map[string]interface{})

		if old.Name != user.Name {
			updates["name"] = user.Name
		}
		if old.UserType != user.UserType {
			updates["user_type"] = user.UserType
		}
		if old.PaymentType != user.PaymentType {
			updates["payment_type"] = user.PaymentType
		}
		if old.Role != user.Role {
			updates["role"] = user.Role
		}
		if old.Email != user.Email {
			updates["email"] = user.Email
		}

		if user.PaymentType == PackagePaymentType {
			et, err := parseTime(*user.ExpireTime)
			if err != nil {
				klog.Errorf("%v", err)
				return err
			}
			if old.ExpireTime != et {
				updates["expire_time"] = et
			}
		} else {
			if old.RemainCount != user.RemainCount {
				updates["remain_count"] = user.RemainCount
			}
		}
		// 无变更时，直接退出
		if len(updates) == 0 {
			return nil
		}
		return s.factory.Task().UpdateUser(ctx, user.UserId, old.ResourceVersion, updates)
	} else {
		if !errors.IsNotFound(err) {
			return err
		}
		// 用户不存在，则创建
		if err = s.CreateUser(ctx, user); err != nil {
			return err
		}
	}

	return nil
}

func (s *ServerController) CreateOrUpdateUsers(ctx context.Context, req *types.CreateUsersRequest) error {
	for _, user := range req.Users {
		if err := s.CreateOrUpdateUser(ctx, &user); err != nil {
			return err
		}
	}

	return nil
}

func (s *ServerController) CreateUser(ctx context.Context, req *types.CreateUserRequest) error {
	// 安全：初始密码必填，bcrypt 加密存储
	if len(req.Password) == 0 {
		return fmt.Errorf("创建用户必须设置初始密码")
	}
	hash, err := passwordutil.EncryptPassword(req.Password)
	if err != nil {
		klog.Errorf("加密用户密码失败 %v", err)
		return err
	}

	userId := req.UserId
	if len(userId) == 0 {
		generated, gErr := util.GenerateAK("rainbow")
		if gErr != nil {
			return gErr
		}
		userId = generated
	}

	obj := &model.User{
		Name:        req.Name,
		UserId:      userId,
		Password:    hash,
		Status:      model.UserStatusNormal,
		UserType:    req.UserType,
		PaymentType: req.PaymentType,
		Role:        req.Role,
		Email:       req.Email,
	}

	if req.Sync {
		obj.PaymentType = DemandPaymentType
		obj.RemainCount = defaultRemainCount
	} else {
		if req.PaymentType == PackagePaymentType {
			et, err := parseTime(*req.ExpireTime)
			if err != nil {
				klog.Errorf("%v", err)
				return err
			}
			obj.ExpireTime = et
		} else {
			obj.RemainCount = req.RemainCount
		}
	}

	if err := s.factory.Task().CreateUser(ctx, obj); err != nil {
		klog.Errorf("创建用户 %s 失败 %v", req.Name, err)
		return err
	}

	return nil
}

func (s *ServerController) ListUsers(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
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
	if listOption.Role == 0 || listOption.Role == 1 {
		opts = append(opts, db.WithRole(listOption.Role))
	}
	if listOption.UserType == 0 || listOption.UserType == 1 {
		opts = append(opts, db.WithUserType(listOption.UserType))
	}
	if listOption.PaymentType == 0 || listOption.PaymentType == 1 {
		opts = append(opts, db.WithPaymentType(listOption.PaymentType))
	}

	var err error
	pageResult.Total, err = s.factory.Task().CountUsers(ctx, opts...)
	if err != nil {
		klog.Errorf("获取用户总数失败 %v", err)
		pageResult.Message = err.Error()
	}
	offset := (listOption.Page - 1) * listOption.Limit
	opts = append(opts, []db.Options{
		db.WithCreateOrderByDesc(),
		db.WithOffset(offset),
		db.WithLimit(listOption.Limit),
	}...)
	pageResult.Items, err = s.factory.Task().ListUsers(ctx, opts...)
	if err != nil {
		klog.Errorf("获取用户列表失败 %v", err)
		pageResult.Message = err.Error()
		return pageResult, err
	}

	return pageResult, nil
}

func (s *ServerController) GetUser(ctx context.Context, userId string) (*model.User, error) {
	return s.factory.Task().GetUser(ctx, userId)
}

func (s *ServerController) UpdateUser(ctx context.Context, req *types.UpdateUserRequest) error {
	return s.factory.Task().UpdateUser(ctx, req.UserId, req.ResourceVersion, map[string]interface{}{"name": req.Name, "user_type": req.UserType, "expire_time": req.ExpireTime})
}

func (s *ServerController) DeleteUser(ctx context.Context, userId string) error {
	return s.factory.Task().DeleteUser(ctx, userId)
}
