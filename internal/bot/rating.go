package bot

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/mymmrac/telego"
)

// handleRatingCommand обрабатывает команду /rating
func (b *Bot) handleRatingCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отправляем стартовое сообщение о рейтинге
	ratingStartText := b.service.GetText("ratingstart", language)

	// Создаем клавиатуру для выбора типа рейтинга
	ratingKeyboard := b.createRatingKeyboard(language)

	// Отправляем сообщение с клавиатурой
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          ratingStartText,
		ReplyKeyboard: ratingKeyboard,
	})
}

// handleWeeklyRating обрабатывает запрос на просмотр недельного рейтинга
func (b *Bot) handleWeeklyRating(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем текущий недельный рейтинг (топ 100)
	ratings, err := b.service.GetWeeklyTopRating(100)
	if err != nil {
		log.Printf("Error getting weekly rating: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: b.service.GetText("rating_error", language),
		})
		return
	}

	var templateKey string
	var resultText string

	// Если рейтинг пуст
	if len(ratings) == 0 {
		templateKey = "weekly_rating_empty"
		resultText = b.service.GetText(templateKey, language)
	} else {
		// Форматируем список рейтинга
		formattedList := b.service.FormatRatingList(ratings, user.ID, language)

		// Ограничиваем количество отображаемых игроков
		maxDisplayCount := 20
		if len(ratings) > maxDisplayCount {
			// Ограничиваем список
			truncatedRatings := ratings
			if len(truncatedRatings) > maxDisplayCount {
				truncatedRatings = truncatedRatings[:maxDisplayCount]
			}
			formattedList = b.service.FormatRatingList(truncatedRatings, user.ID, language)

			// Получаем шаблон с форматированием для топ-игроков
			templateKey = "weekly_rating_top"
			resultText = fmt.Sprintf(
				b.service.GetText(templateKey, language),
				maxDisplayCount,
				formattedList,
			)
		} else {
			// Получаем шаблон для всего рейтинга
			templateKey = "weekly_rating_all"
			resultText = fmt.Sprintf(
				b.service.GetText(templateKey, language),
				formattedList,
			)
		}
	}

	// Отправляем сообщение с рейтингом
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          resultText,
		ReplyKeyboard: b.createRatingKeyboard(language),
	})

	// Отправляем дополнительное сообщение через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		ratingNextText := b.service.GetText("ratingnext", language)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          ratingNextText,
			ReplyKeyboard: b.createRatingKeyboard(language),
		})
	}()
}

// handlePersonalRating обрабатывает запрос на просмотр личной позиции в рейтинге
func (b *Bot) handlePersonalRating(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем позицию пользователя и его соседей в рейтинге
	neighbors, position, err := b.service.GetUserRatingPosition(user.ID, 2)
	if err != nil {
		log.Printf("Error getting user position: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: b.service.GetText("rating_error", language),
		})
		return
	}

	// Сортируем соседей по баллам и эффективности
	sort.Slice(neighbors, func(i, j int) bool {
		// Если баллы разные, сортируем по убыванию баллов
		if neighbors[i].Points != neighbors[j].Points {
			return neighbors[i].Points > neighbors[j].Points
		}
		// Если баллы одинаковые, сортируем по убыванию эффективности
		return neighbors[i].Efficiency > neighbors[j].Efficiency
	})

	var templateKey string
	var resultText string

	// Если рейтинг пуст
	if len(neighbors) == 0 {
		templateKey = "personal_rating_empty"
		resultText = fmt.Sprintf(
			b.service.GetText(templateKey, language),
			position,
		)
	} else {
		// Форматируем список соседей
		formattedList := b.service.FormatRatingList(neighbors, user.ID, language)

		// Получаем количество баллов, необходимое для входа в призовую зону
		pointsNeeded, err := b.service.GetPointsNeededForUser(user.ID)
		if err != nil {
			log.Printf("Error getting points needed: %v", err)
			pointsNeeded = 0
		}

		if pointsNeeded > 0 {
			// Пользователь не в призовой зоне
			templateKey = "personal_rating_need_points"
			resultText = fmt.Sprintf(
				b.service.GetText(templateKey, language),
				position,
				formattedList,
				pointsNeeded,
			)
		} else {
			// Пользователь в призовой зоне
			templateKey = "personal_rating_prize_zone"
			resultText = fmt.Sprintf(
				b.service.GetText(templateKey, language),
				position,
				formattedList,
			)
		}
	}

	// Отправляем сообщение
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          resultText,
		ReplyKeyboard: b.createRatingKeyboard(language),
	})

	// Отправляем дополнительное сообщение через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		ratingNextText := b.service.GetText("ratingnext", language)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          ratingNextText,
			ReplyKeyboard: b.createRatingKeyboard(language),
		})
	}()
}

// createRatingKeyboard создает клавиатуру для выбора типа рейтинга
func (b *Bot) createRatingKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	weekRatingText := b.service.GetText("weekrat", language)
	personalRatingText := b.service.GetText("personalrat", language)
	exitRatingText := b.service.GetText("exitrat", language)

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
