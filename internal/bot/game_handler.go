package bot

import (
	"context"
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

// Интерфейс для уведомления о новом хеше (для обратной совместимости)
type HashChangeNotifier interface {
	HandleNewRound(hashEntry *models.HashEntry)
}

// GameHandler управляет игровым процессом рулетки
type GameHandler struct {
	bot              *Bot
	service          service.Service
	currentRound     *models.HashEntry
	mutex            sync.RWMutex
	waitingPlayers   map[int64]bool             // Карта игроков, ожидающих результатов
	activeBets       map[int64]models.BetOption // Карта активных ставок игроков
	activePlayers    map[int64]bool             // Карта активных игроков в режиме /play
	stopChan         chan struct{}              // Канал для остановки игры
	checkRoundTicker *time.Ticker               // Тикер для периодической проверки раундов
	rabbitmq         *messaging.RabbitMQ        // Клиент RabbitMQ
	processedRounds  map[uint]bool              // Хранит ID обработанных раундов для избежания дублирования
	processMutex     sync.Mutex                 // Мьютекс для доступа к processedRounds
}

// NewGameHandler создает новый обработчик игры
func NewGameHandler(bot *Bot, service service.Service, rabbitmqURL string) (*GameHandler, error) {
	// Создаем клиент RabbitMQ
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, "roulette_events", "bot")
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	handler := &GameHandler{
		bot:              bot,
		service:          service,
		waitingPlayers:   make(map[int64]bool),
		activeBets:       make(map[int64]models.BetOption),
		activePlayers:    make(map[int64]bool),
		stopChan:         make(chan struct{}),
		checkRoundTicker: time.NewTicker(1 * time.Second), // Проверка каждую секунду
		rabbitmq:         rmq,
		processedRounds:  make(map[uint]bool),
	}

	// Запускаем горутину для периодической проверки раундов (для обратной совместимости)
	go handler.startRoundCheckLoop()

	// Подписываемся на события от RabbitMQ
	if err := handler.subscribeToRoundEvents(); err != nil {
		return nil, fmt.Errorf("failed to subscribe to round events: %w", err)
	}

	return handler, nil
}

