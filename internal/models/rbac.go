package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	ModDashboard     = "dashboard"
	ModUsers         = "users"
	ModActivity      = "activity_analyzer"
	ModStatistics    = "statistics"
	ModSources       = "sources"
	ModRatings       = "ratings"
	ModWithdrawals   = "withdrawals"
	ModSettings      = "settings"
	ModHashes        = "hashes"
	ModRBAC          = "rbac_management"
	ModAdmins        = "administrator_management"
	ModNotifications = "notifications"
	ModLocalizations = "localizations"
)

const (
	PermRead       = "can_read"
	PermWrite      = "can_write"
	PermEdit       = "can_edit"
	PermDelete     = "can_delete"
	PermAddBalance = "can_add_balance"
)

type Module struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:50;uniqueIndex" json:"code"`
	Name        string    `gorm:"size:100" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Code        string       `gorm:"size:50;uniqueIndex" json:"code"`
	Name        string       `gorm:"size:100" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"`
	IsActive    bool         `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []RoleModule `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"permissions,omitempty"`
}

type RoleModule struct {
	RoleID        uint `gorm:"primaryKey;column:role_id" json:"role_id"`
	ModuleID      uint `gorm:"primaryKey;column:module_id" json:"module_id"`
	CanRead       bool `gorm:"default:false" json:"can_read"`
	CanWrite      bool `gorm:"default:false" json:"can_write"`
	CanEdit       bool `gorm:"default:false" json:"can_edit"`
	CanDelete     bool `gorm:"default:false" json:"can_delete"`
	CanAddBalance bool `gorm:"default:false;column:can_add_balance" json:"can_add_balance"`

	CreatedAt time.Time `json:"created_at"`
	Role      Role      `gorm:"foreignKey:RoleID" json:"-"`
	Module    Module    `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
}

type AdminUser struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PasswordHash string         `gorm:"size:255" json:"-"`
	Email        string         `gorm:"size:255;uniqueIndex" json:"email"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time     `json:"last_login_at"`
	CreatedBy    *uint          `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Roles        []Role         `gorm:"many2many:admin_user_roles;joinForeignKey:user_id;joinReferences:role_id" json:"roles,omitempty"`
}

type AdminUserRole struct {
	UserID    uint      `gorm:"primaryKey;column:user_id" json:"user_id"`
	RoleID    uint      `gorm:"primaryKey;column:role_id" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;column:user_id" json:"user_id"`
	ModuleCode string    `gorm:"size:50;index" json:"module_code"`
	Action     string    `gorm:"size:100" json:"action"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	UserAgent  string    `gorm:"type:text" json:"user_agent"`
	IsAllowed  bool      `json:"is_allowed"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminUserWithAccess struct {
	AdminUser
	IsSuperAdmin bool                       `json:"-"`
	ModuleCodes  map[string]bool            `json:"-"` // Fast lookup for modules user has any access to
	Permissions  map[string]map[string]bool `json:"-"`
}

func (u *AdminUserWithAccess) HasPermission(moduleCode string, perm string) bool {
	if u.IsSuperAdmin {
		return true
	}

	if u.Permissions == nil {
		return false
	}

	mod, ok := u.Permissions[moduleCode]
	if !ok {
		return false
	}
	if perm == PermRead {
		return mod[PermRead] || mod[PermWrite] || mod[PermEdit] || mod[PermDelete] || mod[PermAddBalance]
	}

	return mod[perm]
}

func (u *AdminUserWithAccess) HasAccess(moduleCode string) bool {
	return u.HasPermission(moduleCode, PermRead)
}

func (AdminUser) TableName() string     { return "admin_users" }
func (Role) TableName() string          { return "roles" }
func (Module) TableName() string        { return "modules" }
func (RoleModule) TableName() string    { return "role_modules" }
func (AdminUserRole) TableName() string { return "admin_user_roles" }
func (AccessLog) TableName() string     { return "access_logs" }
