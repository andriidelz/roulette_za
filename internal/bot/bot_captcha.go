package bot

import (
	"context"
	"fmt"
	"math/rand/v2"
	"roulette/internal/captcha-go"
	"roulette/internal/logger"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key для измерения активности пользователей за период userActivityExpiration
	// В случае превышения пользователем кол-ва действий выше userActivityLimit
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	userActivityKeyPrefix  = "user:%d:activity"
	userActivityExpiration = 10 * time.Second // Время периода
	userActivityLimit      = 10               // Лимит действий за период userActivityExpiration

	// Redis key для измерения ставок пользователей за период betActivityExpiration
	// В случае превышения пользователем кол-ва ставок выше betActivityLimit
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	betActivityKeyPrefix  = "user:%d:bet_activity"
	betActivityExpiration = 3 * time.Minute // Время периода
	betActivityLimit      = 9               // Лимит ставок за период betActivityExpiration

	// Redis key для проверки одинаковости ставок за период betDuplicateExpiration
	// В случае превышения если все время betDuplicateExpiration пользователь делает ставки только на 1 опцию
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	betDuplicateKeyPrefix  = "user:%d:bet_duplicate"
	betDuplicateExpiration = 30 * time.Minute // Время периода

	// Redis key для проверки через каждые userBetPointsLimit набранных баллов
	userBetPointsPrefix = "user:%d:bet_points"
	userBetPointsLimit  = 50 // Лимит баллов для запуска капчи

	// Redis key для пользователей которые ожидают на проверку капчи
	// В случае нахождения пользователя все дальнейшие действия будут заблокированы
	// до прохождения капчи или истечения userCaptchaExpiration
	userCaptchaKeyPrefix         = "user:%d:captcha"              // необходимо пройти капчу, значение - правильный ответ
	userCaptchaUpdateKey         = "users:captcha_update"         // пользователи которым нужно обновить капчу
	userCaptchaUpdateCountPrefix = "user:%d:captcha_update_count" // кол-во обновлений капчи если нет ответа
)

