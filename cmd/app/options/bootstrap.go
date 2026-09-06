package options

import (
	"context"
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/util"
	rainbowerrors "github.com/caoyingjunz/rainbow/pkg/util/errors"
	"github.com/caoyingjunz/rainbow/pkg/util/passwordutil"
)

const (
	defaultAdminName = "root"
	// 环境变量：首次启动时设置初始管理员账号密码（不设置则拒绝启动，避免空凭据弱配置）
	envRootUserName     = "RAINBOW_ROOT_USER"
	envRootUserPassword = "RAINBOW_ROOT_PASSWORD"
)

// bootstrap 首次启动引导：用户表为空时创建初始管理员账号
func (o *ServerOptions) bootstrap() error {
	// 用户表已有数据则跳过
	_, err := o.Factory.Task().GetUserBy(context.TODO())
	if err == nil {
		return nil
	}
	if !rainbowerrors.IsNotFound(err) {
		return fmt.Errorf("检查用户表失败: %v", err)
	}

	name := os.Getenv(envRootUserName)
	if len(name) == 0 {
		name = defaultAdminName
	}
	password := os.Getenv(envRootUserPassword)
	if len(password) == 0 {
		return fmt.Errorf("首次启动检测到用户表为空，请通过环境变量 %s 设置初始管理员密码（%s 可自定义用户名，默认 %s）",
			envRootUserPassword, envRootUserName, defaultAdminName)
	}

	hash, err := passwordutil.EncryptPassword(password)
	if err != nil {
		return fmt.Errorf("加密初始管理员密码失败: %v", err)
	}

	userId, err := util.GenerateAK("rainbow")
	if err != nil {
		return fmt.Errorf("生成初始管理员 UserId 失败: %v", err)
	}

	admin := &model.User{
		Name:     name,
		UserId:   userId,
		Role:     model.RoleAdmin,
		Password: hash,
		Status:   model.UserStatusNormal,
	}
	if err := o.Factory.Task().CreateUser(context.TODO(), admin); err != nil {
		return fmt.Errorf("创建初始管理员失败: %v", err)
	}

	// 初始化内置管理员角色并关联到账号（RBAC 链路：用户 → 角色 → 权限 → 资源）
	adminRole, err := o.Factory.Rbac().CreateRole(context.TODO(), &model.Role{
		Name:        "admin",
		Description: "内置管理员角色",
		RoleStatus:  true,
		Editable:    false,
	})
	if err != nil {
		return fmt.Errorf("创建内置管理员角色失败: %v", err)
	}
	if err := o.Factory.Rbac().CreateUserRoleMapping(context.TODO(), userId, adminRole.Id); err != nil {
		return fmt.Errorf("关联管理员角色失败: %v", err)
	}

	klog.Infof("已创建初始管理员账号(%s)并绑定 admin 角色，请及时修改默认密码", name)
	return nil
}
