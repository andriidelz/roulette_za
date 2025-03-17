package service

import (
	"encoding/hex"
	"fmt"
	"time"

	"roulette/internal/models"
	"roulette/internal/repository"
	"roulette/internal/utils"
)

// Service інтерфейс для бізнес-логіки
type Service interface {
	// Користувачі
	RegisterUser(telegramID int64, username, firstName, lastName, languageCode string) (*models.User, error)
	GetUser(telegramID int64) (*models.User, error)
	GetUserStats(telegramID int64) (*models.UserStats, error)

	// Гра та раунди
	MakeBet(telegramID int64, option models.BetOption) error
	GetUserBets(telegramID int64, limit int) ([]models.Bet, error)
	CanBetZero(telegramID int64) (bool, int, error)
	GetCurrentRound() (*models.HashEntry, error)
	StartNewRound() (*models.HashEntry, error)
	StartNewRoundFromRotator() (*models.HashEntry, error)
	CompleteRound(hashEntryID uint) error
	GetRoundResult(hashEntryID uint) (models.BetOption, error)
	ProcessBets(hashEntryID uint) error
	GetUserBetsForRound(telegramID int64, hashEntryID uint) ([]models.Bet, error)
	GetHashEntryByID(id uint) (*models.HashEntry, error)

	// Рейтинги
	GetWeeklyRating(limit int) ([]models.WeeklyRating, error)
	GetUserPosition(telegramID int64) (int, error)
	GetSuperRating(limit int) ([]models.SuperRating, error)
	UpdateWeeklyRatings() error
	DistributePrizes() error

	// Налаштування та локалізація
	GetText(key string, languageCode string) string
	GetSettings() (map[string]string, error)
	UpdateSetting(key, value string) error

	// Хеші та історія раундів
	GetHashEntries(page, limit int) ([]models.HashEntry, int, error)
	GetLatestHashEntry() (*models.HashEntry, error)

	// Методы для работы со страной пользователя
	SetUserCountry(telegramID int64, country string) error
	GetUserCountry(telegramID int64) (string, error)

	UpdateUserLanguage(telegramID int64, languageCode string) error
	UpdateUser(user *models.User) error
}

type ServiceImpl struct {
	repo repository.Repository
}

// NewService створює новий екземпляр сервісу
func NewService(repo repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

// Реалізація методів для користувачів

func (s *ServiceImpl) RegisterUser(telegramID int64, username, firstName, lastName, languageCode string) (*models.User, error) {
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

		// Обновляем пользователя только если были изменения
		if updateNeeded {
			if err := s.repo.UpdateUser(existingUser); err != nil {
				return nil, err
			}
		}

		return existingUser, nil
	}

	// Создаем нового пользователя
	user := &models.User{
		TelegramID:   telegramID,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		LanguageCode: languageCode,
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

func (s *ServiceImpl) GetUserStats(telegramID int64) (*models.UserStats, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserStats(user.ID)
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
		// Если ошибка - "запись не найдена", это нормально для первого запуска
		if err.Error() == "record not found" {
			// Продолжаем выполнение для создания первого раунда
		} else {
			// Если другая ошибка - возвращаем её
			return nil, err
		}
	} else {
		// Если активный раунд найден, завершаем его
		err = s.CompleteRound(currentRound.ID)
		if err != nil {
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
	err = s.ProcessBets(hashEntryID)
	if err != nil {
		return err
	}

	// Помечаем раунд как завершенный
	return s.repo.CompleteHashEntry(hashEntryID, time.Now())
}

// GetRoundResult получает результат раунда (цвет)
func (s *ServiceImpl) GetRoundResult(hashEntryID uint) (models.BetOption, error) {
	round, err := s.repo.GetHashEntryByID(hashEntryID)
	if err != nil {
		return "", err
	}

	// Определяем результат по числу
	if round.Number == 0 {
		return models.Zero, nil
	}

	// Определяем красное или черное
	redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
	for _, n := range redNumbers {
		if round.Number == n {
			return models.Red, nil
		}
	}
	return models.Black, nil
}

// ProcessBets обрабатывает все ставки для завершенного раунда
func (s *ServiceImpl) ProcessBets(hashEntryID uint) error {
	// Получаем все ставки для этого раунда
	bets, err := s.repo.GetBetsByHashEntryID(hashEntryID)
	if err != nil {
		return err
	}

	// Получаем результат раунда
	result, err := s.GetRoundResult(hashEntryID)
	if err != nil {
		return err
	}

	// Обрабатываем каждую ставку
	for _, bet := range bets {
		// Определяем, выиграла ли ставка
		won := bet.Option == result

		// Рассчитываем количество полученных баллов
		points := 0
		if won {
			if result == models.Zero {
				points = 10
			} else {
				points = 1
			}
		}

		// Обновляем ставку
		bet.Won = won
		bet.Points = points
		if err := s.repo.UpdateBet(&bet); err != nil {
			return err
		}

		// Обновляем статистику пользователя
		stats, err := s.repo.GetUserStats(bet.UserID)
		if err != nil {
			return err
		}

		stats.TotalBets++
		stats.DailyBets++
		stats.WeeklyBets++
		stats.MonthlyBets++

		if won {
			stats.WonBets++
			stats.TotalPoints += points
			stats.DailyPoints += points
			stats.WeeklyPoints += points
			stats.MonthlyPoints += points
		}

		if err := s.repo.UpdateUserStats(stats); err != nil {
			return err
		}
	}

	return nil
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

	// Увеличиваем счетчик ставок за день
	user.TodayBets++
	if err := s.repo.UpdateUser(user); err != nil {
		return err
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

	// Перевіряємо кількість ставок за день
	dailyBetsLimit := 100 // можна винести в налаштування

	if user.TodayBets >= dailyBetsLimit {
		return true, 0, nil
	}

	return false, dailyBetsLimit - user.TodayBets, nil
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

func (s *ServiceImpl) UpdateWeeklyRatings() error {
	year, week := time.Now().ISOWeek()
	return s.repo.CalculateWeeklyRatings(year, week)
}

func (s *ServiceImpl) DistributePrizes() error {
	year, week := time.Now().ISOWeek()

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

		// Створюємо сповіщення про виграш
		notification := &models.Notification{
			UserID:    user.ID,
			Type:      "prize",
			Message:   fmt.Sprintf("You won %.2f in the weekly rating! Position: %d", prize, rating.Position),
			CreatedAt: time.Now(),
		}

		if err := s.repo.CreateNotification(notification); err != nil {
			return err
		}
	}

	// Позначаємо призовий фонд як оброблений
	prizeFund.Processed = true
	return s.repo.UpdatePrizeFund(prizeFund)
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

func (s *ServiceImpl) GetSettings() (map[string]string, error) {
	// Цей метод для адмін-панелі, можна розширити пізніше
	return map[string]string{}, nil
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

// Реализация методов для работы со страной пользователя
func (s *ServiceImpl) SetUserCountry(telegramID int64, country string) error {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	return s.repo.SetUserCountry(user.ID, country)
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