// captchaUserActivity - Проверка активности и если она слишком высокая - вывод капчи
func (b *Bot) captchaUserActivity(telegramID int64) string {

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Если пользователь в списке на прохожение капчи то никак не реагируем до прохождения капчи
	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, telegramID)
	count, err := b.redisDB.Exists(cont, captchaKey).Result()

	if err != nil {
		logger.Error.Printf("Error check captchaKey %d: %v", telegramID, err)
		return "wait"
	}
	if count > 0 {
		return "wait"
	}

	// Проверяем активность пользователя за период
	userActivityKey := fmt.Sprintf(userActivityKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userActivityKey).Int64()
	if err == redis.Nil {
		// За текущий период еще не было активности пользователя, создаем запись
		err = b.redisDB.Set(cont, userActivityKey, 1, userActivityExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	val++

	if val <= userActivityLimit {

		// Пользователь не превышает активность
		// Обновляем активность пользователя за минуту
		// с указанием redis.KeepTTL для того чтобы не обновлялось время expiration
		err = b.redisDB.Set(cont, userActivityKey, val, redis.KeepTTL).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}
	logger.Error.Println(telegramID, " captchaUserActivity")
	// Превышение активности за период выше лимита - необходимо пройти капчу
	return "needCaptcha"
}

// captchaBetActivity - Проверка активности - беспрерывная игра
func (b *Bot) captchaBetActivity(telegramID int64) string {

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем ставки пользователя за период
	betActivityKey := fmt.Sprintf(betActivityKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, betActivityKey).Int64()
	if err == redis.Nil {
		// За текущий период еще не было ставок пользователя, создаем запись
		err = b.redisDB.Set(cont, betActivityKey, 1, betActivityExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	val++

	if val < betActivityLimit {

		// Пользователь не превышает активность
		// Обновляем активность пользователя за период
		// с указанием redis.KeepTTL для того чтобы не обновлялось время expiration
		err = b.redisDB.Set(cont, betActivityKey, val, redis.KeepTTL).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	logger.Error.Println(telegramID, " captchaBetActivity")
	// Превышение ставок за период выше лимита - необходимо пройти капчу
	// Удаляем ключ
	_, err = b.redisDB.Del(cont, betActivityKey).Result()
	if err != nil {
		logger.Error.Printf("Error Del %d: %v", telegramID, err)
	}

	return "needCaptcha"
}

// captchaBetDuplicate - Проверка активности - ставка на одну и ту же опцию
func (b *Bot) captchaBetDuplicate(telegramID int64, option string) string {

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	userBetKey := fmt.Sprintf(betDuplicateKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userBetKey).Result()
	if err == redis.Nil {
		// За период еще не было ставок пользователя, создаем запись
		err = b.redisDB.Set(cont, userBetKey, option, betDuplicateExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	// Ставка не повторилась с последней
	if val != option {
		// Создаем новую проверку с отсчетом времени сначала
		err = b.redisDB.Set(cont, userBetKey, option, betDuplicateExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	// Ставка повторилась, проверяем время истечения.
	ttl, err := b.redisDB.TTL(cont, userBetKey).Result()
	if err != nil {
		logger.Error.Printf("Error TTL %d: %v", telegramID, err)
		return ""
	}
	if ttl.Seconds() < 20 {
		// Если время подходит к указанному expiration то он делал ставки все последнее время одинаковые.
		// Удаляем ключ
		_, err = b.redisDB.Del(cont, userBetKey).Result()
		if err != nil {
			logger.Error.Printf("Error del %d: %v", telegramID, err)
			return ""
		}
		logger.Error.Println(telegramID, " captchaBetDuplicate")
		// Выводим капчу
		return "needCaptcha"
	}

	return ""
}

// captchaBetPoints - Проверка активности - кол-во набранных баллов
func (b *Bot) captchaBetPoints(telegramID int64, point int) string {

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	userBetPointKey := fmt.Sprintf(userBetPointsPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userBetPointKey).Int()
	if err == redis.Nil {
		// За период еще не было получено баллов пользователя, создаем запись
		err = b.redisDB.Set(cont, userBetPointKey, point, 0).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}

	val += point

	if val <= userBetPointsLimit {

		// Пользователь не превышает лимит балов для запуска капчи
		// Обновляем кол-во балов
		err = b.redisDB.Set(cont, userBetPointKey, val, 0).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return ""
	}
	logger.Error.Println(telegramID, " captchaBetPoints")
	// Превышение активности за период выше лимита - необходимо пройти капчу
	return "needCaptcha"
}

// captchaMessage - Создание капчи
func (b *Bot) captchaMessage(telegramID int64, language string) MessageOptions {
	// Остановка игры и возврат в главное меню
	b.gameHandler.HandleStopGameButton(telegramID)
	b.sendMainMenu(telegramID, language)

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	textLen := 4
	correctText := captcha.RandomText(textLen)
	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, telegramID)

	// Список пользователей которые ожидают капчу
	// Удаляем чтобы избежать дублирования
	// `count = 0` means remove all elements matching 'value'.
	_, err := b.redisDB.LRem(cont, userCaptchaUpdateKey, 0, telegramID).Result()
	if err != nil {
		logger.Error.Printf("Failed to LREM: %d: %v", telegramID, err)
	}

	// Добавляем пользователя в список ожидающих на подтверждения капчи
	err = b.redisDB.RPush(cont, userCaptchaUpdateKey, telegramID).Err()
	if err != nil {
		logger.Error.Printf("Failed to RPUSH: %d: %v", telegramID, err)
	}

	// Добавляем пользователя с указанием правильного ответа
	err = b.redisDB.Set(cont, captchaKey, correctText, 0).Err()
	if err != nil {
		logger.Error.Printf("Error Set %d: %v", telegramID, err)
	}

	wrongTextOne := captcha.RandomText(textLen)
	wrongTextTwo := captcha.RandomText(textLen)
	wrongTextThree := captcha.RandomText(textLen)
	// Создаем линию кнопок
	lines := []telego.InlineKeyboardButton{
		{Text: correctText, CallbackData: CallbackCaptcha + correctText},
		{Text: wrongTextOne, CallbackData: CallbackCaptcha + wrongTextOne},
		{Text: wrongTextTwo, CallbackData: CallbackCaptcha + wrongTextTwo},
		{Text: wrongTextThree, CallbackData: CallbackCaptcha + wrongTextThree},
	}
	// Перемешиваем кнопки
	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})
	// Создаем inline-клавиатуру для выбора
	nicknameKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{lines},
	}
	captchaText := b.service.GetText("captcha_text", language)
	mess := MessageOptions{
		Text:           captchaText,
		InlineKeyboard: nicknameKeyboard,
	}
	filepath := "./internal/captcha-go/"
	filename := correctText + ".png"

	// Генерируем капчу как картинку
	if err := captcha.GenerateCaptcha(correctText, filename,
		captcha.DefaultOption(
			filepath,
			filepath+"fonts/")); err != nil {
		logger.Error.Println("Error GenerateCaptcha:", err)
		// Если была ошибка генерации вставляем как текст в сообщение
		mess.Text += "\n " + correctText
	} else {
		mess.PhotoPath = filepath + filename
		mess.DelPhoto = true // удаляем сгенерированное фото капчи
	}

	return mess
}

// captchaCheck - Проверка капчи
func (b *Bot) captchaCheck(query *telego.CallbackQuery) {

	user := query.From
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		// Регистрация пользователя, если он не найден
		dbUser, err = b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, "", user.LanguageCode)
		if err != nil {
			logger.Error.Printf("Error registering user: %v", err)
			return
		}
	}

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, user.ID)

	val, err := b.redisDB.Get(cont, captchaKey).Result()
	if err != nil {
		logger.Error.Printf("Error Get %d: %v", user.ID, err)
		return
	}

	// Извлекаем код капчи
	option := strings.TrimPrefix(query.Data, CallbackCaptcha)
	if val != option {
		// невірна відповідь на капчу
		// Відповідаєм на callback текстом что капча невірна
		b.answerCallbackQuery(query.ID, b.service.GetText("wrongcapcha_mes", language), true)

		// присилаєм нову капчу
		b.SendMessage(user.ID, b.captchaMessage(user.ID, language))
		return
	}

	// Капча коректна

	// Используем Redis Pipeline для выполнения нескольких команд за один сетевой запрос
	pipe := b.redisDB.Pipeline()

	// Убираем пользователя из списка ожидающих на подтверждения капчи
	pipe.LRem(cont, userCaptchaUpdateKey, 0, user.ID)
	pipe.Del(cont, fmt.Sprintf(userCaptchaUpdateCountPrefix, user.ID))
	// Убираем пользователю правильный ответ капчи
	pipe.Del(cont, captchaKey)

	// Очищаем все условия
	// Удаляем активность пользователя
	pipe.Del(cont, fmt.Sprintf(userActivityKeyPrefix, user.ID))
	// Удаляем ставки пользователя
	pipe.Del(cont, fmt.Sprintf(betActivityKeyPrefix, user.ID))
	// Удаляем ставки на одну и ту же опцию
	pipe.Del(cont, fmt.Sprintf(betDuplicateKeyPrefix, user.ID))
	// Удаляем кол-во набранных баллов
	pipe.Del(cont, fmt.Sprintf(userBetPointsPrefix, user.ID))

	// Выполняем все операции за один запрос
	_, pipeErr := pipe.Exec(cont)
	if pipeErr != nil {
		logger.Error.Printf("Error in pipeline execution: %v", pipeErr)
	}

	correctText := b.service.GetText("captcha_correct", language)

	// Отвечаем на callback c текстом что капча успешно решена
	b.answerCallbackQuery(query.ID, correctText, true)
}