// startRoundCheckLoop запускает цикл проверки раундов
func (h *GameHandler) startRoundCheckLoop() {
	log.Printf("Starting round check loop")
	var notifiedFifteen, notifiedFive bool
	var lastRoundID uint

	for {
		select {
		case <-h.stopChan:
			log.Printf("Round check loop stopped")
			return
		case <-h.checkRoundTicker.C:
			// Получаем текущий раунд из сервиса
			currentRound, err := h.service.GetCurrentRound()
			if err != nil {
				log.Printf("Error getting current round: %v", err)
				continue
			}

			if currentRound == nil {
				log.Printf("Current round is nil, skipping check")
				continue
			}

			// Сохраняем текущий раунд и состояние ожидающих игроков
			h.mutex.Lock()
			oldRound := h.currentRound

			// Определяем, изменился ли раунд
			roundChanged := oldRound == nil || oldRound.ID != currentRound.ID

			// Если раунд изменился
			if roundChanged {
				var oldRoundID uint
				if oldRound != nil {
					oldRoundID = oldRound.ID
				}
				log.Printf("Round changed: old=%v, new=%d", oldRoundID, currentRound.ID)

				// Если у нас был предыдущий раунд и есть ожидающие игроки
				if oldRound != nil && len(h.waitingPlayers) > 0 {
					// Создаем копии данных для обработки результатов
					waitingPlayersToProcess := make(map[int64]bool, len(h.waitingPlayers))
					activeBetsToProcess := make(map[int64]models.BetOption, len(h.activeBets))

					// Копируем данные из текущих карт
					for userID, waiting := range h.waitingPlayers {
						waitingPlayersToProcess[userID] = waiting
						log.Printf("Copying player %d to waiting list for results", userID)
					}

					for userID, bet := range h.activeBets {
						activeBetsToProcess[userID] = bet
						log.Printf("Copying bet %s for player %d", bet, userID)
					}

					// Обрабатываем результаты синхронно перед обновлением текущего раунда
					roundIDToProcess := oldRound.ID
					h.mutex.Unlock() // Разблокируем мьютекс перед обработкой результатов

					// Обрабатываем результаты предыдущего раунда
					h.processRoundResults(roundIDToProcess, waitingPlayersToProcess, activeBetsToProcess)

					// Снова блокируем мьютекс для обновления состояния
					h.mutex.Lock()
				}

				// Обновляем текущий раунд
				h.currentRound = currentRound
				lastRoundID = currentRound.ID

				// Сбрасываем флаги уведомлений
				notifiedFifteen = false
				notifiedFive = false

				// Очищаем карты только если раунд изменился
				h.waitingPlayers = make(map[int64]bool)
				h.activeBets = make(map[int64]models.BetOption)

				// Создаем копию для передачи в горутину, чтобы избежать race condition
				roundInfo := currentRound
				h.mutex.Unlock() // Разблокируем мьютекс перед уведомлением

				// Уведомляем активных игроков о новом раунде ПОСЛЕ обработки результатов
				h.notifyActivePlayers(roundInfo)

				// Снова блокируем мьютекс для оставшейся части цикла
				h.mutex.Lock()
			} else if lastRoundID != currentRound.ID {
				// Иногда раунд может не обновиться в h.currentRound, но изменилось lastRoundID
				log.Printf("Round changed but not detected in mutex: lastRoundID=%d, currentRound.ID=%d",
					lastRoundID, currentRound.ID)
				lastRoundID = currentRound.ID
			}
			h.mutex.Unlock()

			// Получаем время с момента создания раунда
			elapsedTime := time.Since(currentRound.CreatedAt)
			remainingTime := time.Duration(30)*time.Second - elapsedTime

			// Уведомляем игроков за 15 секунд до окончания
			if remainingTime <= 15*time.Second && remainingTime > 14*time.Second && !notifiedFifteen {
				notifiedFifteen = true
				go h.notifyTimeRemaining(currentRound, 15)
			}

			// Уведомляем игроков за 5 секунд до окончания
			if remainingTime <= 5*time.Second && remainingTime > 4*time.Second && !notifiedFive {
				notifiedFive = true
				go h.notifyTimeRemaining(currentRound, 5)
			}
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
		roundDuration := 30 * time.Second // Продолжительность раунда
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

	// Проверяем, не обрабатывали ли мы уже этот раунд (для избежания дублирования)
	h.processMutex.Lock()
	if processed, exists := h.processedRounds[roundID]; exists && processed {
		h.processMutex.Unlock()
		log.Printf("Round #%d already processed, skipping duplicate message", roundID)
		return nil
	}
	h.processMutex.Unlock()

	switch message.Type {
	case messaging.EventRoundCompleted:
		// Обрабатываем сообщение о завершении раунда
		log.Printf("Processing RabbitMQ message: round #%d completed", roundID)

		// Получаем данные о раунде
		round, err := h.service.GetHashEntryByID(roundID)
		if err != nil {
			return fmt.Errorf("failed to get round #%d: %w", roundID, err)
		}

		// Обрабатываем результаты раунда
		h.handleRoundCompletion(round)

		// Помечаем раунд как обработанный
		h.processMutex.Lock()
		h.processedRounds[roundID] = true
		h.processMutex.Unlock()

	case messaging.EventRoundStarted:
		// Обрабатываем сообщение о начале нового раунда
		log.Printf("Processing RabbitMQ message: round #%d started", roundID)

		// Получаем данные о новом раунде
		newRound, err := h.service.GetHashEntryByID(roundID)
		if err != nil {
			return fmt.Errorf("failed to get new round #%d: %w", roundID, err)
		}

		// Уведомляем активных игроков о новом раунде
		h.handleNewRound(newRound)

		// Помечаем раунд как обработанный
		h.processMutex.Lock()
		h.processedRounds[roundID] = true
		h.processMutex.Unlock()
	}

	return nil
}

// handleRoundCompletion обрабатывает завершение раунда
func (h *GameHandler) handleRoundCompletion(round *models.HashEntry) {
	log.Printf("Handling round #%d completion", round.ID)

	// Получаем игроков, ожидающих результатов
	h.mutex.Lock()
	waitingPlayersToProcess := make(map[int64]bool)
	activeBetsToProcess := make(map[int64]models.BetOption)

	// Копируем данные о ставках и ожидающих игроках
	for userID, waiting := range h.waitingPlayers {
		waitingPlayersToProcess[userID] = waiting
		log.Printf("Player %d is waiting for round #%d results", userID, round.ID)
	}

	for userID, bet := range h.activeBets {
		activeBetsToProcess[userID] = bet
		log.Printf("Player %d has bet %s in round #%d", userID, bet, round.ID)
	}

	// Очищаем карты ожидающих и ставок для нового раунда
	h.waitingPlayers = make(map[int64]bool)
	h.activeBets = make(map[int64]models.BetOption)
	h.mutex.Unlock()

	// Определяем, есть ли игроки, ожидающие результатов
	if len(waitingPlayersToProcess) == 0 {
		log.Printf("No players waiting for round #%d results", round.ID)
		return
	}

	// Получаем результат раунда
	result, err := h.service.GetRoundResult(round.ID)
	if err != nil {
		log.Printf("Error getting result for round #%d: %v", round.ID, err)
		return
	}

	log.Printf("Round #%d result: %s (number: %d)", round.ID, result, round.Number)

	// Уведомляем игроков о результатах
	for userID := range waitingPlayersToProcess {
		bet, found := activeBetsToProcess[userID]
		if !found {
			log.Printf("Warning: No bet found for user %d in the local cache", userID)

			// Пробуем получить ставку из базы данных
			userBets, err := h.service.GetUserBetsForRound(userID, round.ID)
			if err != nil {
				log.Printf("Error getting bets for user %d in round %d: %v", userID, round.ID, err)
				continue
			}

			if len(userBets) == 0 {
				log.Printf("No bets found in database for user %d in round %d", userID, round.ID)
				continue
			}

			// Используем первую найденную ставку
			log.Printf("Found bet in database for user %d: option=%s", userID, userBets[0].Option)
			bet = userBets[0].Option
		}

		// Отправляем уведомление через отдельную горутину
		go func(uid int64, betOption models.BetOption) {
			err := h.notifyPlayerAboutResult(uid, round.ID, round, result, betOption)
			if err != nil {
				log.Printf("Error notifying player %d: %v", uid, err)
			} else {
				log.Printf("Successfully notified player %d about round #%d results", uid, round.ID)

				// Публикуем событие об уведомлении пользователя
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				notification := map[string]interface{}{
					"user_id":  uid,
					"round_id": round.ID,
					"bet":      string(betOption),
					"result":   string(result),
					"won":      betOption == result,
				}

				if err := h.rabbitmq.PublishUserNotified(ctx, round.ID, notification); err != nil {
					log.Printf("Error publishing user notification: %v", err)
				}
			}
		}(userID, bet)
	}
}

// handleNewRound обрабатывает событие нового раунда
func (h *GameHandler) handleNewRound(newRound *models.HashEntry) {
	log.Printf("Handling new round #%d", newRound.ID)

	// Обновляем текущий раунд
	h.mutex.Lock()
	h.currentRound = newRound
	h.mutex.Unlock()

	// Отправляем уведомления всем активным игрокам
	h.notifyActivePlayers(newRound)
}

// HandleNewRound обрабатывает событие нового раунда (вызывается из ротатора)
func (h *GameHandler) HandleNewRound(hashEntry *models.HashEntry) {
	// Теперь обработка происходит через RabbitMQ, но оставляем этот метод для совместимости
	log.Printf("HandleNewRound called via legacy notifier interface for round #%d", hashEntry.ID)

	// Проверяем, не обрабатывали ли мы уже этот раунд
	h.processMutex.Lock()
	if processed, exists := h.processedRounds[hashEntry.ID]; exists && processed {
		h.processMutex.Unlock()
		log.Printf("Round #%d already processed, skipping duplicate notification", hashEntry.ID)
		return
	}
	h.processedRounds[hashEntry.ID] = true
	h.processMutex.Unlock()

	// Обрабатываем новый раунд
	h.handleNewRound(hashEntry)
}

// processRoundResults обрабатывает результаты раунда
func (h *GameHandler) processRoundResults(roundID uint, waitingPlayers map[int64]bool, activeBets map[int64]models.BetOption) {
	// Выводим список всех ожидающих игроков
	if len(waitingPlayers) > 0 {
		log.Printf("Waiting players in processRoundResults:")
		for playerID := range waitingPlayers {
			bet, hasBet := activeBets[playerID]
			if hasBet {
				log.Printf("  - Player %d is waiting for results with bet %s", playerID, bet)
			} else {
				log.Printf("  - Player %d is waiting for results but has no bet in activeBets map", playerID)
			}
		}
	}

	// Пытаемся завершить раунд, но не останавливаемся, если он уже завершен
	log.Printf("Calling CompleteRound for round #%d", roundID)
	err := h.service.CompleteRound(roundID)
	if err != nil {
		log.Printf("Notice: CompleteRound error for round %d: %v", roundID, err)
		// Продолжаем даже при ошибке завершения раунда - он может быть уже завершен
	}

	// Получаем данные о раунде
	log.Printf("Getting hash entry for round #%d", roundID)
	round, err := h.service.GetHashEntryByID(roundID)
	if err != nil {
		log.Printf("Error getting round %d: %v", roundID, err)
		return
	}

	// Получаем результат
	log.Printf("Getting result for round #%d", roundID)
	result, err := h.service.GetRoundResult(roundID)
	if err != nil {
		log.Printf("Error getting result for round %d: %v", roundID, err)
		return
	}

	log.Printf("Round #%d result: %s (number: %d)", roundID, result, round.Number)

	// Уведомляем игроков о результатах
	for userID := range waitingPlayers {
		log.Printf("Preparing to notify player %d about results", userID)

		bet, found := activeBets[userID]
		if !found {
			log.Printf("Warning: No bet found for user %d in the local cache", userID)

			// Попробуем получить ставку из базы данных
			userBets, err := h.service.GetUserBetsForRound(userID, roundID)
			if err != nil {
				log.Printf("Error getting bets for user %d in round %d: %v", userID, roundID, err)
				continue
			}

			if len(userBets) == 0 {
				log.Printf("No bets found in database for user %d in round %d", userID, roundID)
				continue
			}

			// Используем первую найденную ставку
			log.Printf("Found bet in database for user %d: option=%s", userID, userBets[0].Option)
			switch userBets[0].Option {
			case models.Red:
				bet = models.Red
			case models.Black:
				bet = models.Black
			case models.Zero:
				bet = models.Zero
			default:
				log.Printf("Unknown bet option: %s", userBets[0].Option)
				continue
			}
		}

		// Отправляем уведомление через отдельную горутину
		log.Printf("Launching goroutine to notify player %d with bet %s", userID, bet)
		go func(uid int64, b models.BetOption) {
			err := h.notifyPlayerAboutResult(uid, roundID, round, result, b)
			if err != nil {
				log.Printf("Error notifying player %d: %v", uid, err)
			} else {
				log.Printf("Successfully notified player %d about results", uid)
			}
		}(userID, bet)
	}
}

// notifyPlayerAboutResult уведомляет игрока о результате раунда
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

	// 1. Отправляем сообщение о результате (цвет)
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
	h.bot.SendMessage(userID, MessageOptions{
		Text: resultText,
	})

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

	// 2. Отправляем сообщение о выигрыше/проигрыше
	var winLoseText string
	if won {
		winMessageTemplate := h.service.GetText("winmessage", language)
		winLoseText = fmt.Sprintf(winMessageTemplate, points)
	} else {
		loseMsgText := h.service.GetText("losemessage", language)
		winLoseText = loseMsgText
	}

	// Создаем кнопку для проверки раунда в системе
	checkSystemText := h.service.GetText("systemcheck", language)
	roundIDBase62 := utils.ToBase62(uint(roundID))
	checkSystemURL := fmt.Sprintf("%s/hashes/?hash=%s", webPage, roundIDBase62)

	checkSystemKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf(checkSystemText, roundIDBase62), URL: checkSystemURL},
			},
		},
	}

	h.bot.SendMessage(userID, MessageOptions{
		Text:           winLoseText,
		InlineKeyboard: checkSystemKeyboard,
	})

	// 3. Отправляем сообщение о рейтинге
	// Получаем текущий рейтинг пользователя
	position, err := h.service.GetUserPosition(userID)
	if err != nil {
		log.Printf("Error getting user position: %v", err)
		position = 0
	}

	// Получаем статистику пользователя
	stats, err := h.service.GetUserStats(userID)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		stats = map[string]int{}
	}

	// Получаем информацию о призовом фонде
	year, week := time.Now().ISOWeek()

	// Получаем призовой фонд через репозиторий
	prizeFund, err := h.service.GetPrizeFund(year, week)
	if err != nil {
		log.Printf("Error getting prize fund: %v", err)
		// Значения по умолчанию
		prizeFundAmount := 1000.0
		userShare := 0.0

		if position > 0 && position <= 100 {
			// Упрощенный расчет доли пользователя
			userShare = prizeFundAmount / 100.0 * (float64(100-position+1) / 100.0)
		}

		// Формируем сообщение о рейтинге
		ratingTemplate := h.service.GetText("bidrating", language)
		ratingText := fmt.Sprintf(ratingTemplate, stats["totalPoints"], position, userShare, prizeFundAmount)

		// Создаем кнопку для просмотра рейтинга
		viewRatingText := h.service.GetText("viewrating", language)

		ratingKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: viewRatingText, CallbackData: "view_rating"},
				},
			},
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:           ratingText,
			InlineKeyboard: ratingKeyboard,
		})
	} else {
		// Получаем данные о призовом фонде из БД
		prizeFundAmount := prizeFund.Amount
		topCount := prizeFund.TopCount
		userShare := 0.0

		if position > 0 && position <= topCount {
			// Упрощенный расчет доли пользователя
			userShare = prizeFundAmount / float64(topCount) * (float64(topCount-position+1) / float64(topCount))
		}

		// Формируем сообщение о рейтинге
		ratingTemplate := h.service.GetText("bidrating", language)
		ratingText := fmt.Sprintf(ratingTemplate, stats["totalPoints"], position, userShare, prizeFundAmount)

		// Создаем кнопку для просмотра рейтинга
		viewRatingText := h.service.GetText("viewrating", language)

		ratingKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: viewRatingText, CallbackData: "view_rating"},
				},
			},
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:           ratingText,
			InlineKeyboard: ratingKeyboard,
		})
	}

	// 4. Проверяем баланс ставок и отправляем соответствующее сообщение
	betsBalance, err := h.service.GetUserRemainingBets(userID)
	if err != nil {
		log.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	if betsBalance <= 0 {
		// Недостаточно ставок
		betsBalanceLowText := h.service.GetText("betsbalancelow", language)

		topUpBalanceText := h.service.GetText("topupbalance", language)

		// Неактивная кнопка для пополнения
		betsBalanceLowKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: topUpBalanceText, CallbackData: "noop"}, // Неактивная кнопка
				},
			},
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:           betsBalanceLowText,
			InlineKeyboard: betsBalanceLowKeyboard,
		})

		// Отправляем дополнительное сообщение при недостаточном балансе
		nextBidLowText := h.service.GetText("nextbidlow", language)

		stopGameText := h.service.GetText("stopgame", language)

		nextBidLowKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: topUpBalanceText, CallbackData: "noop"}, // Неактивная кнопка
					{Text: stopGameText, CallbackData: "stop_game"},
				},
			},
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:           nextBidLowText,
			InlineKeyboard: nextBidLowKeyboard,
		})

	} else {
		// Достаточно ставок
		betsBalanceOkTemplate := h.service.GetText("betsbalanceok", language)
		betsBalanceOkText := fmt.Sprintf(betsBalanceOkTemplate, betsBalance)

		topUpBalanceText := h.service.GetText("topupbalance", language)

		// Неактивная кнопка для пополнения
		betsBalanceOkKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: topUpBalanceText, CallbackData: "noop"}, // Неактивная кнопка
				},
			},
		}

		h.bot.SendMessage(userID, MessageOptions{
			Text:           betsBalanceOkText,
			InlineKeyboard: betsBalanceOkKeyboard,
		})

		// В случае достаточного баланса возвращаем детальную клавиатуру для следующего раунда
		return nil // Сообщение о новом раунде будет отправлено в другом методе
	}

	return nil
}

