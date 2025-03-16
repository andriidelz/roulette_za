package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"

	"github.com/mymmrac/telego"
)

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
}

// NewGameHandler создает новый обработчик игры
func NewGameHandler(bot *Bot, service service.Service) *GameHandler {
	handler := &GameHandler{
		bot:              bot,
		service:          service,
		waitingPlayers:   make(map[int64]bool),
		activeBets:       make(map[int64]models.BetOption),
		activePlayers:    make(map[int64]bool),
		stopChan:         make(chan struct{}),
		checkRoundTicker: time.NewTicker(1 * time.Second), // Проверка каждую секунду
	}

	// Запустите горутину для периодической проверки раундов
	go handler.startRoundCheckLoop()

	return handler
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

		// Получаем локализированное сообщение для нового раунда
		newRoundTemplate := h.service.GetText("new_round", language)
		newRoundText := fmt.Sprintf(newRoundTemplate, roundIDBase62, round.Hash)

		h.bot.SendMessage(userID, MessageOptions{
			Text:          newRoundText,
			ReplyKeyboard: h.createBetKeyboard(language, userID),
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

		// Получаем локализированное сообщение для времени
		var timeTemplate string
		if seconds == 15 {
			timeTemplate = h.service.GetText("nextbid15", language)
		} else {
			timeTemplate = h.service.GetText("nextbid5", language)
		}

		// Заменяем %s на идентификатор раунда в base62
		timeText := fmt.Sprintf(timeTemplate, roundIDBase62)

		h.bot.SendMessage(userID, MessageOptions{
			Text:          timeText,
			ReplyKeyboard: h.createBetKeyboard(language, userID),
		})
	}
}

// HandleNewRound обрабатывает событие нового раунда (вызывается из ротатора)
func (h *GameHandler) HandleNewRound(hashEntry *models.HashEntry) {

	// Создадим локальные копии данных перед изменением
	var previousRound *models.HashEntry
	var waitingPlayersToProcess map[int64]bool
	var activeBetsToProcess map[int64]models.BetOption

	// Выводим информацию о внутреннем состоянии до блокировки
	h.mutex.RLock()
	log.Printf("Current state before locking: currentRound=%v, waitingPlayers=%d, activeBets=%d",
		h.currentRound, len(h.waitingPlayers), len(h.activeBets))

	// Выводим список всех ожидающих игроков
	if len(h.waitingPlayers) > 0 {
		log.Printf("Waiting players before locking:")
		for playerID := range h.waitingPlayers {
			log.Printf("  - Player %d is waiting for results", playerID)
		}
	}
	h.mutex.RUnlock()

	// Блокируем доступ к shared state, чтобы безопасно скопировать и изменить данные
	h.mutex.Lock()

	// Сохраняем старый раунд
	previousRound = h.currentRound

	// Копируем данные о ставках для последующей обработки, но только если есть ожидающие игроки
	// и предыдущий раунд существует
	if len(h.waitingPlayers) > 0 && previousRound != nil {
		waitingPlayersToProcess = make(map[int64]bool, len(h.waitingPlayers))
		activeBetsToProcess = make(map[int64]models.BetOption, len(h.activeBets))

		log.Printf("Copying %d waiting players for processing", len(h.waitingPlayers))

		for userID, waiting := range h.waitingPlayers {
			waitingPlayersToProcess[userID] = waiting
			log.Printf("  - Copying player %d to waiting list for results", userID)
		}

		for userID, bet := range h.activeBets {
			activeBetsToProcess[userID] = bet
			log.Printf("  - Copying bet %s for player %d", bet, userID)
		}
	} else {
		log.Printf("No waiting players (%d) or previous round is nil (isNil=%v)",
			len(h.waitingPlayers), previousRound == nil)
	}

	// Очищаем карты для нового раунда
	h.waitingPlayers = make(map[int64]bool)
	h.activeBets = make(map[int64]models.BetOption)

	h.mutex.Unlock()

	// Логируем что получилось после блокировки
	log.Printf("After unlock: previousRound=%v, waitingPlayersToProcess=%d",
		previousRound != nil, len(waitingPlayersToProcess))

	// Обрабатываем результаты предыдущего раунда ПЕРЕД обновлением текущего раунда
	if previousRound != nil && waitingPlayersToProcess != nil && len(waitingPlayersToProcess) > 0 {
		log.Printf("Calling processRoundResults for previous round #%d with %d waiting players",
			previousRound.ID, len(waitingPlayersToProcess))

		// Обрабатываем результаты синхронно
		h.processRoundResults(previousRound.ID, waitingPlayersToProcess, activeBetsToProcess)
	} else {
		log.Printf("Skipping processing results: previousRound=%v, waitingPlayersToProcess=%v",
			previousRound != nil, waitingPlayersToProcess != nil && len(waitingPlayersToProcess) > 0)
	}

	// Теперь, после обработки результатов, обновляем текущий раунд
	h.mutex.Lock()
	h.currentRound = hashEntry
	h.mutex.Unlock()

	log.Printf("New round #%s started with hash %s", utils.ToBase62(hashEntry.ID), hashEntry.Hash)

	// Задержка перед уведомлением о новом раунде
	time.Sleep(1 * time.Second)

	// Уведомляем о новом раунде ПОСЛЕ обработки результатов
	h.notifyActivePlayers(hashEntry)
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
func (h *GameHandler) notifyPlayerAboutResult(userID int64, roundID uint, round *models.HashEntry, result models.BetOption, userBet models.BetOption) error {
	log.Printf("notifyPlayerAboutResult called for user %d, round #%d", userID, roundID)

	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
		return fmt.Errorf("error getting user: %w", err)
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для результата
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

	// Формируем более подробное сообщение о результате
	resultDetailsTemplate := "%s\nResult: %d (%s)"
	resultColorText := ""
	switch result {
	case models.Red:
		resultColorText = h.service.GetText("btn_bet_red", language)
	case models.Black:
		resultColorText = h.service.GetText("btn_bet_black", language)
	case models.Zero:
		resultColorText = h.service.GetText("btn_bet_zero", language)
	}

	resultDetails := fmt.Sprintf(resultDetailsTemplate, resultText, round.Number, resultColorText)

	// Добавляем информацию для проверки результата
	verificationTemplate := h.service.GetText("verification_info", language)
	roundIDBase62 := utils.ToBase62(uint(roundID))
	verificationText := fmt.Sprintf(verificationTemplate, roundIDBase62, round.Number, round.SaltHEX, round.Hash)

	// Готовим информацию о выигрыше/проигрыше
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

	var betResultText string
	if won {
		if result == models.Zero {
			winZeroTemplate := h.service.GetText("win_zero", language)
			betResultText = fmt.Sprintf(winZeroTemplate, points)
		} else {
			winTemplate := h.service.GetText("win", language)
			betResultText = fmt.Sprintf(winTemplate, getOptionText(userBet, language), points)
		}
	} else {
		loseTemplate := h.service.GetText("lose", language)
		betResultText = fmt.Sprintf(loseTemplate, getOptionText(userBet, language), getOptionText(result, language))
	}

	// Отправляем результат игры с полной информацией
	// Используем обычные \n между блоками - обработка будет в SendMessage
	fullResultText := resultDetails + "\n\n" + verificationText + "\n\n" + betResultText

	_, err = h.bot.SendMessage(userID, MessageOptions{
		Text:          fullResultText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})

	return err
}

// MakeBet делает ставку в текущем раунде
func (h *GameHandler) MakeBet(userID int64, betOption models.BetOption) error {
	log.Printf("MakeBet called for user %d with option %s", userID, betOption)

	// Получаем текущий раунд
	currentRound, err := h.service.GetCurrentRound()
	if err != nil {
		log.Printf("Error getting current round: %v", err)
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
	if betOption == models.Zero {
		canBetZero, _, err := h.service.CanBetZero(userID)
		if err != nil {
			log.Printf("Error checking zero bet: %v", err)
			return fmt.Errorf("error checking zero bet: %w", err)
		}

		if !canBetZero {
			return fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Делаем ставку через сервис
	if err := h.service.MakeBet(userID, betOption); err != nil {
		log.Printf("Error making bet: %v", err)
		return fmt.Errorf("error making bet: %w", err)
	}

	// Регистрируем пользователя как ожидающего результата и сохраняем его ставку
	h.mutex.Lock()
	h.waitingPlayers[userID] = true
	h.activeBets[userID] = betOption
	h.mutex.Unlock()

	log.Printf("Bet created successfully for user %d in round %d (waitingPlayers count: %d)", userID, currentRound.ID, len(h.waitingPlayers))

	// ВАЖНО: Эта функция НЕ отправляет сообщение о принятии ставки,
	// сообщение отправляется в методе handleMakeBet класса Bot

	return nil
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Добавляем пользователя в список активных игроков
	h.mutex.Lock()
	h.activePlayers[user.ID] = true
	h.mutex.Unlock()

	log.Printf("Added user %d to active players", user.ID)

	// Пытаемся получить текущий раунд, если он еще не установлен
	if h.currentRound == nil {
		currentRound, err := h.service.GetCurrentRound()
		if err == nil {
			h.mutex.Lock()
			h.currentRound = currentRound
			h.mutex.Unlock()
		}
	}

	h.mutex.RLock()
	currentRound := h.currentRound
	h.mutex.RUnlock()

	// Получаем локализированный текст инструкций
	instructionsText := h.service.GetText("game_instructions", language)

	// Добавляем информацию о текущем раунде, если он есть
	if currentRound != nil {
		roundInfoTemplate := h.service.GetText("round_info", language)
		roundIDBase62 := utils.ToBase62(uint(currentRound.ID))
		roundInfo := fmt.Sprintf("\n\n%s", fmt.Sprintf(roundInfoTemplate, roundIDBase62, currentRound.Hash))
		instructionsText += roundInfo
	} else {
		instructionsText += "\n\nОжидаем начала игры. Пожалуйста, подождите немного."
	}

	// Отправляем сообщение с инструкциями и клавиатурой для ставок
	h.bot.SendMessage(message.Chat.ID, MessageOptions{
		Text:          instructionsText,
		ReplyKeyboard: h.createBetKeyboard(language, user.ID),
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

// Stop останавливает обработчик игры
func (h *GameHandler) Stop() {
	if h.checkRoundTicker != nil {
		h.checkRoundTicker.Stop()
	}
	close(h.stopChan)

	log.Println("Game handler stopped")
}

// createBetKeyboard создает клавиатуру для ставок
func (h *GameHandler) createBetKeyboard(language string, userID int64) *telego.ReplyKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	btnRedText := h.service.GetText("btn_bet_red", language)
	btnBlackText := h.service.GetText("btn_bet_black", language)
	btnZeroText := h.service.GetText("btn_bet_zero", language)
	btnZeroLockedText := h.service.GetText("btn_bet_zero_locked", language)
	btnBackText := h.service.GetText("btn_back", language)

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
			},
			{
				zeroButton,
			},
			{
				{Text: btnBackText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       true,
	}
}

// Вспомогательная функция для получения текстового представления опции ставки
func getOptionText(option models.BetOption, language string) string {
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
