package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"

	"github.com/mymmrac/telego"
)

const webPage = "https://roulette.myapps.vip"

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
	activePlayers   map[int64]bool             // Карта активных игроков в режиме /play
	rabbitmq        *messaging.RabbitMQ        // Клиент RabbitMQ
	processedRounds map[uint]bool              // Хранит ID обработанных раундов для избежания дублирования
	processMutex    sync.Mutex                 // Мьютекс для доступа к processedRounds

	roundMsgChan   chan RoundMessage
	processingLock sync.Mutex
	stopWorker     chan struct{}
}

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
		activePlayers:   make(map[int64]bool),
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
	log.Println("Starting message processing worker")

	for {
		select {
		case <-h.stopWorker:
			log.Println("Message processing worker stopped")
			return
		case msg := <-h.roundMsgChan:
			// Блокируем на время обработки сообщения
			h.processingLock.Lock()

			switch msg.Type {
			case RoundCompletedMessage:
				log.Printf("Processing RoundCompletedMessage for round #%d", msg.RoundID)
				// Обрабатываем завершение раунда
				if msg.Round != nil {
					// Вызываем обработчик завершения раунда
					h.handleRoundCompletion(msg.Round)

					// Если доступны дополнительные данные, логируем их
					if msg.Data != nil {
						log.Printf("Round #%d completed with result: %s", msg.RoundID, msg.Data.Result)

						// Здесь можно использовать данные для дополнительной обработки,
						// например, для верификации результата или отображения доп. информации
					}

					// Помечаем раунд как обработанный
					h.processMutex.Lock()
					h.processedRounds[msg.RoundID] = true
					h.processMutex.Unlock()
				}
			case RoundStartedMessage:
				log.Printf("Processing RoundStartedMessage for round #%d", msg.RoundID)
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
						log.Printf("Round #%d started with hash: %s", msg.RoundID, msg.Data.Hash)
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
	h.mutex.RLock()
	players := make([]int64, 0, len(h.activePlayers))
	for userID := range h.activePlayers {
		players = append(players, userID)
	}
	h.mutex.RUnlock()

	roundIDBase62 := utils.ToBase62(uint(round.ID))

	// Запускаем таймер для одного уведомления - 5 секунд до конца раунда
	go func() {
		// Вычисляем время до уведомления
		createdAt := round.CreatedAt
		// roundDuration := 15 * time.Second // Раунд длится 15 секунд

		// Отправляем уведомление за 5 секунд до конца раунда (на 10-й секунде)
		fiveSecondsMark := createdAt.Add(10 * time.Second)

		// Вычисляем время до уведомления
		timeToFiveSeconds := time.Until(fiveSecondsMark)

		// Отправляем уведомление за 5 секунд до конца раунда
		if timeToFiveSeconds > 0 {
			time.Sleep(timeToFiveSeconds)
			// Отправляем только одно уведомление - о 5 секундах до конца раунда
			log.Printf("Sending 5 seconds remaining notification for round #%d", round.ID)
			h.notifyTimeRemaining(round, 5)
		}
	}()

	for _, userID := range players {
		user, err := h.service.GetUser(userID)
		if err != nil {
			log.Printf("Error getting user %d: %v", userID, err)
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
		roundInfoTemplate := h.service.GetText("round_info_countdown", language)
		roundInfoText := fmt.Sprintf(roundInfoTemplate, roundIDBase62, round.Hash, remainingSeconds)

		// Получаем доступное количество ставок
		betsBalance, err := h.service.GetUserRemainingBets(userID)
		if err != nil {
			log.Printf("Error getting user remaining bets: %v", err)
			betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:          roundInfoText,
			ReplyKeyboard: h.createDetailedBetKeyboard(language, userID, betsBalance),
		})
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
			log.Printf("Error getting user bets: %v", err)
			continue
		}

		// Если пользователь уже сделал ставку, не отправляем уведомление
		if len(bets) > 0 {
			continue
		}

		user, err := h.service.GetUser(userID)
		if err != nil {
			log.Printf("Error getting user %d: %v", userID, err)
			continue
		}

		language := user.LanguageCode
		if language == "" {
			language = "en"
		}

		// Получаем локализированный шаблон для времени
		var timeTemplate string
		if seconds == 15 {
			timeTemplate = h.service.GetText("nextbid15", language)
		} else {
			timeTemplate = h.service.GetText("nextbid5", language)
		}

		// Формируем текст в новом формате
		timeText := fmt.Sprintf(timeTemplate, roundIDBase62)

		// Узнаем доступное количество ставок
		betsBalance, err := h.service.GetUserRemainingBets(userID)
		if err != nil {
			log.Printf("Error getting user remaining bets: %v", err)
			betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:          timeText,
			ReplyKeyboard: h.createDetailedBetKeyboard(language, userID, betsBalance),
		})
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
			log.Printf("Round #%d already processed, skipping duplicate message", roundID)
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
		log.Printf("Processing round completed message for round #%d", roundID)

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
		log.Printf("Sending round started message to processing queue for round #%d", roundID)
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
	log.Printf("Handling round #%d completion", round.ID)

	// Получаем всех пользователей, которые сделали ставки в этом раунде
	bets, err := h.service.ProcessAndGetBets(round.ID)
	if err != nil {
		log.Printf("Error processing bets for round #%d: %v", round.ID, err)
		return
	}

	if len(bets) == 0 {
		log.Printf("No bets found for round #%d", round.ID)
		return
	}

	// Получаем результат раунда
	result, err := h.service.GetRoundResult(round.ID)
	if err != nil {
		log.Printf("Error getting result for round #%d: %v", round.ID, err)
		return
	}

	log.Printf("Round #%d result: %s (number: %d)", round.ID, result, round.Number)

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
				log.Printf("Error notifying player %d: %v", uid, err)
			} else {
				log.Printf("Successfully notified player %d about round #%d results", uid, round.ID)
			}
		}(userID, bet)
	}

	// Ожидаем завершения всех уведомлений
	wg.Wait()
	log.Printf("All players have been notified about round #%d results", round.ID)
}

