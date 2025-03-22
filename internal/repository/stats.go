package repository

import (
	"roulette/internal/models"

	"gorm.io/gorm"
)

// GetDetailedStatsByDate получает детальную статистику по ставкам пользователя за указанный период
func (r *PostgresRepository) GetDetailedStatsByDate(userID uint, startDate string, endDate string) (map[string]int, error) {
	// Подготавливаем базовый SQL-запрос
	baseQuery := r.db.Table("bets").Where("user_id = ?", userID)

	// Добавляем условие по дате начала, если оно указано
	if startDate != "" {
		baseQuery = baseQuery.Where("DATE(created_at) >= ?", startDate)
	}

	// Добавляем условие по дате окончания, если оно указано
	if endDate != "" {
		baseQuery = baseQuery.Where("DATE(created_at) <= ?", endDate)
	}

	// Создаем копии запроса для разных подсчетов
	query := baseQuery.Session(&gorm.Session{})

	// Получаем общее количество ставок
	var totalBets int64
	if err := query.Count(&totalBets).Error; err != nil {
		return nil, err
	}

	// Создаем карту для хранения статистики
	stats := map[string]int{
		"totalBets": int(totalBets),
	}

	// Получаем количество ставок на черное
	var blackBets int64
	if err := query.Where("option = ?", string(models.Black)).Count(&blackBets).Error; err != nil {
		return nil, err
	}
	stats["blackBets"] = int(blackBets)

	// Получаем количество ставок на красное
	var redBets int64
	if err := query.Where("option = ?", string(models.Red)).Count(&redBets).Error; err != nil {
		return nil, err
	}
	stats["redBets"] = int(redBets)

	// Получаем количество ставок на зеро
	var zeroBets int64
	if err := query.Where("option = ?", string(models.Zero)).Count(&zeroBets).Error; err != nil {
		return nil, err
	}
	stats["zeroBets"] = int(zeroBets)

	// Получаем количество выигрышных ставок
	var wonBets int64
	if err := query.Where("won = ?", true).Count(&wonBets).Error; err != nil {
		return nil, err
	}
	stats["wonBets"] = int(wonBets)

	// Получаем количество выигрышных ставок на черное
	var wonBlackBets int64
	if err := query.Where("option = ? AND won = ?", string(models.Black), true).Count(&wonBlackBets).Error; err != nil {
		return nil, err
	}
	stats["wonBlackBets"] = int(wonBlackBets)

	// Получаем количество выигрышных ставок на красное
	var wonRedBets int64
	if err := query.Where("option = ? AND won = ?", string(models.Red), true).Count(&wonRedBets).Error; err != nil {
		return nil, err
	}
	stats["wonRedBets"] = int(wonRedBets)

	// Получаем количество выигрышных ставок на зеро
	var wonZeroBets int64
	if err := query.Where("option = ? AND won = ?", string(models.Zero), true).Count(&wonZeroBets).Error; err != nil {
		return nil, err
	}
	stats["wonZeroBets"] = int(wonZeroBets)

	// Вычисляем проигрышные ставки
	stats["lostBets"] = stats["totalBets"] - stats["wonBets"]
	stats["lostBlackBets"] = stats["blackBets"] - stats["wonBlackBets"]
	stats["lostRedBets"] = stats["redBets"] - stats["wonRedBets"]
	stats["lostZeroBets"] = stats["zeroBets"] - stats["wonZeroBets"]

	// Получаем общее количество заработанных баллов
	var totalPoints int
	if err := query.Where("won = ?", true).Select("COALESCE(SUM(points), 0)").Scan(&totalPoints).Error; err != nil {
		return nil, err
	}
	stats["totalPoints"] = totalPoints

	return stats, nil
}
