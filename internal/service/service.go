package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
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

	// Гра
	MakeBet(telegramID int64, option models.BetOption) (*models.Game, *models.Bet, error)
	GetUserBets(telegramID int64, limit int) ([]models.Bet, error)
	CanBetZero(telegramID int64) (bool, int, error)

	// Рейтинги
	// GetWeeklyRating(limit int) ([]models.WeeklyRating, error)
	// GetUserPosition(telegramID int64) (int, error)
	// GetSuperRating(limit int) ([]models.SuperRating, error)
	// UpdateWeeklyRatings() error
	DistributePrizes() error

	// Налаштування та локалізація
	GetText(key string, languageCode string) string
	GetSettings() (map[string]string, error)
	UpdateSetting(key, value string) error

	// Сповіщення та робота з рахунком
	// AddNotification(telegramID int64, notificationType string, message string) error
	// GetNotifications(telegramID int64, limit int) ([]models.Notification, error)
	// RequestWithdrawal(telegramID int64, amount float64, wallet string) error

	GenerateHashEntry() (*models.HashEntry, error)
	GetHashEntries(page, limit int) ([]models.HashEntry, int, error)

	SaveGame(game *models.Game) error
	SaveBet(bet *models.Bet) error
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
	// Перевіряємо, чи існує користувач
	existingUser, err := s.repo.GetUserByTelegramID(telegramID)
	if err == nil {
		// Користувач вже існує, оновлюємо його дані
		existingUser.Username = username
		existingUser.FirstName = firstName
		existingUser.LastName = lastName
		existingUser.LanguageCode = languageCode

		if err := s.repo.UpdateUser(existingUser); err != nil {
			return nil, err
		}

		return existingUser, nil
	}

	// Створюємо нового користувача
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

// Реалізація методів для гри

func (s *ServiceImpl) MakeBet(telegramID int64, option models.BetOption) (*models.Game, *models.Bet, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, nil, err
	}

	// Перевіряємо, чи може користувач зробити ставку на Zero
	if option == models.Zero {
		canBetZero, _, err := s.CanBetZero(telegramID)
		if err != nil {
			return nil, nil, err
		}

		if !canBetZero {
			return nil, nil, fmt.Errorf("cannot bet on zero yet")
		}
	}

	// Генеруємо результат гри
	result := s.generateGameResult()

	// Створюємо хеш для верифікації результату
	hash := s.generateHash(result)

	// Створюємо нову гру
	game := &models.Game{
		Result:    result,
		Hash:      hash,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateGame(game); err != nil {
		return nil, nil, err
	}

	// Визначаємо, чи виграв користувач
	won := option == result

	// Розраховуємо кількість балів
	points := 0
	if won {
		if result == models.Zero {
			points = 10
		} else {
			points = 1
		}
	}

	// Створюємо ставку
	bet := &models.Bet{
		UserID:    user.ID,
		GameID:    game.ID,
		Option:    option,
		Won:       won,
		Points:    points,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateBet(bet); err != nil {
		return nil, nil, err
	}

	return game, bet, nil
}

func (s *ServiceImpl) GetUserBets(telegramID int64, limit int) ([]models.Bet, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserBets(user.ID, limit)
}

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

// Генерація результату гри
func (s *ServiceImpl) generateGameResult() models.BetOption {
	rand.Seed(time.Now().UnixNano())
	roll := rand.Intn(37) + 1 // 1-37

	if roll == 37 {
		return models.Zero
	} else if roll%2 == 0 {
		return models.Red
	} else {
		return models.Black
	}
}

// Генерація хешу для верифікації
func (s *ServiceImpl) generateHash(result models.BetOption) string {
	data := fmt.Sprintf("%s_%d", result, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
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
		user, err := s.repo.GetUserByTelegramID(rating.User.TelegramID)
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

// Реалізація методів для сповіщень та роботи з рахунком

func (s *ServiceImpl) AddNotification(telegramID int64, notificationType string, message string) error {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	notification := &models.Notification{
		UserID:    user.ID,
		Type:      notificationType,
		Message:   message,
		CreatedAt: time.Now(),
	}

	return s.repo.CreateNotification(notification)
}

func (s *ServiceImpl) GetNotifications(telegramID int64, limit int) ([]models.Notification, error) {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserNotifications(user.ID, limit)
}

func (s *ServiceImpl) RequestWithdrawal(telegramID int64, amount float64, wallet string) error {
	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	if user.Balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	// Створюємо запит на виведення коштів
	withdrawal := &models.Withdrawal{
		UserID:    user.ID,
		Amount:    amount,
		Status:    "pending",
		Wallet:    wallet,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Знімаємо кошти з балансу
	user.Balance -= amount
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}

	return s.repo.CreateWithdrawal(withdrawal)
}

// GenerateHashEntry генерує новий запис хешу та зберігає його в базу даних
func (s *ServiceImpl) GenerateHashEntry() (*models.HashEntry, error) {
	// Генеруємо випадкове число від 0 до 36
	randomNumber := utils.GenerateRandomNumber(37)

	// Генеруємо сіль
	salt := utils.GenerateSalt()
	saltHEX := hex.EncodeToString(salt)

	// Створюємо хеш
	hash := utils.CreateHash(randomNumber, salt)

	// Створюємо новий запис
	entry := &models.HashEntry{
		Number:  randomNumber,
		SaltHEX: saltHEX,
		Hash:    hash,
	}

	// Зберігаємо в базу даних
	err := s.repo.SaveHashEntry(entry)
	if err != nil {
		return nil, err
	}

	// log.Printf("Generated hash: %s, number: %d, color: %s", hash, randomNumber, utils.GetColorForNumber(randomNumber))

	return entry, nil
}

// GetHashEntries отримує записи хешів з пагінацією
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

// SaveGame сохраняет информацию об игре в БД
func (s *ServiceImpl) SaveGame(game *models.Game) error {
	return s.repo.CreateGame(game)
}

// SaveBet сохраняет информацию о ставке в БД
func (s *ServiceImpl) SaveBet(bet *models.Bet) error {
	return s.repo.CreateBet(bet)
}
