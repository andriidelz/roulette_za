package bot

import (
	"fmt"
	"sync"
	"time"

	"roulette/internal/logger"
	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"

	"github.com/mymmrac/telego"
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
	waitingPlayers  map[int64]bool             // Карта игроков, ожидающих результатов
	activeBets      map[int64]models.BetOption // Карта активных ставок игроков
	activePlayers   map[int64]int              // Карта активных игроков в режиме /play
	rabbitmq        *messaging.RabbitMQ        // Клиент RabbitMQ
	processedRounds map[uint]bool              // Хранит ID обработанных раундов для избежания дублирования
	processMutex    sync.Mutex                 // Мьютекс для доступа к processedRounds

	roundMsgChan   chan RoundMessage
	processingLock sync.Mutex
	stopWorker     chan struct{}
}

const (
	CallbackStartRound    = "startround"
	CallbackBetRed        = "bet_red"
	CallbackBetBlack      = "bet_black"
	CallbackBetZero       = "bet_zero"
	CallbackBetZeroLocked = "bet_zero_locked"
	CallbackBetAvailable  = "availablebets"

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
		waitingPlayers:  make(map[int64]bool),
		activeBets:      make(map[int64]models.BetOption),
		activePlayers:   make(map[int64]int),
		rabbitmq:        rmq,
		processedRounds: make(map[uint]bool),
		roundMsgChan:    make(chan RoundMessage, 100), // Буфер для сообщений
		stopWorker:      make(chan struct{}),
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
					h.handleRoundCompletion(msg.Round)

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
					h.notifyActivePlayers(msg.Round)

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

// notifyActivePlayers уведомляет всех активных игроков о новом раунде
func (h *GameHandler) notifyActivePlayers(round *models.HashEntry) {
	stopPlayers := []int64{}
	h.mutex.RLock()
	players := make([]int64, 0, len(h.activePlayers))
	for userID, val := range h.activePlayers {
		// Перед отправлением раунда проверяем активный ли юзер
		// Если юзер пропустил 10 раундов останавливаем для него игру
		if val >= 10 {
			stopPlayers = append(stopPlayers, userID)
		} else {
			// Инкрементим ему кол-во раундов. В случае если он сделает ставку это число обнулится
			h.activePlayers[userID] = val + 1
			players = append(players, userID)
		}
	}
	h.mutex.RUnlock()

	// Останавливаем всех неактивных игроков
	for i := range stopPlayers {
		logger.Error.Println("Stop user ", stopPlayers[i])
		h.stopGame(stopPlayers[i])
	}

	roundIDBase62 := utils.ToBase62(uint(round.ID))

	// Запускаем таймер для одного уведомления - 5 секунд до конца раунда
	go func() {
		// Вычисляем время до уведомления
		createdAt := round.CreatedAt
		// roundDuration := 15 * time.Second // Раунд длится 15 секунд

		// Отправляем уведомление за 5 секунд до конца раунда (на 10-й секунде)
		fiveSecondsMark := createdAt.Add(10 * time.Second)

		// Вычисляем время до уведомления
		timeToFive := time.Until(fiveSecondsMark)

		// Отправляем уведомление за 5 секунд до конца раунда
		if timeToFive > 0 {
			time.Sleep(timeToFive)
			h.notifyTimeRemaining(round, 5)
		}
	}()

	for _, userID := range players {
		user, err := h.service.GetUser(userID)
		if err != nil {
			logger.Error.Printf("Error getting user %d: %v", userID, err)
			continue
		}

		language := user.LanguageCode
		if language == "" {
			language = "en"
		}

		// Вычисляем оставшееся время до конца раунда
		elapsedTime := time.Since(round.CreatedAt)
		roundDuration := 15 * time.Second // 15-секундный раунд
		remainingSeconds := int((roundDuration - elapsedTime).Seconds())

		if remainingSeconds < 0 {
			remainingSeconds = 0
		}

		// Получаем локализированный шаблон для нового раунда с обратным отсчетом
		options := h.bot.prepareMessage("round_info_countdown", language)
		options.Text = fmt.Sprintf(options.Text, roundIDBase62, round.Hash, remainingSeconds)

		options.InlineKeyboard = h.createBetKeyboard(language, userID)
		h.bot.SendMessage(userID, options)
	}
}

// notifyTimeRemaining уведомляет игроков об оставшемся времени раунда
func (h *GameHandler) notifyTimeRemaining(round *models.HashEntry, seconds int) {
	h.mutex.RLock()
	// Получаем активных игроков
	players := make([]int64, 0, len(h.activePlayers))
	for userID := range h.activePlayers {
		players = append(players, userID)
	}
	h.mutex.RUnlock()

	roundIDBase62 := utils.ToBase62(uint(round.ID))

	for _, userID := range players {
		// Проверяем, сделал ли игрок уже ставку в текущем раунде
		bets, err := h.service.GetUserBetsForRound(userID, round.ID)
		if err != nil {
			logger.Error.Printf("Error getting user bets: %v", err)
			continue
		}

		// Если пользователь уже сделал ставку, не отправляем уведомление
		if len(bets) > 0 {
			continue
		}

		user, err := h.service.GetUser(userID)
		if err != nil {
			logger.Error.Printf("Error getting user %d: %v", userID, err)
			continue
		}

		language := user.LanguageCode
		if language == "" {
			language = "en"
		}

		// Получаем локализированный шаблон для времени
		var options MessageOptions
		if seconds == 15 {
			options = h.bot.prepareMessage("nextbid15", language)
		} else {
			options = h.bot.prepareMessage("nextbid5", language)
		}

		// Формируем текст в новом формате
		options.Text = fmt.Sprintf(options.Text, roundIDBase62)
		options.InlineKeyboard = h.createBetKeyboard(language, userID)

		h.bot.SendMessage(userID, options)
	}
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

		// Получаем данные о раунде
		round, err := h.service.GetHashEntryByID(roundID)
		if err != nil {
			return fmt.Errorf("failed to get round #%d: %w", roundID, err)
		}

		// Важно! Сначала обрабатываем результаты и уведомляем игроков напрямую
		// вместо помещения в канал сообщений
		h.handleRoundCompletion(round)

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
func (h *GameHandler) handleRoundCompletion(round *models.HashEntry) {
	// Получаем всех пользователей, которые сделали ставки в этом раунде
	bets, err := h.service.ProcessAndGetBets(round.ID)
	if err != nil {
		logger.Error.Printf("Error processing bets for round #%d: %v", round.ID, err)
		return
	}

	if len(bets) == 0 {
		return
	}

	// Получаем результат раунда
	result, err := h.service.GetRoundResult(round.ID)
	if err != nil {
		logger.Error.Printf("Error getting result for round #%d: %v", round.ID, err)
		return
	}

	logger.Info.Printf("Round #%d result: %s (number: %d)", round.ID, result, round.Number)

	// Группируем ставки по пользователям
	userBets := make(map[int64]models.Bet)
	for _, bet := range bets {
		userBets[bet.User.TelegramID] = bet
	}

	// Создаем WaitGroup для ожидания завершения всех уведомлений
	var wg sync.WaitGroup

	// Уведомляем каждого пользователя
	for userID, bet := range userBets {
		wg.Add(1)
		go func(uid int64, b models.Bet) {
			defer wg.Done()

			// Отправляем уведомления о результатах синхронно
			if err := h.notifyPlayerAboutResult(uid, round.ID, round, result, b.Option); err != nil {
				logger.Error.Printf("Error notifying player %d: %v", uid, err)
			} else {
				logger.Info.Printf("Successfully notified player %d about round #%d results", uid, round.ID)
			}
		}(userID, bet)
	}

	// Ожидаем завершения всех уведомлений
	wg.Wait()
	logger.Info.Printf("All players have been notified about round #%d results", round.ID)
}

// notifyPlayerAboutResult уведомляет игрока о результате раунда
func (h *GameHandler) notifyPlayerAboutResult(userID int64, roundID uint, round *models.HashEntry, result models.BetOption, userBet models.BetOption) error {
	logger.Info.Printf("notifyPlayerAboutResult called for user %d, round #%d", userID, roundID)

	// Не отправляем результаты если пользователя нет в списке активных игроков
	h.mutex.RLock()
	_, exists := h.activePlayers[userID]
	h.mutex.RUnlock()
	if !exists {
		return nil
	}

	// Получаем пользователя
	userInfo, err := h.service.GetUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", userID, err)
		return fmt.Errorf("error getting user: %w", err)
	}

	language := userInfo.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем информацию о ставке
	userBets, err := h.service.GetUserBetsForRound(userID, roundID)
	if err != nil {
		return fmt.Errorf("error getting bets: %w", err)
	}

	if len(userBets) == 0 {
		return fmt.Errorf("no bets found for user %d in round %d", userID, roundID)
	}

	bet := userBets[0]
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
	var resultSticker string
	switch result {
	case models.Red:
		resultSticker = getRandomSticker(StickerRedRes1, StickerRedRes2)
	case models.Black:
		resultSticker = getRandomSticker(StickerBlackRes1, StickerBlackRes2)
	case models.Zero:
		resultSticker = getRandomSticker(StickerZeroRes1, StickerZeroRes2)
	}
	// Отправляем стикер результата
	h.bot.MakeRequestDeferred(userID, 0, MessageOptions{
		Text:       resultSticker,
		MethodName: sendSticker,
	})

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
	}
	// Отправляем текст результата
	h.bot.SendMessage(userID, h.bot.prepareMessage(resultLangKey, language))

	// 3. Отправляем стикер выигрыша/проигрыша на 19 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	if won {
		// Отправляем стикер выигрыша
		h.bot.MakeRequestDeferred(userID, 0, MessageOptions{
			Text:       StickerWin,
			MethodName: sendSticker,
		})
	} else {
		// Отправляем стикер проигрыша
		h.bot.MakeRequestDeferred(userID, 0, MessageOptions{
			Text:       StickerLose,
			MethodName: sendSticker,
		})
	}

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

	// Формируем часть о рейтинге

	// TMP: нужно запускать 1 раз после конца раунда но перед выводом
	// Пересчитываем балы и эффективность пользователя и
	// обновляем позиции всех пользователей
	if err := h.service.GetRepo().UpdateWeeklyRatingForUser(userInfo.ID); err != nil {
		logger.Error.Printf("Error refreshing ratings before getting position: %v", err)
	}

	// Получаем текущий рейтинг пользователя
	year, week := time.Now().ISOWeek()
	rating, err := h.service.GetRepo().GetUserWeeklyRating(userInfo.ID, year, week)
	if err != nil {
		logger.Error.Printf("Error get rating: %v", err)
	}

	// Получаем информацию о призовом фонде

	// Переменные для текста рейтинга
	var ratingText string
	var prizeFundAmount float64 = 1000.0 // По умолчанию
	var userShare float64 = 0.0
	var topCount int = 100 // По умолчанию

	// Получаем призовой фонд через репозиторий
	prizeFund, err := h.service.GetPrizeFund(year, week)
	if err == nil {
		// Получаем данные о призовом фонде из БД
		prizeFundAmount = prizeFund.Amount
		topCount = prizeFund.TopCount

		if rating.Position > 0 && rating.Position <= topCount {
			// Расчет доли пользователя
			userShare = prizeFundAmount / float64(topCount) * (float64(topCount-rating.Position+1) / float64(topCount))
		}
	} else {
		logger.Error.Printf("Error getting prize fund: %v", err)

		// Если не удалось получить данные о призовом фонде, используем значения по умолчанию
		if rating.Position > 0 && rating.Position <= 100 {
			// Упрощенный расчет доли пользователя
			userShare = prizeFundAmount / 100.0 * (float64(100-rating.Position+1) / 100.0)
		}
	}

	// Формируем сообщение о рейтинге
	ratingTemplate := h.service.GetText("bidrating", language)
	ratingText = fmt.Sprintf(ratingTemplate, rating.Points, rating.Position, userShare, prizeFundAmount)

	// Часть проверки баланса ставок
	betsBalance, err := h.service.GetUserRemainingBets(userID)
	if err != nil {
		logger.Error.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	var betsBalanceText string
	var additionalMessage string

	if betsBalance <= 0 {
		// Недостаточно ставок
		betsBalanceText = h.service.GetText("betsbalancelow", language)
		additionalMessage = h.service.GetText("nextbidlow", language)
	} else {
		// Достаточно ставок
		betsBalanceTemplate := h.service.GetText("betsbalanceok", language)
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
	checkSystemText := h.service.GetText("systemcheck", language)
	roundIDBase62 := utils.ToBase62(uint(roundID))
	checkSystemURL := fmt.Sprintf("%s/hashes/?id=%s", webPage, roundIDBase62)

	viewRatingText := h.service.GetText("viewrating", language)

	// Верхний ряд из 2 кнопок
	inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
		{Text: checkSystemText, URL: checkSystemURL},
		{Text: viewRatingText, CallbackData: "view_rating"},
	})

	// TODO: кнопка временно скрыта
	// Второй ряд только с кнопкой пополнения баланса
	// topUpBalanceText := h.service.GetText("topupbalance", language)
	// inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
	// 	{Text: topUpBalanceText, CallbackData: "noop"},
	// })

	// Если баланс ставок недостаточен, добавляем кнопку остановки игры в третий ряд
	if betsBalance <= 0 {
		stopGameText := h.service.GetText("stopgame", language)
		inlineButtons = append(inlineButtons, []telego.InlineKeyboardButton{
			{Text: stopGameText, CallbackData: "stop_game"},
		})
	}

	// Создаем inline клавиатуру с кнопками
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: inlineButtons,
	}

	// Отправляем объединенное сообщение с клавиатурой
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
func (h *GameHandler) MakeBet(userID int64, option models.BetOption) error {
	logger.Info.Printf("MakeBet called for user %d with option %s", userID, option)

	// Получаем пользователя
	user, err := h.service.GetUser(userID)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	// Проверяем статус бана
	if user.Banned {
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
	if currentRound.IsCompleted {
		return fmt.Errorf("round is already completed")
	}

	// Проверяем, не делал ли пользователь уже ставку в этом раунде
	existingBets, err := h.service.GetUserBetsForRound(userID, currentRound.ID)
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
		canBetZero, _, err := h.service.CanBetZero(userID)
		if err != nil {
			logger.Error.Printf("Error checking zero bet: %v", err)
			return fmt.Errorf("error checking zero bet: %w", err)
		}

		if !canBetZero {
			return fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Проверяем доступное количество ставок
	betsRemaining, err := h.service.GetUserRemainingBets(userID)
	if err != nil {
		logger.Error.Printf("Error checking remaining bets: %v", err)
		// Не возвращаем ошибку, так как это не критично
	} else if betsRemaining == 0 {
		return fmt.Errorf("no bets left for today")
	}

	// Делаем ставку через сервис
	if err := h.service.MakeBet(userID, option); err != nil {
		logger.Error.Printf("Error making bet: %v", err)
		return fmt.Errorf("error making bet: %w", err)
	}

	// Записываем метрику ставки
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.RecordBet(string(option))
	}

	// Получаем язык пользователя
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Регистрируем пользователя как ожидающего результата и сохраняем его ставку
	h.mutex.Lock()
	h.waitingPlayers[userID] = true
	h.activeBets[userID] = option
	h.activePlayers[userID] = 1 // обнуляем кол-во пропущенных раундов
	h.mutex.Unlock()

	logger.Info.Printf("Bet created successfully for user %d in round %d (waitingPlayers count: %d)", userID, currentRound.ID, len(h.waitingPlayers))

	// Сразу отправляем стикер "Ставки больше не принимаются"
	h.bot.MakeRequestDeferred(userID, 0, MessageOptions{
		Text:       StickerNoBids,
		MethodName: sendSticker,
	})

	// После короткой паузы отправляем сообщение о принятии ставки
	go func() {
		time.Sleep(1000 * time.Millisecond)
		options := h.bot.prepareMessage("nomorebids", language)
		options.InlineKeyboard = h.createBetKeyboard(language, userID)

		h.bot.SendMessage(userID, options)
	}()

	return nil
}

// handleAvailableBets присылаем игроку доступное количество ставок
func (h *GameHandler) handleAvailableBets(query *telego.CallbackQuery) {

	h.bot.answerCallbackQuery(query.ID, "", false)
	user := query.From
	dbUser, err := h.service.GetUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error get user: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Получаем доступное количество ставок
	betsBalance, err := h.service.GetUserRemainingBets(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	var options MessageOptions
	if betsBalance <= 0 {
		options = h.bot.prepareMessage("betsbalancelow", language)
	} else {
		options = h.bot.prepareMessage("betsbalanceok", language)
		options.Text = fmt.Sprintf(options.Text, betsBalance)
	}
	options.InlineKeyboard = h.createBetKeyboard(language, user.ID)
	h.bot.SendMessage(query.Message.GetChat().ID, options)
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	dbUser, err := h.bot.service.GetUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Сначала отправляем сообщение с описанием игры
	options := h.bot.prepareMessage("playstart1", language)

	// Создаем inline клавиатуру
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: h.service.GetText("btn_awards", language), CallbackData: "awards"},
				{Text: h.service.GetText("btn_payments", language), CallbackData: "payments"},
				{Text: h.service.GetText("btn_fairplay", language), CallbackData: "fairplay"},
			},
			{
				{Text: h.service.GetText("btn_startround", language), CallbackData: CallbackStartRound},
			},
		},
	}

	// Отправляем первое сообщение с описанием игры и кнопкой на правила
	h.bot.SendMessage(message.Chat.ID, options)
}

