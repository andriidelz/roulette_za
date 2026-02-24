package admin

import (
	"net/http"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============== MODULES ==============

func (a *AdminPanel) rbacModulesPage(c *gin.Context) {
	modules, err := a.repo.GetAllModules()
	if err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{
			"title":   "Error",
			"message": err.Error(),
		})
		return
	}

	a.render(c, http.StatusOK, "rbac_modules", gin.H{
		"title":      "RBAC - Modules",
		"modules":    modules,
		"admin_user": getAdminUser(c),
	})
}

func (a *AdminPanel) rbacRolesPage(c *gin.Context) {
	roles, err := a.repo.GetAllRoles()
	if err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{"message": err.Error()})
		return
	}

	a.render(c, http.StatusOK, "rbac_roles", gin.H{
		"title":      "RBAC - Roles",
		"roles":      roles,
		"activeTab":  "rbac",
		"admin_user": getAdminUser(c),
	})
}

// returns role details page with permissions matrix
func (a *AdminPanel) rbacRoleDetails(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, _ := strconv.ParseUint(roleIDStr, 10, 32)

	role, err := a.repo.GetRoleByID(uint(roleID))
	if err != nil || role == nil {
		a.render(c, http.StatusNotFound, "error.html", gin.H{"message": "Role not found"})
		return
	}

	modules, _ := a.repo.GetAllModules()

	currentPerms := make(map[uint]models.RoleModule)
	for _, p := range role.Permissions {
		currentPerms[p.ModuleID] = p
	}

	a.render(c, http.StatusOK, "rbac_role_details", gin.H{
		"title":        "Налаштування прав",
		"role":         role,
		"modules":      modules, // send all modules to the template to build the permissions matrix
		"currentPerms": currentPerms,
		"activeTab":    "rbac",
		"user":         getAdminUser(c),
	})
}

func (a *AdminPanel) rbacCreateRole(c *gin.Context) {
	name := c.PostForm("name")
	code := c.PostForm("code")
	desc := c.PostForm("description")

	if name == "" || code == "" {
		a.render(c, http.StatusBadRequest, "error.html", gin.H{
			"message": "Назва та код ролі є обов'язковими",
		})
		return
	}

	role := &models.Role{
		Name:        name,
		Code:        code,
		Description: desc,
		IsActive:    true,
		IsSystem:    false,
	}

	if err := a.repo.CreateRole(role, make(map[uint]models.RoleModule)); err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{
			"message": "Помилка БД: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/rbac/roles")
}

