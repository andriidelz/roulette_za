package utils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	logger "roulette/internal/logger"
)

const (
	chars    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	lenChars = len(chars)
)

func DumpInterface(in interface{}) {
	b, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		logger.Info.Println("Error: ", err)
	}
	log.Print(string(b))
}

func ContainsString(s []string, e string) bool {
	for i := range s {
		if s[i] == e {
			return true
		}
	}
	return false
}

func IndexOf(arr interface{}, v interface{}) int {
	V := reflect.ValueOf(v)
	Arr := reflect.ValueOf(arr)
	if t := reflect.TypeOf(arr).Kind(); t != reflect.Slice && t != reflect.Array {
		panic("Type Error! Second argument must be an array or a slice.")
	}
	for i := 0; i < Arr.Len(); i++ {
		if Arr.Index(i).Interface() == V.Interface() {
			return i
		}
	}
	return -1
}

func IndexOfUint64(s []uint64, e uint64) (int, bool) {
	for i, a := range s {
		if a == e {
			return i, true
		}
	}
	return 0, false
}

func Env(envName string, defaultValue string) (value string) {
	value = os.Getenv(envName)
	if len(value) == 0 {
		value = defaultValue
	}
	return
}

func DoEvery(d time.Duration, f func()) {
	go func() {
		for range time.Tick(d) {
			f()
		}
	}()
}

func IsNumber(text string) bool {
	_, err := strconv.ParseFloat(text, 64)
	return err == nil
}

func ConvertStringToUnix(from string, add int) int64 {

	date, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0
	}

	return date.AddDate(0, 0, add).Unix()
}

func IPtoIntUniversal(ipStr string) *big.Int {
	ip := net.ParseIP(ipStr)
	IPInt := big.NewInt(0)

	if ip != nil && strings.Contains(ipStr, ":") { // IPv6
		IPInt.SetBytes(ip.To16())
	} else {
		IPInt.SetBytes(ip.To4())
	}

	return IPInt
}

func HasSubString(str string, strArr []string) bool {
	for i := range strArr {
		if strings.Contains(str, strArr[i]) {
			return true
		}
	}
	return false
}

func GetKeysFromMap(m map[string]string) []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func CheckURL(u string) (bool, error) {
	res, err := http.Head(u)
	if err != nil {
		return false, err
	}
	if res.StatusCode == 404 {
		return false, errors.New(res.Status)
	}
	return true, nil
}

func ShortenString(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func GetPWD() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}

	return filepath.Dir(ex)
}

// input string in HH:MM format
func TimeDelta(start, stop string) (time.Duration, error) {

	tStart, err := time.Parse("15:04", start)
	if err != nil {
		return 0, err
	}

	tStop, err := time.Parse("15:04", stop)
	if err != nil {
		return 0, err
	}

	return tStop.Sub(tStart), nil
}

func GenerateCSV(records [][]string) ([]byte, error) {

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)

	for _, record := range records {
		if err := w.Write(record); err != nil {
			return b.Bytes(), err
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return b.Bytes(), err
	}

	return b.Bytes(), nil
}

func ParseCSV(s string) ([][]string, error) {
	ret := [][]string{}

	r := csv.NewReader(bytes.NewReader([]byte(s)))

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ret, err
		}

		ret = append(ret, record)
	}

	return ret, nil
}

func CheckErr(e error) {
	if e != nil {
		panic(e)
	}
}

func ParseTelegramURL(input string) string {

	input = strings.Trim(strings.ToLower(input), " ")

	r := []rune(input)
	if string(r[0:1]) == "@" {
		return input[1:]
	}
	if strings.Contains(input, "t.me/") && string(r[0:1]) == "t" {
		return input[strings.Index(input, "t.me/")+5:]
	}

	return input
}

func GetFileContentType(f *multipart.FileHeader) (string, error) {

	file, err := f.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// to sniff the content type only the first
	// 512 bytes are used.
	buf := make([]byte, 512)
	file.Read(buf)

	return http.DetectContentType(buf), nil
}

func RemoveFromSliceByValue(slice []string, value string) ([]string, error) {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...), nil
		}
	}
	return slice, errors.New("value not found")
}

func RemoveEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func ToLowerCase(input []string) []string {
	lowerCased := make([]string, len(input))
	for i, s := range input {
		lowerCased[i] = strings.ToLower(s)
	}
	return lowerCased
}

