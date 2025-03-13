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
}

// NewGameHandler создает новый обработчик игры
func NewGameHandler(bot *Bot, service service.Service) *GameHandler {
	handler := &GameHandler{
		bot:            bot,
		service:        service,
		waitingPlayers: make(map[int64]bool),
		activeBets:     make(map[int64]models.BetOption),
	}

	// Получаем последний хеш при инициализации
	handler.updateCurrentHash()

	return handler
}

// updateCurrentHash обновляет текущий хеш
func (h *GameHandler) updateCurrentHash() {
	entries, _, err := h.service.GetHashEntries(1, 1)
	if err != nil || len(entries) == 0 {
		log.Printf("Error getting current hash: %v", err)
		return
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.currentHashEntry = &entries[0]
	log.Printf("Updated current hash: %v (Number: %d)", h.currentHashEntry.Hash, h.currentHashEntry.Number)
}

// HandleNewHash обрабатывает событие нового хеша (вызывается ротатором)
func (h *GameHandler) HandleNewHash(hashEntry *models.HashEntry) {
	// Сохраняем предыдущий хеш для определения результатов
	prevHashEntry := h.currentHashEntry

	// Обновляем текущий хеш
	h.mutex.Lock()
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
	if prevHashEntry != nil {
		go h.processGameResults(prevHashEntry, waitingPlayers, activeBets)
	}

	// Отправляем уведомление о начале нового раунда
	go h.notifyNewRound()
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
	}

	// Обрабатываем ставки игроков
	for userID, betOption := range activeBets {
		if waitingPlayers[userID] {
			go h.processBetResult(userID, betOption, result, game.ID)
		}
	}
}

// processBetResult обрабатывает результат ставки для конкретного игрока
func (h *GameHandler) processBetResult(userID int64, betOption, result models.BetOption, gameID uint) {
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
	resultText := ""
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

	// Получаем локализированный текст для результата
	language := user.LanguageCode
	if language == "" {
		language = "uk" // Украинский по умолчанию
	}

	// Готовим информацию о выигрыше/проигрыше
	switch winStatus {
	case "win":
		resultText = h.service.GetText("win", language)
		resultText = fmt.Sprintf(resultText, getOptionText(betOption, language), points)
	case "win_zero":
		resultText = h.service.GetText("win_zero", language)
		resultText = fmt.Sprintf(resultText, points)
	case "lose":
		resultText = h.service.GetText("lose", language)
		resultText = fmt.Sprintf(resultText, getOptionText(betOption, language), getOptionText(result, language))
	}

	// Отправляем сообщение о результате и предлагаем сделать новую ставку
	h.bot.SendMessage(userID, MessageOptions{
		Text:          resultText,
		ReplyKeyboard: h.createBetKeyboard(language, userID),
	})
}

// notifyNewRound отправляет уведомление о начале нового раунда всем активным игрокам
func (h *GameHandler) notifyNewRound() {
	// Здесь можно добавить дополнительную логику для уведомления игроков о новом раунде
	// Например, отправку сообщения всем активным пользователям
}

// MakeBet позволяет игроку сделать ставку
func (h *GameHandler) MakeBet(userID int64, betOption models.BetOption) {
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
				language = "uk"
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
		language = "uk"
	}

	// Получаем локализированное сообщение о принятии ставки
	nomorebidsText := h.service.GetText("nomorebids", language)

	// Отправляем сообщение о принятии ставки
	h.bot.SendMessage(userID, MessageOptions{
		Text: nomorebidsText,
	})
}

// HandlePlayCommand обрабатывает команду /play
func (h *GameHandler) HandlePlayCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "uk"
	}

	// Получаем локализированный текст инструкций
	instructionsText := h.service.GetText("game_instructions", language)

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
		language = "uk"
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
