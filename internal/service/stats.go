package service

import (
	"fmt"
	"time"
)

// GetDetailedUserStats получает детальную статистику пользователя по ставкам за указанный период
func (s *ServiceImpl) GetDetailedUserStats(telegramID int64, period string) (map[string]int, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	// Получаем статистику в зависимости от периода
	switch period {
	case "day":
		return s.getDailyDetailedStats(user.ID)
	case "week":
		return s.getWeeklyDetailedStats(user.ID)
	case "month":
		return s.getMonthlyDetailedStats(user.ID)
	case "all":
		return s.getAllTimeDetailedStats(user.ID)
	default:
		return nil, fmt.Errorf("unknown period: %s", period)
	}
}

// getDailyDetailedStats получает детальную статистику за день
func (s *ServiceImpl) getDailyDetailedStats(userID uint) (map[string]int, error) {
	// Определяем временной диапазон (сегодня)
	today := time.Now().Format("2006-01-02")
	return s.getDetailedStatsByDate(userID, today, "")
}

// getWeeklyDetailedStats получает детальную статистику за неделю
func (s *ServiceImpl) getWeeklyDetailedStats(userID uint) (map[string]int, error) {
	// Определяем начало недели (понедельник)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Воскресенье в Go имеет номер 0
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1).Format("2006-01-02")

	return s.getDetailedStatsByDate(userID, startOfWeek, "")
}

// getMonthlyDetailedStats получает детальную статистику за месяц
func (s *ServiceImpl) getMonthlyDetailedStats(userID uint) (map[string]int, error) {
	// Определяем начало месяца
	startOfMonth := time.Now().Format("2006-01") + "-01"

	return s.getDetailedStatsByDate(userID, startOfMonth, "")
}

// getAllTimeDetailedStats получает детальную статистику за все время
func (s *ServiceImpl) getAllTimeDetailedStats(userID uint) (map[string]int, error) {
	return s.getDetailedStatsByDate(userID, "", "")
}

// getDetailedStatsByDate получает детальную статистику по указанной дате
func (s *ServiceImpl) getDetailedStatsByDate(userID uint, startDate string, endDate string) (map[string]int, error) {
	// Запрос к репозиторию для получения детальной статистики
	return s.repo.GetDetailedStatsByDate(userID, startDate, endDate)
}

// GetTotalStats возвращает общую статистику ставок
func (s *ServiceImpl) GetTotalStats() (map[string]int64, error) {
	return s.repo.GetTotalStats()
}

// GetSuccessRateStats возвращает статистику успешных угадываний
func (s *ServiceImpl) GetSuccessRateStats() (map[string]float64, error) {
	return s.repo.GetSuccessRateStats()
}

// GetTopPlayersBySuccessRate возвращает топ игроков по успешным угадываниям
func (s *ServiceImpl) GetTopPlayersBySuccessRate(limit int) ([]map[string]interface{}, error) {
	return s.repo.GetTopPlayersBySuccessRate(limit)
}

// GetTopPlayersByAttempts возвращает топ игроков по количеству попыток
func (s *ServiceImpl) GetTopPlayersByAttempts(limit int) ([]map[string]interface{}, error) {
	return s.repo.GetTopPlayersByAttempts(limit)
}

// GetSource возвращает кол-во регистраций по источнику
func (s *ServiceImpl) GetSource(dateFrom, dateTo string) ([]map[string]interface{}, error) {
	return s.repo.GetSource(dateFrom, dateTo)
}
