package admin

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"roulette/internal/logger"
	"roulette/internal/metrics"
	"roulette/internal/models"
	"roulette/internal/repository"
	"roulette/internal/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// Админ-панель
type AdminPanel struct {
	router         *gin.Engine
	service        service.Service
	repo           repository.Repository
	settings       *Settings
	paymentService *service.PaymentService
	metrics        *metrics.Metrics
}

// Настройки админ-панели
type Settings struct {
	Port             string
	SessionSecret    string
	AdminEmail       string
	AdminPassword    string
	BotName          string
	AllowedIPs       []string
	DisableIPFilters bool
	DebugMode        bool
}

func NewAdminPanel(service service.Service, repo repository.Repository, settings *Settings,
	paymentService *service.PaymentService, appMetrics *metrics.Metrics,
) *AdminPanel {
	if settings.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	store := cookie.NewStore([]byte(settings.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
	})

	router.Use(sessions.Sessions("roulette_admin", store))

	// Создаем админ-панель
	admin := &AdminPanel{
		router:         router,
		service:        service,
		repo:           repo,
		settings:       settings,
		paymentService: paymentService,
		metrics:        appMetrics,
	}

	if err := admin.ensureSuperAdminExists(); err != nil {
		logger.Warning.Printf("[WARNING] Failed to ensure super admin exists: %v", err)
	}

	admin.setupRoutes()

	return admin
}

func (a *AdminPanel) ensureSuperAdminExists() error {
	user, _ := a.repo.GetAdminUserByEmail(a.settings.AdminEmail)
	if user != nil {
		return nil
	}

	logger.Info.Printf("[INIT] Admin user '%s' not found. Creating system admin...", a.settings.AdminEmail)

	roleCode := "super_admin"
	role, err := a.repo.GetRoleByCode(roleCode)

	allModules, err := a.repo.GetAllModules()
	if err != nil {
		return fmt.Errorf("failed to get modules for seed: %w", err)
	}

	fullPermissions := make(map[uint]models.RoleModule)
	for _, mod := range allModules {
		fullPermissions[mod.ID] = models.RoleModule{
			ModuleID:      mod.ID,
			CanRead:       true,
			CanWrite:      true,
			CanEdit:       true,
			CanDelete:     true,
			CanAddBalance: true,
		}
	}

	if err != nil || role == nil {
		logger.Info.Printf("[INIT] Role '%s' not found. Creating with full access...", roleCode)
		newRole := &models.Role{
			Name:        "Super Administrator",
			Code:        roleCode,
			Description: "Full System Access",
			IsActive:    true,
			IsSystem:    true,
		}
		if err := a.repo.CreateRole(newRole, fullPermissions); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
		role = newRole
	}

	_, err = a.repo.CreateAdminUser(
		a.settings.AdminPassword,
		a.settings.AdminEmail,
		"Super",
		"Administrator",
		[]uint{role.ID},
		nil,
	)

	return err
}

func (a *AdminPanel) Start() error {
	return a.router.Run(":" + a.settings.Port)
}

// Настройка роутов
func (a *AdminPanel) setupRoutes() {
	a.router.StaticFS("/static", http.Dir("./web/public/static"))
	a.router.StaticFS("/admin/static", http.Dir("./web/admin/static"))

	a.router.SetFuncMap(tplFuncMap)

	files1, _ := filepath.Glob("web/**/templates/*.html")
	files2, _ := filepath.Glob("web/**/templates/**/*.html")
	allFiles := append(files1, files2...)

	if len(allFiles) > 0 {
		logger.Info.Printf("[TEMPLATES] Loading %d template files", len(allFiles))
		a.router.LoadHTMLFiles(allFiles...)
	} else {
		logger.Warning.Printf("[WARNING] No template files found!")
	}

	a.setupAuthRoutes()

	// Настраиваем публичные маршруты
	a.setupPublicRoutes()

	// Настройка роутов уведомлений
	a.setupNotificationsRoutes()

	admin := a.router.Group("/admin")
	admin.Use(a.ipFilterMiddleware(), a.rbacAuthRequired(), a.globalTemplateData(), a.addPaymentServiceToContext())
	{

		admin.GET("/", a.requireRead("dashboard"), a.dashboard)

		a.setupUserRoutes(admin)
		a.setupFinanceRoutes(admin)
		a.setupSystemRoutes(admin)
		a.setupAnalyzerRoutes(admin)
		a.setupRBACRoutes(admin)
	}

	a.setupActivityAnalyzerAPIRoutes(admin)
}

func (a *AdminPanel) setupAuthRoutes() {
	auth := a.router.Group("/")
	{
		auth.GET("/login", a.loginPage)
		auth.POST("/login", a.login)
		auth.GET("/logout", a.logout)
	}
}

