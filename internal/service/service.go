package service

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"

	"roulette/internal/models"
	"roulette/internal/repository"
	"roulette/internal/utils"
)

// Service інтерфейс для бізнес-логіки
type Service interface {
	// Пользователи
	RegisterUser(telegramID int64, username, firstName, lastName, source, languageCode string) (*models.User, error)
	GetUser(telegramID int64) (*models.User, error)
	GetUserStats(telegramID int64) (map[string]int, error)
	GetDetailedUserStats(telegramID int64, period string) (map[string]int, error)

	// Игра и раунды
	MakeBet(telegramID int64, option models.BetOption) error
	GetUserBets(telegramID int64, limit int) ([]models.Bet, error)
	CanBetZero(telegramID int64) (bool, int, error)
	GetUserRemainingBets(telegramID int64) (int, error) // Добавленный метод для проверки доступных ставок
	GetCurrentRound() (*models.HashEntry, error)
	StartNewRound() (*models.HashEntry, error)
	StartNewRoundFromRotator() (*models.HashEntry, error)
	CompleteRound(hashEntryID uint) error
	GetRoundResult(hashEntryID uint) (models.BetOption, error)
	ProcessAndGetBets(hashEntryID uint) ([]models.Bet, error)
	GetUserBetsForRound(telegramID int64, hashEntryID uint) ([]models.Bet, error)
	GetHashEntryByID(id uint) (*models.HashEntry, error)

	// Статистика
	GetTotalStats() (map[string]int64, error)
	GetSuccessRateStats() (map[string]float64, error)
	GetTopPlayersBySuccessRate(limit int) ([]map[string]interface{}, error)
	GetTopPlayersByAttempts(limit int) ([]map[string]interface{}, error)
	GetSource(dateFrom, dateTo string) ([]map[string]interface{}, error)

	// Рейтинги
	GetWeeklyRating(limit int) ([]models.WeeklyRating, error)
	GetUserPosition(telegramID int64) (int, error)
	GetSuperRating(limit int) ([]models.SuperRating, error)
	UpdateWeeklyRatings() error
	DistributePrizes(year, week int) error
	CancelPrizeDistribution(year, week int) error
	GetPrizeFund(year, week int) (*models.PrizeFund, error)
	GetWeeklyTopRating(limit int) ([]models.WeeklyRating, error)
	GetUserRatingPosition(telegramID int64, neighborsCount int) ([]models.WeeklyRating, int, error)
	GetPointsToReachPrizeZone() (int, error)
	GetPointsNeededForUser(telegramID int64) (int, error)
	RefreshAllRatings() error
	FormatRatingForDisplay(ratings []models.WeeklyRating, currentUserID int64) []string
	GetPrizeDistributionStatus(year, week int) (string, error)
	FormatRatingList(ratings []models.WeeklyRating, currentUserID int64, language string) string
	CreateNewPrizeFund(year, week int) error
	UpdateCurrentPrizeFund(amount float64, topCount int) error

	// Настройки и локализация
	GetText(key string, languageCode string) string
	GetSettings() (map[string]string, error)
	UpdateSetting(key, value string) error
	GetSettingsWithInfo() (map[string]models.SettingInfo, error)
	SaveSettings(settings map[string]string) error

	// Хеши и история раундов
	GetHashEntries(page, limit int) ([]models.HashEntry, int, error)
	GetLatestHashEntry() (*models.HashEntry, error)

	// Методы для работы со страной пользователя
	GetUserCountry(telegramID int64) (string, error)

	UpdateUserLanguage(telegramID int64, languageCode string) error
	UpdateUser(user *models.User) error

	// Вывод средств
	CreateWithdrawal(withdrawal *models.Withdrawal) error

	GetCurrentYearWeek() (int, int)

	// Методы для работы с уведомлениями
	GetNotificationTemplates(templateType string, page, perPage int) ([]models.NotificationTemplate, int64, error)
	GetNotificationTemplateByID(id uint) (*models.NotificationTemplate, error)
	CreateNotificationTemplate(template *models.NotificationTemplate) error
	UpdateNotificationTemplate(template *models.NotificationTemplate) error
	DeleteNotificationTemplate(id uint) error
	GetTemplateWithLocalizations(templateID uint) (*models.NotificationTemplateWithLocalizations, error)

	// Автоматические уведомления
	HandleBalanceUpdate(userID uint, amount float64) error
	HandleTopRatingEntry(userID uint, position int) error

	GetNotificationTasks(status string, page, perPage int) ([]models.NotificationTask, int64, error)
	GetEnhancedNotificationTask(id uint) (*models.EnhancedNotificationTask, error)
	CreateNotificationTask(templateID uint, targetType string, targetParams models.NotificationTargetParams, scheduledAt *time.Time) (*models.NotificationTask, error)
	CancelNotificationTask(id uint) error
	SendNotifications(taskID uint) error
	GetPendingNotificationTasks() ([]models.NotificationTask, error)
	GetNotificationTasksStats(period string) (*models.NotificationStatistics, error)
	GetCountriesWithUserCounts() ([]models.CountryOption, error)
	CheckTopRatingEntries() error

	// Вспомогательный метод для доступа к репозиторию
	GetRepo() repository.Repository
}

