package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"roulette/internal/models"
	"roulette/internal/service"

	"github.com/mymmrac/telego"
)

// GameHandler управляет игровым процессом рулетки
type GameHandler struct {
	bot              *Bot
	service          service.Service
	currentHashEntry *models.HashEntry
	mutex            sync.RWMutex
	waitingPlayers   map[int64]bool             // Карта игроков, ожидающих результатов
	activeBets       map[int64]models.BetOption // Карта активных ставок игроков
	lastCheckedID    uint                       // ID последнего проверенного хеша
	stopChan         chan struct{}              // Канал для остановки проверки хешей
}

// NewGameHandler создает новый обработчик игры
func NewGameHandler(bot *Bot, service service.Service) *GameHandler {
	handler := &GameHandler{
		bot:            bot,
		service:        service,
		waitingPlayers: make(map[int64]bool),
		activeBets:     make(map[int64]models.BetOption),
		stopChan:       make(chan struct{}),
	}

	// Загружаем последний хеш при инициализации
	handler.loadLatestHash()

	// Запускаем периодическую проверку новых хешей
	go handler.startHashCheckLoop()

	return handler
}

// loadLatestHash загружает последний хеш из базы данных
func (h *GameHandler) loadLatestHash() {
	entry, err := h.service.GetLatestHashEntry()
	if err != nil {
		log.Printf("Error loading latest hash: %v", err)
		return
	}

	h.mutex.Lock()
	h.currentHashEntry = entry
	h.lastCheckedID = entry.ID
	h.mutex.Unlock()

	log.Printf("Loaded latest hash #%d", entry.ID)
}

// startHashCheckLoop запускает цикл проверки новых хешей
func (h *GameHandler) startHashCheckLoop() {
	ticker := time.NewTicker(5 * time.Second) // Проверяем каждые 5 секунд
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.checkForNewHash()
		}
	}
}

// Stop останавливает проверку хешей
func (h *GameHandler) Stop() {
	close(h.stopChan)
}

// checkForNewHash проверяет наличие нового хеша в базе данных
func (h *GameHandler) checkForNewHash() {
	// Получаем последний хеш
	entry, err := h.service.GetLatestHashEntry()
	if err != nil {
		log.Printf("Error getting latest hash: %v", err)
		return
	}

	h.mutex.RLock()
	lastID := h.lastCheckedID
	h.mutex.RUnlock()

	// Проверяем, появился ли новый хеш
	if entry.ID > lastID {
		h.mutex.Lock()

		// Сохраняем предыдущий хеш для определения результатов
		prevHashEntry := h.currentHashEntry

		// Обновляем текущий хеш
		h.currentHashEntry = entry
		h.lastCheckedID = entry.ID

		// Копируем карты ожидающих игроков и ставок для обработки
		waitingPlayers := make(map[int64]bool)
		activeBets := make(map[int64]models.BetOption)

		for userID, waiting := range h.waitingPlayers {
			if waiting {
				waitingPlayers[userID] = true
			}
		}

		for userID, betOption := range h.activeBets {
			activeBets[userID] = betOption
		}

		// Очищаем карты после копирования
		h.waitingPlayers = make(map[int64]bool)
		h.activeBets = make(map[int64]models.BetOption)
		h.mutex.Unlock()

		// Если был предыдущий хеш, определяем результаты для игроков
		if prevHashEntry != nil && len(waitingPlayers) > 0 {
			go h.processGameResults(prevHashEntry, waitingPlayers, activeBets)
		}

		// Отправляем уведомление о начале нового раунда
		go h.notifyNewRound(entry.ID)
	}
}