// notifyPlayerAboutResult уведомляет игрока о результате раунда
func (h *GameHandler) notifyPlayerAboutResult(userID int64, roundID uint, round *models.HashEntry, result models.BetOption, userBet models.BetOption) error {
	log.Printf("notifyPlayerAboutResult called for user %d, round #%d", userID, roundID)

	// Получаем пользователя
	userInfo, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
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
	h.bot.SendSticker(userID, resultSticker)

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
	resultText := h.service.GetText(resultLangKey, language)

	// Отправляем текст результата
	h.bot.SendMessage(userID, MessageOptions{
		Text: resultText,
	})

	// 3. Отправляем стикер выигрыша/проигрыша на 19 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	if won {
		// Отправляем стикер выигрыша
		h.bot.SendSticker(userID, StickerWin)
	} else {
		// Отправляем стикер проигрыша
		h.bot.SendSticker(userID, StickerLose)
	}

	// 4. Отправляем полное сообщение о выигрыше/проигрыше на 20 секунде (через 1 секунду)
	time.Sleep(1 * time.Second)

	// Формируем сообщение о выигрыше/проигрыше
	var winLoseText string
	if won {
		winMessageTemplate := h.service.GetText("winmessage", language)
		winLoseText = fmt.Sprintf(winMessageTemplate, points)
	} else {
		loseMsgText := h.service.GetText("losemessage", language)
		winLoseText = loseMsgText
	}

	// Формируем часть о рейтинге

	// TMP: нужно запускать 1 раз после конца раунда но перед выводом
	// Пересчитываем балы и эффективность пользователя и
	// обновляем позиции всех пользователей
	if err := h.service.GetRepo().UpdateWeeklyRatingForUser(userInfo.ID); err != nil {
		log.Printf("Error refreshing ratings before getting position: %v", err)
	}

	// Получаем текущий рейтинг пользователя
	year, week := time.Now().ISOWeek()
	rating, err := h.service.GetRepo().GetUserWeeklyRating(userInfo.ID, year, week)
	if err != nil {
		log.Printf("Error get rating: %v", err)
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
		log.Printf("Error getting prize fund: %v", err)

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
		log.Printf("Error getting user remaining bets: %v", err)
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
	combinedMessage := winLoseText + "\n\n" + ratingText + "\n\n" + betsBalanceText

	// Если есть дополнительное сообщение для недостаточного баланса, добавляем его
	if additionalMessage != "" {
		combinedMessage += "\n\n" + additionalMessage
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
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: inlineButtons,
	}

	// Отправляем объединенное сообщение с клавиатурой
	h.bot.SendMessage(userID, MessageOptions{
		Text:           combinedMessage,
		InlineKeyboard: inlineKeyboard,
	})

	return nil
}

