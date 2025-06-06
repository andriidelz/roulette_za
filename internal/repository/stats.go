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

// GetTotalStats получает общую статистику по всем ставкам
func (r *PostgresRepository) GetTotalStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Получаем общее количество ставок
	var totalBets int64
	if err := r.db.Model(&models.Bet{}).Count(&totalBets).Error; err != nil {
		return nil, err
	}
	stats["totalBets"] = totalBets

	// Получаем количество выигрышных ставок
	var wonBets int64
	if err := r.db.Model(&models.Bet{}).Where("won = ?", true).Count(&wonBets).Error; err != nil {
		return nil, err
	}
	stats["wonBets"] = wonBets

	// Получаем количество проигрышных ставок
	stats["lostBets"] = totalBets - wonBets

	// Получаем количество уникальных пользователей, сделавших ставки
	var uniqueUsers int64
	if err := r.db.Model(&models.Bet{}).Select("COUNT(DISTINCT user_id)").Scan(&uniqueUsers).Error; err != nil {
		return nil, err
	}
	stats["uniqueUsers"] = uniqueUsers

	// Получаем общее количество выигранных баллов
	var totalPoints int64
	if err := r.db.Model(&models.Bet{}).Where("won = ?", true).Select("COALESCE(SUM(points), 0)").Scan(&totalPoints).Error; err != nil {
		return nil, err
	}
	stats["totalPoints"] = totalPoints

	// Получаем статистику по типам ставок
	var redBets, blackBets, zeroBets int64

	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Red)).Count(&redBets).Error; err != nil {
		return nil, err
	}
	stats["redBets"] = redBets

	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Black)).Count(&blackBets).Error; err != nil {
		return nil, err
	}
	stats["blackBets"] = blackBets

	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Zero)).Count(&zeroBets).Error; err != nil {
		return nil, err
	}
	stats["zeroBets"] = zeroBets

	return stats, nil
}

// GetSuccessRateStats получает статистику успешных угадываний
func (r *PostgresRepository) GetSuccessRateStats() (map[string]float64, error) {
	stats := make(map[string]float64)

	// Получаем общее количество ставок
	var totalBets int64
	if err := r.db.Model(&models.Bet{}).Count(&totalBets).Error; err != nil {
		return nil, err
	}

	// Получаем количество выигрышных ставок
	var wonBets int64
	if err := r.db.Model(&models.Bet{}).Where("won = ?", true).Count(&wonBets).Error; err != nil {
		return nil, err
	}

	// Вычисляем общий процент успешных ставок
	if totalBets > 0 {
		stats["overallSuccessRate"] = float64(wonBets) / float64(totalBets) * 100
	} else {
		stats["overallSuccessRate"] = 0
	}

	// Получаем статистику по типам ставок
	var redTotal, redWon, blackTotal, blackWon, zeroTotal, zeroWon int64

	// Красное
	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Red)).Count(&redTotal).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.Bet{}).Where("option = ? AND won = ?", string(models.Red), true).Count(&redWon).Error; err != nil {
		return nil, err
	}

	if redTotal > 0 {
		stats["redSuccessRate"] = float64(redWon) / float64(redTotal) * 100
	} else {
		stats["redSuccessRate"] = 0
	}

	// Черное
	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Black)).Count(&blackTotal).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.Bet{}).Where("option = ? AND won = ?", string(models.Black), true).Count(&blackWon).Error; err != nil {
		return nil, err
	}

	if blackTotal > 0 {
		stats["blackSuccessRate"] = float64(blackWon) / float64(blackTotal) * 100
	} else {
		stats["blackSuccessRate"] = 0
	}

	// Zero
	if err := r.db.Model(&models.Bet{}).Where("option = ?", string(models.Zero)).Count(&zeroTotal).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.Bet{}).Where("option = ? AND won = ?", string(models.Zero), true).Count(&zeroWon).Error; err != nil {
		return nil, err
	}

	if zeroTotal > 0 {
		stats["zeroSuccessRate"] = float64(zeroWon) / float64(zeroTotal) * 100
	} else {
		stats["zeroSuccessRate"] = 0
	}

	return stats, nil
}