// HandleNewHash обрабатывает событие нового хеша (вызывается ротатором)
func (h *GameHandler) HandleNewHash(hashEntry *models.HashEntry) {
	h.mutex.Lock()

	// Сохраняем предыдущий хеш для определения результатов
	prevHashEntry := h.currentHashEntry

	// Обновляем текущий хеш
	h.currentHashEntry = hashEntry

	// Копируем карты ожидающих игроков и ставок для обработки
	waitingPlayers := make(map[int64]bool)
	activeBets := make(map[int64]models.BetOption)

	for userID, waiting := range h.waitingPlayers {
		if waiting {
			waitingPlayers[userID] = true
		}
	}

	for userID, betOption := range h.activeBets {
		activeBets[userID] = betOption
	}

	// Очищаем карты после копирования
	h.waitingPlayers = make(map[int64]bool)
	h.activeBets = make(map[int64]models.BetOption)
	h.mutex.Unlock()

	// Если был предыдущий хеш, определяем результаты для игроков
	if prevHashEntry != nil && len(waitingPlayers) > 0 {
		go h.processGameResults(prevHashEntry, waitingPlayers, activeBets)
	}

	// Отправляем уведомление о начале нового раунда
	go h.notifyNewRound(hashEntry.ID)
}

// processGameResults обрабатывает результаты игры для предыдущего хеша
func (h *GameHandler) processGameResults(hashEntry *models.HashEntry, waitingPlayers map[int64]bool, activeBets map[int64]models.BetOption) {
	// Определяем цвет по числу
	var result models.BetOption
	if hashEntry.Number == 0 {
		result = models.Zero
	} else {
		// Определяем красное или черное
		redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
		isRed := false
		for _, n := range redNumbers {
			if hashEntry.Number == n {
				isRed = true
				break
			}
		}

		if isRed {
			result = models.Red
		} else {
			result = models.Black
		}
	}

	// Создаем запись об игре в БД
	game := &models.Game{
		Result:    result,
		Hash:      hashEntry.Hash,
		CreatedAt: time.Now(),
	}
	if err := h.service.SaveGame(game); err != nil {
		log.Printf("Error saving game: %v", err)
		return
	}

	// Отправляем общее сообщение о результате всем игрокам
	// var resultMsg string
	var langKey string

	switch result {
	case models.Red:
		langKey = "redresult"
	case models.Black:
		langKey = "blackresult"
	case models.Zero:
		langKey = "zeroresult"
	}

	// Обрабатываем ставки игроков
	for userID, betOption := range activeBets {
		if waitingPlayers[userID] {
			go h.processBetResult(userID, betOption, result, game.ID, langKey)
		}
	}
}

// processBetResult обрабатывает результат ставки для конкретного игрока
func (h *GameHandler) processBetResult(userID int64, betOption, result models.BetOption, gameID uint, resultKey string) {
	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %d: %v", userID, err)
		return
	}

	won := betOption == result
	points := 0

	if won {
		if result == models.Zero {
			points = 10
		} else {
			points = 1
		}
	}

	// Создаем запись о ставке
	bet := &models.Bet{
		UserID:    user.ID,
		GameID:    gameID,
		Option:    betOption,
		Won:       won,
		Points:    points,
		CreatedAt: time.Now(),
	}

	if err := h.service.SaveBet(bet); err != nil {
		log.Printf("Error saving bet: %v", err)
	}

	// Определяем текст результата
	language := user.LanguageCode
	if language == "" {
		language = "en" // Украинский по умолчанию
	}

	// Получаем локализированный текст для результата рулетки
	resultText := h.service.GetText(resultKey, language)

	// Определяем сообщение о результате ставки
	var winStatus string
	if won {
		if result == models.Zero {
			winStatus = "win_zero"
		} else {
			winStatus = "win"
		}
	} else {
		winStatus = "lose"
	}

	// Готовим информацию о выигрыше/проигрыше
	var betResultText string

	switch winStatus {
	case "win":
		winTemplate := h.service.GetText("win", language)
		betResultText = fmt.Sprintf(winTemplate, getOptionText(betOption, language), points)
	case "win_zero":
		winZeroTemplate := h.service.GetText("win_zero", language)
		betResultText = fmt.Sprintf(winZeroTemplate, points)
	case "lose":
		loseTemplate := h.service.GetText("lose", language)
		betResultText = fmt.Sprintf(loseTemplate, getOptionText(betOption, language), getOptionText(result, language))
	}

	// Отправляем результат игры: сначала сообщение о цвете рулетки, затем результат ставки
	h.bot.SendMessage(userID, MessageOptions{
		Text: resultText,
	})

	// Небольшая задержка между сообщениями
	time.Sleep(500 * time.Millisecond)

	// Теперь отправляем результат ставки и предлагаем сделать новую
	h.bot.SendMessage(userID, MessageOptions{
		Text:          betResultText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})
}

