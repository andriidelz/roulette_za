package service

import (
	"fmt"
	"roulette/internal/models"
	"sort"
	"strings"
	"time"
)

// GetWeeklyTopRating получает текущий недельный рейтинг (топ игроков)
func (s *ServiceImpl) GetWeeklyTopRating(limit int) ([]models.WeeklyRating, error) {
	// Получаем текущий недельный рейтинг
	return s.repo.GetCurrentWeekRating(limit)
}

// GetUserRatingPosition получает текущую позицию пользователя в рейтинге
// и его соседей (для отображения в интерфейсе)
func (s *ServiceImpl) GetUserRatingPosition(telegramID int64, neighborsCount int) ([]models.WeeklyRating, int, error) {
	// Получаем пользователя
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, 0, err
	}

	// Обновляем рейтинг пользователя
	if err := s.repo.UpdateWeeklyRatingForUser(user.ID); err != nil {
		return nil, 0, err
	}

	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	// Получаем рейтинг пользователя и его соседей
	neighbors, position, err := s.repo.GetUserRankAndNeighbors(user.ID, year, week, neighborsCount)
	if err != nil {
		return nil, 0, err
	}

	return neighbors, position, nil
}

// GetPointsToReachPrizeZone возвращает количество баллов, необходимое для входа в призовую зону
func (s *ServiceImpl) GetPointsToReachPrizeZone() (int, error) {
	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	// Получаем призовой фонд для определения кол-ва призовых мест
	prizeFund, err := s.repo.GetPrizeFund(year, week)
	if err != nil {
		// Используем значение по умолчанию, если призовой фонд не найден
		return s.repo.GetPointsToReachPrizeZone(year, week, 100)
	}

	// Используем topCount из призового фонда
	return s.repo.GetPointsToReachPrizeZone(year, week, prizeFund.TopCount)
}

// GetPointsNeededForUser возвращает количество баллов, которое пользователю необходимо
// набрать для входа в призовую зону
func (s *ServiceImpl) GetPointsNeededForUser(telegramID int64) (int, error) {
	// Получаем пользователя
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return 0, err
	}

	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	// Получаем текущие баллы пользователя
	userRating, err := s.repo.GetUserWeeklyRating(user.ID, year, week)
	if err != nil {
		return 0, err
	}

	// Получаем призовой фонд для определения кол-ва призовых мест
	prizeFund, err := s.repo.GetPrizeFund(year, week)
	if err != nil {
		// Используем значение по умолчанию, если призовой фонд не найден
		prizeFund = &models.PrizeFund{
			TopCount: 100,
		}
	}

	// Получаем минимальное количество баллов для входа в призовую зону
	minPoints, err := s.repo.GetPointsToReachPrizeZone(year, week, prizeFund.TopCount)
	if err != nil {
		return 0, err
	}

	// Если у пользователя достаточно баллов, возвращаем 0
	if userRating.Points >= minPoints {
		return 0, nil
	}

	// Возвращаем разницу между минимальным количеством баллов и текущими баллами пользователя
	return minPoints - userRating.Points, nil
}

// RefreshAllRatings обновляет позиции всех пользователей в рейтинге
func (s *ServiceImpl) RefreshAllRatings() error {
	return s.repo.RefreshAllWeeklyRatings()
}

// FormatRatingForDisplay форматирует рейтинг для отображения
// с анонимизацией всех пользователей, кроме указанного
func (s *ServiceImpl) FormatRatingForDisplay(ratings []models.WeeklyRating, currentUserID int64) []string {
	// Форматируем каждую запись рейтинга
	var formattedRatings []string

	for _, rating := range ratings {
		var displayName string

		// Проверяем, является ли запись текущим пользователем
		if rating.User.TelegramID == currentUserID {
			// Для текущего пользователя используем его реальное имя
			if rating.User.Username != "" {
				displayName = "@" + rating.User.Username
			} else if rating.User.FirstName != "" {
				if rating.User.LastName != "" {
					displayName = fmt.Sprintf("%s %s", rating.User.FirstName, rating.User.LastName)
				} else {
					displayName = rating.User.FirstName
				}
			} else {
				displayName = fmt.Sprintf("Игрок %d", rating.Position)
			}

			// Добавляем звездочки для выделения текущего пользователя
			displayName = "*" + displayName + "*"
		} else {
			// Для остальных пользователей используем анонимное имя
			displayName = fmt.Sprintf("Игрок %d", rating.Position)
		}

		// Форматируем запись
		formattedRating := fmt.Sprintf("%s - %d баллов", displayName, rating.Points)

		// Опционально можем добавить информацию об эффективности
		if rating.Bets > 0 {
			efficiencyPercent := (rating.Efficiency * 100)
			formattedRating += fmt.Sprintf(" (%.1f%%)", efficiencyPercent)
		}

		formattedRatings = append(formattedRatings, formattedRating)
	}

	return formattedRatings
}

