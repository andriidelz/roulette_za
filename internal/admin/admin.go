package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"roulette/internal/models"
	"roulette/internal/repository"
	"roulette/internal/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// Адмін-панель
type AdminPanel struct {
	router   *gin.Engine
	service  service.Service
	repo     repository.Repository
	settings *Settings
}

// Налаштування адмін-панелі
type Settings struct {
	Port             string
	SessionSecret    string
	AdminUsername    string
	AdminPassword    string
	AllowedIPs       []string
	DisableIPFilters bool
}

// Створення нової адмін-панелі
func NewAdminPanel(service service.Service, repo repository.Repository, settings *Settings) *AdminPanel {
	// Створюємо роутер
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Налаштовуємо сесії
	store := cookie.NewStore([]byte(settings.SessionSecret))
	router.Use(sessions.Sessions("roulette_admin", store))

	// Створюємо адмін-панель
	admin := &AdminPanel{
		router:   router,
		service:  service,
		repo:     repo,
		settings: settings,
	}

	// Налаштовуємо роути
	admin.setupRoutes()

	return admin
}

// Запуск адмін-панелі
func (a *AdminPanel) Start() error {
	return a.router.Run(":" + a.settings.Port)
}

// Налаштування роутів
func (a *AdminPanel) setupRoutes() {

	a.router.SetFuncMap(template.FuncMap{
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
		"multiply": func(a, b int) int {
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

	// Статичні файли
	a.router.Static("/static", "./web/static")

	// Авторизація
	auth := a.router.Group("/")
	auth.Use(a.ipFilterMiddleware())
	{
		auth.GET("/login", a.loginPage)
		auth.POST("/login", a.login)
		auth.GET("/logout", a.logout)
	}

	// Защищенные роуты
	admin := a.router.Group("/admin")
	admin.Use(a.ipFilterMiddleware(), a.authRequired())
	{
		admin.GET("/", a.dashboard)

		admin.GET("/users", a.usersList)
		admin.GET("/user/:id", a.userDetails)
		admin.POST("/user/:id/ban", a.userBan)
		admin.POST("/user/:id/unban", a.userUnban)

		admin.GET("/stats", a.statistics)

		admin.GET("/ratings", a.ratingsList)
		admin.GET("/rating/:year/:week", a.ratingDetails)
		admin.POST("/rating/:year/:week/distribute", a.distributeRatingPrizes)

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

		admin.GET("/hashes", a.hashesPage)

		// API ендпоінти
		api := a.router.Group("/admin/api")
		api.Use(a.ipFilterMiddleware(), a.authRequired())
		{
			api.GET("/hashes", a.getHashesAPI)
			api.GET("/hashes/:id", a.getHashDetailsAPI)
			api.POST("/verify-hash", a.verifyHashAPI)
		}
	}
}

// Middleware для фільтрації за IP
func (a *AdminPanel) ipFilterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаємо, якщо фільтри відключені
		if a.settings.DisableIPFilters {
			c.Next()
			return
		}

		// Отримуємо IP клієнта
		clientIP := c.ClientIP()

		// Перевіряємо, чи дозволений IP
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

// Middleware для перевірки авторизації
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

// Страница атворизации
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

	// Перевіряємо логін і пароль
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

// Вихід
func (a *AdminPanel) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Save()

	c.Redirect(http.StatusFound, "/login")
}

func (a *AdminPanel) dashboard(c *gin.Context) {
	// Отримуємо загальну статистику
	userCount, err := a.repo.GetUserCount()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Рік і тиждень для рейтингу
	year, week := time.Now().ISOWeek()

	// Отримуємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		// Якщо запис не знайдено, створюємо значення за замовчуванням
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}

		// Можемо спробувати зберегти його в базу даних
		_ = a.repo.UpdatePrizeFund(prizeFund) // Ігноруємо помилку, якщо вона виникне
	}

	// Поточна дата
	currentDateTime := time.Now().Format("2006-01-02")

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"title":           "Admin-panel - Головна",
		"userCount":       userCount,
		"year":            year,
		"week":            week,
		"prizeFund":       prizeFund,
		"activeTab":       "dashboard",
		"currentDateTime": currentDateTime,
	})
}

// Список користувачів
func (a *AdminPanel) usersList(c *gin.Context) {
	// Отримуємо параметри пагінації
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20

	// Отримуємо користувачів
	users, totalUsers, err := a.repo.GetUsers(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Розраховуємо кількість сторінок
	totalPages := (int(totalUsers) + perPage - 1) / perPage

	// Розраховуємо номери попередньої та наступної сторінок
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}

	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	c.HTML(http.StatusOK, "users", gin.H{
		"title":      "Admin-panel - Користувачі",
		"users":      users,
		"page":       page,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"activeTab":  "users",
	})
}

// Деталі користувача
func (a *AdminPanel) userDetails(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Wrong user ID",
		})
		return
	}

	// Отримуємо інформацію про користувача
	// Цей метод треба додати в репозиторій
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо статистику користувача
	stats, err := a.repo.GetUserStats(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо останні ставки користувача
	bets, err := a.repo.GetUserBets(uint(userID), 20)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "user_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Користувач %s", user.Username),
		"user":      user,
		"stats":     stats,
		"bets":      bets,
		"activeTab": "users",
	})
}

