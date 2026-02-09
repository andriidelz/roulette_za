package bot

import (
	"fmt"
	"time"

	"roulette/internal/logger"

	"github.com/mymmrac/telego"
)

// handleRatingCommand обрабатывает команду /rating
func (b *Bot) handleRatingCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Отправляем стартовое сообщение о рейтинге
	options := b.prepareMessage("ratingstart", language)
	options.ReplyKeyboard = b.createRatingKeyboard(language)

	// Отправляем сообщение с клавиатурой
	b.SendMessage(message.Chat.ID, options)
}

func (b *Bot) handleRatingCallbackQuery(query *telego.CallbackQuery) {
	user := query.From

	options, language := b.getWeeklyRating(user.ID, user.LanguageCode)

	if query.Message != nil {
		// Создаем inline клавиатуру с кнопками
		options.InlineKeyboard = &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{{Text: b.getText("next_round", language), CallbackData: CallbackStartRound}},
			},
		}

		b.SendMessage(query.Message.GetChat().ID, options)
	}
}

// getWeeklyRating используется для вывода рейтинга в меню и в игре
func (b *Bot) getWeeklyRating(telegramID int64, appLanguage string) (MessageOptions, string) {
	dbUser, err := b.getUser(telegramID)
	if err != nil {
		// Если не удалось получить пользователя, используем язык из параметра
		return b.prepareMessage("rating_error", appLanguage), appLanguage
	}

	language := getLanguage(dbUser.LanguageCode, appLanguage)

	// Получаем текущий недельный рейтинг (топ 100)
	ratings, err := b.service.GetWeeklyTopRating(100)
	if err != nil {
		logger.Error.Printf("Error getting weekly rating: %v", err)
		return b.prepareMessage("rating_error", language), language
	}

	var options MessageOptions
	// Ограничиваем количество отображаемых игроков
	maxDisplayCount := 100

	// Если рейтинг пуст
	if len(ratings) == 0 {
		options = b.prepareMessage("weekly_rating_empty", language)
	} else if len(ratings) > maxDisplayCount {
		// Ограничиваем список
		truncatedRatings := ratings
		if len(truncatedRatings) > maxDisplayCount {
			truncatedRatings = truncatedRatings[:maxDisplayCount]
		}
		formattedList := b.service.FormatRatingList(truncatedRatings, telegramID, language)

		// Получаем шаблон с форматированием для топ-игроков
		options = b.prepareMessage("weekly_rating_top", language)
		options.Text = fmt.Sprintf(
			options.Text,
			maxDisplayCount,
			formattedList,
		)
	} else {
		// Форматируем список рейтинга
		formattedList := b.service.FormatRatingList(ratings, telegramID, language)

		// Получаем шаблон для всего рейтинга
		options = b.prepareMessage("weekly_rating_all", language)
		options.Text = fmt.Sprintf(
			options.Text,
			formattedList,
		)
	}
	return options, language
}

// handleWeeklyRating обрабатывает запрос на просмотр недельного рейтинга
func (b *Bot) handleWeeklyRating(message *telego.Message) {
	user := message.From

	options, language := b.getWeeklyRating(user.ID, user.LanguageCode)
	options.ReplyKeyboard = b.createRatingKeyboard(language)

	// Отправляем сообщение с рейтингом
	b.SendMessage(message.Chat.ID, options)

	// Отправляем дополнительное сообщение через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		options := b.prepareMessage("ratingnext", language)
		options.ReplyKeyboard = b.createRatingKeyboard(language)
		b.SendMessage(message.Chat.ID, options)
	}()
}

