package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/logger"
	"roulette/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func (a *AdminPanel) rbacAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("rbac_user")

		if username == nil {
			logger.Info.Println("[AUTH DEBUG] Session 'rbac_user' is NIL.")

			if isAJAX(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			} else {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		user, err := a.repo.GetAdminUserByUsername(username.(string))

		if err != nil || user == nil || !user.IsActive {
			logger.Warning.Printf("[AUTH SECURITY] Access blocked: User=%v, Active=%v", username, user != nil && user.IsActive)

			session.Clear()
			session.Save()

			if isAJAX(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Account disabled or not found"})
			} else {
				c.Redirect(http.StatusFound, "/login?error=account_disabled")
				c.Abort()
			}
			return
		}

		c.Set("admin_user", user)
		c.Next()
	}
}

func (a *AdminPanel) globalTemplateData() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (a *AdminPanel) requireRead(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		hasAnyAccess := user != nil && (user.HasPermission(moduleCode, "can_read") ||
			user.HasPermission(moduleCode, "can_write") ||
			user.HasPermission(moduleCode, "can_edit") ||
			user.HasPermission(moduleCode, "can_delete"))

		if !hasAnyAccess {
			a.handlePermissionDenied(c, "Ви не маєте прав на перегляд цього розділу")
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=any_read", user.Username, moduleCode)
			return
		}
		c.Next()
	}
}

// requireWrite - wrapper
func (a *AdminPanel) requireWrite(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		if user == nil || !user.HasPermission(moduleCode, "can_write") {
			a.handlePermissionDenied(c, "Ви не маєте прав на створення записів у цьому розділі")
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=can_write", user.Username, moduleCode)
			return
		}
		c.Next()
	}
}

// requireEdit - wrapper
func (a *AdminPanel) requireEdit(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		if user == nil || !user.HasPermission(moduleCode, "can_edit") {
			a.handlePermissionDenied(c, "Ви не маєте прав на редагування записів у цьому розділі")
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=can_edit", user.Username, moduleCode)
			return
		}
		c.Next()
	}
}

// requireDelete - wrapper
func (a *AdminPanel) requireDelete(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		if user == nil || !user.HasPermission(moduleCode, "can_delete") {
			a.handlePermissionDenied(c, "Ви не маєте прав на видалення записів у цьому розділі")
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=can_delete", user.Username, moduleCode)
			return
		}
		c.Next()
	}
}

// requireAddBalance - wrapper для validate can_add_balance
func (a *AdminPanel) requireAddBalance(moduleCode string) gin.HandlerFunc {
	return a.requirePermission(moduleCode, models.PermAddBalance)
}

// requireModule - validate at least read access to the module (for menu items, etc.)
func (a *AdminPanel) requireModule(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		hasAnyAccess := user != nil && (user.HasPermission(moduleCode, "can_read") ||
			user.HasPermission(moduleCode, "can_write") ||
			user.HasPermission(moduleCode, "can_edit") ||
			user.HasPermission(moduleCode, "can_delete"))

		if !hasAnyAccess {
			a.handlePermissionDenied(c, "Ви не маєте доступу до цього модуля")
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=any_module", user.Username, moduleCode)
			return
		}
		c.Next()
	}
}

// requirePermission validation of right RBAC
func (a *AdminPanel) requirePermission(moduleCode string, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getAdminUser(c)
		if user == nil {
			logger.Warning.Println("[RBAC] User missing in context")
			a.handleUnauthorized(c)
			return
		}

		if user.IsSuperAdmin {
			c.Next()
			return
		}

		if !user.HasPermission(moduleCode, permission) {
			logger.Warning.Printf("[RBAC] Access DENIED: User=%s, Module=%s, Perm=%s", user.Username, moduleCode, permission)

			go a.repo.LogAccess(user.ID, moduleCode, permission, c.ClientIP(), c.Request.UserAgent(), false)

			a.handleForbidden(c, moduleCode, permission, user)
			return
		}

		if c.Request.Method != http.MethodGet {
			go a.repo.LogAccess(user.ID, moduleCode, permission, c.ClientIP(), c.Request.UserAgent(), true)
		}

		c.Next()
	}
}

func (a *AdminPanel) handleUnauthorized(c *gin.Context) {
	if isAJAX(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	} else {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
	}
}

// handleForbidden - calculating 403
func (a *AdminPanel) handleForbidden(c *gin.Context, module, permission string, user *models.AdminUserWithAccess) {
	errorMessage := fmt.Sprintf("Access Denied: Module %s, Permission %s", module, permission)

	if isAJAX(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":      "Forbidden",
			"module":     module,
			"permission": permission,
			"message":    errorMessage,
		})
		return
	}

	if a.router == nil || a.router.HTMLRender == nil {
		c.AbortWithStatus(http.StatusForbidden)
		logger.Error.Printf("[CRITICAL] Attempted to render 403 page but HTMLRender is nil")
		return
	}

	a.render(c, http.StatusForbidden, "error.html", gin.H{
		"title":      "Access Denied",
		"code":       403,
		"message":    "You do not have permission to perform this action.",
		"detail":     "Required: " + module + " -> " + permission,
		"user":       user,
		"currentUrl": c.Request.URL.Path,
	})
	c.Abort()
}

func getAdminUser(c *gin.Context) *models.AdminUserWithAccess {
	if user, exists := c.Get("admin_user"); exists {
		if adminUser, ok := user.(*models.AdminUserWithAccess); ok {
			return adminUser
		}
	}
	return nil
}

func isAJAX(c *gin.Context) bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest" ||
		c.ContentType() == "application/json"
}

func (a *AdminPanel) handlePermissionDenied(c *gin.Context, message string) {
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" || c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   message,
		})
		logger.Warning.Printf("[RBAC] Access DENIED (AJAX): %s", message)
		c.Abort()
		return
	}

	a.render(c, http.StatusForbidden, "error.html", gin.H{
		"title":   "Доступ заборонено",
		"message": message,
	})
	logger.Warning.Printf("[RBAC] Access DENIED: %s", message)
	c.Abort()
}
