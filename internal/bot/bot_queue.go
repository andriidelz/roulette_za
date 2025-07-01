package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"roulette/internal/logger"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
)

// https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this
// Лимиты приблизительные, настоящие отличаются по времени, кол-ву и от типа сообщения
// Около 30 сообщений на секунду, и не больше 1 сообщения пользователю в секунду
// Телеграм позволяет краткосрочные всплески превышения лимитов
// но при длительной активности сверх лимитов будет 429 ошибка
// Время запрета на взаимодействие устанавливается для каждого пользователя индивидуально
// С кем лимиты не превышены будет отправляться без задержек
// При превышении лимитов отправки конкретному пользователю бан на отправку может быть 20 мин - 3 часа и больше

// **********
// TODO Возможно удаление сообщений и отправка стикеров тоже стоит сюда перенести
// точные лимиты на них не понятны
// **********

const (
	sendInterval                 = time.Second // Интервал срабатывания отправки
	limitMessPerInterval         = 30          // Максимум сообщений которые могут быть отправлены за интервал
	coolDownInterval       int64 = 1           // Время задержки отправки
	coolDownIntervalErr    int64 = 5           // Время задержки отправки при ошибке
	coolDownIntervalErr429 int64 = 60          // Время задержки отправки при ошибке 429 - Too Many Requests

	// Redis key для получения информации по пользователю - время следующей отправки(Unix timestamp)
	userNextSendTimeKeyPrefix = "user:%d:next_send_time"
	// Pattern чтобы найти всех пользователей
	userNextSendTimeKeyPattern = "user:*:next_send_time"
	// Redis key для очереди
	userQueueKeyPrefix = "user:%d:queue"
	// Redis key для времени отправки ошибок
	userErrorKeyPrefix  = "user:%d:error"
	userErrorExpiration = 40 * time.Second // Redis key для времени отправки ошибок
)

// MakeRequestDeferred Постановка сообщения в очередь на отправку
func (b *Bot) MakeRequestDeferred(chatID int64, firstInOrder bool, param MessageOptions) error {

	// Создание активного юзера если его не существует и установка немедленной отправки
	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nextTimeKey := fmt.Sprintf(userNextSendTimeKeyPrefix, chatID)
	err := b.redisDB.SetNX(cont, nextTimeKey, time.Now().Unix(), 0).Err() // Текущее время
	if err != nil {
		logger.Error.Printf("Error update user %d: %v", chatID, err)
	}

	// Отправка сообщения в очередь
	// Превращаем структуру
	queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
	message, err := json.Marshal(param)
	if err != nil {
		logger.Error.Println("Error Marshal:", err, chatID)
		return err
	}

	// Добавляем в очередь на отправку
	var length int64
	if firstInOrder {
		// Поставка впереди очереди
		length, err = b.redisDB.LPush(cont, queueKey, message).Result()
	} else {
		// По дефолту в конец list
		length, err = b.redisDB.RPush(cont, queueKey, message).Result()
	}
	if err != nil {
		logger.Error.Printf("Error Push message for user %d: %v", chatID, err)
		return err
	}

	if length > 100 {
		b.MakeRequestDeferredErr(chatID, "send_queue_error")

		logger.Error.Printf("user %d More than 100 messages are waiting", chatID)
		return fmt.Errorf("user %d More than 100 messages are waiting", chatID)
	}

	return nil
}

func (b *Bot) MakeRequestDeferredErr(chatID int64, errText string) {

	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Проверяем была ли отправка за период указанный в userErrorExpiration сообщений об ошибке
	errKey := fmt.Sprintf(userErrorKeyPrefix, chatID)
	_, err := b.redisDB.Get(cont, errKey).Result()
	if err == redis.Nil {
		// Если не было таких сообщений то создаем сообщение об ошибке и ставим ключ

		// Получаем пользователя для определения языка
		user, userErr := b.service.GetUser(chatID)
		if userErr != nil {
			logger.Error.Printf("Error getting user %d: %v", chatID, userErr)
		}

		language := user.LanguageCode
		if language == "" {
			language = "en"
		}

		errSendText := b.service.GetText(errText, language)
		options := MessageOptions{
			Text:      fmt.Sprintf(errSendText, 100),
			ParseMode: telego.ModeHTML,
		}

		// Устанавливаем ключ отправки сообщения об ошибке
		// value не важно, проверка идет по наличию ключа
		err = b.redisDB.Set(cont, errKey, "value", userErrorExpiration).Err()
		if err != nil {
			logger.Error.Printf("Error Set userErrorExpiration %d: %v", chatID, err)
		}
		// Ставим в приоритетные уведомление об ошибке
		b.MakeRequestDeferred(chatID, true, options)
	}
}

