package admin

import (
	"html/template"
	"roulette/internal/logger"
	"roulette/internal/models"
	"roulette/internal/utils"
	"time"

	"github.com/kataras/i18n"
)

var (
	localesPath               = utils.GetPWD() + "/locales/*/*.yml"
	localesMap, localesMapErr = utils.YAMLFiles2Map(localesPath)
	I18n, I18nErr             = i18n.New(i18n.Glob(localesPath), "uk_UA", "en-US", "ru-RU") // "en-US", "uk_UA", "ru-RU", ...
)

var tplFuncMap = template.FuncMap{
	// Функция для перевода текста с поддержкой множественных форм и аргументов
	"i18n": func(lang, format string, args ...interface{}) string {
		result := I18n.Tr(lang, format, args...)
		if len(result) == 0 {
			return format
		}
		return result
	},
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
	// Функция seq генерирует последовательность чисел от start до end включительно
	// Используется для создания диапазона страниц в пагинации
	"seq": func(start, end int) []int {
		if end < start {
			return []int{}
		}
		seq := make([]int, end-start+1)
		for i := range seq {
			seq[i] = start + i
		}
		return seq
	},
	// hasPermission chack right for user
	"hasPermission": func(user interface{}, module, permission string) bool {
		if user == nil {
			return false
		}
		admin, ok := user.(*models.AdminUserWithAccess)
		if !ok {
			logger.Error.Printf("TEMPLATE ERROR: expected *AdminUserWithAccess, got %T", user)
			val, ok := user.(models.AdminUserWithAccess)
			if !ok {
				return false
			}
			admin = &val
		}
		return admin.HasPermission(module, permission)
	},

	"can": func(user interface{}, module, permission string) bool {
		if user == nil {
			return false
		}
		adminUser, ok := user.(*models.AdminUserWithAccess)
		if !ok {
			return false
		}

		if permission == "can_read" {
			return adminUser.HasPermission(module, "can_read") ||
				adminUser.HasPermission(module, "can_write") ||
				adminUser.HasPermission(module, "can_edit") ||
				adminUser.HasPermission(module, "can_delete") ||
				adminUser.HasPermission(module, "can_add_balance")
		}

		return adminUser.HasPermission(module, permission)
	},

	// hasModule - validate if user has access to module
	"hasModule": func(user interface{}, module string) bool {
		if user == nil {
			return false
		}
		adminUser, ok := user.(*models.AdminUserWithAccess)
		if !ok {
			return false
		}

		return adminUser.HasPermission(module, "can_read") ||
			adminUser.HasPermission(module, "can_write") ||
			adminUser.HasPermission(module, "can_edit") ||
			adminUser.HasPermission(module, "can_delete")
	},
}
