package bot

import (
	"fmt"
	"roulette/internal/logger"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
)

type deferredUsers struct {
	channel           chan MessageOptions // буферный канал с сообщениями для пользователя
	importantMessages []MessageOptions    // массив с приоритетными сообщениями которые будут отправлены в первую очередь
	LastMessageTime   int64               // время последней отправки сообщения (Unix)
	SleepTime         int64               // время задержки отправки сообщения (Unix)
}

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
)

// MakeRequestDeferred Постановка сообщения в очередь на отправку
func (b *Bot) MakeRequestDeferred(chatID int64, param MessageOptions) error {

	b.deferredMU.Lock()
	defer b.deferredMU.Unlock()

	// Проверка и если нужно то добавление пользователя в очередь
	if _, exists := b.deferredMessages[chatID]; !exists {
		b.deferredMessages[chatID] = deferredUsers{
			channel:           make(chan MessageOptions, 100), // Буферный канал с возможностью принять до 100 сообщений
			importantMessages: []MessageOptions{},
			LastMessageTime:   time.Now().Unix(), // Текущее время
			SleepTime:         0,                 // Первое сообщение - немедленная отправка
		}
	}

	if len(b.deferredMessages[chatID].channel) < 100 {
		// Добавляем сообщение пользователю в очередь на отправку
		b.deferredMessages[chatID].channel <- param
		return nil
	}

	// Превышение очереди на отправку
	data, ok := b.deferredMessages[chatID]
	if !ok {
		logger.Error.Printf("%d No user to read\n", chatID)
	} else if len(data.importantMessages) == 0 {

		// Получаем пользователя для определения языка
		user, userErr := b.service.GetUser(chatID)
		if userErr != nil {
			logger.Error.Printf("Error getting user %d: %v", chatID, userErr)
		}

		language := user.LanguageCode
		if language == "" {
			language = "en"
		}

		// Ставим в приоритетные уведомление об ошибке
		errSendText := b.service.GetText("send_queue_error", language)
		params := MessageOptions{
			Text:      fmt.Sprintf(errSendText, 100),
			ParseMode: telego.ModeHTML,
		}

		data.importantMessages = append(data.importantMessages, params)
		b.deferredMessages[chatID] = data
	}

	logger.Error.Printf("user %d More than 100 messages are waiting", chatID)
	return fmt.Errorf("user %d More than 100 messages are waiting", chatID)
}

// sendBotQueue цикл для равномерной отправки сообщений пользователям
func (b *Bot) sendBotQueue() {
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	for range ticker.C {
		im := 0
		all := 0

		for chatID, data := range b.deferredMessages {
			// Проверяем кол-во отправленных сообщений и
			// проверяем пользователя и
			// если пришло время то отправляем сообщение из канала
			if all <= limitMessPerInterval &&
				data.LastMessageTime+data.SleepTime <= time.Now().Unix() {

				// В первую очередь обрабатываем приоритетные
				if len(data.importantMessages) > 0 {

					options := MessageOptions{}

					b.deferredMU.Lock()
					d, ok := b.deferredMessages[chatID]
					if !ok || len(d.importantMessages) == 0 {
						logger.Error.Printf("%d No user to read\n", chatID)
					} else {
						options = d.importantMessages[0]
						d.importantMessages = d.importantMessages[1:]
						b.deferredMessages[chatID] = d
					}
					b.deferredMU.Unlock()
					im++
					all++
					b.sendDeferredMessage(chatID, options)

				} else {

					select {
					case options := <-data.channel:
						all++
						b.sendDeferredMessage(chatID, options)
					default:
						// Нет сообщений
					}
				}
			}
		}
		logger.Info.Println("End im: ", im, ", all: ", all)
	}
}

// Раз в 5 минут проверяем очередь сообщений для каждого юзера
func (b *Bot) checkDeferredMessages() {

	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for range ticker.C {
		for chatID, data := range b.deferredMessages {
			l := len(data.channel) + len(data.importantMessages)
			if l > 50 {

				// Получаем пользователя для определения языка
				user, userErr := b.service.GetUser(chatID)
				if userErr != nil {
					logger.Error.Printf("Error getting user %d: %v", chatID, userErr)
					continue
				}
				language := user.LanguageCode
				if language == "" {
					language = "en"
				}

				if len(data.importantMessages) == 0 {
					b.deferredMU.Lock()
					d, ok := b.deferredMessages[chatID]
					if !ok {
						logger.Error.Printf("%d No user to read\n", chatID)
					} else {

						// Ставим в приоритетные уведомление об ошибке
						errSendText := b.service.GetText("send_queue_error", language)

						params := MessageOptions{
							Text:      fmt.Sprintf(errSendText, l),
							ParseMode: telego.ModeHTML,
						}

						d.importantMessages = append(d.importantMessages, params)
						b.deferredMessages[chatID] = d
					}
					b.deferredMU.Unlock()
				}
				logger.Error.Printf("User %d More than 50 messages are waiting", chatID)
			}
		}
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

			// Получаем пользователя для определения языка
			user, userErr := b.service.GetUser(chatID)
			if userErr != nil {
				logger.Error.Printf("Error getting user %d: %v", chatID, userErr)
				return
			}
			language := user.LanguageCode
			if language == "" {
				language = "en"
			}

			b.deferredMU.Lock()
			d, ok := b.deferredMessages[chatID]
			if !ok {
				logger.Error.Printf("%d No user to read\n", chatID)
			} else {

				// Отправляем уведомление об ошибке при первой возможности
				errSendText := b.service.GetText("send_error", language)

				// Добавляем в начало списка приоритетных сообщений уведомление
				d.importantMessages = append([]MessageOptions{
					{
						Text:      errSendText,
						ParseMode: telego.ModeHTML,
					}},
					d.importantMessages...)

				// Добавляем в конец списка приоритетных сообщений то сообщение которое не получилось доставить
				d.importantMessages = append(d.importantMessages, options)
				b.deferredMessages[chatID] = d
			}
			b.deferredMU.Unlock()

		} else {
			logger.Error.Printf("Error %d sending message: %v", chatID, err)
			sleep = coolDownIntervalErr // Если была другая ошибка отправки то немного подождем
		}
	}

	// если ошибок не было при отправке то устанавливаем стандартный интервал
	if sleep == 0 {
		sleep = coolDownInterval
	}

	// обновляем данные
	b.deferredMU.Lock()
	defer b.deferredMU.Unlock()

	d, ok := b.deferredMessages[chatID]
	if !ok {
		logger.Error.Printf("%d No user to read\n", chatID)
		return
	}

	d.SleepTime = sleep
	d.LastMessageTime = time.Now().Unix()
	b.deferredMessages[chatID] = d

	logger.Info.Println("SendMessage ", chatID, options)
}
