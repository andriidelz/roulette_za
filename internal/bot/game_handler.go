package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"roulette/internal/logger"
	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
)

const webPage = "https://games.sprut.net"

// Типы сообщений для внутренней обработки
type MessageType int

const (
	RoundCompletedMessage MessageType = iota // Сообщение о завершении раунда
	RoundStartedMessage                      // Сообщение о начале нового раунда
)

// RoundData представляет структурированные данные о раунде
type RoundData struct {
	Number       int64                  `json:"number,omitempty"`        // Номер, выпавший на рулетке
	SaltHex      string                 `json:"salt_hex,omitempty"`      // Соль в HEX формате
	Hash         string                 `json:"hash,omitempty"`          // Хеш для проверки
	Result       string                 `json:"result,omitempty"`        // Результат (red, black, zero)
	CreatedAt    time.Time              `json:"created_at,omitempty"`    // Время создания раунда
	CompletedAt  time.Time              `json:"completed_at,omitempty"`  // Время завершения раунда
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // Дополнительные поля (если нужны)
}

// RoundMessage представляет сообщение о раунде для синхронной обработки
type RoundMessage struct {
	Type    MessageType       // Тип сообщения (завершение/начало раунда)
	RoundID uint              // ID раунда
	Round   *models.HashEntry // Данные о раунде из БД
	Data    *RoundData        // Структурированные данные о раунде
}

// GameHandler управляет игровым процессом рулетки
type GameHandler struct {
	bot             *Bot
	service         service.Service
	currentRound    *models.HashEntry
	mutex           sync.RWMutex
	activePlayers   map[int64]int       // Карта активных игроков в режиме /play
	rabbitmq        *messaging.RabbitMQ // Клиент RabbitMQ
	processedRounds map[uint]bool       // Хранит ID обработанных раундов для избежания дублирования
	processMutex    sync.Mutex          // Мьютекс для доступа к processedRounds

	prizeFundAmount float64 // значення з GetPrizeFund
	// topCount        int        // значення з GetPrizeFund
	totalPoints    int        // значення з GetWeeklyRating
	prizeFundMutex sync.Mutex // Мьютекс для доступа к призового фонду

	roundMsgChan   chan RoundMessage
	processingLock sync.Mutex
	stopWorker     chan struct{}
}

const (
	CallbackPlay           = "startplay"
	CallbackStartRound     = "startround"
	CallbackGetResultRound = "get_result_"
	CallbackBetRed         = "bet_red_"
	CallbackBetBlack       = "bet_black_"
	CallbackBetZero        = "bet_zero_"
	CallbackBetZeroLocked  = "locked_bet_zero"
	CallbackBetAvailable   = "availablebets"

	userWaitBetResultPrefix = "game:waiting_bet_result" // Карта игроков, ожидающих результатов
	userWaitNewRoundPrefix  = "game:waiting_new_round"  // Карта игроков, ожидающих результатов
	gameAnimationPrefix     = "game:animation"          // Карта анимаций

	StickerNoBids    = "CAACAgUAAxkBAAEORLpn9lEBwqSME7WwehtZBLt5ybqSrAACKRUAAvWxqVeH8hhzfq9SEjYE" // nomorebids
	StickerWin       = "CAACAgUAAxkBAAEORLxn9lEJolSTKIZrUxOLZbkMChpdWwACuBcAArzBqVdjiSsft06GCjYE" // win
	StickerLose      = "CAACAgUAAxkBAAEORL5n9lEOq_kczbL1CGpgN5-UhhhgqQAC3BIAAtGwqVdlepoFId2tMzYE" // lose
	StickerBlackRes1 = "CAACAgUAAxkBAAEORMBn9lEUB8KMRJ8nduCQ-y32y5ns4AACNBUAArIsqVfUvoMXgG8VvzYE" // blackresult (вариант 1)
	StickerBlackRes2 = "CAACAgUAAxkBAAEORMJn9lEXC6ByJRCY4_8Mu5vQQP-1zgACOxYAAnWOqVesBnNzFycGfDYE" // blackresult (вариант 2)
	StickerRedRes1   = "CAACAgUAAxkBAAEORMhn9lEfopgbb8y7qi__V8deZr0MpAACYBcAAs4bqVeFX-l3HDBIFjYE" // redresult (вариант 1)
	StickerRedRes2   = "CAACAgUAAxkBAAEORMpn9lEiRobEQnz4qg6GFSmfZQmjbwACiRgAAhuTqVdgysjb-Y-sLTYE" // redresult (вариант 2)
	StickerZeroRes1  = "CAACAgUAAxkBAAEORMRn9lEar58eDwvent8Lp3TvMRvF5AACtxEAAlRRsFdySRXPzXyVqzYE" // zeroresult (вариант 1)
	StickerZeroRes2  = "CAACAgUAAxkBAAEORMZn9lEd12gNsWFFxGXLAZoeJbSEsgACCxYAAmDwqVdsE7WC-rayWDYE" // zeroresult (вариант 2)

)

