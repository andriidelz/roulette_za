package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	logger "roulette/internal/logger"
)

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
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
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
	return from.Before(to), false
}
