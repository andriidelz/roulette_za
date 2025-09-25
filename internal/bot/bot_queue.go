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
// TODO Возможно удаление сообщений тоже стоит сюда перенести
// точные лимиты на них не понятны
// **********

const (
	// Интервалы отправки сообщений
	sendInterval                 = time.Second // Интервал срабатывания отправки
	limitMessPerInterval         = 10000       // Максимум сообщений которые могут быть отправлены за интервал
	coolDownInterval       int64 = 1           // Время задержки отправки
	coolDownIntervalErr    int64 = 5           // Время задержки отправки при ошибке
	coolDownIntervalErr429 int64 = 60          // Время задержки отправки при ошибке 429 - Too Many Requests

	// Ключи для Redis

	// Redis key для получения информации по пользователю - время следующей отправки(Unix timestamp)
	userNextSendTimeKeyPrefix = "user:%d:next_send_time"
	// Redis key для задач на отправку в sorted set
	userSendTaskKey = "telegram:ready_users"
	// Redis key для sort set очереди
	userQueueKeyPrefix = "user:%d:set_queue"
	// Время жизни сообщения по дефолту
	// (если не будет доставлено до указанного времени то удаляем сообщение)
	userQueueExpiration = 15 * time.Minute
	// Redis key для времени отправки ошибок
	userErrorKeyPrefix  = "user:%d:error"
	userErrorExpiration = 40 * time.Second // Redis key для времени отправки ошибок

)

// MakeRequestDeferred Постановка сообщения в очередь на отправку
func (b *Bot) MakeRequestDeferred(chatID, order int64, param MessageOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	param.CreatedAt = time.Now().UnixNano()
	ttl := param.TTL
	if ttl == 0 {
		ttl = userQueueExpiration
	}

	// Маршалим сообщение заранее
	message, err := json.Marshal(param)
	if err != nil {
		logger.Error.Println("Error Marshal:", err, chatID)
		return err
	}

	// Используем Redis Pipeline для выполнения нескольких команд за один сетевой запрос
	pipe := b.redisDB.Pipeline()

	// Подготавливаем команды
	nextTimeKey := fmt.Sprintf(userNextSendTimeKeyPrefix, chatID)
	getCmd := pipe.Get(ctx, nextTimeKey)

	queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
	score := float64(time.Now().Add(ttl).UnixNano()) // TTL
	if order > 0 && order < 10 {
		score = float64(order) //  если указана очередность то идет как приоритетное сообщение. Оно не удаляется и будет отправлено в первую очередь
	}

	// Записываем в виде Sorted Sets для контроля времени жизни сообщения
	err = pipe.ZAdd(ctx, queueKey, redis.Z{Score: score, Member: string(message)}).Err()
	if err != nil {
		logger.Error.Println("Error adding element", queueKey, message, err)
	}

	// Выполняем команды
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		logger.Error.Printf("Pipeline execution error: %v", err)
		return err
	}

	// Обрабатываем результаты
	var sleep int64
	val, err := getCmd.Result()
	if err == redis.Nil {
		sleep = time.Now().Unix()
		// Создание активного юзера если его не существует и установка немедленной отправки
		err := b.redisDB.Set(ctx, nextTimeKey, sleep, 0).Err() // Текущее время
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

	// Получаем длину очереди
	length, err := b.redisDB.ZCard(ctx, queueKey).Result()
	if err != nil {
		logger.Error.Printf("Error getting sorted set cardinality for user %d: %v", chatID, err)
		return err
	}

	// Создание задачи на отправку
	b.createTaskToSend(chatID, sleep)

	if length > 100 {
		b.MakeRequestDeferredErr(chatID, "send_queue_error")

		logger.Error.Printf("user %d More than 100 messages are waiting", chatID)
		return fmt.Errorf("user %d More than 100 messages are waiting", chatID)
	}

	// Обновляем метрику размера очереди
	if metrics := b.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.UpdateQueueSize(float64(length))
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

		language := getLanguage(user.LanguageCode, "")

		errSendText := b.getText(errText, language)
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
		b.MakeRequestDeferred(chatID, 1, options)
	}
}