// NewGameHandler создает новый обработчик игры
func NewGameHandler(bot *Bot, service service.Service, rabbitmqURL string) (*GameHandler, error) {
	// Создаем клиент RabbitMQ
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, "roulette_events", "bot")
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	handler := &GameHandler{
		bot:             bot,
		service:         service,
		activePlayers:   make(map[int64]int),
		rabbitmq:        rmq,
		processedRounds: make(map[uint]bool),
		roundMsgChan:    make(chan RoundMessage, 500), // Буфер для сообщений
		stopWorker:      make(chan struct{}),

		prizeFundAmount: 1000.0, // По умолчанию
		// topCount:        100,    // По умолчанию
		totalPoints: 0, // По умолчанию
	}

	// Запускаем обработчик сообщений в отдельной горутине
	go handler.processMessagesWorker()

	// Подписываемся на события от RabbitMQ
	if err := handler.subscribeToRoundEvents(); err != nil {
		handler.Stop() // Останавливаем запущенные горутины и ресурсы
		return nil, fmt.Errorf("failed to subscribe to round events: %w", err)
	}

	return handler, nil
}

func (h *GameHandler) processMessagesWorker() {
	logger.Info.Println("Starting message processing worker")

	for {
		select {
		case <-h.stopWorker:
			logger.Info.Println("Message processing worker stopped")
			return
		case msg := <-h.roundMsgChan:
			// Блокируем на время обработки сообщения
			h.processingLock.Lock()

			switch msg.Type {
			case RoundCompletedMessage:
				logger.Info.Printf("Processing RoundCompletedMessage for round #%d", msg.RoundID)
				// Обрабатываем завершение раунда
				if msg.Round != nil {
					// Вызываем обработчик завершения раунда
					h.handleRoundCompletion()

					// Если доступны дополнительные данные, логируем их
					if msg.Data != nil {
						logger.Info.Printf("Round #%d completed with result: %s", msg.RoundID, msg.Data.Result)

						// Здесь можно использовать данные для дополнительной обработки,
						// например, для верификации результата или отображения доп. информации
					}

					// Помечаем раунд как обработанный
					h.processMutex.Lock()
					h.processedRounds[msg.RoundID] = true
					h.processMutex.Unlock()
				}
			case RoundStartedMessage:
				logger.Info.Printf("Processing RoundStartedMessage for round #%d", msg.RoundID)
				// Обрабатываем новый раунд только после обработки всех сообщений о завершении раунда
				if msg.Round != nil {
					// Обновляем текущий раунд
					h.mutex.Lock()
					h.currentRound = msg.Round
					h.mutex.Unlock()

					// Уведомляем активных игроков
					h.checkActivePlayers()

					// Если есть дополнительные данные о раунде, можно использовать их
					if msg.Data != nil && msg.Data.Hash != "" {
						logger.Info.Printf("Round #%d started with hash: %s", msg.RoundID, msg.Data.Hash)
					}

					// Помечаем раунд как обработанный
					h.processMutex.Lock()
					h.processedRounds[msg.RoundID] = true
					h.processMutex.Unlock()
				}
			}

			h.processingLock.Unlock()
		}
	}
}

// checkActivePlayers проверяет активных игроков
func (h *GameHandler) checkActivePlayers() {
	stopPlayers := []int64{}
	h.mutex.RLock()
	for userID, val := range h.activePlayers {
		// Перед отправлением раунда проверяем активный ли юзер
		// Если юзер пропустил 10 раундов останавливаем для него игру
		if val >= 10 {
			stopPlayers = append(stopPlayers, userID)
		} else {
			// Инкрементим ему кол-во раундов. В случае если он сделает ставку это число обнулится
			h.activePlayers[userID] = val + 1
		}
	}
	h.mutex.RUnlock()

	// Останавливаем всех неактивных игроков
	for i := range stopPlayers {
		logger.Error.Println("Stop user ", stopPlayers[i])
		h.stopGame(stopPlayers[i])
	}

	// Відправляємо повідомлення всім хто очікує нового раунду
	// Відправляємо їм результат по першій можливості
	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Список гравців що очікують на отримання
	players, err := h.bot.redisDB.HGetAll(cont, userWaitNewRoundPrefix).Result()
	if err != nil {
		logger.Error.Println("Failed to HGetAll", err)
	}
	if len(players) == 0 {
		return
	}

	// Видаляєм список гравців
	_, err = h.bot.redisDB.Del(cont, userWaitNewRoundPrefix).Result()
	if err != nil {
		logger.Error.Printf("Error Del: %v", err)
	}

	for userText := range players {
		userID, err := strconv.ParseInt(userText, 10, 64)
		if err != nil {
			logger.Error.Printf("failed to parse #%s: %v", userText, err)
			continue
		}

		dbUser, err := h.bot.getUser(userID)
		if err != nil {
			logger.Error.Printf("Error get user: %v", err)
			continue
		}

		language := dbUser.LanguageCode
		if language == "" {
			language = "en"
		}

		err = h.sendNewRound(language, userID, dbUser.ID)
		if err != nil {
			logger.Error.Printf("Error sendNewRound: %v", err)
			continue
		}
	}

	logger.Info.Printf("All players have been notified about results")
}

