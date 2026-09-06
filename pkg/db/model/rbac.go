package model

import (
	"time"

	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
)

// RBAC 数据模型（参考 rainbow-java 的 lcap 权限体系）：
// 用户 --(user_role_mapping)-- 角色 --(role_per_mapping)-- 权限 --(per_res_mapping)-- 资源
// 资源即受保护的路径：菜单（type=page）与按钮/接口（type=component）

func init() {
	register(&Role{}, &Permission{}, &Resource{}, &UserRoleMapping{}, &RolePerMapping{}, &PerResMapping{})
}

// Role 角色
type Role struct {
	rainbow.Model

	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	RoleStatus  bool      `gorm:"default:true" json:"role_status"` // true 启用，false 禁用
	Editable    bool      `gorm:"default:true" json:"editable"`    // 内置角色不可编辑
	CreateTime  time.Time `gorm:"autoCreateTime" json:"create_time"`
}

func (r *Role) TableName() string {
	return "roles"
}

// Permission 权限（一组资源的集合）
type Permission struct {
	rainbow.Model

	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:varchar(255)" json:"description"`
}

func (p *Permission) TableName() string {
	return "permissions"
}

// Resource 资源（受保护路径，菜单或按钮/接口）
type Resource struct {
	rainbow.Model

	Name        string `gorm:"type:varchar(255);not null" json:"name"` // 资源路径，如 /api/v2/images
	Description string `gorm:"type:varchar(255)" json:"description"`
	Type        string `gorm:"type:varchar(32)" json:"type"`        // page / component
	ClientType  string `gorm:"type:varchar(32)" json:"client_type"` // pc / mobile
}

func (r *Resource) TableName() string {
	return "resources"
}

// UserRoleMapping 用户-角色关联
type UserRoleMapping struct {
	rainbow.Model

	UserId string `gorm:"type:varchar(255);index:idx_user_role" json:"user_id"`
	RoleId int64  `gorm:"index:idx_user_role" json:"role_id"`
}

func (m *UserRoleMapping) TableName() string {
	return "user_role_mappings"
}

// RolePerMapping 角色-权限关联
type RolePerMapping struct {
	rainbow.Model

	RoleId       int64 `gorm:"index:idx_role_per" json:"role_id"`
	PermissionId int64 `gorm:"index:idx_role_per" json:"permission_id"`
}

func (m *RolePerMapping) TableName() string {
	return "role_per_mappings"
}

// PerResMapping 权限-资源关联
type PerResMapping struct {
	rainbow.Model

	PermissionId int64 `gorm:"index:idx_per_res" json:"permission_id"`
	ResourceId   int64 `gorm:"index:idx_per_res" json:"resource_id"`
}

func (m *PerResMapping) TableName() string {
	return "per_res_mappings"
}
