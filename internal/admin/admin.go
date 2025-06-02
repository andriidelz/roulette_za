package admin

import (
	"html/template"
	"net/http"
	"path/filepath"
	"time"

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
}

// Настройки админ-панели
type Settings struct {
	Port             string
	SessionSecret    string
	AdminUsername    string
	AdminPassword    string
	AllowedIPs       []string
	DisableIPFilters bool
}

// Создание новой админ-панели
func NewAdminPanel(service service.Service, repo repository.Repository, settings *Settings, paymentService *service.PaymentService) *AdminPanel {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Настройка сессий
	store := cookie.NewStore([]byte(settings.SessionSecret))
	router.Use(sessions.Sessions("roulette_admin", store))

	// Создаем админ-панель
	admin := &AdminPanel{
		router:         router,
		service:        service,
		repo:           repo,
		settings:       settings,
		paymentService: paymentService,
	}

	admin.setupRoutes()

	return admin
}

// Запуск админ-панели
func (a *AdminPanel) Start() error {
	return a.router.Run(":" + a.settings.Port)
}

// Настройка роутов
func (a *AdminPanel) setupRoutes() {

	a.router.SetFuncMap(template.FuncMap{
		"abs": func(x float64) float64 {
			if x < 0 {
				return -x
			}
			return x
		},
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"divide": func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b)
		},
		"multiply": func(a, b float64) float64 {
			return a * b
		},
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006 15:04")
		},
	})

	files1, _ := filepath.Glob("web/templates/*.html")
	files2, _ := filepath.Glob("web/templates/**/*.html")
	allFiles := append(files1, files2...)
	a.router.LoadHTMLFiles(allFiles...)

	// Статические файлы
	a.router.Static("/static", "./web/static")

	// Настраиваем публичные маршруты
	a.setupPublicRoutes()

	// Настройка роутов уведомлений
	a.setupNotificationsRoutes()

	// Авторизация
	auth := a.router.Group("/")
	auth.Use(a.ipFilterMiddleware())
	{
		auth.GET("/login", a.loginPage)
		auth.POST("/login", a.login)
		auth.GET("/logout", a.logout)
	}

	// Защищенные роуты
	admin := a.router.Group("/admin")
	admin.Use(a.ipFilterMiddleware(), a.authRequired(), a.addPaymentServiceToContext())
	{
		admin.GET("/", a.dashboard)

		// Раздел управления пользователями
		admin.GET("/users", a.usersList)
		admin.GET("/user/:id", a.userDetails)
		admin.POST("/user/:id/ban", a.userBan)
		admin.POST("/user/:id/unban", a.userUnban)
		admin.POST("/user/:id/ref", a.userRef)
		admin.POST("/user/:id/update", a.updateUserProfile)
		admin.POST("/user/:id/balance", a.updateUserBalance)

		// JSON API для данных пользователя
		admin.GET("/user/:id/json", a.getUserJSON)
		admin.GET("/user/:id/stats/json", a.getUserDetailedStats)

		admin.GET("/stats", a.statistics)

		admin.GET("/ratings", a.ratingsList)
		admin.GET("/rating/:year/:week", a.ratingDetails)
		admin.POST("/rating/:year/:week/distribute", a.distributeRatingPrizes)
		admin.POST("/rating/:year/:week/cancel", a.cancelRatingPrizes)

		admin.GET("/super-ratings", a.superRatingsList)
		admin.GET("/super-rating/:period", a.superRatingDetails)

		admin.GET("/withdrawals", a.withdrawalsList)
		admin.POST("/withdrawal/:id/approve", a.withdrawalApprove)
		admin.POST("/withdrawal/:id/reject", a.withdrawalReject)

		admin.GET("/settings", a.settingsPage)
		admin.POST("/settings", a.saveSettings)

		admin.GET("/localizations", a.localizationsList)
		admin.GET("/localization/:key", a.localizationEdit)
		admin.POST("/localization/:key", a.localizationSave)
		admin.POST("/localization/:key/delete", a.localizationDelete)
		admin.POST("/localization/add", a.localizationAdd)
		admin.GET("/localization/export", a.localizationExport) // Export with ?lang=en/ru/uk or all
		admin.POST("/localization/import", a.localizationImport)

		admin.GET("/hashes", a.hashesPage)

		// API эндпоинты для админ-панели (защищенный доступ)
		api := a.router.Group("/admin/api")
		api.Use(a.ipFilterMiddleware(), a.authRequired())
		{
			api.GET("/hashes", a.getHashesAPI)
			api.GET("/hashes/:id", a.getHashDetailsAPI)
			api.POST("/verify-hash", a.verifyHashAPI)
			api.GET("/localizations/:key", a.getLocalizationsByKey)
		}
	}
}

// Middleware для фильтрации по IP
func (a *AdminPanel) ipFilterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаємо, якщо фільтри відключені
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

// Middleware для проверки авторизации
func (a *AdminPanel) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")

		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Next()
	}
}

// Добавления paymentService в контекст
func (a *AdminPanel) addPaymentServiceToContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("paymentService", a.paymentService)
		c.Next()
	}
}

// Страница авторизации
func (a *AdminPanel) loginPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	if user != nil {
		c.Redirect(http.StatusFound, "/admin")
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login into Admin-panel",
	})
}

// Авторизация
func (a *AdminPanel) login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Проверяем логин и пароль
	if username == a.settings.AdminUsername && password == a.settings.AdminPassword {
		session := sessions.Default(c)
		session.Set("user", username)
		session.Save()

		c.Redirect(http.StatusFound, "/admin")
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":    "Login into Admin-panel",
		"error":    "Wrong login or password",
		"username": username,
	})
}

// Выход
func (a *AdminPanel) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Save()

	c.Redirect(http.StatusFound, "/login")
}