// subscribeToRoundEvents подписывается на события завершения и начала раундов
func (h *GameHandler) subscribeToRoundEvents() error {
	// Создаем уникальное имя очереди для бота
	queueName := fmt.Sprintf("roulette_bot_queue_%d", time.Now().UnixNano())

	// Подписываемся на события завершения и начала раундов
	return h.rabbitmq.SubscribeToQueue(queueName,
		[]string{messaging.RoutingRoundCompleted, messaging.RoutingRoundStarted},
		h.handleRabbitMQMessage)
}

// handleRabbitMQMessage обрабатывает сообщения от RabbitMQ
func (h *GameHandler) handleRabbitMQMessage(message messaging.RouletteMessage) error {
	roundID := message.RoundID

	// Не пропускаем EventRoundCompleted даже если уже обработан
	if message.Type != messaging.EventRoundCompleted {
		// Проверяем, не обрабатывали ли мы уже этот раунд (только для не-завершенных раундов)
		h.processMutex.Lock()
		if processed, exists := h.processedRounds[roundID]; exists && processed {
			h.processMutex.Unlock()
			logger.Info.Printf("Round #%d already processed, skipping duplicate message", roundID)
			return nil
		}
		h.processMutex.Unlock()
	}

	// Преобразуем данные из общего интерфейса в нашу типизированную структуру
	var roundData *RoundData
	if message.Data != nil {
		// Пытаемся конвертировать данные в map
		if dataMap, ok := message.Data.(map[string]interface{}); ok {
			roundData = &RoundData{
				CustomFields: make(map[string]interface{}),
			}

			// Обрабатываем известные поля
			if val, exists := dataMap["number"]; exists {
				if num, ok := val.(int64); ok {
					roundData.Number = num
				} else if num, ok := val.(float64); ok {
					roundData.Number = int64(num)
				}
			}

			if val, exists := dataMap["salt_hex"]; exists {
				if str, ok := val.(string); ok {
					roundData.SaltHex = str
				}
			}

			if val, exists := dataMap["hash"]; exists {
				if str, ok := val.(string); ok {
					roundData.Hash = str
				}
			}

			if val, exists := dataMap["result"]; exists {
				if str, ok := val.(string); ok {
					roundData.Result = str
				}
			}

			// Копируем все остальные поля в CustomFields
			for k, v := range dataMap {
				switch k {
				case "number", "salt_hex", "hash", "result", "created_at", "completed_at":
					// Эти поля уже обработаны выше
					continue
				default:
					roundData.CustomFields[k] = v
				}
			}
		}
	}

	switch message.Type {
	case messaging.EventRoundCompleted:
		logger.Info.Printf("Processing round completed message for round #%d", roundID)

		// Важно! Сначала обрабатываем результаты и уведомляем игроков напрямую
		// вместо помещения в канал сообщений
		h.handleRoundCompletion()

		// Помечаем раунд как обработанный
		h.processMutex.Lock()
		h.processedRounds[roundID] = true
		h.processMutex.Unlock()

	case messaging.EventRoundStarted:
		// Получаем данные о новом раунде
		newRound, err := h.service.GetHashEntryByID(roundID)
		if err != nil {
			return fmt.Errorf("failed to get new round #%d: %w", roundID, err)
		}

		// Отправляем сообщение в канал для синхронной обработки
		logger.Info.Printf("Sending round started message to processing queue for round #%d", roundID)
		h.roundMsgChan <- RoundMessage{
			Type:    RoundStartedMessage,
			RoundID: roundID,
			Round:   newRound,
			Data:    roundData,
		}
	}

	return nil
}

