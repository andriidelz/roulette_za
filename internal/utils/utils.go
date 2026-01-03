package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	logger "roulette/internal/logger"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func DumpInterface(in interface{}) {
	b, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		logger.Info.Println("Error: ", err)
	}
	log.Print(string(b))
}

func GenerateRandomNumber(max int64) int64 {
	randomNumber, _ := rand.Int(rand.Reader, big.NewInt(max))
	return randomNumber.Int64()
}

// GenerateSalt генерує випадкову сіль
func GenerateSalt() []byte {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		panic(err)
	}
	return salt
}

// CreateHash створює хеш на основі числа та солі
func CreateHash(number int64, salt []byte) string {
	saltHex := hex.EncodeToString(salt)
	data := fmt.Sprintf("%d:%s", number, saltHex)
	hash := sha256.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

// VerifyHash перевіряє хеш
func VerifyHash(originalHash, saltHex string, number int64) bool {
	data := fmt.Sprintf("%d:%s", number, saltHex)
	hash := sha256.New()
	hash.Write([]byte(data))
	computedHash := hex.EncodeToString(hash.Sum(nil))
	return originalHash == computedHash
}

// ToBase62 конвертує число в Base62 рядок
func ToBase62(num uint) string {
	length := len(charset)

	if num == 0 {
		return string(charset[0])
	}

	var result strings.Builder

	for num > 0 {
		result.WriteByte(charset[num%uint(length)])
		num = num / uint(length)
	}

	// Перевертаємо рядок
	runes := []rune(result.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// FromBase62 decodes a base62 encoded string to an uint64.
func FromBase62(token string) uint64 {
	var val uint64
	var base uint64 = 62
	for index, char := range token {
		pow := len(token) - (index + 1)
		pos := strings.IndexRune(charset, char)
		if pos == -1 {
			return 0
		}

		val += uint64(pos) * uint64(math.Pow(float64(base), float64(pow)))
	}

	return val
}

// GetColorForNumber повертає колір для числа на рулетці
func GetColorForNumber(number int64) string {
	if number == 0 {
		return "zero (green)"
	}
	redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36} // European-style layout
	for _, n := range redNumbers {
		if number == n {
			return "red"
		}
	}
	return "black"
}

// PeriodControl - check dates
func PeriodControl(dateFrom, dateTo, period *string) (bool, error) {
	if *dateTo == "" && *dateFrom == "" {
		year, month, day := time.Now().Date()
		switch *period {
		case "week":
			weekday := time.Now().Weekday()
			res := time.Date(year, month, day-int(weekday)+1, 0, 0, 0, 0, time.UTC).Local()
			*dateFrom = res.Format("2006-01-02")
			*dateTo = res.AddDate(0, 0, 7).Format("2006-01-02")
		case "month":
			*dateFrom = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Local().Format("2006-01-02")
			*dateTo = time.Date(year, month+1, -1, 0, 0, 0, 0, time.UTC).Local().Format("2006-01-02")
		case "year":
			*dateFrom = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Local().Format("2006-01-02")
			*dateTo = time.Date(year+1, 1, -1, 0, 0, 0, 0, time.UTC).Local().Format("2006-01-02")
		default:
			res := time.Date(year, month, day-1, 0, 0, 0, 0, time.UTC).Local()
			*dateFrom = res.Format("2006-01-02")
			*dateTo = res.AddDate(0, 0, 1).Format("2006-01-02")
			return false, nil
		}

	} else if *dateTo == "" || *dateFrom == "" {
		return false, errors.New("введенна только одна дата, введите вторую")
	} else {
		before, err := CheckTimeBefore(dateTo, dateFrom)
		if err {
			return false, errors.New("формат даты указан неправильно, нужно YYYY-MM-DD")
		}
		if !before {
			return false, errors.New("период 'from' больше периода 'to'")
		}
	}
	return true, nil
}

// CheckTimeBefore - checkTimeBefore dates
func CheckTimeBefore(dateTo, dateFrom *string) (bool, bool) {
	from, err1 := time.Parse("2006-01-02", *dateFrom)
	to, err2 := time.Parse("2006-01-02", *dateTo)
	if err2 != nil || err1 != nil {
		return false, true
	}
	to = to.AddDate(0, 0, 1)
	*dateTo = to.Format("2006-01-02")
	return from.Before(to), false
}

func ReplaceMacrosInTexts(title, message, buttonText string, params map[string]interface{}) (string, string, string) {
	// Заменяем все макросы в текстах
	for key, value := range params {
		placeholder := "{" + key + "}"
		var strValue string

		switch v := value.(type) {
		case int:
			strValue = fmt.Sprintf("%d", v)
		case float64:
			// Проверяем, является ли число фактически целым
			if v == float64(int(v)) && (key == "position" || key == "points") {
				// Для позиции и очков выводим как целое число
				strValue = fmt.Sprintf("%d", int(v))
			} else {
				// Форматируем числа с плавающей точкой с двумя знаками после запятой
				strValue = fmt.Sprintf("%.2f", v)
			}
		case string:
			strValue = v
		default:
			strValue = fmt.Sprintf("%v", v)
		}

		title = strings.Replace(title, placeholder, strValue, -1)
		message = strings.Replace(message, placeholder, strValue, -1)
		if buttonText != "" {
			buttonText = strings.Replace(buttonText, placeholder, strValue, -1)
		}
	}

	return title, message, buttonText
}

func GetUserLang(c *gin.Context) string {
	matcher := language.NewMatcher([]language.Tag{
		language.Ukrainian, // The first language is used as fallback
		language.English,
		language.Russian,
	})

	header := c.Request.Header.Get("X-Language")
	tag, _ := language.MatchStrings(matcher, header)
	baseLang, _ := tag.Base()

	return baseLang.String()
}

func GetLangPath(c *gin.Context) string {
	path := ""
	lang := GetUserLang(c)
	if lang != "uk" {
		path = "/" + lang
	}
	return path
}

func GetLangPathFromLang(lang string) string {
	path := ""
	if lang != "uk" {
		path = "/" + lang
	}
	return path
}

func GetPWD() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}

	return filepath.Dir(ex)
}