func (a *AdminPanel) setupUserRoutes(parent *gin.RouterGroup) {
	uRead := parent.Group("/", a.requireRead(models.ModUsers))
	{
		uRead.GET("/users", a.usersList)
		uRead.GET("/user/:id", a.userDetails)
		uRead.GET("/user/:id/json", a.getUserJSON)
		uRead.GET("/user/:id/stats/json", a.getUserDetailedStats)
	}
	uWrite := parent.Group("/", a.requireWrite(models.ModUsers))
	{
		uWrite.POST("/user/:id/status", a.userStatus)
		uWrite.POST("/user/:id/ref", a.userRef)
	}
	uEdit := parent.Group("/", a.requireEdit(models.ModUsers))
	{
		uEdit.POST("/user/:id/update", a.updateUserProfile)
	}
	uBalance := parent.Group("/", a.requireAddBalance(models.ModUsers))
	{
		uBalance.POST("/user/:id/balance", a.updateUserBalance)
	}
}

func (a *AdminPanel) setupFinanceRoutes(parent *gin.RouterGroup) {
	// --- SOURCES ---
	sources := parent.Group("/sources", a.requireRead(models.ModSources))
	{
		sources.GET("", a.sourcesPage)
		sources.GET("/keys", a.sourcesKeysPage)

		sActions := sources.Group("/")
		{
			sActions.POST("/get_all", a.requireWrite(models.ModSources), a.sourcesGetAll)
			sActions.POST("/keys/:key", a.requireEdit(models.ModSources), a.sourceKeysSave)
			sActions.POST("/keys/add", a.requireWrite(models.ModSources), a.sourceKeysAdd)
		}
	}

	// --- RATINGS ---
	ratings := parent.Group("/ratings", a.requireRead(models.ModRatings))
	{
		ratings.GET("", a.ratingsList)
		ratings.GET("/:year/:week", a.ratingDetails)
		ratings.GET("/super", a.superRatingsList)
		ratings.GET("/super-details/:period", a.superRatingDetails)

		ratings.POST("/:year/:week/distribute", a.requireEdit(models.ModRatings), a.distributeRatingPrizes)
		ratings.POST("/:year/:week/cancel", a.requireEdit(models.ModRatings), a.cancelRatingPrizes)
	}

	// --- WITHDRAWALS ---
	{
		wRead := parent.Group("/", a.requireRead(models.ModWithdrawals))
		{
			wRead.GET("/withdrawals", a.withdrawalsList)
			wRead.GET("/withdrawals/stat", a.withdrawalsStat)
		}
		wEdit := parent.Group("/", a.requireEdit(models.ModWithdrawals))
		{
			wEdit.POST("/withdrawal/:id/approve", a.withdrawalApprove)
			wEdit.POST("/withdrawal/:id/reject", a.withdrawalReject)
		}
	}
}

func (a *AdminPanel) setupSystemRoutes(parent *gin.RouterGroup) {
	parent.GET("/stats", a.requireRead(models.ModStatistics), a.statistics)

	settings := parent.Group("/settings", a.requireRead(models.ModSettings))
	{
		settings.GET("", a.settingsPage)
		settings.POST("", a.requireEdit(models.ModSettings), a.saveSettings)
	}

	locs := parent.Group("/localizations", a.requireRead(models.ModLocalizations))
	{
		locs.GET("", a.localizationsList)
		locs.GET("/export", a.localizationExport)

		lEdit := locs.Group("/", a.requireEdit(models.ModLocalizations))
		{
			lEdit.POST("/:key", a.localizationSave)
			lEdit.POST("/add", a.localizationAdd)
			lEdit.POST("/import", a.localizationImport)
		}

		locs.POST("/:key/delete", a.requireDelete(models.ModLocalizations), a.localizationDelete)
	}

	parent.GET("/hashes", a.requireRead(models.ModHashes), a.hashesPage)

	api := parent.Group("/api")
	{
		api.GET("/hashes", a.requireRead(models.ModHashes), a.getHashesAPI)
		api.POST("/verify-hash", a.requireRead(models.ModHashes), a.verifyHashAPI)
		api.GET("/localizations/:key", a.requireRead(models.ModLocalizations), a.getLocalizationsByKey)
	}
}

func (a *AdminPanel) setupAnalyzerRoutes(parent *gin.RouterGroup) {
	analyzerMod := parent.Group("/", a.requireRead(models.ModActivity))
	{
		analyzerMod.GET("/activity-analyzer", a.activityAnalyzerDashboardPage)
		analyzerMod.GET("/activity-analyzer/user/:telegram_id", a.userActivityDetailPage)
	}
}