// handleStartRound - старт нового раунду гри
func (h *GameHandler) handleStartRound(query *telego.CallbackQuery) {

	user := query.From
	dbUser, err := h.service.GetUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error get user: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Пытаемся получить текущий раунд
	currentRound, err := h.service.GetCurrentRound()
	if err != nil {
		logger.Error.Printf("Error getting current round: %v", err)
		return
	}

	// Проверяем, что раунд существует
	if currentRound == nil {
		logger.Warning.Printf("Current round is nil, waiting for a new round")
		// Виводимо pop-up toast без підтвердження
		h.bot.answerCallbackQuery(query.ID, h.service.GetText("waiting_for_round", language), false)
		return
	}

	// Обновляем текущий раунд в хендлере
	h.mutex.Lock()
	h.currentRound = currentRound
	h.mutex.Unlock()

	// Получаем ID раунда в Base62 формате
	roundIDBase62 := utils.ToBase62(uint(currentRound.ID))

	// Вычисляем оставшееся время до конца раунда
	elapsedTime := time.Since(currentRound.CreatedAt)
	roundDuration := 15 * time.Second // Изменено с 30 на 15 секунд
	remainingTime := roundDuration - elapsedTime

	// Если осталось меньше 0 секунд, ждем следующий раунд
	if remainingTime < 0 {
		// Виводимо pop-up toast без підтвердження
		h.bot.answerCallbackQuery(query.ID, h.service.GetText("waiting_for_round", language), false)
		return
	}

	// Добавляем пользователя в список активных игроков
	h.mutex.Lock()
	h.activePlayers[user.ID] = 1

	// Обновляем метрику активных игроков
	if metrics := h.bot.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.SetActivePlayers(float64(len(h.activePlayers)))
	}
	h.mutex.Unlock()

	h.bot.answerCallbackQuery(query.ID, "", false)

	// Формируем текст в новом формате
	remainingSeconds := int(remainingTime.Seconds())
	options := h.bot.prepareMessage("round_info_countdown", language)
	options.Text = fmt.Sprintf(options.Text, roundIDBase62, currentRound.Hash, remainingSeconds)
	options.InlineKeyboard = h.createBetKeyboard(language, user.ID)

	// Отправляем сообщение с информацией о раунде и клавиатурой для ставок
	h.bot.SendMessage(query.Message.GetChat().ID, options)
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
func (h *GameHandler) createBetKeyboard(language string, userID int64) *telego.InlineKeyboardMarkup {

	// Проверяем, может ли пользователь ставить на Zero
	canBetZero, _, err := h.service.CanBetZero(userID)
	if err != nil {
		logger.Error.Printf("Error checking zero bet: %v", err)
		canBetZero = false
	}

	// Создаем клавиатуру с соответствующими кнопками
	var zeroButton telego.InlineKeyboardButton
	if canBetZero {
		zeroButton = telego.InlineKeyboardButton{Text: h.service.GetText("btn_bet_zero", language), CallbackData: CallbackBetZero}
	} else {
		zeroButton = telego.InlineKeyboardButton{Text: h.service.GetText("btn_bet_zero_locked", language), CallbackData: CallbackBetZeroLocked}
	}

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: h.service.GetText("btn_bet_red", language), CallbackData: CallbackBetRed},
				{Text: h.service.GetText("btn_bet_black", language), CallbackData: CallbackBetBlack},
				zeroButton,
			},
			{
				{Text: h.service.GetText("availablebets", language), CallbackData: CallbackBetAvailable},
			},
		},
	}
}