// GetTopPlayersBySuccessRate получает топ игроков по успешным угадываниям
func (r *PostgresRepository) GetTopPlayersBySuccessRate(limit int) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// SQL запрос для получения топа игроков по проценту успешных ставок
	// с минимальным количеством ставок (10) для исключения случайных результатов
	rows, err := r.db.Raw(`
		WITH user_stats AS (
			SELECT 
				u.id AS user_id,
				u.username,
				u.first_name,
				u.last_name,
				COUNT(b.id) AS total_bets,
				SUM(CASE WHEN b.won THEN 1 ELSE 0 END) AS won_bets,
				SUM(CASE WHEN b.won THEN b.points ELSE 0 END) AS total_points,
				CASE 
					WHEN COUNT(b.id) > 0 THEN 
						CAST(SUM(CASE WHEN b.won THEN 1 ELSE 0 END) AS FLOAT) / COUNT(b.id) * 100 
					ELSE 0 
				END AS success_rate
			FROM users u
			LEFT JOIN bets b ON u.id = b.user_id
			GROUP BY u.id, u.username, u.first_name, u.last_name
			HAVING COUNT(b.id) >= 3
		)
		SELECT 
			user_id,
			COALESCE(NULLIF(username, ''), CONCAT(first_name, ' ', last_name)) AS display_name,
			total_bets,
			won_bets,
			total_points,
			success_rate
		FROM user_stats
		ORDER BY success_rate DESC, total_bets DESC
		LIMIT ?
	`, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userId uint
		var displayName string
		var totalBets, wonBets, totalPoints int
		var successRate float64

		if err := rows.Scan(&userId, &displayName, &totalBets, &wonBets, &totalPoints, &successRate); err != nil {
			return nil, err
		}

		player := map[string]interface{}{
			"userId":      userId,
			"displayName": displayName,
			"totalBets":   totalBets,
			"wonBets":     wonBets,
			"totalPoints": totalPoints,
			"successRate": successRate,
		}

		result = append(result, player)
	}

	return result, nil
}

// GetTopPlayersByAttempts получает топ игроков по количеству попыток
func (r *PostgresRepository) GetTopPlayersByAttempts(limit int) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// SQL запрос для получения топа игроков по количеству ставок
	rows, err := r.db.Raw(`
		SELECT 
			u.id AS user_id,
			COALESCE(NULLIF(u.username, ''), CONCAT(u.first_name, ' ', u.last_name)) AS display_name,
			COUNT(b.id) AS total_bets,
			SUM(CASE WHEN b.won THEN 1 ELSE 0 END) AS won_bets,
			SUM(CASE WHEN b.won THEN b.points ELSE 0 END) AS total_points,
			CASE 
				WHEN COUNT(b.id) > 0 THEN 
					CAST(SUM(CASE WHEN b.won THEN 1 ELSE 0 END) AS FLOAT) / COUNT(b.id) * 100 
				ELSE 0 
			END AS success_rate
		FROM users u
		JOIN bets b ON u.id = b.user_id
		GROUP BY u.id, u.username, u.first_name, u.last_name
		ORDER BY total_bets DESC, total_points DESC
		LIMIT ?
	`, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userId uint
		var displayName string
		var totalBets, wonBets, totalPoints int
		var successRate float64

		if err := rows.Scan(&userId, &displayName, &totalBets, &wonBets, &totalPoints, &successRate); err != nil {
			return nil, err
		}

		player := map[string]interface{}{
			"userId":      userId,
			"displayName": displayName,
			"totalBets":   totalBets,
			"wonBets":     wonBets,
			"totalPoints": totalPoints,
			"successRate": successRate,
		}

		result = append(result, player)
	}

	return result, nil
}

// GetSource возвращает кол-во регистраций по источникам
func (r *PostgresRepository) GetSource(dateFrom, dateTo string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// SQL запрос для получения кол-ва регистраций по источнику
	// ::date скорочена нотація Postgres для приведення значення в формат
	// WHERE source = ref_key
	rows, err := r.db.Raw(`
		SELECT created_at::date, COALESCE(source, '') AS source, COUNT(*)
		FROM public.users
		WHERE created_at >= ? and created_at <= ?
		GROUP BY created_at::date, source
		ORDER BY created_at DESC, source DESC;
	`, dateFrom, dateTo).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var created_at, source string
		var count int

		if err := rows.Scan(&created_at, &source, &count); err != nil {
			return nil, err
		}

		player := map[string]interface{}{
			"created_at": created_at,
			"source":     source,
			"count":      count,
		}
		result = append(result, player)
	}

	return result, nil
}