func getOptionTextSimple(option models.BetOption, language string) string {
	switch option {
	case models.Red:
		return "🔴 " + (map[string]string{"uk": "Червоне", "en": "Red", "ru": "Красное"})[language]
	case models.Black:
		return "⚫ " + (map[string]string{"uk": "Чорне", "en": "Black", "ru": "Черное"})[language]
	case models.Zero:
		return "0️⃣ " + (map[string]string{"uk": "Зеро", "en": "Zero", "ru": "Зеро"})[language]
	default:
		return string(option)
	}
}

// MakeBet делает ставку в текущем раунде
func (h *GameHandler) MakeBet(userID int64, option models.BetOption) error {
	log.Printf("MakeBet called for user %d with option %s", userID, option)

	// Получаем пользователя
	_, err := h.service.GetUser(userID)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
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

	// Регистрируем пользователя как ожидающего результата и сохраняем его ставку
	h.mutex.Lock()
	h.waitingPlayers[userID] = true
	h.activeBets[userID] = option
	h.mutex.Unlock()

	log.Printf("Bet created successfully for user %d in round %d (waitingPlayers count: %d)", userID, currentRound.ID, len(h.waitingPlayers))

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
				{Text: rulesButtonText, URL: webPage + "/rules/"},
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
	roundDuration := 30 * time.Second // Продолжительность раунда
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
	if h.checkRoundTicker != nil {
		h.checkRoundTicker.Stop()
	}
	close(h.stopChan)

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

// getOptionText получает текстовое представление опции ставки
func (h *GameHandler) getOptionText(option models.BetOption, language string) string {
	switch option {
	case models.Red:
		return "🔴 " + (map[string]string{"uk": "Червоне", "en": "Red", "ru": "Красное"})[language]
	case models.Black:
		return "⚫ " + (map[string]string{"uk": "Чорне", "en": "Black", "ru": "Черное"})[language]
	case models.Zero:
		return "0️⃣ " + (map[string]string{"uk": "Зеро", "en": "Zero", "ru": "Зеро"})[language]
	default:
		return string(option)
	}
}
