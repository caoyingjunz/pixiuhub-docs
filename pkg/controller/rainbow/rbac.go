package rainbow

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
)

// CreateRole 创建角色
func (s *ServerController) CreateRole(ctx context.Context, req *types.CreateRoleRequest) (*model.Role, error) {
	role, err := s.factory.Rbac().CreateRole(ctx, &model.Role{
		Name:        req.Name,
		Description: req.Description,
		RoleStatus:  req.RoleStatus,
		Editable:    true,
	})
	if err != nil {
		klog.Errorf("创建角色 %s 失败 %v", req.Name, err)
		return nil, fmt.Errorf("创建角色失败: %v", err)
	}
	return role, nil
}

// UpdateRole 更新角色
func (s *ServerController) UpdateRole(ctx context.Context, req *types.UpdateRoleRequest) error {
	updates := make(map[string]interface{})
	if len(req.Description) != 0 {
		updates["description"] = req.Description
	}
	if req.RoleStatus != nil {
		updates["role_status"] = *req.RoleStatus
	}
	if len(updates) == 0 {
		return nil
	}
	if err := s.factory.Rbac().UpdateRole(ctx, req.Id, updates); err != nil {
		klog.Errorf("更新角色 %d 失败 %v", req.Id, err)
		return fmt.Errorf("更新角色失败: %v", err)
	}
	return nil
}

// DeleteRole 删除角色
func (s *ServerController) DeleteRole(ctx context.Context, roleId int64) error {
	if err := s.factory.Rbac().DeleteRole(ctx, roleId); err != nil {
		klog.Errorf("删除角色 %d 失败 %v", roleId, err)
		return fmt.Errorf("删除角色失败: %v", err)
	}
	return nil
}

// ListRoles 角色列表
func (s *ServerController) ListRoles(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	roles, err := s.factory.Rbac().ListRoles(ctx)
	if err != nil {
		klog.Errorf("获取角色列表失败 %v", err)
		return nil, fmt.Errorf("获取角色列表失败: %v", err)
	}
	pageResult := types.PageResult{
		PageRequest: types.PageRequest{Page: listOption.Page, Limit: listOption.Limit},
		Total:       int64(len(roles)),
		Items:       roles,
	}
	return pageResult, nil
}

// AssignUserRole 为用户绑定角色
func (s *ServerController) AssignUserRole(ctx context.Context, userId string, roleId int64) error {
	if err := s.factory.Rbac().CreateUserRoleMapping(ctx, userId, roleId); err != nil {
		klog.Errorf("为用户 %s 绑定角色 %d 失败 %v", userId, roleId, err)
		return fmt.Errorf("绑定角色失败: %v", err)
	}
	return nil
}

// RemoveUserRole 移除用户角色
func (s *ServerController) RemoveUserRole(ctx context.Context, userId string, roleId int64) error {
	if err := s.factory.Rbac().DeleteUserRoleMapping(ctx, userId, roleId); err != nil {
		klog.Errorf("移除用户 %s 角色 %d 失败 %v", userId, roleId, err)
		return fmt.Errorf("移除角色失败: %v", err)
	}
	return nil
}

// ListPermissions 权限列表
func (s *ServerController) ListPermissions(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	permissions, err := s.factory.Rbac().ListPermissions(ctx)
	if err != nil {
		klog.Errorf("获取权限列表失败 %v", err)
		return nil, fmt.Errorf("获取权限列表失败: %v", err)
	}
	return types.PageResult{
		PageRequest: types.PageRequest{Page: listOption.Page, Limit: listOption.Limit},
		Total:       int64(len(permissions)),
		Items:       permissions,
	}, nil
}

// ListResources 资源列表
func (s *ServerController) ListResources(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	resources, err := s.factory.Rbac().ListResources(ctx)
	if err != nil {
		klog.Errorf("获取资源列表失败 %v", err)
		return nil, fmt.Errorf("获取资源列表失败: %v", err)
	}
	return types.PageResult{
		PageRequest: types.PageRequest{Page: listOption.Page, Limit: listOption.Limit},
		Total:       int64(len(resources)),
		Items:       resources,
	}, nil
}