// handleRoundCompletion обрабатывает завершение раунда
func (h *GameHandler) handleRoundCompletion() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Получаем список игроков, ожидающих результат
	players, err := h.bot.redisDB.HGetAll(ctx, userWaitBetResultPrefix).Result()
	if err != nil {
		logger.Error.Println("Failed to HGetAll", err)
		return
	}

	if len(players) == 0 {
		return
	}

	// Удаляем список игроков сразу
	_, err = h.bot.redisDB.Del(ctx, userWaitBetResultPrefix).Result()
	if err != nil {
		logger.Error.Printf("Error Del: %v", err)
	}

	// Параллельная обработка с ограничением
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Максимум 10 параллельных уведомлений

	for userText, roundText := range players {
		wg.Add(1)
		semaphore <- struct{}{} // Захватываем слот

		go func(ut, rt string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Освобождаем слот

			userID, err := strconv.ParseInt(ut, 10, 64)
			if err != nil {
				logger.Error.Printf("failed to parse user ID '%s': %v", ut, err)
				return
			}

			roundID, err := strconv.ParseUint(rt, 10, 64)
			if err != nil {
				logger.Error.Printf("failed to parse round ID '%s': %v", rt, err)
				return
			}

			round, err := h.service.GetHashEntryByID(uint(roundID))
			if err != nil {
				logger.Error.Printf("failed to get round #%d: %v", roundID, err)
				return
			}

			// Получаем результат раунда
			option, err := h.service.GetRoundResult(round.Number)
			if err != nil {
				logger.Error.Printf("Error getting result for round #%d: %v", roundID, err)
				return
			}

			logger.Info.Printf("Round #%d result: %s (number: %d)", roundID, option, round.Number)

			// Уведомляем игрока о результате
			if err := h.notifyPlayerAboutResult(userID, round, option); err != nil {
				logger.Error.Printf("Error notifying player %d: %v", userID, err)
			} else {
				logger.Info.Printf("Successfully notified player %d about round #%d results", userID, round.ID)
			}
		}(userText, roundText)
	}

	wg.Wait()
	logger.Info.Printf("All players have been notified about results")
}

// handleGetResultRound присилаємо користувачу результат раунда в якому він робив ставку і не отримав результат
func (h *GameHandler) handleGetResultRound(query *telego.CallbackQuery) {
	user := query.From

	// Отримуєм раунд
	roundID, err := strconv.ParseUint(strings.TrimPrefix(query.Data, CallbackGetResultRound), 10, 64)
	if err != nil {
		logger.Error.Printf("failed to parse #%s: %v", query.Data, err)
		return
	}

	round, err := h.service.GetHashEntryByID(uint(roundID))
	if err != nil {
		logger.Error.Printf("failed to get round #%d: %v", roundID, err)
		return
	}

	// Получаем пользователя
	language, err := h.bot.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	//  Користувач натискає кнопку отримати результат раніше, ніж результати готові для виводу
	if !round.IsCompleted {

		cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// записуємо в список гравців що очікують на отримання
		err = h.bot.redisDB.HSet(cont, userWaitBetResultPrefix, fmt.Sprint(user.ID), round.ID).Err()
		if err != nil {
			logger.Error.Printf("Error Set %d: %v", user.ID, err)
		}

		// Виводимо pop-up toast без підтвердження
		h.bot.answerCallbackQuery(query.ID, h.bot.getText("wait_result", language), false)
		return
	}

	h.bot.answerCallbackQuery(query.ID, "", false)

	// Отримуєм результат раунда
	option, err := h.service.GetRoundResult(round.Number)
	if err != nil {
		logger.Error.Printf("Error getting result for round #%d: %v", round.ID, err)
		return
	}

	if err := h.notifyPlayerAboutResult(user.ID, round, option); err != nil {
		logger.Error.Printf("Error notifying player %d: %v", user.ID, err)
	}
}