// Блокування користувача
func (a *AdminPanel) userBan(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID користувача"})
		return
	}

	// Отримуємо інформацію про користувача
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Блокуємо користувача
	user.Banned = true
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Розблокування користувача
func (a *AdminPanel) userUnban(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID користувача"})
		return
	}

	// Отримуємо інформацію про користувача
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Розблокуємо користувача
	user.Banned = false
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Статистика
func (a *AdminPanel) statistics(c *gin.Context) {
	// Отримуємо статистику за день, тиждень, місяць
	// Тут потрібно додати методи для отримання статистики

	c.HTML(http.StatusOK, "statistics", gin.H{
		"title":     "Admin-panel - Statistics",
		"activeTab": "stats",
	})
}

// Список рейтингів
func (a *AdminPanel) ratingsList(c *gin.Context) {
	// Отримуємо список рейтингів за останні тижні
	// Тут потрібно додати метод для отримання списку рейтингів

	c.HTML(http.StatusOK, "ratings", gin.H{
		"title":     "Admin-panel - Ratings",
		"activeTab": "ratings",
	})
}

// Деталі рейтингу
func (a *AdminPanel) ratingDetails(c *gin.Context) {
	// Отримуємо рік і тиждень
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Невірний рік",
		})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Невірний тиждень",
		})
		return
	}

	// Отримуємо рейтинг
	ratings, err := a.repo.GetWeeklyRating(year, week, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}
	}

	c.HTML(http.StatusOK, "rating_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Rating %d/%d", year, week),
		"ratings":   ratings,
		"year":      year,
		"week":      week,
		"prizeFund": prizeFund,
		"activeTab": "ratings",
	})
}

// Розподіл призів рейтингу
func (a *AdminPanel) distributeRatingPrizes(c *gin.Context) {
	// Отримуємо рік і тиждень
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong year"})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong week"})
		return
	}

	// Отримуємо суму призового фонду
	amount, err := strconv.ParseFloat(c.PostForm("amount"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірна сума призового фонду"})
		return
	}

	topCount, err := strconv.Atoi(c.PostForm("top_count"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірна кількість призових місць"})
		return
	}

	// Оновлюємо або створюємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   amount,
			TopCount: topCount,
		}
		if err := a.repo.UpdatePrizeFund(prizeFund); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		prizeFund.Amount = amount
		prizeFund.TopCount = topCount
		if err := a.repo.UpdatePrizeFund(prizeFund); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Розподіляємо призи
	if err := a.service.DistributePrizes(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Список супер-рейтингів
func (a *AdminPanel) superRatingsList(c *gin.Context) {
	// Отримуємо список супер-рейтингів
	// Тут потрібно додати метод для отримання списку супер-рейтингів

	c.HTML(http.StatusOK, "super_ratings", gin.H{
		"title":     "Admin-panel - Super-ratings",
		"activeTab": "super_ratings",
	})
}

// Деталі супер-рейтингу
func (a *AdminPanel) superRatingDetails(c *gin.Context) {
	// Отримуємо період
	period := c.Param("period")

	// Отримуємо супер-рейтинг
	ratings, err := a.repo.GetSuperRating(period, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "super_rating_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Super-rating %s", period),
		"ratings":   ratings,
		"period":    period,
		"activeTab": "super_ratings",
	})
}

// Список запитів на виведення коштів
func (a *AdminPanel) withdrawalsList(c *gin.Context) {
	// Отримуємо список запитів на виведення коштів
	withdrawals, err := a.repo.GetPendingWithdrawals()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "withdrawals", gin.H{
		"title":       "Admin-panel - Withdrawals",
		"withdrawals": withdrawals,
		"activeTab":   "withdrawals",
	})
}

// Підтвердження виведення коштів
func (a *AdminPanel) withdrawalApprove(c *gin.Context) {
	// Отримуємо ID запиту
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID запиту"})
		return
	}

	// Підтверджуємо запит
	if err := a.repo.UpdateWithdrawalStatus(uint(withdrawalID), "approved"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Відхилення виведення коштів
func (a *AdminPanel) withdrawalReject(c *gin.Context) {
	// Отримуємо ID запиту
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID запиту"})
		return
	}

	// Отримуємо запит
	withdrawal, err := a.repo.GetWithdrawalByID(uint(withdrawalID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Повертаємо кошти на баланс користувача
	user, err := a.repo.GetUserByID(withdrawal.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Balance += withdrawal.Amount
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Відхиляємо запит
	if err := a.repo.UpdateWithdrawalStatus(uint(withdrawalID), "rejected"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Сторінка налаштувань
func (a *AdminPanel) settingsPage(c *gin.Context) {
	// Отримуємо налаштування
	settings, err := a.service.GetSettings()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "settings", gin.H{
		"title":     "Admin-panel - Налаштування",
		"settings":  settings,
		"activeTab": "settings",
	})
}

// Збереження налаштувань
func (a *AdminPanel) saveSettings(c *gin.Context) {
	// Отримуємо налаштування з форми
	// Зберігаємо налаштування
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			if err := a.service.UpdateSetting(key, values[0]); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
