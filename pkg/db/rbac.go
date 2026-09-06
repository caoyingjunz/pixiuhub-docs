package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/caoyingjunz/rainbow/pkg/db/model"
)

// RbacInterface RBAC 数据访问：角色 / 权限 / 资源 / 三张映射表
type RbacInterface interface {
	// 角色
	CreateRole(ctx context.Context, object *model.Role) (*model.Role, error)
	UpdateRole(ctx context.Context, roleId int64, updates map[string]interface{}) error
	DeleteRole(ctx context.Context, roleId int64) error
	GetRole(ctx context.Context, roleId int64) (*model.Role, error)
	ListRoles(ctx context.Context, opts ...Options) ([]model.Role, error)

	// 权限
	CreatePermission(ctx context.Context, object *model.Permission) (*model.Permission, error)
	DeletePermission(ctx context.Context, permissionId int64) error
	ListPermissions(ctx context.Context, opts ...Options) ([]model.Permission, error)

	// 资源
	CreateResource(ctx context.Context, object *model.Resource) (*model.Resource, error)
	DeleteResource(ctx context.Context, resourceId int64) error
	ListResources(ctx context.Context, opts ...Options) ([]model.Resource, error)

	// 映射
	CreateUserRoleMapping(ctx context.Context, userId string, roleId int64) error
	DeleteUserRoleMapping(ctx context.Context, userId string, roleId int64) error
	ListRoleIdsByUser(ctx context.Context, userId string) ([]int64, error)

	CreateRolePerMapping(ctx context.Context, roleId, permissionId int64) error
	DeleteRolePerMapping(ctx context.Context, roleId, permissionId int64) error
	ListPermissionIdsByRole(ctx context.Context, roleId int64) ([]int64, error)

	CreatePerResMapping(ctx context.Context, permissionId, resourceId int64) error
	DeletePerResMapping(ctx context.Context, permissionId, resourceId int64) error
	ListResourceIdsByPermission(ctx context.Context, permissionId int64) ([]int64, error)

	// 用户可见资源链查询：用户 → 角色 → 权限 → 资源
	ListResourcesByUser(ctx context.Context, userId string) ([]model.Resource, error)
}

type rbac struct {
	db *gorm.DB
}

func newRbac(db *gorm.DB) RbacInterface {
	return &rbac{db: db}
}

func (r *rbac) CreateRole(ctx context.Context, object *model.Role) (*model.Role, error) {
	now := time.Now()
	object.GmtCreate = now
	object.GmtModified = now

	if err := r.db.WithContext(ctx).Create(object).Error; err != nil {
		return nil, err
	}
	return object, nil
}

func (r *rbac) UpdateRole(ctx context.Context, roleId int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", roleId).Updates(updates).Error
}

func (r *rbac) DeleteRole(ctx context.Context, roleId int64) error {
	return r.db.WithContext(ctx).Delete(&model.Role{}, roleId).Error
}

func (r *rbac) GetRole(ctx context.Context, roleId int64) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, roleId).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *rbac) ListRoles(ctx context.Context, opts ...Options) ([]model.Role, error) {
	var list []model.Role
	tx := r.db.WithContext(ctx)
	for _, opt := range opts {
		tx = opt(tx)
	}
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *rbac) CreatePermission(ctx context.Context, object *model.Permission) (*model.Permission, error) {
	now := time.Now()
	object.GmtCreate = now
	object.GmtModified = now

	if err := r.db.WithContext(ctx).Create(object).Error; err != nil {
		return nil, err
	}
	return object, nil
}

func (r *rbac) DeletePermission(ctx context.Context, permissionId int64) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, permissionId).Error
}

func (r *rbac) ListPermissions(ctx context.Context, opts ...Options) ([]model.Permission, error) {
	var list []model.Permission
	tx := r.db.WithContext(ctx)
	for _, opt := range opts {
		tx = opt(tx)
	}
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *rbac) CreateResource(ctx context.Context, object *model.Resource) (*model.Resource, error) {
	now := time.Now()
	object.GmtCreate = now
	object.GmtModified = now

	if err := r.db.WithContext(ctx).Create(object).Error; err != nil {
		return nil, err
	}
	return object, nil
}

func (r *rbac) DeleteResource(ctx context.Context, resourceId int64) error {
	return r.db.WithContext(ctx).Delete(&model.Resource{}, resourceId).Error
}

