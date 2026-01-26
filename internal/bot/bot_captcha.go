package bot

import (
	"context"
	"fmt"
	"math/rand/v2"
	"roulette/internal/captcha-go"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key для измерения активности пользователей за период captcha_bet_activity_ttl
	// В случае превышения пользователем кол-ва действий выше captcha_user_activity
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	userActivityKeyPrefix = "user:%d:activity"

	// Redis key для измерения ставок пользователей за период captcha_bet_activity_ttl
	// В случае превышения пользователем кол-ва ставок выше captcha_bet_activity
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	betActivityKeyPrefix = "user:%d:bet_activity"

	// Redis key для проверки одинаковости ставок за период captcha_bet_duplicate_ttl
	// В случае превышения если все время captcha_bet_duplicate_ttl пользователь делает ставки только на 1 опцию
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	betDuplicateKeyPrefix = "user:%d:bet_duplicate"

	// Redis key для проверки через каждые userBetPointsLimit набранных баллов
	userBetPointsPrefix = "user:%d:bet_points"

	// Redis key для пользователей которые ожидают на проверку капчи
	// В случае нахождения пользователя все дальнейшие действия будут заблокированы
	// до прохождения капчи
	userCaptchaKeyPrefix  = "user:%d:captcha"     // необходимо пройти капчу, значение - правильный ответ
	userCaptchaMessPrefix = "user:%d:captcha_mes" // telegram id останнього повідомлення з капчею для оновлення повідомлення

	// Redis key для пользователей которые временно забанены
	// В случае нахождения пользователя все дальнейшие действия будут заблокированы
	// до прохождения истечения expiration
	userCaptchaBanCountPrefix = "user:%d:captcha_ban_count" // кол-во полученых банов - если больше 3 то блокируем на больший термин

	CallbackCaptcha        = "captcha_"
	CallbackCaptchaRefresh = "captcha_refresh"
)

// captchaUserActivity - Проверка активности и если она слишком высокая - вывод капчи
func (b *Bot) captchaUserActivity(telegramID int64) bool {

	if b.testMode {
		return false
	}

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var limit int64
	b.settingsMutex.Lock()
	limit, _ = b.settings["captcha_user_activity"]
	ttl, _ := b.settings["captcha_bet_activity_ttl"]
	b.settingsMutex.Unlock()

	// Проверяем активность пользователя за период
	userActivityKey := fmt.Sprintf(userActivityKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userActivityKey).Int64()
	if err == redis.Nil {
		// За текущий период еще не было активности пользователя, создаем запись
		err = b.redisDB.Set(cont, userActivityKey, 1, time.Second*time.Duration(ttl)).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}

	val++
	if val <= limit {

		// Пользователь не превышает активность
		// Обновляем активность пользователя за минуту
		// с указанием redis.KeepTTL для того чтобы не обновлялось время expiration
		err = b.redisDB.Set(cont, userActivityKey, val, redis.KeepTTL).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}
	// Превышение активности за период выше лимита - необходимо пройти капчу
	return true
}

// captchaBetActivity - Проверка активности - беспрерывная игра
func (b *Bot) captchaBetActivity(telegramID int64) bool {

	if b.testMode {
		return false
	}

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var limit int64
	b.settingsMutex.Lock()
	limit, _ = b.settings["captcha_bet_activity"]
	ttl, _ := b.settings["captcha_bet_activity_ttl"]
	b.settingsMutex.Unlock()

	// Проверяем ставки пользователя за период
	betActivityKey := fmt.Sprintf(betActivityKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, betActivityKey).Int64()
	if err == redis.Nil {
		// За текущий период еще не было ставок пользователя, создаем запись
		err = b.redisDB.Set(cont, betActivityKey, 1, time.Second*time.Duration(ttl)).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}

	val++

	if val < limit {

		// Пользователь не превышает активность
		// Обновляем активность пользователя за период
		// с указанием redis.KeepTTL для того чтобы не обновлялось время expiration
		err = b.redisDB.Set(cont, betActivityKey, val, redis.KeepTTL).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}

	logger.Error.Println(telegramID, " captchaBetActivity")
	// Превышение ставок за период выше лимита - необходимо пройти капчу
	// Удаляем ключ
	_, err = b.redisDB.Del(cont, betActivityKey).Result()
	if err != nil {
		logger.Error.Printf("Error Del %d: %v", telegramID, err)
	}

	return true
}

