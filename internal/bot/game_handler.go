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
	var notifiedFifteen, notifiedFive bool

	for {
		select {
		case <-h.stopChan:
			return
		case <-h.checkRoundTicker.C:
			currentRound, err := h.service.GetCurrentRound()
			if err != nil {
				log.Printf("Error getting current round: %v", err)
				continue
			}

			h.mutex.Lock()
			oldRound := h.currentRound
			h.currentRound = currentRound
			h.mutex.Unlock()

			// Если раунд изменился, логируем это и сбрасываем флаги уведомлений
			if oldRound == nil || oldRound.ID != currentRound.ID {
				log.Printf("Bot detected new round #%d with hash %s", currentRound.ID, currentRound.Hash)
				notifiedFifteen = false
				notifiedFive = false

				// Уведомляем активных игроков о новом раунде
				go h.notifyActivePlayers(currentRound)
			}

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

		// Получаем локализированное сообщение для времени
		var timeTemplate string
		if seconds == 15 {
			timeTemplate = h.service.GetText("nextbid15", language)
		} else {
			timeTemplate = h.service.GetText("nextbid5", language)
		}

		// Заменяем %d на идентификатор раунда в base62
		timeText := fmt.Sprintf(timeTemplate, roundIDBase62)

		h.bot.SendMessage(userID, MessageOptions{
			Text:          timeText,
			ReplyKeyboard: h.createBetKeyboard(language, userID),
		})
	}
}

