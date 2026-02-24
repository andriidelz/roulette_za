package repository

import (
	"errors"
	"fmt"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrSystemRoleRequired = errors.New("cannot delete or modify a system-critical role")
	ErrPasswordSameAsOld  = errors.New("new password cannot be the same as the old password")
)

// ============== PRIVATE HELPERS ==============

func (r *PostgresRepository) hydrateAdminAccess(user models.AdminUser) *models.AdminUserWithAccess {
	userWithAccess := &models.AdminUserWithAccess{
		AdminUser:   user,
		Permissions: make(map[string]map[string]bool),
		ModuleCodes: make(map[string]bool),
	}

	for _, role := range user.Roles {
		if !role.IsActive {
			continue
		}

		if role.Code == "super_admin" {
			userWithAccess.IsSuperAdmin = true
		}

		for _, p := range role.Permissions {
			if p.Module.Code == "" {
				continue
			}

			if _, ok := userWithAccess.Permissions[p.Module.Code]; !ok {
				userWithAccess.Permissions[p.Module.Code] = make(map[string]bool)
			}

			m := userWithAccess.Permissions[p.Module.Code]
			if p.CanRead {
				m[models.PermRead] = true
			}
			if p.CanWrite {
				m[models.PermWrite] = true
			}
			if p.CanEdit {
				m[models.PermEdit] = true
			}
			if p.CanDelete {
				m[models.PermDelete] = true
			}
			if p.CanAddBalance {
				m[models.PermAddBalance] = true
			}
		}
	}

	return userWithAccess
}

func (r *PostgresRepository) GetAdminUserByUsername(email string) (*models.AdminUserWithAccess, error) {
	var user models.AdminUser

	err := r.db.Preload("Roles.Permissions.Module").
		Where("email = ? AND is_active = ?", email, true).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.hydrateAdminAccess(user), nil
}

func (r *PostgresRepository) GetAdminFullContext(userID uint) (*models.AdminUserWithAccess, error) {
	var user models.AdminUser

	err := r.db.Preload("Roles.Permissions.Module").
		Where("id = ? AND is_active = ?", userID, true).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.hydrateAdminAccess(user), nil
}

// ============== MODULES ==============

func (r *PostgresRepository) GetAllModules() ([]models.Module, error) {
	var modules []models.Module
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&modules).Error
	return modules, err
}

func (r *PostgresRepository) GetModuleByCode(code string) (*models.Module, error) {
	var module models.Module
	err := r.db.Where("code = ?", code).First(&module).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &module, err
}

// ============== ROLES ==============

func (r *PostgresRepository) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Preload("Permissions.Module").Where("is_active = ?", true).Order("name ASC").Find(&roles).Error
	return roles, err
}

func (r *PostgresRepository) GetRoleByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions.Module").First(&role, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &role, err
}

func (r *PostgresRepository) GetRoleByCode(code string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions.Module").Where("code = ?", code).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &role, err
}

func (r *PostgresRepository) CreateRole(role *models.Role, modulePermissions map[uint]models.RoleModule) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}

		for moduleID, perms := range modulePermissions {
			roleModule := models.RoleModule{
				RoleID:    role.ID,
				ModuleID:  moduleID,
				CanRead:   perms.CanRead,
				CanWrite:  perms.CanWrite,
				CanEdit:   perms.CanEdit,
				CanDelete: perms.CanDelete,
			}
			if err := tx.Create(&roleModule).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PostgresRepository) UpdateRole(role *models.Role, modulePermissions map[uint]models.RoleModule) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updateData := map[string]interface{}{
			"name":        role.Name,
			"description": role.Description,
			"is_active":   role.IsActive,
			"updated_at":  time.Now(),
		}

		if err := tx.Model(role).Updates(updateData).Error; err != nil {
			return err
		}

		// Delete old permissions
		if err := tx.Where("role_id = ?", role.ID).Delete(&models.RoleModule{}).Error; err != nil {
			return err
		}

		// Preparing for Batch Insert
		if len(modulePermissions) > 0 {
			newPerms := make([]models.RoleModule, 0, len(modulePermissions))
			for moduleID, perms := range modulePermissions {
				newPerms = append(newPerms, models.RoleModule{
					RoleID:    role.ID,
					ModuleID:  moduleID,
					CanRead:   perms.CanRead,
					CanWrite:  perms.CanWrite,
					CanEdit:   perms.CanEdit,
					CanDelete: perms.CanDelete,
				})
			}

			if err := tx.Create(&newPerms).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PostgresRepository) DeleteRole(id uint) error {
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRoleRequired
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&role).Association("Permissions").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&role).Association("Users").Clear(); err != nil {
			return err
		}
		// Delete the role itself
		return tx.Delete(&role).Error
	})
}