// notifyPlayerAboutResult уведомляет игрока о результате раунда
func (h *GameHandler) notifyPlayerAboutResult(userID int64, round *models.HashEntry, result models.BetOption) error {
	logger.Info.Printf("notifyPlayerAboutResult called for user %d, round #%d", userID, round.ID)

	// Получаем пользователя
	dbUser, err := h.bot.getUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", userID, err)
		return fmt.Errorf("error getting user: %w", err)
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем информацию о ставке
	userBets, err := h.service.GetUserBetsForRound(dbUser.ID, round.ID)
	if err != nil {
		return fmt.Errorf("error getting bets: %w", err)
	}

	if len(userBets) == 0 {
		return fmt.Errorf("no bets found for user %d in round %d", userID, round.ID)
	}

	bet := userBets[0]

	//  Користувач вже отримував результат раунду - присилаєм текст і лінк на сторінку з історією ставок
	if bet.GetResult {
		if time.Since(bet.CreatedAt).Seconds() < 20 {
			// раунд ще не закінчився, ніяк не відповідаємо
			logger.Error.Printf("Not complited %d: ", userID)
			return nil
		}
		// Якщо з моменту створення раунду пройшло більше 20 сек

		options := h.bot.prepareMessage("repeatresultalert", language)
		// Создаем кнопки
		var inlineButtons [][]telego.InlineKeyboardButton

		// Добавляем первый ряд с двумя кнопками: проверка раунда и просмотр рейтинга
		checkSystemText := h.bot.getText("systemcheck", language)
		roundIDBase62 := utils.ToBase62(uint(round.ID))
		options.Text = "#" + roundIDBase62 + "\n\n" + options.Text
		checkSystemURL := fmt.Sprintf("%s/hashes/?id=%s", webPage, roundIDBase62)

		viewRatingText := h.bot.getText("viewrating", language)

		// Верхний ряд из 2 кнопок
		inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
			{Text: checkSystemText, URL: checkSystemURL},
			{Text: viewRatingText, CallbackData: "view_rating"},
		})
		inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
			{Text: h.bot.getText("next_round", language), CallbackData: CallbackStartRound},
		})

		// Создаем inline клавиатуру с кнопками
		options.InlineKeyboard = &telego.InlineKeyboardMarkup{
			InlineKeyboard: inlineButtons,
		}

		// Отправляем объединенное сообщение с клавиатурой
		h.bot.SendMessage(userID, options)
		return nil
	}

	// Сохраняем обновленную ставку в БД
	bet.GetResult = true
	if err := h.service.GetRepo().UpdateBet(&bet); err != nil {
		logger.Error.Printf("Error updating bet for user %d in round %d: %v",
			bet.UserID, round.ID, err)
	}

	won := bet.Won
	points := bet.Points

	// Записываем метрику результата ставки
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		if won {
			metrics.Bot.RecordBetResult("won")
		} else {
			metrics.Bot.RecordBetResult("lost")
		}
	}

	// Задержка для соблюдения правильного таймирования сообщений
	// Разница между завершением раунда и первым стикером (результат)
	// должна составлять примерно 2 секунды (17 секунда раунда)
	// Мы можем рассчитать это относительно времени RevealedAt
	if round.RevealedAt != nil {
		// Время отправки первого стикера - 2 секунды после завершения раунда
		firstMessageTime := round.RevealedAt.Add(2 * time.Second)
		sleepDuration := time.Until(firstMessageTime)
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}
	}

	// 1. Отправляем стикер с результатом (цвет) на 17 секунде

	// 2. Отправляем сообщение о результате на 18 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	// Определяем текст результата
	var resultLangKey string
	switch result {
	case models.Red:
		resultLangKey = "redresult"
	case models.Black:
		resultLangKey = "blackresult"
	case models.Zero:
		resultLangKey = "zeroresult"
	default:
		logger.Error.Println("Error getting prize fund", &round)
		logger.Error.Println("Error getting prize fund", *round)
		return nil
	}

	// 3. Отправляем стикер выигрыша/проигрыша на 19 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	// "black"+"_"+"lose" = black_lose
	var resultAnimation string = string(result) + "_"
	if won {
		resultAnimation += "win"
	} else {
		resultAnimation += "lose"
	}

	// logger.Error.Println(resultAnimation)

	// 4. Отправляем полное сообщение о выигрыше/проигрыше на 20 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	// Формируем сообщение о выигрыше/проигрыше
	var options MessageOptions
	if won {
		options = h.bot.prepareMessage("winmessage", language)
		options.Text = fmt.Sprintf(options.Text, points)
	} else {
		options = h.bot.prepareMessage("losemessage", language)
	}

	options.Text = h.bot.getText(resultLangKey, language) + "\n\n" + options.Text

	// Формируем часть о рейтинге

	// Получаем текущий рейтинг пользователя
	year, week := time.Now().ISOWeek()
	rating, err := h.service.GetRepo().GetUserWeeklyRating(dbUser.ID, year, week)
	if err != nil {
		logger.Error.Printf("Error get rating: %v", err)
	}

	// Получаем информацию о призовом фонде

	// Переменные для текста рейтинга
	var ratingText string
	var userShare float64 = 0.0

	h.prizeFundMutex.Lock()
	totalPoints := h.totalPoints
	prizeFundAmount := h.prizeFundAmount
	h.prizeFundMutex.Unlock()

	if rating.Points > totalPoints {
		totalPoints = rating.Points // totalPoints оновлюється з затримкою і є ймовірність що рейтинг користувача буде більше ніж загальна кількість балів
	}
	if rating.Points > 0 && totalPoints > 0 {
		// Расчет доли пользователя
		userShare = (float64(rating.Points) / float64(totalPoints)) * prizeFundAmount
	}

	// Формируем сообщение о рейтинге
	ratingTemplate := h.bot.getText("bidrating", language)
	ratingText = fmt.Sprintf(ratingTemplate, rating.Points, rating.Position, userShare, prizeFundAmount)

	// Часть проверки баланса ставок
	betsBalance, err := h.service.GetUserRemainingBets(dbUser.ID)
	if err != nil {
		logger.Error.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	var betsBalanceText string
	var additionalMessage string

	if betsBalance <= 0 {
		// Недостаточно ставок
		betsBalanceText = h.bot.getText("betsbalancelow", language)
		additionalMessage = h.bot.getText("nextbidlow", language)
	} else {
		// Достаточно ставок
		betsBalanceTemplate := h.bot.getText("betsbalanceok", language)
		betsBalanceText = fmt.Sprintf(betsBalanceTemplate, betsBalance)
	}

	// Объединяем части сообщения
	options.Text = options.Text + "\n\n" + ratingText + "\n\n" + betsBalanceText

	// Если есть дополнительное сообщение для недостаточного баланса, добавляем его
	if additionalMessage != "" {
		options.Text += "\n\n" + additionalMessage
	}

	// Создаем кнопки
	var inlineButtons [][]telego.InlineKeyboardButton

	// Добавляем первый ряд с двумя кнопками: проверка раунда и просмотр рейтинга
	checkSystemText := h.bot.getText("systemcheck", language)
	roundIDBase62 := utils.ToBase62(uint(round.ID))
	checkSystemURL := fmt.Sprintf("%s/hashes/?id=%s", webPage, roundIDBase62)

	options.Text = "#" + roundIDBase62 + "\n\n" + options.Text

	viewRatingText := h.bot.getText("viewrating", language)

	// Верхний ряд из 2 кнопок
	inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
		{Text: checkSystemText, URL: checkSystemURL},
		{Text: viewRatingText, CallbackData: "view_rating"},
	})
	inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
		{Text: h.bot.getText("next_round", language), CallbackData: CallbackStartRound},
	})

	// TODO: кнопка временно скрыта
	// Второй ряд только с кнопкой пополнения баланса
	// topUpBalanceText := h.bot.getText("topupbalance", language)
	// inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
	// 	{Text: topUpBalanceText, CallbackData: "noop"},
	// })

	// Если баланс ставок недостаточен, добавляем кнопку остановки игры в третий ряд
	if betsBalance <= 0 {
		h.stopGame(userID)
	}

	// Создаем inline клавиатуру с кнопками
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: inlineButtons,
	}

	options.MethodName = sendAnimation

	cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Пробуємо отримати id файла
	value, err := h.bot.redisDB.HGet(cont, gameAnimationPrefix, resultAnimation).Result()
	if err == redis.Nil {
		// Пробуємо відправити файл
		options.VideoPath = resultAnimation
	} else if err != nil {
		logger.Error.Println(err)
	} else {
		options.VideoFileID = value
	}

	h.bot.SendMessage(userID, options)

	// Проверяем активность пользователя - кол-во набранных баллов
	if won {
		switch h.bot.captchaBetPoints(userID, points) {
		case "needCaptcha":
			h.bot.SendMessage(userID, h.bot.captchaMessage(userID, language))
			return nil
		}
	}

	return nil
}

