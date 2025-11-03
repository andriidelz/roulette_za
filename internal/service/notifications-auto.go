package service

import (
	"roulette/internal/data"
	"roulette/internal/logger"
	"roulette/internal/models"
	"time"
)

// AutoNotificationService отвечает за обработку и отправку автоматических уведомлений
type AutoNotificationService struct {
	service Service
}

// NewAutoNotificationService создает новый сервис автоматических уведомлений
func NewAutoNotificationService(service Service) *AutoNotificationService {
	return &AutoNotificationService{
		service: service,
	}
}

// HandleBalanceUpdated обрабатывает событие обновления баланса и отправляет уведомление
func (s *AutoNotificationService) HandleBalanceUpdated(userID uint, amount float64) error {
	if amount <= 0 {
		return nil
	}

	userMacros := map[string]interface{}{
		"amount": amount,
	}

	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	userMacros["balance"] = user.Balance

	templates, _, err := s.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		return err
	}

	var balanceTemplate *models.NotificationTemplate
	for _, tpl := range templates {
		if tpl.TriggerEvent == "balance_updated" && tpl.Active {
			balanceTemplate = &tpl
			break
		}
	}

	if balanceTemplate == nil {
		logger.Info.Printf("No active template found for 'balance_updated' event")
		return nil
	}

	targetParams := models.NotificationTargetParams{
		UserIDs: []uint{user.ID},
	}

	macrosForUsers := map[uint]map[string]interface{}{
		user.ID: userMacros,
	}

	_, err = s.service.CreateNotificationTask(balanceTemplate.ID, "custom", targetParams, nil, macrosForUsers)
	if err != nil {
		logger.Error.Printf("Error creating balance update notification task for user %d: %v", userID, err)
		return err
	}

	return nil
}

// HandleRegistration обробляє повідомлення про незавершену реєстрацію
func (s *AutoNotificationService) HandleRegistration(userID uint) error {
	templates, _, err := s.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		return err
	}
	var ratingTemplate *models.NotificationTemplate
	for _, tpl := range templates {
		if tpl.TriggerEvent == "user_unregistered" && tpl.Active {
			ratingTemplate = &tpl
			break
		}
	}

	if ratingTemplate == nil {
		logger.Info.Printf("No active template found for 'user_unregistered' event")
		return nil
	}

	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	// Рассчитываем время отправки - 20:00 по локальному времени пользователя
	scheduledTime := s.calculateTimeFor8PM(user.Country)

	// Создаем параметры таргетирования для конкретного пользователя
	targetParams := models.NotificationTargetParams{
		UserIDs: []uint{user.ID},
	}

	// Создаем макросы для пользователя
	userMacros := map[string]interface{}{}

	macrosForUsers := map[uint]map[string]interface{}{
		user.ID: userMacros,
	}

	_, err = s.service.CreateNotificationTask(ratingTemplate.ID, "custom", targetParams, &scheduledTime, macrosForUsers)
	if err != nil {
		logger.Error.Printf("Ошибка при создании задачи уведомления: %v", err)
		return err
	}

	return nil
}

// HandleTopRatingEntered обрабатывает событие входа в топ рейтинга и отправляет уведомление
func (s *AutoNotificationService) HandleTopRatingEntry(userID uint, position int) error {
	templates, _, err := s.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		return err
	}
	var ratingTemplate *models.NotificationTemplate
	for _, tpl := range templates {
		if tpl.TriggerEvent == "top_rating_entered" && tpl.Active {
			ratingTemplate = &tpl
			break
		}
	}

	if ratingTemplate == nil {
		logger.Info.Printf("No active template found for 'top_rating_entered' event")
		return nil
	}

	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	// Получаем данные о рейтинге пользователя
	year, week := time.Now().ISOWeek()
	userRating, err := s.service.GetRepo().GetUserWeeklyRating(userID, year, week)
	if err != nil {
		logger.Error.Printf("Error getting user rating: %v", err)
		// Продолжаем, используя только позицию
	}

	// Получаем данные о призовом фонде
	prizeFund, err := s.service.GetRepo().GetPrizeFund(year, week)
	if err != nil {
		logger.Error.Printf("Error getting prize fund: %v", err)
		// Продолжаем без данных о фонде
	}

	// Рассчитываем время отправки - 20:00 по локальному времени пользователя
	scheduledTime := s.calculateTimeFor8PM(user.Country)

	// Создаем параметры таргетирования для конкретного пользователя
	targetParams := models.NotificationTargetParams{
		UserIDs: []uint{user.ID},
	}

	// Создаем макросы для пользователя
	userMacros := map[string]interface{}{
		"position": position,
	}

	if userRating != nil {
		userMacros["points"] = userRating.Points
	}

	if prizeFund != nil {
		userMacros["prize_fund"] = prizeFund.Amount
	}

	macrosForUsers := map[uint]map[string]interface{}{
		user.ID: userMacros,
	}

	_, err = s.service.CreateNotificationTask(ratingTemplate.ID, "custom", targetParams, &scheduledTime, macrosForUsers)
	if err != nil {
		logger.Error.Printf("Ошибка при создании задачи уведомления: %v", err)
		return err
	}

	return nil
}

// calculateTimeFor8PM рассчитывает время для отправки в 20:00 по локальному времени пользователя
func (s *AutoNotificationService) calculateTimeFor8PM(country string) time.Time {
	now := time.Now()

	// Определяем часовой пояс пользователя
	loc := data.GetTimezoneLocation(country)

	// Получаем текущее локальное время пользователя
	userLocalTime := now.In(loc)

	// Устанавливаем время на 20:00 сегодня по локальному времени пользователя
	targetTime := time.Date(
		userLocalTime.Year(), userLocalTime.Month(), userLocalTime.Day(),
		20, 0, 0, 0, loc,
	)

	// Если 20:00 уже прошло сегодня, переносим на завтра
	if userLocalTime.After(targetTime) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	// Преобразуем время обратно в UTC для сохранения в базе
	return targetTime.UTC()
}