func (r *rbac) ListResources(ctx context.Context, opts ...Options) ([]model.Resource, error) {
	var list []model.Resource
	tx := r.db.WithContext(ctx)
	for _, opt := range opts {
		tx = opt(tx)
	}
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *rbac) CreateUserRoleMapping(ctx context.Context, userId string, roleId int64) error {
	mapping := &model.UserRoleMapping{
		UserId: userId,
		RoleId: roleId,
	}
	now := time.Now()
	mapping.GmtCreate = now
	mapping.GmtModified = now
	return r.db.WithContext(ctx).Create(mapping).Error
}

func (r *rbac) DeleteUserRoleMapping(ctx context.Context, userId string, roleId int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userId, roleId).
		Delete(&model.UserRoleMapping{}).Error
}

func (r *rbac) ListRoleIdsByUser(ctx context.Context, userId string) ([]int64, error) {
	var mappings []model.UserRoleMapping
	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Find(&mappings).Error; err != nil {
		return nil, err
	}
	roleIds := make([]int64, 0, len(mappings))
	for _, m := range mappings {
		roleIds = append(roleIds, m.RoleId)
	}
	return roleIds, nil
}

func (r *rbac) CreateRolePerMapping(ctx context.Context, roleId, permissionId int64) error {
	mapping := &model.RolePerMapping{
		RoleId:       roleId,
		PermissionId: permissionId,
	}
	now := time.Now()
	mapping.GmtCreate = now
	mapping.GmtModified = now
	return r.db.WithContext(ctx).Create(mapping).Error
}

func (r *rbac) DeleteRolePerMapping(ctx context.Context, roleId, permissionId int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleId, permissionId).
		Delete(&model.RolePerMapping{}).Error
}

func (r *rbac) ListPermissionIdsByRole(ctx context.Context, roleId int64) ([]int64, error) {
	var mappings []model.RolePerMapping
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleId).Find(&mappings).Error; err != nil {
		return nil, err
	}
	permissionIds := make([]int64, 0, len(mappings))
	for _, m := range mappings {
		permissionIds = append(permissionIds, m.PermissionId)
	}
	return permissionIds, nil
}

func (r *rbac) CreatePerResMapping(ctx context.Context, permissionId, resourceId int64) error {
	mapping := &model.PerResMapping{
		PermissionId: permissionId,
		ResourceId:   resourceId,
	}
	now := time.Now()
	mapping.GmtCreate = now
	mapping.GmtModified = now
	return r.db.WithContext(ctx).Create(mapping).Error
}

func (r *rbac) DeletePerResMapping(ctx context.Context, permissionId, resourceId int64) error {
	return r.db.WithContext(ctx).
		Where("permission_id = ? AND resource_id = ?", permissionId, resourceId).
		Delete(&model.PerResMapping{}).Error
}

func (r *rbac) ListResourceIdsByPermission(ctx context.Context, permissionId int64) ([]int64, error) {
	var mappings []model.PerResMapping
	if err := r.db.WithContext(ctx).Where("permission_id = ?", permissionId).Find(&mappings).Error; err != nil {
		return nil, err
	}
	resourceIds := make([]int64, 0, len(mappings))
	for _, m := range mappings {
		resourceIds = append(resourceIds, m.ResourceId)
	}
	return resourceIds, nil
}

// ListResourcesByUser 用户可见资源链查询：用户 → 角色 → 权限 → 资源
func (r *rbac) ListResourcesByUser(ctx context.Context, userId string) ([]model.Resource, error) {
	roleIds, err := r.ListRoleIdsByUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	if len(roleIds) == 0 {
		return []model.Resource{}, nil
	}

	var perMappings []model.RolePerMapping
	if err := r.db.WithContext(ctx).Where("role_id IN ?", roleIds).Find(&perMappings).Error; err != nil {
		return nil, err
	}
	if len(perMappings) == 0 {
		return []model.Resource{}, nil
	}
	permissionIds := make([]int64, 0, len(perMappings))
	for _, m := range perMappings {
		permissionIds = append(permissionIds, m.PermissionId)
	}

	var resMappings []model.PerResMapping
	if err := r.db.WithContext(ctx).Where("permission_id IN ?", permissionIds).Find(&resMappings).Error; err != nil {
		return nil, err
	}
	if len(resMappings) == 0 {
		return []model.Resource{}, nil
	}
	resourceIds := make([]int64, 0, len(resMappings))
	for _, m := range resMappings {
		resourceIds = append(resourceIds, m.ResourceId)
	}

	var resources []model.Resource
	if err := r.db.WithContext(ctx).Where("id IN ?", resourceIds).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}
