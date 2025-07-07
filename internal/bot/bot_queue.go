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
	// Redis key для задач на отправку в sorted set
	userSendTaskKey = "telegram:ready_users"
	// Redis key для очереди
	userQueueKeyPrefix = "user:%d:queue"
	// Redis key для времени отправки ошибок
	userErrorKeyPrefix  = "user:%d:error"
	userErrorExpiration = 40 * time.Second // Redis key для времени отправки ошибок
)

// MakeRequestDeferred Постановка сообщения в очередь на отправку
func (b *Bot) MakeRequestDeferred(chatID int64, firstInOrder bool, param MessageOptions) error {

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Проверяем юзера по времени последней отправки
	var sleep int64
	nextTimeKey := fmt.Sprintf(userNextSendTimeKeyPrefix, chatID)
	val, err := b.redisDB.Get(cont, nextTimeKey).Result()
	if err == redis.Nil {

		sleep = time.Now().Unix()
		// Создание активного юзера если его не существует и установка немедленной отправки
		err := b.redisDB.Set(cont, nextTimeKey, sleep, 0).Err() // Текущее время
		if err != nil {
			logger.Error.Printf("Error update user %d: %v", chatID, err)
		}
	} else if err != nil {
		logger.Error.Println(err)
	} else {
		// Получаем время следующей отправки
		nextTime, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error.Printf("Could not parse '%s': %v\n", val, err)
		} else {
			sleep = nextTime
		}
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

	// Создание задачи на отправку
	b.createTaskToSend(chatID, sleep)

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

		readyUsers := b.getReadyUsers()

		for _, chatID := range readyUsers {
			logger.Error.Println(chatID)
			// Проверяем кол-во отправленных сообщений и
			// проверяем пользователя и
			// если пришло время то отправляем сообщение из очереди
			if countMess <= limitMessPerInterval {

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

	// Создание задачи на отправку
	b.createTaskToSend(chatID, sleep)

	logger.Info.Println("SendMessage ", chatID, options)
}

func (b *Bot) createTaskToSend(chatID, nextSendTime int64) {
	cont, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем есть ли сообщения
	queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
	hasMore, _ := b.redisDB.LLen(cont, queueKey).Result()

	if hasMore > 0 {
		// Если задачи нет создаем ее
		err := b.redisDB.ZAdd(cont, userSendTaskKey, redis.Z{
			Score:  float64(nextSendTime),
			Member: fmt.Sprint(chatID),
		}).Err()
		if err != nil {
			logger.Error.Printf("Error ZAdd %d: %v", chatID, err)
		}

	} else {
		// Сообщений нет - удаляем из задач
		b.redisDB.ZRem(cont, userSendTaskKey, chatID)
	}
}

// Получение информации по пользователям
func (b *Bot) getReadyUsers() []int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().Unix()
	members, err := b.redisDB.ZRangeByScore(ctx, userSendTaskKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%d", now),
		Count: int64(limitMessPerInterval),
	}).Result()
	if err != nil {
		logger.Error.Println(err)
	}

	userIDs := make([]int64, 0, len(members))
	for _, member := range members {
		if userID, err := strconv.ParseInt(member, 10, 64); err == nil {
			userIDs = append(userIDs, userID)
		} else {
			logger.Error.Printf("Could not parse '%s': %v\n", member, err)
		}
	}
	return userIDs
}