// notifyPlayerAboutResult уведомляет игрока о результате раунда
func (h *GameHandler) notifyPlayerAboutResult(userID int64, roundID uint, round *models.HashEntry, result models.BetOption, userBet models.BetOption) {
	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
		return
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

	// Добавляем информацию для проверки результата
	verificationTemplate := h.service.GetText("verification_info", language)
	roundIDBase62 := utils.ToBase62(uint(roundID))
	verificationText := fmt.Sprintf(verificationTemplate, roundIDBase62, round.Number, round.SaltHEX, round.Hash)

	// Готовим информацию о выигрыше/проигрыше
	userBets, err := h.service.GetUserBetsForRound(userID, roundID)
	if err != nil || len(userBets) == 0 {
		log.Printf("Error getting bets for user %d in round %d: %v", userID, roundID, err)
		return
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

	// Отправляем результат игры
	h.bot.SendMessage(userID, MessageOptions{
		Text:          resultText + "\n\n" + verificationText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})

	// Небольшая задержка между сообщениями
	time.Sleep(500 * time.Millisecond)

	// Теперь отправляем результат ставки
	h.bot.SendMessage(userID, MessageOptions{
		Text:          betResultText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})
}

// HandleNewRound обрабатывает событие нового раунда (вызывается из ротатора)
func (h *GameHandler) HandleNewRound(hashEntry *models.HashEntry) {
	h.mutex.Lock()

	// Сохраняем предыдущий раунд
	previousRound := h.currentRound

	// Обновляем текущий раунд
	h.currentRound = hashEntry

	// Копируем карты ожидающих игроков и ставок
	waitingPlayers := make(map[int64]bool)
	activeBets := make(map[int64]models.BetOption)

	for userID, waiting := range h.waitingPlayers {
		if waiting {
			waitingPlayers[userID] = true
		}
	}

	for userID, bet := range h.activeBets {
		activeBets[userID] = bet
	}

	// Очищаем карты
	h.waitingPlayers = make(map[int64]bool)
	h.activeBets = make(map[int64]models.BetOption)

	h.mutex.Unlock()

	// Если был предыдущий раунд, обрабатываем его результаты
	if previousRound != nil && len(waitingPlayers) > 0 {
		go h.processRoundResults(previousRound.ID, waitingPlayers, activeBets)
	}

	log.Printf("New round #%d started with hash %s", hashEntry.ID, hashEntry.Hash)
}

// processRoundResults обрабатывает результаты раунда
func (h *GameHandler) processRoundResults(roundID uint, waitingPlayers map[int64]bool, activeBets map[int64]models.BetOption) {
	// Завершаем раунд и обрабатываем ставки
	if err := h.service.CompleteRound(roundID); err != nil {
		log.Printf("Error completing round %d: %v", roundID, err)
		return
	}

	// Получаем данные о раунде
	round, err := h.service.GetHashEntryByID(roundID)
	if err != nil {
		log.Printf("Error getting round %d: %v", roundID, err)
		return
	}

	// Получаем результат
	result, err := h.service.GetRoundResult(roundID)
	if err != nil {
		log.Printf("Error getting result for round %d: %v", roundID, err)
		return
	}

	// Уведомляем игроков о результатах
	for userID := range waitingPlayers {
		go h.notifyPlayerAboutResult(userID, roundID, round, result, activeBets[userID])
	}
}

// MakeBet делает ставку в текущем раунде
func (h *GameHandler) MakeBet(userID int64, betOption models.BetOption) {
	h.mutex.RLock()
	currentRound := h.currentRound
	h.mutex.RUnlock()

	// Если текущего раунда нет, пробуем получить его
	if currentRound == nil {
		var err error
		currentRound, err = h.service.GetCurrentRound()
		if err != nil || currentRound == nil {
			user, err := h.service.GetUser(userID)
			if err != nil {
				log.Printf("Error getting user %d: %v", userID, err)
				return
			}

			language := user.LanguageCode
			if language == "" {
				language = "en"
			}

			errorText := h.service.GetText("error", language)
			h.bot.SendMessage(userID, MessageOptions{
				Text: errorText + " Рулетка еще не запущена. Пожалуйста, подождите.",
			})
			return
		}

		// Обновим текущий раунд
		h.mutex.Lock()
		h.currentRound = currentRound
		h.mutex.Unlock()
	}

	// Проверяем, может ли пользователь делать ставку на Zero
	if betOption == models.Zero {
		canBetZero, remaining, err := h.service.CanBetZero(userID)
		if err != nil {
			log.Printf("Error checking zero bet: %v", err)
			h.sendErrorMessage(userID, "bet_error")
			return
		}

		if !canBetZero {
			user, err := h.service.GetUser(userID)
			if err != nil {
				log.Printf("Error getting user %d: %v", userID, err)
				return
			}

			language := user.LanguageCode
			if language == "" {
				language = "en"
			}

			zeroLimitText := h.service.GetText("zero_limit", language)
			zeroLimitText = fmt.Sprintf(zeroLimitText, remaining)

			h.bot.SendMessage(userID, MessageOptions{
				Text:          zeroLimitText,
				ReplyKeyboard: h.createBetKeyboard(language, userID),
			})
			return
		}
	}

	// Делаем ставку
	err := h.service.MakeBet(userID, betOption)
	if err != nil {
		log.Printf("Error making bet for user %d: %v", userID, err)
		h.sendErrorMessage(userID, "bet_error")
		return
	}

	// Регистрируем пользователя как ожидающего результата и сохраняем его ставку
	h.mutex.Lock()
	h.waitingPlayers[userID] = true
	h.activeBets[userID] = betOption
	h.mutex.Unlock()

	// Отправляем подтверждение ставки
	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
		return
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированное сообщение о принятии ставки
	nomorebidsText := h.service.GetText("nomorebids", language)

	// Отправляем сообщение о принятии ставки
	h.bot.SendMessage(userID, MessageOptions{
		Text:          nomorebidsText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})
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
}

// Stop останавливает обработчик игры
func (h *GameHandler) Stop() {
	if h.checkRoundTicker != nil {
		h.checkRoundTicker.Stop()
	}
	close(h.stopChan)
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

// sendErrorMessage отправляет сообщение об ошибке
func (h *GameHandler) sendErrorMessage(userID int64, key string) {
	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
		return
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	errorText := h.service.GetText(key, language)
	h.bot.SendMessage(userID, MessageOptions{
		Text: errorText,
	})
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