func (a *AdminPanel) setupRBACRoutes(parentGroup *gin.RouterGroup) {
	rbac := parentGroup.Group("/rbac", a.requireModule(models.ModRBAC))
	{
		rbac.GET("/modules", a.rbacModulesPage)

		rbac.GET("/roles", a.rbacRolesPage)
		rbac.GET("/roles/:id", a.rbacRoleDetails)
		rbac.POST("/roles", a.requireWrite(models.ModRBAC), a.rbacCreateRole)
		rbac.POST("/roles/:id/permissions", a.requireWrite(models.ModRBAC), a.rbacSaveRolePermissions)
		rbac.POST("/roles/:id/update", a.requireEdit(models.ModRBAC), a.rbacUpdateRole)
		rbac.POST("/roles/:id/delete", a.requireDelete(models.ModRBAC), a.rbacDeleteRole)

		rbac.GET("/users", a.rbacAdminUsersPage)
		rbac.GET("/users/:id", a.rbacAdminUserDetails)
		rbac.POST("/users", a.rbacCreateAdminUser)

		rbac.POST("/users/:id/update", a.requireEdit(models.ModAdmins), a.rbacUpdateAdminUser)
		rbac.POST("/users/:id/password", a.requireEdit(models.ModAdmins), a.rbacUpdateAdminUserPassword)
		rbac.POST("/users/:id/delete", a.requireDelete(models.ModAdmins), a.rbacDeleteAdminUser)

		rbac.GET("/logs", a.requireRead(models.ModRBAC), a.rbacAccessLogsPage)
	}

	// ============ NEW MODULE: Administrators ============
	admins := parentGroup.Group("/admins", a.requireModule(models.ModAdmins))
	{
		admins.GET("", a.rbacAdminUsersPage)
		admins.GET("/:id", a.rbacAdminUserDetails)
		admins.POST("", a.requireWrite(models.ModAdmins), a.rbacCreateAdminUser)
		admins.POST("/:id/update", a.requireEdit(models.ModAdmins), a.rbacUpdateAdminUser)
		admins.POST("/:id/password", a.requireEdit(models.ModAdmins), a.rbacUpdateAdminUserPassword)
		admins.POST("/:id/delete", a.requireDelete(models.ModAdmins), a.rbacDeleteAdminUser)
	}
}

// ==========================================
// MIDDLEWARE & HELPERS
// ==========================================

func (a *AdminPanel) ipFilterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.settings.DisableIPFilters {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		allowed := false
		for _, ip := range a.settings.AllowedIPs {
			if ip == clientIP {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

func (a *AdminPanel) addPaymentServiceToContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("paymentService", a.paymentService)
		c.Next()
	}
}

func (a *AdminPanel) getMetrics() *metrics.Metrics {
	if a == nil || a.metrics == nil {
		return nil
	}
	return a.metrics
}

// ==========================================
// AUTH HANDLERS
// ==========================================

func (a *AdminPanel) loginPage(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("rbac_user") != nil {
		c.Redirect(http.StatusFound, "/admin/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login into Admin-panel",
	})
}

func (a *AdminPanel) login(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		email = c.PostForm("username")
	}
	password := c.PostForm("password")

	user, err := a.repo.ValidateAdminCredentials(email, password)
	if err != nil {
		logger.Error.Printf("[LOGIN ERROR] %v", err)
	}

	if user != nil {
		session := sessions.Default(c)
		session.Set("rbac_user", user.Email)
		session.Set("user", user.Email)

		if err := session.Save(); err != nil {
			logger.Error.Printf("[SESSION ERROR] %v", err)
			return
		}

		_ = a.repo.UpdateAdminUserLastLogin(user.ID)
		logger.Info.Printf("[LOGIN SUCCESS] User %s logged in", user.Email)

		c.Redirect(http.StatusFound, "/admin")
		return
	}

	logger.Error.Printf("[LOGIN FAILED] Attempt for '%s'", email)
	a.render(c, http.StatusUnauthorized, "login.html", gin.H{
		"title": "Login",
		"error": "Невірний email або пароль",
		"email": email,
	})
}

func (a *AdminPanel) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Redirect(http.StatusFound, "/login")
}

type templateWriter struct {
	header http.Header
	buf    *bytes.Buffer
}

func (tw *templateWriter) Header() http.Header         { return tw.header }
func (tw *templateWriter) Write(b []byte) (int, error) { return tw.buf.Write(b) }
func (tw *templateWriter) WriteHeader(statusCode int)  {}

func (a *AdminPanel) render(c *gin.Context, statusCode int, templateName string, data gin.H) {

	if admin, exists := c.Get("admin_user"); exists {
		data["Admin"] = admin
		data["admin_user"] = admin
	}

	if data["admin_user"] == nil {
		session := sessions.Default(c)
		email := session.Get("rbac_user")
		if email != nil {
			user, err := a.repo.GetAdminUserByEmail(email.(string))
			if err == nil && user != nil {
				data["Admin"] = user
				data["admin_user"] = user
				c.Set("admin_user", user)
			}
		}
	}

	if a.router == nil || a.router.HTMLRender == nil {
		c.String(http.StatusInternalServerError, "Template engine error")
		return
	}

	buf := &bytes.Buffer{}
	tw := &templateWriter{header: make(http.Header), buf: buf}
	err := a.router.HTMLRender.Instance(templateName, data).Render(tw)
	if err != nil {
		logger.Error.Printf("[ERROR] Template '%s' failed: %v", templateName, err)
		c.String(http.StatusInternalServerError, "Template Error")
		return
	}
	c.Data(statusCode, "text/html; charset=utf-8", buf.Bytes())
}
