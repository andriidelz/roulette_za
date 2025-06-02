package repository

import (
	"fmt"
	"roulette/internal/models"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// GetCurrentWeekRating получает текущий недельный рейтинг с ограничением по количеству записей
func (r *PostgresRepository) GetCurrentWeekRating(limit int) ([]models.WeeklyRating, error) {
	// Получаем текущую неделю и год
	year, week := time.Now().ISOWeek()

	return r.GetWeeklyRating(year, week, limit)
}

// GetUserRankAndNeighbors получает позицию пользователя в рейтинге и его соседей
func (r *PostgresRepository) GetUserRankAndNeighbors(userID uint, year, week int, neighborsCount int) ([]models.WeeklyRating, int, error) {
	// Получаем рейтинг пользователя
	var userRating models.WeeklyRating
	err := r.db.Where("user_id = ? AND year = ? AND week = ?", userID, year, week).First(&userRating).Error

	// Если рейтинг не найден, возвращаем ошибку
	if err == gorm.ErrRecordNotFound {
		// Создаем новый рейтинг для пользователя с нулевыми значениями
		userRating = models.WeeklyRating{
			UserID:    userID,
			Year:      year,
			Week:      week,
			Points:    0,
			Bets:      0,
			Position:  0, // Позиция пока неизвестна
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := r.db.Create(&userRating).Error; err != nil {
			return nil, 0, err
		}

		// Обновляем позицию пользователя
		position, err := r.calculateUserPosition(userID, year, week)
		if err != nil {
			return nil, 0, err
		}

		userRating.Position = position
		if err := r.db.Save(&userRating).Error; err != nil {
			return nil, 0, err
		}
	} else if err != nil {
		return nil, 0, err
	}

	// Получаем соседей пользователя в рейтинге
	position := userRating.Position

	// Вычисляем диапазон позиций соседей
	startPos := position - neighborsCount
	if startPos < 1 {
		startPos = 1
	}

	endPos := position + neighborsCount

	// ИСПРАВЛЕНИЕ: Сортируем рейтинги сначала по количеству баллов (убывание),
	// затем по эффективности (убывание)
	var ratings []models.WeeklyRating
	err = r.db.Where("year = ? AND week = ? AND position >= ? AND position <= ?",
		year, week, startPos, endPos).
		Order("points DESC, efficiency DESC"). // Сортировка по критериям рейтинга
		Preload("User").
		Find(&ratings).Error

	if err != nil {
		return nil, 0, err
	}

	return ratings, position, nil
}

// calculateUserPosition рассчитывает текущую позицию пользователя в рейтинге
func (r *PostgresRepository) calculateUserPosition(userID uint, year, week int) (int, error) {
	// Получаем баллы и ставки пользователя
	var userRating models.WeeklyRating
	err := r.db.Select("points, bets, efficiency").
		Where("user_id = ? AND year = ? AND week = ?", userID, year, week).
		First(&userRating).Error

	if err != nil {
		return 0, err
	}

	// ИСПРАВЛЕНИЕ: Уточняем логику подсчета пользователей с лучшими результатами
	var count int64
	err = r.db.Model(&models.WeeklyRating{}).
		Where("year = ? AND week = ? AND (points > ? OR (points = ? AND efficiency > ?))",
			year, week, userRating.Points, userRating.Points, userRating.Efficiency).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	// Позиция = количество пользователей с лучшими результатами + 1
	return int(count) + 1, nil
}

// UpdateWeeklyRating обновляет или создает запись еженедельного рейтинга для пользователя
func (r *PostgresRepository) UpdateWeeklyRatingForUser(userID uint) error {
	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()

	// Находим начало текущей недели (понедельник 00:00:00)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Воскресенье в Go имеет индекс 0
		weekday = 7
	}
	startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())

	// Получаем все ставки пользователя за текущую неделю
	var bets []models.Bet
	err := r.db.Where("user_id = ? AND created_at >= ?", userID, startOfWeek).Find(&bets).Error
	if err != nil {
		return err
	}

	// Рассчитываем статистику
	totalBets := len(bets)
	if totalBets == 0 {
		// Если ставок нет, создаем или обновляем запись с нулевыми значениями
		var rating models.WeeklyRating
		result := r.db.Where("user_id = ? AND year = ? AND week = ?", userID, year, week).First(&rating)

		if result.Error == gorm.ErrRecordNotFound {
			// Создаем новую запись
			rating = models.WeeklyRating{
				UserID:     userID,
				Year:       year,
				Week:       week,
				Points:     0,
				Bets:       0,
				Efficiency: 0,
				Position:   0, // Будет обновлено позже
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := r.db.Create(&rating).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			// Обновляем существующую запись
			rating.Points = 0
			rating.Bets = 0
			rating.Efficiency = 0
			rating.UpdatedAt = time.Now()
			if err := r.db.Save(&rating).Error; err != nil {
				return err
			}
		}

		return nil
	}

	// Рассчитываем очки и эффективность
	totalPoints := 0
	wonBets := 0

	for _, bet := range bets {
		if bet.Won {
			totalPoints += bet.Points
			wonBets++
		}
	}

	// Рассчитываем эффективность: очки / количество ставок
	efficiency := float64(totalPoints) / float64(totalBets)

	// Создаем или обновляем запись рейтинга
	var rating models.WeeklyRating
	result := r.db.Where("user_id = ? AND year = ? AND week = ?", userID, year, week).First(&rating)

	if result.Error == gorm.ErrRecordNotFound {
		// Создаем новую запись
		rating = models.WeeklyRating{
			UserID:     userID,
			Year:       year,
			Week:       week,
			Points:     totalPoints,
			Bets:       totalBets,
			Efficiency: efficiency,
			Position:   0, // Будет обновлено позже
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := r.db.Create(&rating).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		// Обновляем существующую запись
		rating.Points = totalPoints
		rating.Bets = totalBets
		rating.Efficiency = efficiency
		rating.UpdatedAt = time.Now()
		if err := r.db.Save(&rating).Error; err != nil {
			return err
		}
	}

	// Обновляем позицию пользователя
	position, err := r.calculateUserPosition(userID, year, week)
	if err != nil {
		return err
	}

	rating.Position = position
	return r.db.Save(&rating).Error
}

// RefreshAllWeeklyRatings обновляет позиции всех пользователей в еженедельном рейтинге
func (r *PostgresRepository) RefreshAllWeeklyRatings() error {
	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()

	// ИСПРАВЛЕНИЕ: Обновляем порядок сортировки при обновлении позиций
	positionQuery := `
        WITH ranked AS (
            SELECT id, user_id, ROW_NUMBER() OVER (
                ORDER BY points DESC, efficiency DESC
            ) AS new_position
            FROM weekly_ratings
            WHERE week = ? AND year = ?
        )
        UPDATE weekly_ratings wr
        SET position = r.new_position
        FROM ranked r
        WHERE wr.id = r.id AND wr.week = ? AND wr.year = ?
    `

	return r.db.Exec(positionQuery, week, year, week, year).Error
}

// GetPointsToReachPrizeZone возвращает количество баллов, необходимое для входа в призовую зону
func (r *PostgresRepository) GetPointsToReachPrizeZone(year, week, topCount int) (int, error) {
	var minPoints int

	err := r.db.Model(&models.WeeklyRating{}).
		Where("year = ? AND week = ?", year, week).
		Order("points DESC").
		Limit(1).
		Offset(topCount-1).
		Pluck("points", &minPoints).Error

	if err == gorm.ErrRecordNotFound {
		// Если записей меньше, чем topCount, то любое количество баллов позволит войти в призовую зону
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return minPoints, nil
}

// GetPrizeFundWithoutCreation получает призовой фонд, не создавая новый, если он отсутствует
func (r *PostgresRepository) GetPrizeFundWithoutCreation(year, week int) (*models.PrizeFund, error) {
	var fund models.PrizeFund
	err := r.db.Where("year = ? AND week = ?", year, week).First(&fund).Error
	if err != nil {
		return nil, err // Возвращаем ошибку, если фонд не найден
	}
	return &fund, nil
}

// GetRecentPrizeFunds получает список последних призовых фондов
func (r *PostgresRepository) GetRecentPrizeFunds(limit int) ([]models.PrizeFund, error) {
	var funds []models.PrizeFund

	// Получаем последние призовые фонды, сортируя по году и неделе в убывающем порядке
	err := r.db.Order("year DESC, week DESC").Limit(limit).Find(&funds).Error
	if err != nil {
		return nil, err
	}

	return funds, nil
}

// CheckIfPrizesAlreadyDistributed проверяет, были ли уже распределены призы за указанную неделю
func (r *PostgresRepository) CheckIfPrizesAlreadyDistributed(year, week int) (bool, error) {
	var count int64
	err := r.db.Model(&models.WeeklyRating{}).
		Where("year = ? AND week = ? AND prize > 0", year, week).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FixPartiallyDistributedPrizes исправляет частично распределенные призы
func (r *PostgresRepository) FixPartiallyDistributedPrizes(year, week int, action string) error {
	// Начать транзакцию
	tx := r.db.Begin()

	// Если действие - сбросить, удаляем все призы
	if action == "reset" {
		// Сбрасываем все призы для указанной недели
		if err := tx.Model(&models.WeeklyRating{}).
			Where("year = ? AND week = ?", year, week).
			Updates(map[string]interface{}{
				"prize": 0,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Сбрасываем флаг обработки призового фонда
		if err := tx.Model(&models.PrizeFund{}).
			Where("year = ? AND week = ?", year, week).
			Updates(map[string]interface{}{
				"processed": false,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else if action == "mark-processed" {
		// Помечаем призовой фонд как обработанный
		if err := tx.Model(&models.PrizeFund{}).
			Where("year = ? AND week = ?", year, week).
			Updates(map[string]interface{}{
				"processed": true,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Фиксируем транзакцию
	return tx.Commit().Error
}

// CreatePrizeFund создает новый призовой фонд
func (r *PostgresRepository) CreatePrizeFund(fund *models.PrizeFund) error {
	// Проверяем, существует ли уже призовой фонд для указанной недели
	var existingFund models.PrizeFund
	result := r.db.Where("year = ? AND week = ?", fund.Year, fund.Week).First(&existingFund)

	if result.Error == nil {
		// Фонд уже существует - обновляем его
		existingFund.Amount = fund.Amount
		existingFund.TopCount = fund.TopCount
		existingFund.UpdatedAt = time.Now()
		return r.db.Save(&existingFund).Error
	} else if result.Error != gorm.ErrRecordNotFound {
		// Произошла ошибка, отличная от "запись не найдена"
		return result.Error
	}

	// Фонд не существует - создаем новый
	return r.db.Create(fund).Error
}

// GetDefaultPrizeSettings получает настройки призового фонда по умолчанию
func (r *PostgresRepository) GetDefaultPrizeSettings() (float64, int, error) {
	// Получаем сумму по умолчанию
	defaultAmountSetting, err := r.GetSetting("default_prize_fund_amount")
	if err != nil {
		return 1000, 100, nil // Значения по умолчанию
	}

	// Получаем количество призовых мест по умолчанию
	defaultTopCountSetting, err := r.GetSetting("default_prize_top_count")
	if err != nil {
		return 1000, 100, nil // Значения по умолчанию
	}

	// Преобразуем значения в нужные типы
	defaultAmount := 1000.0
	if defaultAmountSetting.Value != "" {
		var parseErr error
		defaultAmount, parseErr = strconv.ParseFloat(defaultAmountSetting.Value, 64)
		if parseErr != nil {
			defaultAmount = 1000.0
		}
	}

	defaultTopCount := 100
	if defaultTopCountSetting.Value != "" {
		var parseErr error
		defaultTopCount, parseErr = strconv.Atoi(defaultTopCountSetting.Value)
		if parseErr != nil {
			defaultTopCount = 100
		}
	}

	return defaultAmount, defaultTopCount, nil
}

// GetAllPrizeFunds получает все призовые фонды с сортировкой и пагинацией
func (r *PostgresRepository) GetAllPrizeFunds(page, perPage int) ([]models.PrizeFund, int64, error) {
	var funds []models.PrizeFund
	var count int64

	// Получаем общее количество фондов
	if err := r.db.Model(&models.PrizeFund{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// Получаем фонды с пагинацией
	offset := (page - 1) * perPage
	if err := r.db.Order("year DESC, week DESC").
		Offset(offset).
		Limit(perPage).
		Find(&funds).Error; err != nil {
		return nil, 0, err
	}

	return funds, count, nil
}

// Реализация метода для отмены распределения призов
func (r *PostgresRepository) CancelPrizeDistribution(year, week int) error {
	// Начинаем транзакцию
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Получаем призовой фонд
	var prizeFund models.PrizeFund
	if err := tx.Where("year = ? AND week = ?", year, week).First(&prizeFund).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("error getting prize fund: %w", err)
	}

	// Проверяем, был ли призовой фонд уже распределен
	if !prizeFund.Processed {
		tx.Rollback()
		return fmt.Errorf("prize fund for week %d/%d has not been processed yet", year, week)
	}

	// Получаем пользователей, которые получили призы за эту неделю
	var ratings []models.WeeklyRating
	if err := tx.Where("year = ? AND week = ? AND prize > 0", year, week).Find(&ratings).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("error getting ratings: %w", err)
	}

	// Возвращаем средства из балансов пользователей
	for _, rating := range ratings {
		// Получаем пользователя
		var user models.User
		if err := tx.First(&user, rating.UserID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error getting user %d: %w", rating.UserID, err)
		}

		// Вычитаем приз из баланса
		user.Balance -= rating.Prize
		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error updating user balance: %w", err)
		}

		// Создаем уведомление о возврате приза
		notification := models.Notification{
			UserID:    user.ID,
			Type:      "prize_cancel",
			Message:   fmt.Sprintf("Prize of %.2f for week %d/%d has been cancelled", rating.Prize, year, week),
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&notification).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error creating notification: %w", err)
		}

		// Обнуляем приз в рейтинге
		if err := tx.Model(&models.WeeklyRating{}).
			Where("id = ?", rating.ID).
			Update("prize", 0).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error updating rating prize: %w", err)
		}
	}

	// Обновляем статус призового фонда
	if err := tx.Model(&models.PrizeFund{}).
		Where("id = ?", prizeFund.ID).
		Update("processed", false).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating prize fund status: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