// Функция, которая преобразует map[string]interface{} в JSON строку и кодирует её в Base64
func MapToBase64Json(input map[string]interface{}) string {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(jsonData)
}

// GetYAMLData возвращает карту с данными YAML для указанного языка и списка модулей.
func GetYAMLData(lang string, modules []string) map[string]interface{} {
	result := map[string]interface{}{}

	for _, module := range modules {
		// Строим путь к файлам YAML для указанного языка и модуля
		path := fmt.Sprintf(GetPWD()+"/locales/%s/%s.yml", lang, module)

		// Получаем данные YAML для указанного пути
		data, err := YAMLFiles2Map(path)
		if err != nil {
			return result
		}

		// Объединяем результаты в общую карту
		for _, v := range data[lang] {
			for k, v2 := range v {
				result[k] = v2
			}
		}
	}

	return result
}

// YAMLFiles2Map принимает путь к файлам YAML в формате "/locales/*/*.yml" и возвращает карту с данными, организованными по языкам и модулям.
// Каждый язык содержит данные для разных модулей.
// Пример использования:
//
//	result, err := YAMLFiles2Map("/locales/*/*.yml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Получить данные для английского языка в public модуле
//	data := result["en"]["ad-marker"]
//	fmt.Println(data)
func YAMLFiles2Map(path string) (map[string]map[string]map[string]interface{}, error) {
	matches, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}

	result := map[string]map[string]map[string]interface{}{}

	for _, match := range matches {
		language := filepath.Base(filepath.Dir(match))

		filename := filepath.Base(match)
		module := strings.TrimSuffix(filename, filepath.Ext(filename))

		m, err := YAMLFile2Map(match)
		if err != nil {
			return nil, err
		}

		if _, ok := result[language]; !ok {
			result[language] = map[string]map[string]interface{}{}
		}

		if _, ok := result[language][module]; !ok {
			result[language][module] = map[string]interface{}{}
		}

		for key, value := range m {
			result[language][module][key] = value
		}
	}

	return result, nil
}

func YAMLFile2Map(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