type ServiceImpl struct {
	repo          repository.Repository
	telegramToken string // Токен для доступа к Telegram API
}

// NewService створює новий екземпляр сервісу
func NewService(repo repository.Repository, telegramToken string) Service {
	return &ServiceImpl{
		repo:          repo,
		telegramToken: telegramToken,
	}
}

// Реалізація методів для користувачів

func (s *ServiceImpl) RegisterUser(telegramID int64, username, firstName, lastName, source, languageCode string) (*models.User, error) {
	// Проверяем, существует ли пользователь
	existingUser, err := s.repo.GetUserByTelegramID(telegramID)
	if err == nil {
		// Пользователь уже существует, обновляем только пустые поля
		updateNeeded := false

		// Обновляем имя пользователя только если оно пустое
		if existingUser.Username == "" && username != "" {
			existingUser.Username = username
			updateNeeded = true
		}

		// Не перезаписываем имя, если оно уже установлено
		if existingUser.FirstName == "" && firstName != "" {
			existingUser.FirstName = firstName
			updateNeeded = true
		}

		// Не перезаписываем фамилию, если она уже установлена
		if existingUser.LastName == "" && lastName != "" {
			existingUser.LastName = lastName
			updateNeeded = true
		}

		// Не перезаписываем язык, если он уже установлен
		if existingUser.LanguageCode == "" && languageCode != "" {
			existingUser.LanguageCode = languageCode
			updateNeeded = true
		}

		// Не перезаписываем источник, если он уже установлен
		if existingUser.Source == "" && source != "" {

			exists, _ := s.repo.CheckSourceKeyExists(source)
			if exists {
				existingUser.Source = source
				updateNeeded = true
			} else {
				log.Println("Error find source", telegramID, source)
			}
		}

		// Если у пользователя нет аватарки, попробуем получить ее из Telegram
		if existingUser.AvatarURL == "" {
			if avatarURL, err := utils.GetUserProfilePhoto(s.telegramToken, telegramID); err == nil && avatarURL != "" {
				existingUser.AvatarURL = avatarURL
				updateNeeded = true
			}
		}

		// Обновляем пользователя только если были изменения
		if updateNeeded {
			if err := s.repo.UpdateUser(existingUser); err != nil {
				return nil, err
			}
		}

		return existingUser, nil
	}

	// Получаем аватарку пользователя из Telegram
	avatarURL := ""
	if avatar, err := utils.GetUserProfilePhoto(s.telegramToken, telegramID); err == nil {
		avatarURL = avatar
	}

	exists, _ := s.repo.CheckSourceKeyExists(source)
	if !exists {
		source = ""
		log.Println("Error find source", telegramID, source)
	}

	// Создаем нового пользователя
	user := &models.User{
		TelegramID:   telegramID,
		Username:     username,
		Nickname:     "", // Пустой никнейм для новых пользователей
		FirstName:    firstName,
		LastName:     lastName,
		Source:       source,
		RefKey:       "", // Пустая реферальная ссылка для новых пользователей
		LanguageCode: languageCode,
		AvatarURL:    avatarURL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *ServiceImpl) GetUser(telegramID int64) (*models.User, error) {
	return s.repo.GetUserByTelegramID(telegramID)
}

func (s *ServiceImpl) GetUserStats(telegramID int64) (map[string]int, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	// Получаем общее количество ставок
	totalBets, err := s.repo.GetUserTotalBets(user.ID)
	if err != nil {
		return nil, err
	}

	// Получаем количество выигрышных ставок
	wonBets, err := s.repo.GetUserWonBets(user.ID)
	if err != nil {
		return nil, err
	}

	// Получаем количество баллов
	totalPoints, err := s.repo.GetUserTotalPoints(user.ID)
	if err != nil {
		return nil, err
	}

	// Получаем статистику за день
	dailyBets, dailyPoints, err := s.repo.GetUserDailyStats(user.ID)
	if err != nil {
		return nil, err
	}

	// Получаем статистику за неделю
	weeklyBets, weeklyPoints, err := s.repo.GetUserWeeklyStats(user.ID)
	if err != nil {
		return nil, err
	}

	// Получаем статистику за месяц
	monthlyBets, monthlyPoints, err := s.repo.GetUserMonthlyStats(user.ID)
	if err != nil {
		return nil, err
	}

	// Формируем карту со статистикой
	stats := map[string]int{
		"totalBets":     totalBets,
		"wonBets":       wonBets,
		"totalPoints":   totalPoints,
		"dailyBets":     dailyBets,
		"dailyPoints":   dailyPoints,
		"weeklyBets":    weeklyBets,
		"weeklyPoints":  weeklyPoints,
		"monthlyBets":   monthlyBets,
		"monthlyPoints": monthlyPoints,
	}

	return stats, nil
}

// Реалізація методів для гри та раундів

// GetCurrentRound получает текущий активный раунд
func (s *ServiceImpl) GetCurrentRound() (*models.HashEntry, error) {
	return s.repo.GetActiveHashEntry()
}

// StartNewRound создает новый раунд и завершает предыдущий
func (s *ServiceImpl) StartNewRound() (*models.HashEntry, error) {
	return s.repo.GetActiveHashEntry()
}

// StartNewRoundFromRotator создает новый раунд (только для ротатора)
func (s *ServiceImpl) StartNewRoundFromRotator() (*models.HashEntry, error) {
	// Проверяем, есть ли активный раунд
	currentRound, err := s.repo.GetActiveHashEntry()
	if err != nil {
		// Если ошибка - это не "запись не найдена", возвращаем её
		return nil, err
	}

	// Если активный раунд найден, завершаем его
	if currentRound != nil {
		if err = s.CompleteRound(currentRound.ID); err != nil {
			return nil, err
		}
	}

	// Генерируем новый хеш для нового раунда
	randomNumber := utils.GenerateRandomNumber(37)
	salt := utils.GenerateSalt()
	saltHEX := hex.EncodeToString(salt)
	hash := utils.CreateHash(randomNumber, salt)

	// Создаем новый запись
	entry := &models.HashEntry{
		Number:      randomNumber,
		SaltHEX:     saltHEX,
		Hash:        hash,
		IsCompleted: false,
	}

	// Сохраняем в базу данных
	err = s.repo.CreateHashEntry(entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// CompleteRound завершает текущий раунд и обрабатывает ставки
func (s *ServiceImpl) CompleteRound(hashEntryID uint) error {
	// Получаем данные о раунде
	round, err := s.repo.GetHashEntryByID(hashEntryID)
	if err != nil {
		return err
	}

	// Если раунд уже завершен, возвращаем ошибку
	if round.IsCompleted {
		return fmt.Errorf("round %d is already completed", hashEntryID)
	}

	// Обрабатываем ставки
	_, err = s.ProcessAndGetBets(hashEntryID)
	if err != nil {
		return err
	}

	// Получаем текущее время для отметки завершения раунда
	revealedAt := time.Now()

	// Помечаем раунд как завершенный с указанием времени
	return s.repo.CompleteHashEntry(hashEntryID, revealedAt)
}

// GetRoundResult получает результат раунда (цвет)
func (s *ServiceImpl) GetRoundResult(hashEntryID uint) (models.BetOption, error) {
	round, err := s.repo.GetHashEntryByID(hashEntryID)
	if err != nil {
		return "", err
	}

	// Используем utils.GetColorForNumber для определения цвета
	color := utils.GetColorForNumber(round.Number)

	// Преобразуем строку в models.BetOption
	switch color {
	case "red":
		return models.Red, nil
	case "black":
		return models.Black, nil
	case "zero (green)":
		return models.Zero, nil
	default:
		return "", fmt.Errorf("unknown color: %s", color)
	}
}

// ProcessAndGetBets обрабатывает все ставки и возвращает список обработанных ставок
func (s *ServiceImpl) ProcessAndGetBets(hashEntryID uint) ([]models.Bet, error) {
	// Получаем все ставки для этого раунда с preload пользователей
	bets, err := s.repo.GetBetsByHashEntryIDWithUsers(hashEntryID)
	if err != nil {
		return nil, err
	}

	if len(bets) == 0 {
		return bets, nil
	}

	// Получаем результат раунда
	result, err := s.GetRoundResult(hashEntryID)
	if err != nil {
		return nil, err
	}

	// Обрабатываем каждую ставку
	for i := range bets {
		// Определяем, выиграла ли ставка
		won := bets[i].Option == result

		// Рассчитываем количество полученных баллов
		points := 0
		if won {
			if result == models.Zero {
				points = 10
			} else {
				points = 1
			}
		}

		// Обновляем ставку прямо в срезе
		bets[i].Won = won
		bets[i].Points = points

		// Сохраняем обновленную ставку в БД
		if err := s.repo.UpdateBet(&bets[i]); err != nil {
			log.Printf("Error updating bet for user %d in round %d: %v",
				bets[i].UserID, hashEntryID, err)
			// Продолжаем обработку других ставок
		}
	}

	return bets, nil
}

// MakeBet делает ставку в текущем раунде
func (s *ServiceImpl) MakeBet(telegramID int64, option models.BetOption) error {
	// Получаем пользователя
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	// Получаем текущий раунд
	currentRound, err := s.repo.GetActiveHashEntry()
	if err != nil {
		return err
	}

	// Проверяем, может ли пользователь делать ставку на Zero
	if option == models.Zero {
		canBetZero, _, err := s.CanBetZero(telegramID)
		if err != nil {
			return err
		}

		if !canBetZero {
			return fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Проверяем, не делал ли пользователь уже ставку в этом раунде
	existingBets, err := s.repo.GetUserBetsForHashEntry(user.ID, currentRound.ID)
	if err != nil {
		return err
	}

	if len(existingBets) > 0 {
		return fmt.Errorf("user has already made a bet in this round")
	}

	// Создаем новую ставку
	bet := &models.Bet{
		UserID:      user.ID,
		HashEntryID: currentRound.ID,
		Option:      option,
		CreatedAt:   time.Now(),
	}

	// Сохраняем ставку
	return s.repo.CreateBet(bet)
}

// GetUserBets получает историю ставок пользователя
func (s *ServiceImpl) GetUserBets(telegramID int64, limit int) ([]models.Bet, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserBets(user.ID, limit)
}

// CanBetZero проверяет, может ли пользователь делать ставку на Zero
func (s *ServiceImpl) CanBetZero(telegramID int64) (bool, int, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return false, 0, err
	}

	// Получаем количество ставок за сегодня
	dailyBets, err := s.repo.GetUserDailyBets(user.ID)
	if err != nil {
		return false, 0, err
	}

	// Получаем настройку лимита ставок на Zero
	setting, err := s.repo.GetSetting("daily_bets_zero_limit")
	if err != nil {
		// Если настройки нет, используем значение по умолчанию
		return dailyBets >= 100, 100 - dailyBets, nil
	}

	dailyBetsZeroLimit, err := strconv.Atoi(setting.Value)
	if err != nil {
		// Если значение не числовое, используем значение по умолчанию
		return dailyBets >= 100, 100 - dailyBets, nil
	}

	// Проверяем лимит ставок на Zero
	if dailyBets >= dailyBetsZeroLimit {
		return true, 0, nil
	}

	return false, dailyBetsZeroLimit - dailyBets, nil
}

// GetUserRemainingBets получает количество доступных ставок для пользователя на сегодня
func (s *ServiceImpl) GetUserRemainingBets(telegramID int64) (int, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return 0, err
	}

	// Получаем настройку дневного лимита ставок
	setting, err := s.repo.GetSetting("daily_bets_limit")
	if err != nil {
		// Если настройки нет, используем значение по умолчанию
		return 2880, nil
	}

	dailyLimit, err := strconv.Atoi(setting.Value)
	if err != nil {
		// Если значение не числовое, используем значение по умолчанию
		return 2880, nil
	}

	// Получаем количество ставок за сегодня
	dailyBets, err := s.repo.GetUserDailyBets(user.ID)
	if err != nil {
		return 0, err
	}

	// Вычисляем оставшееся количество ставок
	return dailyLimit - dailyBets, nil
}

// GetUserBetsForRound получает ставки пользователя для конкретного раунда
func (s *ServiceImpl) GetUserBetsForRound(telegramID int64, hashEntryID uint) ([]models.Bet, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserBetsForHashEntry(user.ID, hashEntryID)
}

// GetHashEntryByID получает запись хеша (раунд) по ID
func (s *ServiceImpl) GetHashEntryByID(id uint) (*models.HashEntry, error) {
	return s.repo.GetHashEntryByID(id)
}

// Реалізація методів для рейтингів

func (s *ServiceImpl) GetWeeklyRating(limit int) ([]models.WeeklyRating, error) {
	year, week := time.Now().ISOWeek()
	return s.repo.GetWeeklyRating(year, week, limit)
}

func (s *ServiceImpl) GetUserPosition(telegramID int64) (int, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return 0, err
	}

	year, week := time.Now().ISOWeek()
	rating, err := s.repo.GetUserWeeklyRating(user.ID, year, week)
	if err != nil {
		return 0, err
	}

	return rating.Position, nil
}

func (s *ServiceImpl) GetSuperRating(limit int) ([]models.SuperRating, error) {
	now := time.Now()
	quarter := fmt.Sprintf("%d-Q%d", now.Year(), (now.Month()-1)/3+1)
	return s.repo.GetSuperRating(quarter, limit)
}

// UpdateWeeklyRatings обновляет еженедельные рейтинги при начале новой недели
// и создает новый рейтинг на основе актуальных настроек
func (s *ServiceImpl) UpdateWeeklyRatings() error {
	// Получаем предыдущую неделю (поскольку это вызывается в понедельник)
	now := time.Now()
	previousDate := now.AddDate(0, 0, -1) // Воскресенье
	prevYear, prevWeek := previousDate.ISOWeek()

	// Текущая неделя (началась сегодня)
	currentYear, currentWeek := now.ISOWeek()

	// Обновляем рейтинги для предыдущей недели
	if err := s.repo.CalculateWeeklyRatings(prevYear, prevWeek); err != nil {
		return fmt.Errorf("error calculating ratings for previous week: %w", err)
	}

	// Проверяем существование призового фонда для текущей недели
	_, err := s.repo.GetPrizeFundWithoutCreation(currentYear, currentWeek)
	if err != nil {
		// Если призовой фонд не найден, создаем новый
		if err := s.CreateNewPrizeFund(currentYear, currentWeek); err != nil {
			return fmt.Errorf("error creating new weekly rating: %w", err)
		}
	}

	return nil
}

func (s *ServiceImpl) DistributePrizes(year, week int) error {

	// Отримуємо призовий фонд
	prizeFund, err := s.repo.GetPrizeFund(year, week)
	if err != nil {
		return err
	}

	if prizeFund.Processed {
		return fmt.Errorf("prize fund for week %d/%d already processed", year, week)
	}

	// Отримуємо рейтинг
	ratings, err := s.repo.GetWeeklyRating(year, week, prizeFund.TopCount)
	if err != nil {
		return err
	}

	if len(ratings) == 0 {
		return fmt.Errorf("no ratings found for week %d/%d", year, week)
	}

	// Рахуємо загальну кількість балів у топі
	totalPoints := 0
	for _, rating := range ratings {
		totalPoints += rating.Points
	}

	if totalPoints == 0 {
		return fmt.Errorf("total points is zero for week %d/%d", year, week)
	}

	// Розподіляємо призовий фонд пропорційно балам
	for _, rating := range ratings {
		prize := (float64(rating.Points) / float64(totalPoints)) * prizeFund.Amount
		rating.Prize = prize

		if err := s.repo.UpdateWeeklyRating(&rating); err != nil {
			return err
		}

		// Зараховуємо приз на баланс користувача
		user, err := s.repo.GetUserByID(rating.UserID)
		if err != nil {
			return err
		}

		user.Balance += prize
		if err := s.repo.UpdateUser(user); err != nil {
			return err
		}

		// Отправляем автоматическое уведомление о пополнении баланса
		if err := s.HandleBalanceUpdate(user.ID, prize); err != nil {
			log.Printf("Error sending balance update notification to user %d: %v", user.ID, err)
			// Продолжаем обработку других пользователей
		}

		// Отправляем автоматическое уведомление о входе в топ рейтинга, если позиция пользователя ≤ TopCount
		if rating.Position <= prizeFund.TopCount {
			if err := s.HandleTopRatingEntry(user.ID, rating.Position); err != nil {
				log.Printf("Error sending top rating notification to user %d: %v", user.ID, err)
				// Продолжаем обработку других пользователей
			}
		}
	}

	// Позначаємо призовий фонд як оброблений
	prizeFund.Processed = true
	return s.repo.UpdatePrizeFund(prizeFund)
}

// GetPrizeFund получает информацию о призовом фонде
func (s *ServiceImpl) GetPrizeFund(year, week int) (*models.PrizeFund, error) {
	return s.repo.GetPrizeFund(year, week)
}

// Реалізація методів для налаштувань та локалізації

func (s *ServiceImpl) GetText(key string, languageCode string) string {
	text, err := s.repo.GetLocalization(key, languageCode)
	if err != nil {
		// Повертаємо ключ, якщо локалізація не знайдена
		return key
	}
	return text
}

func (s *ServiceImpl) UpdateSetting(key, value string) error {
	return s.repo.UpdateSetting(key, value)
}

// Реалізація методів для хешів та історії раундів

// GetHashEntries возвращает хеши с пагинацией
func (s *ServiceImpl) GetHashEntries(page, limit int) ([]models.HashEntry, int, error) {
	offset := (page - 1) * limit

	entries, err := s.repo.GetHashEntries(offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountHashEntries()
	if err != nil {
		return nil, 0, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return entries, totalPages, nil
}

// GetLatestHashEntry получает последний хеш из базы данных
func (s *ServiceImpl) GetLatestHashEntry() (*models.HashEntry, error) {
	entries, err := s.repo.GetHashEntries(0, 1)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no hash entries found")
	}

	return &entries[0], nil
}

func (s *ServiceImpl) GetUserCountry(telegramID int64) (string, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return "", err
	}

	return s.repo.GetUserCountry(user.ID)
}

func (s *ServiceImpl) UpdateUserLanguage(telegramID int64, languageCode string) error {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	// Обновляем язык пользователя
	user.LanguageCode = languageCode

	// Обновляем пользователя в базе данных
	return s.repo.UpdateUser(user)
}

// UpdateUser обновляет информацию о пользователе
func (s *ServiceImpl) UpdateUser(user *models.User) error {
	return s.repo.UpdateUser(user)
}

// Реализация метода в ServiceImpl
func (s *ServiceImpl) CreateWithdrawal(withdrawal *models.Withdrawal) error {
	return s.repo.CreateWithdrawal(withdrawal)
}

func (s *ServiceImpl) GetCurrentYearWeek() (int, int) {
	return time.Now().ISOWeek()
}

func (s *ServiceImpl) GetRepo() repository.Repository {
	return s.repo
}