// sendBotQueue цикл для равномерной отправки сообщений пользователям
func (b *Bot) sendBotQueue() {
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	cont := context.Background()

	for range ticker.C {
		countMess := 0

		activeUsers, err := b.getUsersFromRedis()
		if err != nil {
			logger.Error.Printf("Error refreshing active user IDs: %v\n", err)
			continue
		}

		for chatID, nextSend := range activeUsers {
			logger.Error.Println(chatID, nextSend)
			// Проверяем кол-во отправленных сообщений и
			// проверяем пользователя и
			// если пришло время то отправляем сообщение из очереди
			if countMess <= limitMessPerInterval && nextSend <= time.Now().Unix() {

				// Пробуем получить сообщение для пользователя из очереди
				queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
				mesByte, err := b.redisDB.LPop(cont, queueKey).Bytes()
				if err != nil {
					if err != redis.Nil {
						logger.Error.Printf("Error LPop for %s: %v\n", queueKey, err)
					}
					// Нет сообщений или ошибка при получении
					continue
				}
				// Если сообщение есть превращаем его в структуру
				options := MessageOptions{}
				err = json.Unmarshal(mesByte, &options)
				if err != nil {
					logger.Error.Printf("Error Unmarshal for %s: %v\n", queueKey, err)
					continue
				}

				countMess++
				b.sendDeferredMessage(chatID, options)
			}
		}
		logger.Info.Println("End countMess: ", countMess)
	}
}

func (b *Bot) sendDeferredMessage(chatID int64, options MessageOptions) {

	var err error
	switch options.MethodName {
	case editMessageText:
		_, err = b.updateText(chatID, options.MessageID, options)
	case sendPhoto:
		_, err = b.sendPhoto(chatID, options)
	case editMessageMedia:
		_, err = b.updatePhotoByFileID(chatID, options.MessageID, options)
	default:
		_, err = b.sendText(chatID, options)
	}

	var sleep int64
	if err != nil {
		// Если была ошибка отправки то изменяем время ожидания для данного пользователя
		// Проверка ошибок, если Too Many Requests то парсим время задержки
		// api: 429 "Too Many Requests: retry after 9540", migrate to chat ID: 0, retry after: 9540
		if strings.Contains(err.Error(), "retry after") {
			_, after, found := strings.Cut(err.Error(), "retry after:")
			if found {
				// Пробуем очистить и превратить в число
				after = strings.ReplaceAll(strings.TrimSpace(after), `"`, "")
				if retryTime, err := strconv.Atoi(after); err == nil && retryTime >= 1 {
					sleep = int64(retryTime)
				}
			}
			if sleep <= 0 {
				// Не удалось получить время задержки
				sleep = coolDownIntervalErr429
				logger.Error.Printf("Parse 429 error: %v", err)
			}

			// Добавляем в начало очереди сообщений то сообщение которое не получилось доставить
			b.MakeRequestDeferred(chatID, true, options)

			b.MakeRequestDeferredErr(chatID, "send_error")
		} else {
			logger.Error.Printf("Error %d sending message: %v", chatID, err)
			sleep = coolDownIntervalErr // Если была другая ошибка отправки то немного подождем
		}
	}

	// если ошибок не было при отправке то устанавливаем стандартный интервал
	if sleep == 0 {
		sleep = coolDownInterval
	}

	// обновляем данные - время следующей отправки
	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nextTimeKey := fmt.Sprintf(userNextSendTimeKeyPrefix, chatID)
	err = b.redisDB.Set(cont, nextTimeKey, time.Now().Unix()+sleep, 0).Err()
	if err != nil {
		logger.Error.Printf("Error update user %d: %v", chatID, err)
	}

	logger.Info.Println("SendMessage ", chatID, options)
}

// Получение информации по пользователям
func (b *Bot) getUsersFromRedis() (map[int64]int64, error) {
	var cursor uint64
	var keys []string
	var err error

	usersData := make(map[int64]int64)

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Scan for keys matching "user:*:name"
	for {
		keys, cursor, err = b.redisDB.Scan(cont, cursor, userNextSendTimeKeyPattern, 10).Result() // 10 is the COUNT argument
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {

			val, err := b.redisDB.Get(cont, key).Int64()
			if err != nil {
				logger.Error.Printf("Error getting value for key %s: %v", key, err)
				continue
			}

			parts := strings.Split(key, ":") // e.g., "user:123:next_send_time"
			if len(parts) == 3 && parts[0] == "user" && parts[2] == "next_send_time" {
				userID, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					logger.Error.Printf("Could not parse user ID from key '%s': %v\n", key, err)
					continue
				}
				usersData[userID] = val
			}
		}

		if cursor == 0 { // No more keys to scan
			break
		}
	}

	return usersData, nil
}
