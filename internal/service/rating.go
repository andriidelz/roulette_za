package service

import (
	"fmt"
	"roulette/internal/data"
	"roulette/internal/logger"
	"roulette/internal/models"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetWeeklyTopRating получает текущий недельный рейтинг (топ игроков)
func (s *ServiceImpl) GetWeeklyTopRating(limit int) ([]models.WeeklyRating, error) {
	// Обновляем все рейтинги для актуальности данных
	// TODO: добавление кэширования результатов с кратковременным TTL (например, 30 секунд)

	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()
	if err := s.repo.RefreshWeeklyRatingsPosition(year, week); err != nil {
		return nil, err
	}

	// Получаем текущий недельный рейтинг
	return s.repo.GetWeeklyRating(year, week, limit)
}

// GetUserRatingPosition получает текущую позицию пользователя в рейтинге
// и его соседей (для отображения в интерфейсе)
func (s *ServiceImpl) GetUserRatingPosition(userID uint, neighborsCount int) ([]models.WeeklyRating, int, error) {

	// Обновляем рейтинг пользователя и пересчитываем позиции всех в рейтинге
	if err := s.repo.UpdateWeeklyRatingForUser(userID); err != nil {
		return nil, 0, err
	}

	if err := s.RefreshAllRatings(); err != nil {
		return nil, 0, err
	}

	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	// Получаем рейтинг пользователя и его соседей
	neighbors, position, err := s.repo.GetUserRankAndNeighbors(userID, year, week, neighborsCount)
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
func (s *ServiceImpl) GetPointsNeededForUser(userID uint) (int, error) {

	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	// Получаем текущие баллы пользователя
	userRating, err := s.repo.GetUserWeeklyRating(userID, year, week)
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
	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()
	return s.repo.RefreshWeeklyRatingsPosition(year, week)
}

// FormatRatingForDisplay форматирует рейтинг для отображения
// с анонимизацией всех пользователей, кроме указанного
func (s *ServiceImpl) FormatRatingForDisplay(ratings []models.WeeklyRating, currentUserID int64) []string {
	// Форматируем каждую запись рейтинга
	var formattedRatings []string

	for _, rating := range ratings {
		var displayName string

		// Получаем эмодзи флага страны
		var countryFlag string
		if country := data.GetCountryByCode(rating.User.Country); country != nil {
			countryFlag = country.Emoji
		}

		// Проверяем, является ли запись текущим пользователем
		if rating.User.TelegramID == currentUserID {
			// Для текущего пользователя используем nickname или другое доступное имя
			if rating.User.Nickname != "" {
				displayName = rating.User.Nickname
			} else if rating.User.Username != "" {
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

			// Добавляем флаг страны к имени пользователя, если он есть
			if countryFlag != "" {
				displayName = countryFlag + " " + displayName
			}

			// Добавляем звездочки для выделения текущего пользователя
			displayName = "*" + displayName + "*"
		} else {
			// Для остальных пользователей используем nickname или анонимное имя
			if rating.User.Nickname != "" {
				displayName = rating.User.Nickname
			} else {
				// Для пользователей без никнейма используем анонимное имя
				displayName = fmt.Sprintf("Игрок %d", rating.Position)
			}

			// Добавляем флаг страны к имени пользователя, если он есть
			if countryFlag != "" {
				displayName = countryFlag + " " + displayName
			}
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

	// Получаем эмодзи флага страны используя существующую функцию из пакета data
	var countryFlag string
	if country := data.GetCountryByCode(rating.User.Country); country != nil {
		countryFlag = country.Emoji
	}

	// Выбираем соответствующий шаблон в зависимости от типа игрока и наличия эффективности
	var templateKey string
	var args []interface{}

	// Используем nickname вместо username для отображения, если он задан
	var displayName string
	if rating.User.Nickname != "" {
		displayName = rating.User.Nickname
	} else if rating.User.Username != "" {
		displayName = rating.User.Username
	} else if rating.User.FirstName != "" {
		displayName = rating.User.FirstName
		if rating.User.LastName != "" {
			displayName += " " + rating.User.LastName
		}
	} else {
		displayName = fmt.Sprintf("Player%d", rating.User.TelegramID)
	}

	// Добавляем флаг страны к имени пользователя, если он есть
	if countryFlag != "" {
		displayName = countryFlag + " " + displayName
	}

	if isCurrentUser {
		// Для текущего пользователя
		if rating.Bets > 0 {
			templateKey = "username_points_efficiency"
			args = []interface{}{displayName, rating.Points, rating.Efficiency * 100}
		} else {
			templateKey = "username_points"
			args = []interface{}{displayName, rating.Points}
		}
	} else {
		// Для остальных игроков
		if rating.Bets > 0 {
			templateKey = "player_points_efficiency"
			args = []interface{}{position, displayName, rating.Points, rating.Efficiency * 100}
		} else {
			templateKey = "player_points"
			args = []interface{}{position, displayName, rating.Points}
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
		// Сортируем по позиции
		if ratings[i].Position != ratings[j].Position {
			return ratings[i].Position < ratings[j].Position
		}
		// Если баллы разные, сортируем по убыванию баллов
		if ratings[i].Points != ratings[j].Points {
			return ratings[i].Points > ratings[j].Points
		}
		// Если баллы одинаковые, сортируем по убыванию эффективности
		if ratings[i].Efficiency != ratings[j].Efficiency {
			return ratings[i].Efficiency > ratings[j].Efficiency
		}
		return ratings[i].UserID < ratings[j].UserID
	})

	// Форматируем каждую строку и объединяем их
	var lines []string
	for _, rating := range ratings {
		line := s.FormatPlayerLine(rating, rating.Position, currentUserID, language)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// CreateNewPrizeFund создает новый призовой фонд на основе текущих настроек
func (s *ServiceImpl) CreateNewPrizeFund(year, week int) error {
	// Получаем настройки
	settings, err := s.GetSettings()
	if err != nil {
		return fmt.Errorf("error getting settings: %w", err)
	}

	// Получаем значения для призового фонда из настроек
	prizeAmount := 1000.0 // Значение по умолчанию
	if amountStr, ok := settings["weekly_prize_amount"]; ok && amountStr != "" {
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err == nil {
			prizeAmount = amount
		}
	}

	topCount := 100 // Значение по умолчанию
	if countStr, ok := settings["weekly_prize_top"]; ok && countStr != "" {
		count, err := strconv.Atoi(countStr)
		if err == nil {
			topCount = count
		}
	}

	// Создаем призовой фонд
	prizeFund := &models.PrizeFund{
		Year:      year,
		Week:      week,
		Amount:    prizeAmount,
		TopCount:  topCount,
		Processed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Сохраняем призовой фонд, используя существующий метод
	if err := s.repo.UpdatePrizeFund(prizeFund); err != nil {
		return fmt.Errorf("error creating prize fund: %w", err)
	}

	logger.Info.Printf("Created new weekly rating for %d/%d with prize amount %.2f and top count %d",
		year, week, prizeAmount, topCount)

	return nil
}

// UpdateCurrentPrizeFund обновляет призовой фонд для текущей недели
// на основе переданных значений
func (s *ServiceImpl) UpdateCurrentPrizeFund(amount float64, topCount int) error {
	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()

	// Получаем текущий призовой фонд
	prizeFund, err := s.repo.GetPrizeFund(year, week)
	if err != nil {
		return fmt.Errorf("ошибка получения призового фонда: %w", err)
	}

	// Проверяем, не распределен ли уже фонд
	if prizeFund.Processed {
		return fmt.Errorf("призовой фонд для недели %d/%d уже распределен", year, week)
	}

	// Обновляем значения
	prizeFund.Amount = amount
	prizeFund.TopCount = topCount

	// Сохраняем изменения
	if err := s.repo.UpdatePrizeFund(prizeFund); err != nil {
		return fmt.Errorf("ошибка обновления призового фонда: %w", err)
	}

	logger.Info.Printf("Обновлен призовой фонд %d/%d: сумма = %.2f, топ = %d",
		year, week, amount, topCount)

	return nil
}

func (s *ServiceImpl) CancelPrizeDistribution(year, week int) error {
	return s.repo.CancelPrizeDistribution(year, week)
}