// captchaBetDuplicate - Проверка активности - ставка на одну и ту же опцию
func (b *Bot) captchaBetDuplicate(telegramID int64, option string) bool {

	if b.testMode {
		return false
	}

	b.settingsMutex.Lock()
	ttlDuplicate, _ := b.settings["captcha_bet_duplicate_ttl"]
	b.settingsMutex.Unlock()

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	userBetKey := fmt.Sprintf(betDuplicateKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userBetKey).Result()
	if err == redis.Nil {
		// За период еще не было ставок пользователя, создаем запись
		err = b.redisDB.Set(cont, userBetKey, option, time.Second*time.Duration(ttlDuplicate)).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}

	// Ставка не повторилась с последней
	if val != option {
		// Создаем новую проверку с отсчетом времени сначала
		err = b.redisDB.Set(cont, userBetKey, option, time.Second*time.Duration(ttlDuplicate)).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}

	// Ставка повторилась, проверяем время истечения.
	ttl, err := b.redisDB.TTL(cont, userBetKey).Result()
	if err != nil {
		logger.Error.Printf("Error TTL %d: %v", telegramID, err)
		return false
	}
	if ttl.Seconds() < 20 {
		// Если время подходит к указанному expiration то он делал ставки все последнее время одинаковые.
		// Удаляем ключ
		_, err = b.redisDB.Del(cont, userBetKey).Result()
		if err != nil {
			logger.Error.Printf("Error del %d: %v", telegramID, err)
			return false
		}
		logger.Error.Println(telegramID, " captchaBetDuplicate")
		// Выводим капчу
		return true
	}

	return false
}

// captchaBetPoints - Проверка активности - кол-во набранных баллов
func (b *Bot) captchaBetPoints(telegramID int64, point int) bool {

	if b.testMode {
		return false
	}

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
		return false
	}

	val += point

	var limit int64
	b.settingsMutex.Lock()
	limit, _ = b.settings["captcha_bet_points"]
	b.settingsMutex.Unlock()
	if int64(val) <= limit {

		// Пользователь не превышает лимит балов для запуска капчи
		// Обновляем кол-во балов
		err = b.redisDB.Set(cont, userBetPointKey, val, 0).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", telegramID, err)
		}
		return false
	}
	logger.Error.Println(telegramID, " captchaBetPoints")
	// Превышение активности за период выше лимита - необходимо пройти капчу
	return true
}