// Function to check if a string contains only Cyrillic characters
func IsCyrillic(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Cyrillic, r) {
			return false
		}
	}
	return true
}

// Function to check if a string contains only Latin characters
func IsLatin(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Latin, r) {
			return false
		}
	}
	return true
}

// Transliteration from Cyrillic to Latin (simplified version)
func CyrillicToLatin(s string) string {

	translitMap := map[rune]string{
		'і': "i",
		'ї': "ji",

		'а': "a",
		'б': "b",
		'в': "v",
		'г': "h",
		'д': "d",
		'е': "e",
		'ё': "e",
		'ж': "zh",
		'з': "z",
		'и': "i",
		'й': "y",
		'к': "k",
		'л': "l",
		'м': "m",
		'н': "n",
		'о': "o",
		'п': "p",
		'р': "r",
		'с': "s",
		'т': "t",
		'у': "u",
		'ф': "f",
		'х': "kh",
		'ц': "ts",
		'ч': "ch",
		'ш': "sh",
		'щ': "shch",
		'ы': "y",
		'э': "e",
		'ю': "u",
		'я': "ya",
	}

	var result strings.Builder
	for _, r := range s {
		if val, ok := translitMap[r]; ok {
			result.WriteString(val)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Function to "transliterate" Latin to Cyrillic (stub function)
// You will need to define the specific transliteration rules
// Transliteration from Latin to Cyrillic (simplified version)
func LatinToCyrillic(s string) string {

	translitMap := map[string]string{
		"ch":   "ч",
		"ght":  "т",
		"kh":   "х",
		"oo":   "у",
		"sh":   "ш",
		"shch": "щ",
		"th":   "с",
		"ts":   "ц",
		"ya":   "я",
		"yo":   "ё",
		"yu":   "ю",
		"zh":   "ж",

		"a": "а",
		"b": "б",
		"c": "ц",
		"d": "д",
		"e": "e",
		"f": "ф",
		"g": "г",
		"h": "х",
		"i": "и",
		"j": "й",
		"k": "к",
		"l": "л",
		"m": "м",
		"n": "н",
		"o": "о",
		"p": "п",
		"q": "к",
		"r": "р",
		"s": "с",
		"t": "т",
		"u": "у",
		"v": "в",
		"w": "в",
		"x": "кс",
		"y": "й",
		"z": "з",
	}

	var result strings.Builder
	i := 0
	for i < len(s) {
		matched := false
		for k, v := range translitMap {
			if strings.HasPrefix(s[i:], k) {
				result.WriteString(v)
				i += len(k)
				matched = true
				break
			}
		}
		if !matched {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// ConvertBBCodeToHTML преобразует BB-коды в HTML
func ConvertBBCodeToHTML(input string) string {
	replacers := []struct {
		bbCode  string
		htmlTag string
	}{
		{`\[b\](.*?)\[/b\]`, "<strong>$1</strong>"},
		{`\[i\](.*?)\[/i\]`, "<em>$1</em>"},
		{`\[u\](.*?)\[/u\]`, "<u>$1</u>"},
		{`\[url\](.*?)\[/url\]`, "<a href=\"$1\">$1</a>"},
		{`\[url=(.*?)\](.*?)\[/url\]`, "<a href=\"$1\">$2</a>"},
		{`\[img\](.*?)\[/img\]`, "<img src=\"$1\" />"},
	}

	result := input
	for _, replacer := range replacers {
		re := regexp.MustCompile(replacer.bbCode)
		result = re.ReplaceAllString(result, replacer.htmlTag)
	}

	return result
}

// ReadEmailsFromFile читает email-адреса из файла, валидирует их и возвращает в виде массива.
// Email-адреса могут быть разделены запятыми, переносами строк или их сочетанием.
func ReadEmailsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var emails []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Разделяем строки по запятым и переносам строк
		for _, email := range strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == '\n'
		}) {
			email = strings.TrimSpace(email)
			if email != "" {
				// email validation
				if _, err := mail.ParseAddress(email); err == nil {
					emails = append(emails, email)
				} else {
					return nil, errors.New("wrong email-format: " + email)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.New("read file error: " + err.Error())
	}

	return emails, nil
}

// Преобразует структуру или срез структур в interface{}, с учетом кастомного тега.
func ConvertStructureWithCustomTag(v interface{}, tagKey string, accessLevel string) (interface{}, error) {
	val := reflect.ValueOf(v)

	// Обработка случая, если v - это срез
	if val.Kind() == reflect.Slice {
		sliceResult := make([]interface{}, val.Len())

		for i := 0; i < val.Len(); i++ {
			singleResult, err := processStruct(val.Index(i), tagKey, accessLevel)
			if err != nil {
				return nil, err
			}
			sliceResult[i] = singleResult
		}

		return sliceResult, nil
	}

	// Обработка случая, если v - это структура
	return processStruct(val, tagKey, accessLevel)
}

// Сериализует структуру или срез структур в JSON, учитывая кастомный тег и входящую переменную
func MarshalJSONWithCustomTag(v interface{}, tagKey string, accessLevel string) ([]byte, error) {
	val := reflect.ValueOf(v)

	// Обработка случая, если v - это срез
	if val.Kind() == reflect.Slice {
		sliceResult := make([]interface{}, val.Len())

		for i := 0; i < val.Len(); i++ {
			singleResult, err := processStruct(val.Index(i), tagKey, accessLevel)
			if err != nil {
				return nil, err
			}
			sliceResult[i] = singleResult
		}

		return json.Marshal(sliceResult)
	}

	// Обработка случая, если v - это структура
	structResult, err := processStruct(val, tagKey, accessLevel)
	if err != nil {
		return nil, err
	}
	return json.Marshal(structResult)
}

// Процессинг для ConvertStructureWithCustomTag и MarshalJSONWithCustomTag
func processStruct(val reflect.Value, tagKey string, accessLevel string) (interface{}, error) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct or a slice of structs")
	}

	result := make(map[string]interface{})
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		customTag := field.Tag.Get(tagKey)
		if customTag != "" {
			allowed := false
			for _, level := range strings.Split(customTag, ",") {
				if level == accessLevel {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		fieldName := field.Name
		if jsonTag != "" {
			fieldName = strings.Split(jsonTag, ",")[0] // Берем только имя поля, без опций
		}
		result[fieldName] = val.Field(i).Interface()
	}

	return result, nil
}

// Возвращает новый срез, содержащий только уникальные значения из исходного среза
func UniqueSlice(slice interface{}) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return slice
	}

	uniqueMap := make(map[interface{}]bool)
	uniqueSlice := reflect.MakeSlice(sliceValue.Type(), 0, 0)

	for i := 0; i < sliceValue.Len(); i++ {
		elem := sliceValue.Index(i).Interface()
		if _, ok := uniqueMap[elem]; !ok {
			uniqueMap[elem] = true
			uniqueSlice = reflect.Append(uniqueSlice, sliceValue.Index(i))
		}
	}

	return uniqueSlice.Interface()
}

// DeepCloneGob выполняет глубокое клонирование объекта src в объект dst с использованием сериализации gob.
// src: Исходный объект для клонирования. Должен быть сериализуемым через gob.
// dst: Указатель на объект, в который будет скопирован клон src. Также должен быть сериализуемым через gob.
// Возвращает error, если произошла ошибка в процессе кодирования или декодирования.
func DeepCloneGob(src, dst interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err // Возвращение ошибки, если кодирование не удалось
	}
	return gob.NewDecoder(&buf).Decode(dst)
}

func ParsePath(path string) (module, action, id string) {
	segments := strings.Split(strings.Trim(path, "/"), "/")

	if len(segments) >= 1 {
		module = segments[0]
	}
	if len(segments) >= 2 {
		action = segments[1]
	}
	if len(segments) >= 3 {
		id = segments[2]
	}
	return
}

func RemoveElements(slice []string, elementsToRemove []string) []string {
	// Создание map для быстрой проверки наличия элемента во втором слайсе
	toRemove := make(map[string]bool)
	for _, item := range elementsToRemove {
		toRemove[item] = true
	}

	// Создание результирующего слайса
	var result []string
	for _, item := range slice {
		if _, found := toRemove[item]; !found {
			// Если элемент не найден в map, добавляем его в результирующий слайс
			result = append(result, item)
		}
	}
	return result
}

func AddUniqueElements(slice []string, elementsToAdd []string) []string {
	// Создание map для отслеживания уникальных элементов
	uniqueElements := make(map[string]bool)
	for _, item := range slice {
		uniqueElements[item] = true
	}

	// Добавление новых элементов, если они уникальны
	for _, item := range elementsToAdd {
		if _, found := uniqueElements[item]; !found {
			slice = append(slice, item)
			uniqueElements[item] = true // Обновляем map, добавляя новый элемент
		}
	}

	return slice
}

func GetLangPathFromLang(lang string) string {
	path := ""
	if lang != "ru" {
		path = "/" + lang
	}
	return path
}

func IsBot(userAgent string) bool {
	botSignatures := []string{
		"applebot",
		"baiduspider",
		"bingbot",
		"bitlybot",
		"bitrix link preview",
		"bot",
		"chrome-lighthouse",
		"crawl",
		"developers.google.com/+/web/snippet",
		"discordbot",
		"duckduckbot",
		"embedly",
		"exabot",
		"facebookexternalhit",
		"facebot",
		"fetch",
		"flipboard",
		"google page speed",
		"google-inspectiontool",
		"googlebot",
		"ia_archiver",
		"linkedinbot",
		"mediapartners",
		"nuzzel",
		"outbrain",
		"pinterest/0.",
		"pinterestbot",
		"quora link preview",
		"qwantify",
		"redditbot",
		"rogerbot",
		"showyoubot",
		"skypeuripreview",
		"slackbot",
		"slurp",
		"sogou",
		"spider",
		"telegrambot",
		"tumblr",
		"twitterbot",
		"vkshare",
		"w3c_validator",
		"whatsapp",
		"xing-contenttabreceiver",
		"yahoo! slurp",
		"yandex",
		"yandexbot",
	}

	for _, bot := range botSignatures {
		if strings.Contains(strings.ToLower(userAgent), bot) {
			return true
		}
	}

	return false
}

// извлекает короткую версию User-Agent с проверкой на бота
func GetShortUserAgent(fullUserAgent string) string {
	// Проверка на бота по ключевым словам
	if IsBot(fullUserAgent) {
		return "Bot"
	}

	// Регулярное выражение для извлечения названия браузера и основной версии
	re := regexp.MustCompile(`(?i)(firefox|chrome|safari|opera|edg|msie|trident|vivaldi|brave|duckduckgo|yandex|ucbrowser|qqbrowser|baidubrowser|nokia|blackberry|puffin|samsungbrowser|maxthon|seamonkey|silk|line|miuibrowser|iron|palemoon|waterfox|epiphany|konqueror|netscape|lynx|midori|slimjet|coc_coc_browser|tizen|avant)[/ ]?([0-9.]*)`)

	// Ищем совпадение
	matches := re.FindStringSubmatch(fullUserAgent)
	if len(matches) > 2 {
		// matches[1] - название браузера, matches[2] - версия
		browserName := matches[1]
		version := strings.Split(matches[2], ".")[0] // Берем только основную версию
		return browserName + " " + version
	}

	// Если браузер не определился, возвращаем "Unknown"
	return "Unknown"
}

func EscapeSpecialChars(input string) string {
	specialChars := []string{`\`, `[`, `]`, `^`, `$`, `.`, `|`, `?`, `*`, `+`, `(`, `)`, `{`, `}`}
	for _, char := range specialChars {
		escapedChar := "\\" + char
		input = strings.ReplaceAll(input, char, escapedChar)
	}
	return input
}

func ArchiveFolder(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, filepath.Dir(source)+string(filepath.Separator))
		if info.IsDir() {
			return nil
		}

		zipFileEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFileEntry, file)
		return err
	})

	return err
}

func ClearDirFilesWithPrefix(dir string, prefix string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) {
			filePath := filepath.Join(dir, file.Name())
			if err := os.Remove(filePath); err != nil {
				logger.Error.Printf("failed to remove file %s: %v\n", filePath, err)
			} else {
				logger.Info.Printf("removed file: %s\n", filePath)
			}
		}
	}

	return nil
}

func ConverStringSliceToIntSlice(strs []string) []int {
	result := []int{}
	for _, el := range strs {
		if val, err := strconv.Atoi(el); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func ReplaceMasks(text string) string {
	currentYear := time.Now().Year()
	text = strings.ReplaceAll(text, "{YEAR}", strconv.Itoa(currentYear))
	return text
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