// MakeBet делает ставку в текущем раунде
func (h *GameHandler) MakeBet(userID int64, option models.BetOption) error {
	log.Printf("MakeBet called for user %d with option %s", userID, option)

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
		log.Printf("Error checking existing bets: %v", err)
		return fmt.Errorf("error checking existing bets: %w", err)
	}

	if len(existingBets) > 0 {
		log.Printf("User %d already made a bet in round %d", userID, currentRound.ID)
		return fmt.Errorf("user has already made a bet in this round")
	}

	// Проверяем, может ли пользователь делать ставку на Zero
	if option == models.Zero {
		canBetZero, _, err := h.service.CanBetZero(userID)
		if err != nil {
			log.Printf("Error checking zero bet: %v", err)
			return fmt.Errorf("error checking zero bet: %w", err)
		}

		if !canBetZero {
			return fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Проверяем доступное количество ставок
	betsRemaining, err := h.service.GetUserRemainingBets(userID)
	if err != nil {
		log.Printf("Error checking remaining bets: %v", err)
		// Не возвращаем ошибку, так как это не критично
	} else if betsRemaining == 0 {
		return fmt.Errorf("no bets left for today")
	}

	// Делаем ставку через сервис
	if err := h.service.MakeBet(userID, option); err != nil {
		log.Printf("Error making bet: %v", err)
		return fmt.Errorf("error making bet: %w", err)
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
	h.mutex.Unlock()

	log.Printf("Bet created successfully for user %d in round %d (waitingPlayers count: %d)", userID, currentRound.ID, len(h.waitingPlayers))

	// Сразу отправляем стикер "Ставки больше не принимаются"
	h.bot.SendSticker(userID, StickerNoBids)

	// После короткой паузы отправляем сообщение о принятии ставки
	go func() {
		time.Sleep(1000 * time.Millisecond)
		nomorebidsText := h.service.GetText("nomorebids", language)
		h.bot.SendMessage(userID, MessageOptions{
			Text:          nomorebidsText,
			ReplyKeyboard: h.createDetailedBetKeyboard(language, userID, betsRemaining),
		})
	}()

	return nil
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Сначала отправляем сообщение с описанием игры
	playStartText := h.service.GetText("playstart1", language)

	// Создаем inline клавиатуру с кнопкой "Детальные правила"
	rulesButtonText := h.service.GetText("rulesstart", language)
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: rulesButtonText, URL: webPage + "/faq#gameplay"},
			},
		},
	}

	// Отправляем первое сообщение с описанием игры и кнопкой на правила
	h.bot.SendMessage(message.Chat.ID, MessageOptions{
		Text:           playStartText,
		InlineKeyboard: inlineKeyboard,
	})

	// Добавляем пользователя в список активных игроков
	h.mutex.Lock()
	h.activePlayers[user.ID] = true
	h.mutex.Unlock()

	log.Printf("Added user %d to active players", user.ID)

	// Пытаемся получить текущий раунд
	currentRound, err := h.service.GetCurrentRound()
	if err != nil {
		log.Printf("Error getting current round: %v", err)
		return
	}

	// Проверяем, что раунд существует
	if currentRound == nil {
		log.Printf("Current round is nil, waiting for a new round")
		waitingText := h.service.GetText("waiting_for_round", language)
		h.bot.SendMessage(message.Chat.ID, MessageOptions{
			Text: waitingText,
		})
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
		waitingText := h.service.GetText("waiting_for_round", language)
		h.bot.SendMessage(message.Chat.ID, MessageOptions{
			Text: waitingText,
		})
		return
	}

	// Формируем текст в новом формате
	remainingSeconds := int(remainingTime.Seconds())
	roundInfoTemplate := h.service.GetText("round_info_countdown", language)
	roundInfoText := fmt.Sprintf(roundInfoTemplate, roundIDBase62, currentRound.Hash, remainingSeconds)

	// Получаем доступное количество ставок
	betsBalance, err := h.service.GetUserRemainingBets(user.ID)
	if err != nil {
		log.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	// Отправляем сообщение с информацией о раунде и клавиатурой для ставок
	h.bot.SendMessage(message.Chat.ID, MessageOptions{
		Text:          roundInfoText,
		ReplyKeyboard: h.createDetailedBetKeyboard(language, user.ID, betsBalance),
	})
}