// MakeBet делает ставку в текущем раунде
func (h *GameHandler) MakeBet(userID int64, roundID uint64, option models.BetOption) error {
	logger.Info.Printf("MakeBet called for user %d with option %s", userID, option)

	// Получаем пользователя
	dbUser, err := h.bot.getUser(userID)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	// Проверяем статус бана
	if dbUser.Banned {
		return fmt.Errorf("user is banned")
	}

	// Получаем текущий раунд
	currentRound, err := h.service.GetCurrentRound()
	if err != nil {
		return fmt.Errorf("error getting current round: %w", err)
	}

	if currentRound == nil {
		return fmt.Errorf("no active round")
	}

	// Дополнительная проверка - подтверждаем, что раунд не завершен
	if currentRound.IsCompleted || currentRound.ID != uint(roundID) {
		return fmt.Errorf("round is already completed")
	}

	// Проверяем, не делал ли пользователь уже ставку в этом раунде
	existingBets, err := h.service.GetUserBetsForRound(dbUser.ID, currentRound.ID)
	if err != nil {
		logger.Error.Printf("Error checking existing bets: %v", err)
		return fmt.Errorf("error checking existing bets: %w", err)
	}

	if len(existingBets) > 0 {
		logger.Info.Printf("User %d already made a bet in round %d", userID, currentRound.ID)
		return fmt.Errorf("user has already made a bet in this round")
	}

	// Проверяем, может ли пользователь делать ставку на Zero
	if option == models.Zero {
		canBetZero, _, err := h.service.CanBetZero(dbUser.ID)
		if err != nil {
			logger.Error.Printf("Error checking zero bet: %v", err)
			return fmt.Errorf("error checking zero bet: %w", err)
		}

		if !canBetZero {
			return fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Проверяем доступное количество ставок
	betsRemaining, err := h.service.GetUserRemainingBets(dbUser.ID)
	if err != nil {
		logger.Error.Printf("Error checking remaining bets: %v", err)
		// Не возвращаем ошибку, так как это не критично
	} else if betsRemaining == 0 {
		return fmt.Errorf("no bets left for today")
	}

	// Делаем ставку через сервис
	if err := h.service.MakeBet(dbUser.ID, option); err != nil {
		logger.Error.Printf("Error making bet: %v", err)
		return fmt.Errorf("error making bet: %w", err)
	}

	// Записываем метрику ставки
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.RecordBet(string(option))
	}

	// Получаем язык пользователя
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Регистрируем пользователя как активного
	h.mutex.Lock()
	h.activePlayers[userID] = 1 // обнуляем кол-во пропущенных раундов
	h.mutex.Unlock()

	logger.Info.Printf("Bet created successfully for user %d in round %d (activePlayers count: %d)", userID, currentRound.ID, len(h.activePlayers))

	// Сразу отправляем стикер "Ставки больше не принимаются"
	h.bot.MakeRequestDeferred(userID, 0, MessageOptions{
		Text:       StickerNoBids,
		MethodName: sendSticker,
	})

	// После короткой паузы отправляем сообщение о принятии ставки
	go func() {
		time.Sleep(1000 * time.Millisecond)
		options := h.bot.prepareMessage("nomorebids", language)

		// Создаем кнопку для запроса результата
		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{{
				{
					Text:         h.bot.getText("get_result", language),
					CallbackData: CallbackGetResultRound + fmt.Sprint(currentRound.ID),
				},
			}},
		}
		options.InlineKeyboard = inlineKeyboard

		h.bot.SendMessage(userID, options)
	}()

	return nil
}