// sendBotQueue цикл для равномерной отправки сообщений пользователям
func (b *Bot) sendBotQueue() {
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	for range ticker.C {
		countMess := 0
		maxMessages := limitMessPerInterval

		readyUsers := b.getReadyUsers()
		if len(readyUsers) == 0 {
			continue
		}

		logger.Info.Printf("Found %d ready users", len(readyUsers))

		// TODO: мб тут тоже увеличить workerCount?
		// Создаем worker pool для параллельной обработки
		const workerCount = 5 // Оптимальное количество воркеров
		jobs := make(chan int64, len(readyUsers))
		results := make(chan bool, len(readyUsers))

		// Запускаем воркеры
		for w := 1; w <= workerCount; w++ {
			go func(workerId int) {
				for chatID := range jobs {

					// Очищаем старые сообщения которые не получилось отправить
					now := time.Now().UnixNano()
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					// Минимальное 10, с score 0 идут приоритетные сообщения. Они не удаляется и будут отправлены в первую очередь
					removedCount, err := b.redisDB.ZRemRangeByScore(ctx, userQueueKeyPrefix, "10", fmt.Sprintf("%d", now)).Result()
					if err != nil {
						logger.Error.Println("Error removing expired elements:", err)
					} else if removedCount > 0 {
						logger.Error.Printf("Removed %d expired elements from sorted set.\n", removedCount)
					}

					ctx = context.Background()

					// Пробуем получить сообщение для пользователя из очереди
					queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
					msgs, err := b.redisDB.ZRangeByScore(ctx, queueKey, &redis.ZRangeBy{
						Min:   "0",
						Max:   "+inf",
						Count: 1, // Получаем одно сообщение
					}).Result()
					if err != nil {
						logger.Error.Println("Error ZRangeByScore:", err)
						results <- false
						continue
					} else if len(msgs) == 0 {
						logger.Error.Println("There is no messages", queueKey)
						results <- false
						continue
					}

					// Удаляем сообщение из очереди
					err = b.redisDB.ZRem(ctx, queueKey, msgs[0]).Err()
					if err != nil {
						logger.Error.Printf("Error ZRem %s: %v", queueKey, err)
					}

					// Если сообщение есть превращаем его в структуру
					options := MessageOptions{}
					if err = json.Unmarshal([]byte(msgs[0]), &options); err != nil {
						logger.Error.Printf("Worker %d: Error Unmarshal for %s: %v\n", workerId, queueKey, err)
						results <- false
						continue
					}

					// Отправляем сообщение
					b.sendDeferredMessage(chatID, options)
					results <- true
				}
			}(w)
		}

		// Отправляем задания
		for _, chatID := range readyUsers {
			if countMess >= maxMessages {
				break
			}
			jobs <- chatID
			countMess++
		}
		close(jobs)

		// Собираем результаты
		successCount := 0
		for i := 0; i < countMess; i++ {
			if <-results {
				successCount++
			}
		}

		logger.Info.Printf("Processed messages: %d/%d", successCount, countMess)
	}
}