// HandleBackButton обрабатывает нажатие кнопки "Назад"
func (h *GameHandler) HandleBackButton(userID int64) {
	// Удаляем пользователя из списка активных игроков
	h.mutex.Lock()
	delete(h.activePlayers, userID)
	h.mutex.Unlock()

	log.Printf("Removed user %d from active players", userID)
}

// HandleStopGameButton обрабатывает нажатие кнопки "Стоп игра"
func (h *GameHandler) HandleStopGameButton(userID int64) {
	// Удаляем пользователя из списка активных игроков
	h.mutex.Lock()
	delete(h.activePlayers, userID)
	h.mutex.Unlock()

	log.Printf("User %d stopped the game", userID)
}

// Stop останавливает обработчик игры
func (h *GameHandler) Stop() {
	// Оставить закрытие каналов
	close(h.stopWorker)

	// Закрываем соединение с RabbitMQ
	if h.rabbitmq != nil {
		if err := h.rabbitmq.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}

	log.Println("Game handler stopped")
}

// createDetailedBetKeyboard создает расширенную клавиатуру для ставок с дополнительной информацией
func (h *GameHandler) createDetailedBetKeyboard(language string, userID int64, betsBalance int) *telego.ReplyKeyboardMarkup {
	// Получаем локализированные тексты для кнопок
	btnRedText := h.service.GetText("btn_bet_red", language)
	btnBlackText := h.service.GetText("btn_bet_black", language)
	btnZeroText := h.service.GetText("btn_bet_zero", language)
	btnZeroLockedText := h.service.GetText("btn_bet_zero_locked", language)
	btnStopText := h.service.GetText("stop", language)
	betsBalanceText := h.service.GetText("availablebets", language)

	// Проверяем, может ли пользователь ставить на Zero
	canBetZero, _, err := h.service.CanBetZero(userID)
	if err != nil {
		log.Printf("Error checking zero bet: %v", err)
		canBetZero = false
	}

	// Создаем клавиатуру с соответствующими кнопками
	var zeroButton telego.KeyboardButton
	if canBetZero {
		zeroButton = telego.KeyboardButton{Text: btnZeroText}
	} else {
		zeroButton = telego.KeyboardButton{Text: btnZeroLockedText}
	}

	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: btnRedText},
				{Text: btnBlackText},
				zeroButton,
			},
			{
				{Text: betsBalanceText},
				{Text: btnStopText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       true,
	}
}