// GetPrizeDistributionStatus возвращает статус распределения призов для указанной недели
func (s *ServiceImpl) GetPrizeDistributionStatus(year, week int) (string, error) {
	// Проверяем наличие призового фонда
	prizeFund, err := s.repo.GetPrizeFundWithoutCreation(year, week)
	if err != nil {
		return "NOT_CONFIGURED", nil // Фонд не настроен
	}

	// Проверяем, был ли обработан фонд
	if prizeFund.Processed {
		return "DISTRIBUTED", nil // Призы распределены (по флагу в фонде)
	}

	// Проверяем, есть ли записи с ненулевыми призами
	alreadyDistributed, err := s.repo.CheckIfPrizesAlreadyDistributed(year, week)
	if err != nil {
		return "", err
	}

	if alreadyDistributed {
		return "PARTIALLY_DISTRIBUTED", nil // Частично распределены (несоответствие между флагом и фактическими призами)
	}

	return "PENDING", nil // Ожидается распределение
}

// FormatPlayerLine форматирует одну строку с игроком для рейтинга
func (s *ServiceImpl) FormatPlayerLine(rating models.WeeklyRating, position int, currentUserID int64, language string) string {
	// Проверяем, является ли игрок текущим пользователем
	isCurrentUser := rating.User.TelegramID == currentUserID

	// Выбираем соответствующий шаблон в зависимости от типа игрока и наличия эффективности
	var templateKey string
	var args []interface{}

	if isCurrentUser {
		// Для текущего пользователя
		if rating.User.Username != "" {
			// Если есть имя пользователя
			if rating.Bets > 0 {
				templateKey = "username_points_efficiency"
				args = []interface{}{"@" + rating.User.Username, rating.Points, rating.Efficiency * 100}
			} else {
				templateKey = "username_points"
				args = []interface{}{"@" + rating.User.Username, rating.Points}
			}
		} else if rating.User.FirstName != "" {
			// Если есть имя
			displayName := rating.User.FirstName
			if rating.User.LastName != "" {
				displayName += " " + rating.User.LastName
			}

			if rating.Bets > 0 {
				templateKey = "username_points_efficiency"
				args = []interface{}{displayName, rating.Points, rating.Efficiency * 100}
			} else {
				templateKey = "username_points"
				args = []interface{}{displayName, rating.Points}
			}
		} else {
			// Если нет имени, используем "Вы"
			if rating.Bets > 0 {
				templateKey = "you_points_efficiency"
				args = []interface{}{rating.Points, rating.Efficiency * 100}
			} else {
				templateKey = "you_points"
				args = []interface{}{rating.Points}
			}
		}
	} else {
		// Для остальных игроков
		if rating.Bets > 0 {
			templateKey = "player_points_efficiency"
			args = []interface{}{position, rating.Points, rating.Efficiency * 100}
		} else {
			templateKey = "player_points"
			args = []interface{}{position, rating.Points}
		}
	}

	// Получаем шаблон и форматируем строку
	template := s.GetText(templateKey, language)
	return fmt.Sprintf(template, args...)
}

// FormatRatingList форматирует весь список рейтинга с правильной нумерацией
func (s *ServiceImpl) FormatRatingList(ratings []models.WeeklyRating, currentUserID int64, language string) string {
	if len(ratings) == 0 {
		return ""
	}

	// Сначала сортируем рейтинг
	sort.Slice(ratings, func(i, j int) bool {
		// Если баллы разные, сортируем по убыванию баллов
		if ratings[i].Points != ratings[j].Points {
			return ratings[i].Points > ratings[j].Points
		}
		// Если баллы одинаковые, сортируем по убыванию эффективности
		return ratings[i].Efficiency > ratings[j].Efficiency
	})

	// Форматируем каждую строку и объединяем их
	var lines []string
	for i, rating := range ratings {
		position := i + 1 // Позиция начинается с 1
		line := s.FormatPlayerLine(rating, position, currentUserID, language)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