// handlePersonalRating обрабатывает запрос на просмотр личной позиции в рейтинге
func (b *Bot) handlePersonalRating(message *telego.Message) {
	user := message.From
	dbUser, err := b.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Получаем позицию пользователя и его соседей в рейтинге
	neighbors, position, err := b.service.GetUserRatingPosition(dbUser.ID, 2)
	if err != nil {
		logger.Error.Printf("Error getting user position: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("rating_error", language))
		return
	}

	var options MessageOptions

	// Если рейтинг пуст
	if len(neighbors) == 0 {
		options = b.prepareMessage("personal_rating_empty", language)
		options.Text = fmt.Sprintf(
			options.Text,
			position,
		)
	} else {
		// Форматируем список соседей
		formattedList := b.service.FormatRatingList(neighbors, user.ID, language)

		// Получаем количество баллов, необходимое для входа в призовую зону
		pointsNeeded, err := b.service.GetPointsNeededForUser(dbUser.ID)
		if err != nil {
			logger.Error.Printf("Error getting points needed: %v", err)
			pointsNeeded = 0
		}

		if pointsNeeded > 0 {
			// Пользователь не в призовой зоне
			options = b.prepareMessage("personal_rating_need_points", language)
			options.Text = fmt.Sprintf(
				options.Text,
				position,
				formattedList,
				pointsNeeded,
			)
		} else {
			// Пользователь в призовой зоне
			options = b.prepareMessage("personal_rating_prize_zone", language)
			options.Text = fmt.Sprintf(
				options.Text,
				position,
				formattedList,
			)
		}
	}

	// Отправляем сообщение
	options.ReplyKeyboard = b.createRatingKeyboard(language)
	b.SendMessage(message.Chat.ID, options)

	// Отправляем дополнительное сообщение через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		options := b.prepareMessage("ratingnext", language)
		options.ReplyKeyboard = b.createRatingKeyboard(language)
		b.SendMessage(message.Chat.ID, options)
	}()
}

// createRatingKeyboard создает клавиатуру для выбора типа рейтинга
func (b *Bot) createRatingKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	weekRatingText := b.getText("weekrat", language)
	personalRatingText := b.getText("personalrat", language)
	exitRatingText := b.getText("exitrat", language)

	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: weekRatingText},
				{Text: personalRatingText},
			},
			{
				{Text: exitRatingText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

func (b *Bot) handleStatsCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем локализированный текст для стартового сообщения статистики
	options := b.prepareMessage("statisticsstart", language)

	// Создаем клавиатуру выбора периода статистики
	options.ReplyKeyboard = b.createStatsKeyboard(language)

	// Отправляем сообщение с клавиатурой для выбора периода
	b.SendMessage(message.Chat.ID, options)
}

// handleDayStatistics обрабатывает показ статистики за день
func (b *Bot) handleDayStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "day")
}

// handleWeekStatistics обрабатывает показ статистики за неделю
func (b *Bot) handleWeekStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "week")
}

// handleMonthStatistics обрабатывает показ статистики за месяц
func (b *Bot) handleMonthStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "month")
}

// handleAllStatistics обрабатывает показ статистики за все время
func (b *Bot) handleAllStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "all")
}

// showStatisticsForPeriod показывает статистику для выбранного периода
func (b *Bot) showStatisticsForPeriod(message *telego.Message, period string) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем подробную статистику пользователя
	detailedStats, err := b.service.GetDetailedUserStats(user.ID, period)
	if err != nil {
		logger.Error.Printf("Error getting detailed stats: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving statistics. Please try again.",
		})
		return
	}

	// Формируем ключ для текста статистики в зависимости от периода
	statsMsgKey := period + "statm"

	// Получаем шаблон для соответствующего периода
	options := b.prepareMessage(statsMsgKey, language)

	// Сформируем текст для вставки в шаблон
	totalBets := detailedStats["totalBets"]
	blackBets := detailedStats["blackBets"]
	redBets := detailedStats["redBets"]
	zeroBets := detailedStats["zeroBets"]

	wonBets := detailedStats["wonBets"]
	wonBlackBets := detailedStats["wonBlackBets"]
	wonRedBets := detailedStats["wonRedBets"]
	wonZeroBets := detailedStats["wonZeroBets"]

	lostBets := detailedStats["lostBets"]
	lostBlackBets := detailedStats["lostBlackBets"]
	lostRedBets := detailedStats["lostRedBets"]
	lostZeroBets := detailedStats["lostZeroBets"]

	totalPoints := detailedStats["totalPoints"]

	// Заполняем шаблон данными
	options.Text = fmt.Sprintf(
		options.Text,
		totalBets, blackBets, redBets, zeroBets,
		wonBets, wonBlackBets, wonRedBets, wonZeroBets,
		lostBets, lostBlackBets, lostRedBets, lostZeroBets,
		totalPoints,
	)
	options.ReplyKeyboard = b.createStatsKeyboard(language)

	// Отправляем статистику пользователю
	b.SendMessage(message.Chat.ID, options)

	// Отправляем сообщение с предложением выбрать другой период
	options = b.prepareMessage("statistics_next", language)
	options.ReplyKeyboard = b.createStatsKeyboard(language)
	b.SendMessage(message.Chat.ID, options)
}