// handleAvailableBets присылаем игроку доступное количество ставок
func (h *GameHandler) handleAvailableBets(query *telego.CallbackQuery) {
	user := query.From
	dbUser, err := h.bot.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error get user: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Получаем доступное количество ставок
	betsBalance, err := h.service.GetUserRemainingBets(dbUser.ID)
	if err != nil {
		logger.Error.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	if betsBalance <= 0 {
		// Виводимо pop-up toast без підтвердження
		h.bot.answerCallbackQuery(query.ID, h.bot.getText("betsbalancelow", language), false)
		return
	}

	text := h.bot.getText("betsbalanceok", language)
	text = fmt.Sprintf(text, betsBalance)

	// Виводимо pop-up toast без підтвердження
	h.bot.answerCallbackQuery(query.ID, text, false)
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	language, err := h.bot.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Сначала отправляем сообщение с описанием игры
	options := h.bot.prepareMessage("playstart1", language)
	options.InlineKeyboard = h.createStartPlayKeyboard(language)

	// Отправляем первое сообщение с описанием игры и кнопкой на правила
	h.bot.SendMessage(message.Chat.ID, options)
}

// handlePlay обробляє колбек startplay
func (h *GameHandler) handlePlay(query *telego.CallbackQuery) {
	// Отвечаем на callback, чтобы убрать индикатор загрузки
	h.bot.answerCallbackQuery(query.ID, "", false)

	user := query.From

	language, err := h.bot.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Сначала отправляем сообщение с описанием игры
	options := h.bot.prepareMessage("playstart1", language)
	options.InlineKeyboard = h.createStartPlayKeyboard(language)

	// Обновляем сообщение
	if query.Message != nil {
		h.bot.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), options)
	} else {
		// Если сообщение недоступно, отправляем новое
		h.bot.SendMessage(user.ID, options)
	}
}

// handleStartRound - старт нового раунду гри
func (h *GameHandler) handleStartRound(query *telego.CallbackQuery) {
	user := query.From
	dbUser, err := h.bot.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)
	err = h.sendNewRound(language, user.ID, dbUser.ID)
	if err != nil {
		var errorKey string
		if strings.Contains(err.Error(), "already made a bet") {
			// Пользователь уже сделал ставку в этом раунде
			errorKey = "bet_already_made"
		} else if strings.Contains(err.Error(), "waiting_for_round") {
			errorKey = "waiting_for_round"

			cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// записуємо в список гравців що очікують на отримання
			err = h.bot.redisDB.HSet(cont, userWaitNewRoundPrefix, fmt.Sprint(user.ID), "").Err()
			if err != nil {
				logger.Error.Printf("Error Set %d: %v", user.ID, err)
			}
		}

		// Виводимо pop-up toast без підтвердження
		h.bot.answerCallbackQuery(query.ID, h.bot.getText(errorKey, language), false)
		return
	}

	h.bot.answerCallbackQuery(query.ID, "", false)
}