// notifyNewRound отправляет уведомление о начале нового раунда всем активным игрокам
func (h *GameHandler) notifyNewRound(roundID uint) {
	// Здесь можно добавить логику для уведомления активных игроков о новом раунде
	// Например, если игрок недавно делал ставку, можно отправить ему сообщение о новом раунде

	log.Printf("New round #%d started", roundID)

	// Если есть предыдущие игроки, можно отправить им уведомление
	// Но тут нужно решить, как хранить список активных игроков
}

// MakeBet позволяет игроку сделать ставку
func (h *GameHandler) MakeBet(userID int64, betOption models.BetOption) {
	// Проверяем, есть ли текущий хеш (то есть, идет ли сейчас раунд)
	h.mutex.RLock()
	currentHash := h.currentHashEntry
	h.mutex.RUnlock()

	if currentHash == nil {
		// Если текущего хеша нет, значит рулетка еще не запущена
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

	// Проверяем, может ли пользователь сделать ставку на Zero
	if betOption == models.Zero {
		canBetZero, remaining, err := h.service.CanBetZero(userID)
		if err != nil {
			log.Printf("Error checking zero bet: %v", err)
			h.sendErrorMessage(userID, "bet_error")
			return
		}

		if !canBetZero {
			// Отправляем сообщение о невозможности ставки на Zero
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

	// Получаем локализированное сообщение о принятии ставки и информацию о раунде
	nomorebidsText := h.service.GetText("nomorebids", language)

	// Определяем, сколько примерно времени осталось до нового хеша
	// Это приблизительно, так как мы не знаем точно, когда был создан текущий хеш
	remainingTime := 30 - time.Since(currentHash.CreatedAt).Seconds()
	timeMsg := ""

	if remainingTime > 15 {
		nextBidText := h.service.GetText("nextbid15", language)
		timeMsg = fmt.Sprintf(nextBidText, currentHash.ID)
	} else if remainingTime > 0 {
		nextBidText := h.service.GetText("nextbid5", language)
		timeMsg = fmt.Sprintf(nextBidText, currentHash.ID)
	}

	// Отправляем сообщение о принятии ставки
	h.bot.SendMessage(userID, MessageOptions{
		Text: nomorebidsText + "\n\n" + timeMsg,
	})
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Проверяем, есть ли текущий хеш
	h.mutex.RLock()
	currentHash := h.currentHashEntry
	h.mutex.RUnlock()

	// Получаем локализированный текст инструкций
	instructionsText := h.service.GetText("game_instructions", language)

	// Если есть текущий хеш, добавляем информацию о текущем раунде
	if currentHash != nil {
		// Определяем, сколько примерно времени осталось до нового хеша
		remainingTime := 30 - time.Since(currentHash.CreatedAt).Seconds()
		if remainingTime > 0 {
			var timeMsg string
			if remainingTime > 15 {
				nextBidText := h.service.GetText("nextbid15", language)
				timeMsg = fmt.Sprintf(nextBidText, currentHash.ID)
			} else {
				nextBidText := h.service.GetText("nextbid5", language)
				timeMsg = fmt.Sprintf(nextBidText, currentHash.ID)
			}

			instructionsText += "\n\n" + timeMsg
		}
	} else {
		// Если текущего хеша нет, добавляем сообщение об ожидании
		instructionsText += "\n\nОжидаем начала игры. Пожалуйста, подождите немного."
	}

	// Отправляем сообщение с инструкциями и клавиатурой для ставок
	h.bot.SendMessage(message.Chat.ID, MessageOptions{
		Text:          instructionsText,
		ReplyKeyboard: h.createBetKeyboard(language, user.ID),
	})
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
		return "🔴 " + (map[string]string{"uk": "Червоне", "en": "Red"})[language]
	case models.Black:
		return "⚫ " + (map[string]string{"uk": "Чорне", "en": "Black"})[language]
	case models.Zero:
		return "0️⃣ " + (map[string]string{"uk": "Зеро", "en": "Zero"})[language]
	default:
		return string(option)
	}
}
