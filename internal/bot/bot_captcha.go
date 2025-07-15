package bot

import (
	"context"
	"fmt"
	"math/rand/v2"
	"roulette/internal/captcha-go"
	"roulette/internal/logger"
	"time"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key для измерения активности пользователей за период userActivityExpiration
	// В случае превышения пользователем кол-ва действий выше userActivityLimit
	// он будет записан в userCaptchaKeyPrefix и ему будет отправлена капча
	userActivityKeyPrefix  = "user:%d:activity"
	userActivityExpiration = time.Minute // Время периода
	userActivityLimit      = 15          // Лимит действий за период userActivityExpiration
	// Redis key для пользователей которые ожидают на проверку капчи
	// В случае нахождения пользователя все дальнейшие действия будут заблокированы
	// до прохождения капчи или истечения userCaptchaExpiration
	userCaptchaKeyPrefix  = "user:%d:captcha"
	userCaptchaExpiration = time.Hour // Время удаления проверки капчи
)

// checkUserActivity - Проверка активности и если она слишком высокая - вывод капчи
func (b *Bot) checkUserActivity(telegramID int64, language string) (string, MessageOptions) {

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Если пользователь в списке на прохожение капчи то никак не реагируем до прохождения капчи
	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, telegramID)
	count, err := b.redisDB.Exists(cont, captchaKey).Result()

	if err != nil {
		logger.Error.Printf("Error check captchaKey %d: %v", telegramID, err)
		return "wait", MessageOptions{}
	}
	if count > 0 {
		return "wait", MessageOptions{}
	}

	// Проверяем активность пользователя за период
	userActivityKey := fmt.Sprintf(userActivityKeyPrefix, telegramID)
	val, err := b.redisDB.Get(cont, userActivityKey).Int64()
	if err == redis.Nil {
		// За текущий период еще не было активности пользователя, создаем запись
		err = b.redisDB.Set(cont, userActivityKey, 1, userActivityExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set userActivityKey %d: %v", telegramID, err)
		}
		return "", MessageOptions{}
	}

	val++

	if val <= userActivityLimit {

		// Пользователь не превышает активность
		// Обновляем активность пользователя за минуту
		// с указанием redis.KeepTTL для того чтобы не обновлялось время expiration
		err = b.redisDB.Set(cont, userActivityKey, val, redis.KeepTTL).Err()
		if err != nil {
			logger.Error.Printf("Error Set userActivityKey %d: %v", telegramID, err)
		}
		return "", MessageOptions{}
	}

	// Превышение активности за период выше лимита - необходимо пройти капчу

	// Остановка игры и возврат в главное меню
	b.gameHandler.HandleStopGameButton(telegramID)
	b.sendMainMenu(telegramID, language)

	// Добавляем пользователя в список ожидающих на подтверждения капчи
	// value не важно, проверка идет по наличию ключа
	err = b.redisDB.Set(cont, captchaKey, "value", userCaptchaExpiration).Err()
	if err != nil {
		logger.Error.Printf("Error Set userCaptchaExpiration %d: %v", telegramID, err)
	}

	textLen := 4
	correctText := captcha.RandomText(textLen)

	// Создаем линию кнопок
	lines := []telego.InlineKeyboardButton{
		{Text: correctText, CallbackData: CallbackCaptchaCorrect},
		{Text: captcha.RandomText(textLen), CallbackData: CallbackCaptchaIncorrect},
		{Text: captcha.RandomText(textLen), CallbackData: CallbackCaptchaIncorrect},
		{Text: captcha.RandomText(textLen), CallbackData: CallbackCaptchaIncorrect},
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
		mess.DelPhoto = true
		logger.Info.Println("CAPTCHA successful:", filepath+filename, telegramID)
	}

	return "needCaptcha", mess
}

// captchaCorrect - Отправка уведомления об успешном прохождении капчи
func (b *Bot) captchaCorrect(query *telego.CallbackQuery) {

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

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Убираем пользователя из списка ожидающих на подтверждения капчи
	captchaKey := fmt.Sprintf(userCaptchaKeyPrefix, user.ID)
	_, err = b.redisDB.Del(cont, captchaKey).Result()
	if err != nil {
		logger.Error.Printf("Error del captchaKey %d: %v", user.ID, err)
		return
	}

	// Удаляем активность пользователя
	userActivityKey := fmt.Sprintf(userActivityKeyPrefix, user.ID)
	_, err = b.redisDB.Del(cont, userActivityKey).Result()
	if err != nil {
		logger.Error.Printf("Error del userActivityKey %d: %v", user.ID, err)
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	correctText := b.service.GetText("captcha_correct", language)

	// Отвечаем на callback c текстом что капча успешно решена
	b.answerCallbackQuery(query.ID, correctText, true)
}