func (h *GameHandler) sendNewRound(language string, telegramID int64, userID uint) error {
	currentRound, err := h.service.GetCurrentRound()
	if err != nil {
		logger.Error.Printf("Error getting current round: %v", err)
		return fmt.Errorf("Error getting current round")
	}

	if currentRound == nil {
		return fmt.Errorf("waiting_for_round")
	}

	// БЫСТРАЯ проверка через Redis вместо БД (экономия ~50-200ms)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	betKey := fmt.Sprintf("bet:%d:%d", userID, currentRound.ID)
	exists, err := h.bot.redisDB.Exists(ctx, betKey).Result()
	if err == nil && exists > 0 {
		return fmt.Errorf("user has already made a bet in this round")
	}

	h.mutex.Lock()
	h.currentRound = currentRound
	h.mutex.Unlock()

	roundIDBase62 := utils.ToBase62(uint(currentRound.ID))
	elapsedTime := time.Since(currentRound.CreatedAt)
	roundDuration := 15 * time.Second
	remainingTime := roundDuration - elapsedTime

	if remainingTime < 0 {
		return fmt.Errorf("waiting_for_round")
	}

	h.mutex.Lock()
	h.activePlayers[telegramID] = 1
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.SetActivePlayers(float64(len(h.activePlayers)))
	}
	h.mutex.Unlock()

	remainingSeconds := int(remainingTime.Seconds())
	options := h.bot.prepareMessage("round_info_countdown", language)
	options.Text = fmt.Sprintf(options.Text, roundIDBase62, currentRound.Hash, remainingSeconds)
	options.InlineKeyboard = h.createBetKeyboard(language, userID, currentRound.ID)

	// ВЫСОКИЙ ПРИОРИТЕТ для игровых сообщений
	options.TTL = 5 * time.Second
	return h.bot.MakeRequestDeferred(telegramID, 3, options)
}

// stopGame видаляє зі списку активних гравців
func (h *GameHandler) stopGame(userID int64) {
	h.mutex.Lock()
	delete(h.activePlayers, userID)

	// Обновляем метрику активных игроков
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.SetActivePlayers(float64(len(h.activePlayers)))
	}
	h.mutex.Unlock()
}

// Stop останавливает обработчик игры
func (h *GameHandler) Stop() {
	// Оставить закрытие каналов
	close(h.stopWorker)

	// Закрываем соединение с RabbitMQ
	if h.rabbitmq != nil {
		if err := h.rabbitmq.Close(); err != nil {
			logger.Error.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}

	logger.Info.Println("Game handler stopped")
}

// createBetKeyboard создает клавиатуру для ставок
func (h *GameHandler) createBetKeyboard(language string, userID uint, currentRoundID uint) *telego.InlineKeyboardMarkup {
	// Проверяем, может ли пользователь ставить на Zero
	canBetZero, _, err := h.service.CanBetZero(userID)
	if err != nil {
		logger.Error.Printf("Error checking zero bet: %v", err)
		canBetZero = false
	}
	round := fmt.Sprint(currentRoundID)

	// Создаем клавиатуру с соответствующими кнопками
	var zeroButton telego.InlineKeyboardButton
	if canBetZero {
		zeroButton = telego.InlineKeyboardButton{Text: h.bot.getText("btn_bet_zero", language), CallbackData: CallbackBetZero + round}
	} else {
		zeroButton = telego.InlineKeyboardButton{Text: h.bot.getText("btn_bet_zero_locked", language), CallbackData: CallbackBetZeroLocked}
	}
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: h.bot.getText("btn_bet_red", language), CallbackData: CallbackBetRed + round},
				{Text: h.bot.getText("btn_bet_black", language), CallbackData: CallbackBetBlack + round},
				zeroButton,
			},
			{
				{Text: h.bot.getText("availablebets", language), CallbackData: CallbackBetAvailable},
			},
		},
	}
}

// Створює клавіатуру для стартового меню гри
func (h *GameHandler) createStartPlayKeyboard(language string) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: h.bot.getText("btn_awards", language), CallbackData: "awards_play"},
				{Text: h.bot.getText("btn_payments", language), CallbackData: "payments_play"},
				{Text: h.bot.getText("btn_fairplay", language), CallbackData: "fairplay_play"},
			},
			{
				{Text: h.bot.getText("btn_startround", language), CallbackData: CallbackStartRound},
			},
		},
	}
}