func (b *Bot) sendDeferredMessage(chatID int64, options MessageOptions) {

	var err error
	var messageType string
	mes := &telego.Message{}
	switch options.MethodName {
	case editMessageText:
		messageType = "text"
		_, err = b.updateText(chatID, options.MessageID, options)
	case sendVideo:
		messageType = "video"
		_, err = b.sendVideo(chatID, options)
	case sendPhoto:
		messageType = "photo"
		_, err = b.sendPhoto(chatID, options)
	case sendCaptcha:
		messageType = "photo"

		// отримуємо ID останньої капчі
		cont, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		captchaMess := fmt.Sprintf(userCaptchaMessPrefix, chatID)
		messageID, err := b.redisDB.Get(cont, captchaMess).Int()
		if err != nil {
			if err == redis.Nil {
				logger.Error.Printf("Error Set %d: %v", chatID, err)
			}
			mes, err = b.sendPhoto(chatID, options)
		} else {
			mes, err = b.updatePhoto(chatID, messageID, options)
		}
		messageID = mes.MessageID

		// для капчі зберігаємо id останньої відправленої капчі щоб замінити її
		err = b.redisDB.Set(cont, captchaMess, messageID, 0).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", chatID, err)
		}

	case editMessageMedia:
		messageType = "photo"
		_, err = b.updatePhoto(chatID, options.MessageID, options)
	case sendSticker:
		messageType = "sticker"
		err = b.SendSticker(chatID, options.Text)
	case sendAnimation:
		messageType = "animation"
		_, err = b.sendAnimation(chatID, options)

	default:
		messageType = "text"
		_, err = b.sendText(chatID, options)
	}

	// Записываем метрики после отправки
	if err == nil {
		// Успешная отправка
		// Рассчитываем время в очереди
		queueTime := time.Since(time.Unix(0, options.CreatedAt)).Seconds()
		priority := "normal"
		if options.TTL > 0 && options.TTL < userQueueExpiration/2 {
			priority = "high"
		}

		// Записываем метрики
		if metrics := b.getMetrics(); metrics != nil && metrics.Bot != nil {
			metrics.Bot.RecordMessageSent(messageType)
			metrics.Bot.RecordQueueTime(priority, queueTime)
		}
	} else {
		// Ошибка отправки
		errorType := "other"
		if strings.Contains(err.Error(), "retry after") {
			errorType = "429"
		} else if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		}

		if metrics := b.getMetrics(); metrics != nil && metrics.Bot != nil {
			metrics.Bot.RecordTelegramError(errorType)
		}
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

			// Немедленно отправляем стикер об ошибке
			// На стикеры свои лимиты и он дойдет даже если отправка сообщений забанена
			b.SendSticker(chatID, StickerError)

			// Добавляем в начало очереди сообщений то сообщение которое не получилось доставить
			b.MakeRequestDeferred(chatID, 2, options)

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

	// Обновляем данные в Redis без глобальной блокировки
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Используем Redis Pipeline для атомарного выполнения обновлений
	pipe := b.redisDB.Pipeline()

	// Подготавливаем операции в pipeline
	nextTimeKey := fmt.Sprintf(userNextSendTimeKeyPrefix, chatID)
	pipe.Set(ctx, nextTimeKey, time.Now().Unix()+sleep, 0)

	// Проверяем длину очереди
	queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
	llenCmd := pipe.ZCard(ctx, queueKey)

	// Выполняем все операции за один запрос
	_, pipeErr := pipe.Exec(ctx)
	if pipeErr != nil {
		logger.Error.Printf("Error in pipeline execution: %v", pipeErr)
	}

	// Подготавливаем запрос на получение длины очереди
	hasMore, llenErr := llenCmd.Result()
	if llenErr != nil {
		logger.Error.Printf("Error getting sorted set cardinality for user %d: %v", chatID, err)
	} else {
		// На основе результата делаем соответствующее действие
		if hasMore > 0 {
			// Если есть еще сообщения, планируем следующую отправку
			err := b.redisDB.ZAddNX(ctx, userSendTaskKey, redis.Z{
				Score:  float64(time.Now().Unix() + sleep),
				Member: fmt.Sprint(chatID),
			}).Err()
			if err != nil {
				logger.Error.Printf("Error ZAdd for user %d: %v", chatID, err)
			}
		} else {
			// Если сообщений больше нет, удаляем из очереди задач
			err := b.redisDB.ZRem(ctx, userSendTaskKey, chatID).Err()
			if err != nil {
				logger.Error.Printf("Error ZRem for user %d: %v", chatID, err)
			}
		}
	}

	logger.Info.Println("SendMessage ", chatID, options)
}

func (b *Bot) createTaskToSend(chatID, nextSendTime int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Подготавливаем запрос на получение длины очереди
	queueKey := fmt.Sprintf(userQueueKeyPrefix, chatID)
	hasMore, err := b.redisDB.ZCard(ctx, queueKey).Result()
	if err != nil {
		logger.Error.Printf("Error getting sorted set cardinality for user %d: %v", chatID, err)
		return
	}

	// Выполняем следующую операцию на основе результата
	if hasMore > 0 {
		// Используем ZADD с опцией NX (добавлять только если элемента нет)
		// для оптимизации при высоких нагрузках
		err := b.redisDB.ZAddNX(ctx, userSendTaskKey, redis.Z{
			Score:  float64(nextSendTime),
			Member: fmt.Sprint(chatID),
		}).Err()
		if err != nil {
			logger.Error.Printf("Error ZAdd %d: %v", chatID, err)
		}
	} else {
		// Сообщений нет - удаляем из задач
		err := b.redisDB.ZRem(ctx, userSendTaskKey, chatID).Err()
		if err != nil {
			logger.Error.Printf("Error ZRem %d: %v", chatID, err)
		}
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