// ============== ADMIN USERS ==============

func (r *PostgresRepository) GetAllAdminUsers() ([]models.AdminUser, error) {
	var users []models.AdminUser
	err := r.db.Unscoped().
		Preload("Roles.Permissions.Module").
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

func (r *PostgresRepository) GetAdminUserByID(id uint) (*models.AdminUser, error) {
	var user models.AdminUser
	err := r.db.Unscoped().
		Preload("Roles.Permissions.Module").
		First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (r *PostgresRepository) CreateAdminUser(username, password, email, firstName, lastName string, roleIDs []uint, creatorID *uint) (*models.AdminUser, error) {
	// Password hashing using bcrypt for secure storage
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.AdminUser{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedBy:    creatorID,
	}

	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
				return fmt.Errorf("email %s is already exist", email)
			}
			return err
		}

		if len(roleIDs) > 0 {
			var roles []models.Role
			if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
			if err := tx.Model(user).Association("Roles").Replace(roles); err != nil {
				return err
			}
		}

		return nil
	})

	return user, err
}

func (r *PostgresRepository) UpdateAdminUser(id uint, email, firstName, lastName string, isActive bool) error {
	return r.db.Model(&models.AdminUser{}).Where("id = ?", id).Updates(map[string]interface{}{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"is_active":  isActive,
		"updated_at": time.Now(),
	}).Error
}

func (r *PostgresRepository) UpdateAdminUserPassword(id uint, newPassword string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user models.AdminUser
		if err := tx.Select("password_hash").First(&user, id).Error; err != nil {
			return err
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
			return ErrPasswordSameAsOld
		}

		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		return tx.Model(&models.AdminUser{}).Where("id = ?", id).Update("password_hash", string(newHashedPassword)).Error
	})
}

func (r *PostgresRepository) UpdateAdminUserRoles(userID uint, roleIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user models.AdminUser
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}

		var roles []models.Role
		if len(roleIDs) > 0 {
			if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
		}

		return tx.Model(&user).Association("Roles").Replace(roles)
	})
}

func (r *PostgresRepository) UpdateAdminUserLastLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&models.AdminUser{}).Where("id = ?", id).Update("last_login_at", &now).Error
}

func (r *PostgresRepository) DeleteAdminUser(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user models.AdminUser
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}

		if err := tx.Model(&user).Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&user).Association("Roles").Clear(); err != nil {
			return err
		}

		return tx.Delete(&user).Error
	})
}

func (r *PostgresRepository) DeactivateAdminUser(id uint) error {
	return r.db.Model(&models.AdminUser{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":  false,
		"deleted_at": time.Now(),
	}).Error
}

// ============== ACCESS LOGGING ==============

func (r *PostgresRepository) LogAccess(userID uint, moduleCode, action, ipAddress, userAgent string, isAllowed bool) error {
	log := &models.AccessLog{
		UserID:     userID,
		ModuleCode: moduleCode,
		Action:     action,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		IsAllowed:  isAllowed,
	}
	return r.db.Create(log).Error
}

func (r *PostgresRepository) GetAccessLogs(page, perPage int, userID *uint, moduleCode *string) ([]models.AccessLog, int64, error) {
	var logs []models.AccessLog
	var total int64

	query := r.db.Model(&models.AccessLog{})

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if moduleCode != nil && *moduleCode != "" {
		query = query.Where("module_code = ?", *moduleCode)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	err := query.Preload("User").Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error

	return logs, total, err
}

// ============== AUTH HELPER ==============

func (r *PostgresRepository) ValidateAdminCredentials(email, password string) (*models.AdminUserWithAccess, error) {
	logger.Info.Printf("[RBAC AUTH] Validating credentials for email: %s", email)

	userWithAccess, err := r.GetAdminUserByUsername(email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error.Printf("[RBAC AUTH] Email not found: %s", email)
			return nil, nil
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userWithAccess.PasswordHash), []byte(password)); err != nil {
		logger.Error.Printf("[RBAC AUTH] Password mismatch for %s", email)
		return nil, err
	}

	logger.Info.Printf("[RBAC AUTH] Login successful: %s", email)
	return userWithAccess, nil
}