func (a *AdminPanel) rbacUpdateRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	var req struct {
		Name        string                     `json:"name" binding:"required"`
		Description string                     `json:"description"`
		IsActive    bool                       `json:"is_active"`
		Permissions map[uint]models.RoleModule `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role, err := a.repo.GetRoleByID(uint(roleID))
	if err != nil || role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot edit system role"})
		return
	}

	role.Name = req.Name
	role.Description = req.Description
	role.IsActive = req.IsActive

	if err := a.repo.UpdateRole(role, req.Permissions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (a *AdminPanel) rbacDeleteRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	if err := a.repo.DeleteRole(uint(roleID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============== ADMIN USERS ==============

func (a *AdminPanel) rbacAdminUsersPage(c *gin.Context) {
	users, err := a.repo.GetAllAdminUsers()
	if err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{
			"title":   "Error",
			"message": err.Error(),
		})
		return
	}

	roles, _ := a.repo.GetAllRoles()

	a.render(c, http.StatusOK, "rbac_admin_users", gin.H{
		"title":      "RBAC - Admin Users",
		"adminUsers": users,
		"roles":      roles,
		"activeTab":  "admins",
		"admin_user": getAdminUser(c),
	})
}

func (a *AdminPanel) rbacAdminUserDetails(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := a.repo.GetAdminUserByID(uint(userID))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (a *AdminPanel) rbacCreateAdminUser(c *gin.Context) {
	var req struct {
		Password  string `form:"password" binding:"required,min=6"`
		Email     string `form:"email" binding:"required,email"`
		FirstName string `form:"first_name" binding:"required"`
		LastName  string `form:"last_name" binding:"required"`
		RoleIDs   []uint `form:"role_ids"`
	}

	if err := c.ShouldBind(&req); err != nil {
		logger.Error.Printf("TEST BINDING: err=%v, req.Password=%s, req.Email=%s", err, req.Password, req.Email)
		a.render(c, http.StatusBadRequest, "error.html", gin.H{"message": "Некоректні дані: " + err.Error()})
		return
	}

	username := req.Email

	currentUser := getAdminUser(c)
	var creatorID *uint
	if currentUser != nil {
		id := currentUser.ID
		creatorID = &id
	}

	_, err := a.repo.CreateAdminUser(username, req.Password, req.Email, req.FirstName, req.LastName, req.RoleIDs, creatorID)
	if err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{"message": "Помилка створення: " + err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/admin/rbac/users")
}

func (a *AdminPanel) rbacUpdateAdminUser(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Email     string `form:"email" binding:"required,email"`
		FirstName string `form:"first_name" binding:"required"`
		LastName  string `form:"last_name" binding:"required"`
		IsActive  string `form:"is_active"`
		RoleIDs   []uint `form:"role_ids"`
	}

	if err := c.ShouldBind(&req); err != nil {
		logger.Error.Printf("[ERROR] Binding update admin: %v", err)
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		a.render(c, http.StatusBadRequest, "error.html", gin.H{"message": "Некоректні дані: " + err.Error()})
		return
	}

	isActiveStr := req.IsActive
	if isActiveStr == "" {
		isActiveStr = c.PostForm("is_active")
	}
	isActive := (isActiveStr == "true")

	// Renew the user with new details (except password, which is handled separately)
	if err := a.repo.UpdateAdminUser(uint(userID), req.Email, req.FirstName, req.LastName, isActive); err != nil {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{"message": err.Error()})
		return
	}

	// Roles are updated separately to avoid complications with the main user update logic
	if err := a.repo.UpdateAdminUserRoles(uint(userID), req.RoleIDs); err != nil {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{"message": err.Error()})
		return
	}

	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.Redirect(http.StatusFound, "/admin/rbac/users")
}

func (a *AdminPanel) rbacUpdateAdminUserPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := a.repo.UpdateAdminUserPassword(uint(userID), req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *AdminPanel) rbacDeleteAdminUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	// Get the user to be deleted
	userToDelete, err := a.repo.GetAdminUserByID(uint(userID))
	if err != nil || userToDelete == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	hasSuperAdminRole := false
	for _, role := range userToDelete.Roles {
		if role.Code == "super_admin" {
			hasSuperAdminRole = true
			break
		}
	}

	if hasSuperAdminRole {
		allAdmins, err := a.repo.GetAllAdminUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to check admin users",
			})
			return
		}

		superAdminCount := 0
		for _, admin := range allAdmins {
			if !admin.IsActive {
				continue
			}
			for _, role := range admin.Roles {
				if role.Code == "super_admin" {
					superAdminCount++
					break
				}
			}
		}

		if superAdminCount <= 1 {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Неможливо видалити останнього Super Admin. Система повинна мати хоча б одного Super Admin.",
			})
			return
		}
	}

	if err := a.repo.DeactivateAdminUser(uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ============== ACCESS LOGS ==============

func (a *AdminPanel) rbacAccessLogsPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 50

	var userID *uint
	if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 32); err == nil {
			uidUint := uint(id)
			userID = &uidUint
		}
	}

	var moduleCode *string
	if mc := c.Query("module_code"); mc != "" {
		moduleCode = &mc
	}

	logs, total, err := a.repo.GetAccessLogs(page, perPage, userID, moduleCode)
	if err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{
			"title":   "Error",
			"message": err.Error(),
		})
		return
	}

	a.render(c, http.StatusOK, "rbac_access_logs", gin.H{
		"title":      "RBAC - Access Logs",
		"logs":       logs,
		"page":       page,
		"totalPages": (total + int64(perPage) - 1) / int64(perPage),
		"user":       getAdminUser(c),
	})
}

func (a *AdminPanel) rbacSaveRolePermissions(c *gin.Context) {
	roleID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	role, err := a.repo.GetRoleByID(uint(roleID))
	if err != nil || role == nil {
		a.render(c, http.StatusNotFound, "error.html", gin.H{"message": "Role not found"})
		return
	}

	modules, _ := a.repo.GetAllModules()
	permissions := make(map[uint]models.RoleModule)

	for _, m := range modules {
		permissions[m.ID] = models.RoleModule{
			CanRead:       c.PostForm("perm_"+m.Code+"_read") == "on",
			CanWrite:      c.PostForm("perm_"+m.Code+"_write") == "on",
			CanEdit:       c.PostForm("perm_"+m.Code+"_edit") == "on",
			CanDelete:     c.PostForm("perm_"+m.Code+"_delete") == "on",
			CanAddBalance: c.PostForm("perm_"+m.Code+"_add_balance") == "on",
		}
	}

	if err := a.repo.UpdateRole(role, permissions); err != nil {
		a.render(c, http.StatusInternalServerError, "error.html", gin.H{"message": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/admin/rbac/roles/"+strconv.Itoa(int(roleID)))
}