// setCaptchaStatus - переведення користувача в статус UserStatusCaptcha
func (b *Bot) setCaptchaStatus(telegramID int64, reason string) {
	// reason - captcha create:
	// "captcha_user_activity"
	// "captcha_bet_points"
	// "captcha_bet_activity"
	// "captcha_bet_duplicate"

	dbUser, err := b.getUser(telegramID)
	if err != nil {
		logger.Error.Printf("Failed to getUser: %v", err)
	}

	b.settingsMutex.Lock()
	captchaTTL, _ := b.settings["captcha_ttl"]
	b.settingsMutex.Unlock()

	// create new record
	log := &models.UserBanLog{
		UserID:     dbUser.ID,
		TypeStatus: UserStatusCaptcha,
		Reason:     reason,
		Active:     true,
		UntilTo:    time.Now().Add(time.Minute * time.Duration(captchaTTL)),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	// Save to database
	if err := b.service.GetRepo().CreateBanLog(log); err != nil {
		logger.Error.Printf("Failed to create ban log: %v", err)
	}

	// Оновлюєм статус на той що очікує капчу
	dbUser.Status = UserStatusCaptcha
	err = b.service.UpdateUser(dbUser)
	if err != nil {
		logger.Error.Printf("Error updating user: %v", err)
	}
	b.updateUserCache(telegramID)

}

// captchaMessage - Відправка капчі
func (b *Bot) captchaMessage(telegramID int64, language, reason string) MessageOptions {

	// "refresh"
	// "wrong"
	// "stage"
	// "new"

	b.settingsMutex.Lock()
	captchaTTL, _ := b.settings["captcha_ttl"]
	countRefreshCaptcha, _ := b.settings["captcha_refresh_count"]
	b.settingsMutex.Unlock()

	showRefresh := true

	dbUser, err := b.getUser(telegramID)
	if err != nil {
		logger.Error.Printf("Failed to getUser: %v", err)
	}

	switch reason {
	case "refresh", "wrong", "stage":
		// update log
		res, err := b.service.GetRepo().GetActiveBanLog(dbUser.ID)
		if err != nil {
			logger.Error.Printf("Failed to get ban log: %v", err)
		}

		switch reason {
		case "refresh":
			res.Refresh += 1
		case "wrong":
			res.Wrong += 1
		case "stage":
			res.Stage += 1
		}

		// Перевірка кількості оновлень капчі
		if res.Refresh >= int(countRefreshCaptcha) {
			showRefresh = false
		}

		// Save to database
		if err := b.service.GetRepo().UpdateBanLog(&res); err != nil {
			logger.Error.Printf("Failed to create ban log: %v", err)
		}

	case "new":
		// пропускаємо
	default:
		logger.Error.Println("Unknown captcha status: ", reason)
	}

	// Остановка игры и возврат в главное меню
	b.gameHandler.stopGame(telegramID)
	b.sendMainMenu(telegramID, language)

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	textLen := 4
	correctText := captcha.RandomText(textLen)
	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, telegramID)

	// Добавляем пользователя с указанием правильного ответа
	err = b.redisDB.Set(cont, captchaKey, correctText, time.Minute*time.Duration(captchaTTL)).Err()
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

	if showRefresh {
		nicknameKeyboard.InlineKeyboard = append(nicknameKeyboard.InlineKeyboard,
			[]telego.InlineKeyboardButton{
				{Text: b.getText("captcha_refresh", language),
					CallbackData: CallbackCaptchaRefresh}})
	}

	captchaText := b.getText("captcha_text", language)
	mess := MessageOptions{
		Text:           captchaText,
		InlineKeyboard: nicknameKeyboard,
		MethodName:     sendCaptcha,
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

	dbUser, err := b.getUser(user.ID)
	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
	}

	res, err := b.service.GetRepo().GetActiveBanLog(dbUser.ID)
	if err != nil {
		logger.Error.Printf("Failed to get ban log: %v", err)
	}

	nextCaptcha := false

	cont, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, user.ID)
	captchaBanCountKey := fmt.Sprintf(userCaptchaBanCountPrefix, user.ID)

	// Извлекаем код капчи
	option := strings.TrimPrefix(query.Data, CallbackCaptcha)

	val, err := b.redisDB.Get(cont, captchaKey).Result()
	if err != nil {
		logger.Error.Printf("Error Get %d: %v", user.ID, err)
		// Ключ не знайдений, можливо - капча була вже вирішена і це повторний запит
		// або пройшов час captchaTTL
	} else if val != option {
		// невірна відповідь на капчу

		b.settingsMutex.Lock()
		userCaptchaWrongLimit, _ := b.settings["captcha_wrong_count"]
		userCaptchaBanLimit, _ := b.settings["captcha_ban_count"]
		shortTTL, _ := b.settings["captcha_ban_short_ttl"]
		longTTL, _ := b.settings["captcha_ban_long_ttl"]
		b.settingsMutex.Unlock()

		if res.Wrong <= int(userCaptchaWrongLimit) {

			// Користувач не перевищує ліміт
			// Відповідаєм на callback текстом что капча невірна
			b.answerCallbackQuery(query.ID, b.getText("wrongcapcha_mes", language), true)

			// присилаєм нову капчу
			b.MakeRequestDeferred(user.ID, 9, b.captchaMessage(user.ID, language, "wrong"))
			return
		}

		// неправильна відповідь на капчу 3 рази підряд - бан
		durationTTL := shortTTL

		// Оновлюєм статус на заблокований
		dbUser.Status = UserStatusLockout
		err = b.service.UpdateUser(dbUser)
		if err != nil {
			logger.Error.Printf("Error updating user: %v", err)
		}
		b.updateUserCache(user.ID)

		banCount, err := b.redisDB.Get(cont, captchaBanCountKey).Int64()
		if err == redis.Nil {
			// За період ще не було банів, створюємо запис
			err = b.redisDB.Set(cont, captchaBanCountKey, 0, 24*time.Hour).Err()
			if err != nil {
				logger.Error.Printf("Error Set %d: %v", user.ID, err)
			}
		}

		banCount++

		if banCount <= userCaptchaBanLimit {
			// якщо кількість банів за день менше userCaptchaBanLimit то бан на shortTTL

			// збільшуєм кількість банів
			// з вказанням redis.KeepTTL для того щоб не оновлювався expiration
			err = b.redisDB.Set(cont, captchaBanCountKey, banCount, redis.KeepTTL).Err()
			if err != nil {
				logger.Error.Printf("Error Set %d: %v", user.ID, err)
			}

			// Капча невірна і користувач іде в бан
			b.SendMessage(query.Message.GetChat().ID, b.prepareMessage("banactive_mes", language))
		} else {
			// більше userCaptchaBanLimit банів на userBanShortExpiration за день
			// - бан на longTTL
			durationTTL = longTTL

			// Капча невірна і користувач іде в бан
			b.SendMessage(query.Message.GetChat().ID, b.prepareMessage("banactive_mes3", language))
		}

		// create new record
		log := &models.UserBanLog{
			UserID:     dbUser.ID,
			TypeStatus: UserStatusLockout,
			Reason:     "wrongcapcha",
			Active:     true,
			UntilTo:    time.Now().Add(time.Minute * time.Duration(durationTTL)),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		// Save to database
		if err := b.service.GetRepo().CreateBanLog(log); err != nil {
			logger.Error.Printf("Failed to create ban log: %v", err)
		}

		// Отвечаем на callback, чтобы убрать индикатор загрузки
		b.answerCallbackQuery(query.ID, "", false)
		return
	} else {
		// Капча коректна

		b.settingsMutex.Lock()
		countNeedCaptcha, _ := b.settings["captcha_need_count"]
		b.settingsMutex.Unlock()

		// **** Перевірка мультифакторності ****
		if res.Wrong > 0 {
			if int(countNeedCaptcha) > res.Stage+1 {
				// кількість необхідних капч менше необхідного
				nextCaptcha = true
			}
		}
		// ************

		if nextCaptcha {
			// По умовам користувач має пройти кілька капч підряд

			// Відповідаєм на callback текстом что капча вірна але потрібно пройти ще
			text := b.getText("captcha_stage_title", language)
			text = fmt.Sprintf(text, res.Stage+1, countNeedCaptcha) + " " + b.getText("captcha_next", language)
			b.answerCallbackQuery(query.ID, text, true)

			// присилаєм нову капчу
			b.MakeRequestDeferred(user.ID, 9, b.captchaMessage(user.ID, language, "stage"))
			return
		}
	}

	res.Active = false
	// Save to database
	if err := b.service.GetRepo().UpdateBanLog(&res); err != nil {
		logger.Error.Printf("Failed to create ban log: %v", err)
	}

	// отримуємо ID останньої капчі
	captchaMess := fmt.Sprintf(userCaptchaMessPrefix, user.ID)
	messageID, err := b.redisDB.Get(cont, captchaMess).Int()
	if err != nil && err != redis.Nil {
		logger.Error.Printf("Error Get %d: %v", user.ID, err)
	}

	// Всі умови виконані, очищаєм все що пов'язано з капчею

	// Оновлюєм статус на активний
	dbUser.Status = UserStatusActive
	err = b.service.UpdateUser(dbUser)
	if err != nil {
		logger.Error.Printf("Error updating user: %v", err)
	}
	b.updateUserCache(user.ID)

	// Используем Redis Pipeline для выполнения нескольких команд за один сетевой запрос
	pipe := b.redisDB.Pipeline()

	// Убираем пользователю правильный ответ капчи
	pipe.Del(cont, captchaKey)
	pipe.Del(cont, fmt.Sprintf(userCaptchaMessPrefix, user.ID))

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

	b.answerCallbackQuery(query.ID, "", false)

	options := b.prepareMessage("captcha_correct", language)
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: b.getText("next_round", language), CallbackData: CallbackStartRound},
			},
		},
	}

	if messageID > 0 {
		// 	Видаляємо повідомлення з капчею
		err = b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID: messageID,
		})
		if err != nil {
			logger.Error.Printf("failed to delete message: %v", err)
		}
	}

	b.SendMessage(query.Message.GetChat().ID, options)
}
