package repository

import (
	"log"
	"time"

	"roulette/internal/models"

	"gorm.io/gorm"
)

// SaveHashEntry сохраняет запись хеша в базу данных
func (r *PostgresRepository) SaveHashEntry(entry *models.HashEntry) error {
	return r.db.Create(entry).Error
}

// GetHashEntries получает записи хешей с пагинацией
func (r *PostgresRepository) GetHashEntries(offset, limit int) ([]models.HashEntry, error) {
	var entries []models.HashEntry
	err := r.db.Order("id desc").Offset(offset).Limit(limit).Find(&entries).Error
	return entries, err
}

// CountHashEntries подсчитывает общую количество записей хешей
func (r *PostgresRepository) CountHashEntries() (int64, error) {
	var count int64
	err := r.db.Model(&models.HashEntry{}).Count(&count).Error
	return count, err
}

// GetCurrentHashEntry получает последний активный хеш
func (r *PostgresRepository) GetCurrentHashEntry() (*models.HashEntry, error) {
	var entry models.HashEntry
	err := r.db.Where("is_completed = ?", false).Order("id desc").First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// CreateHashEntry создает новый хеш (начало нового раунда)
func (r *PostgresRepository) CreateHashEntry(entry *models.HashEntry) error {
	return r.db.Create(entry).Error
}

// CompleteHashEntry помечает хеш как завершенный (конец раунда)
func (r *PostgresRepository) CompleteHashEntry(id uint, revealedAt time.Time) error {
	return r.db.Model(&models.HashEntry{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_completed": true,
			"revealed_at":  revealedAt,
		}).Error
}

// GetHashEntryByID получает хеш по ID
func (r *PostgresRepository) GetHashEntryByID(id uint) (*models.HashEntry, error) {
	var entry models.HashEntry
	err := r.db.First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// unused
// GetBetsByHashEntryID получает все ставки для указанного хеша (раунда)
func (r *PostgresRepository) GetBetsByHashEntryID(hashEntryID uint) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.db.Where("hash_entry_id = ?", hashEntryID).Find(&bets).Error
	return bets, err
}

// GetActiveHashEntry получает текущий активный хеш (раунд)
func (r *PostgresRepository) GetActiveHashEntry() (*models.HashEntry, error) {
	var entry models.HashEntry
	err := r.db.Where("is_completed = ?", false).Order("created_at desc").First(&entry).Error

	// При отсутствии активного раунда просто возвращаем nil без ошибки
	if err == gorm.ErrRecordNotFound {
		return nil, nil // Возвращаем nil, nil вместо ошибки
	}

	if err != nil {
		// Логируем только реальные ошибки БД
		log.Printf("Error getting active hash entry: %v", err)
		return nil, err
	}

	return &entry, nil
}

// GetUserBetsForHashEntry получает ставки пользователя для конкретного хеша (раунда)
func (r *PostgresRepository) GetUserBetsForHashEntry(userID, hashEntryID uint) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.db.Where("user_id = ? AND hash_entry_id = ?", userID, hashEntryID).Find(&bets).Error
	return bets, err
}

// UpdateBet обновляет информацию о ставке
func (r *PostgresRepository) UpdateBet(bet *models.Bet) error {
	return r.db.Save(bet).Error
}

// GetCompletedRounds возвращает список завершенных раундов с пагинацией
func (r *PostgresRepository) GetCompletedRounds(page, perPage int) ([]models.HashEntry, int64, error) {
	var entries []models.HashEntry
	var totalCount int64

	// Получаем общее количество завершенных раундов
	if err := r.db.Model(&models.HashEntry{}).Where("is_completed = ?", true).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Получаем записи с пагинацией
	offset := (page - 1) * perPage
	err := r.db.Where("is_completed = ?", true).
		Order("id desc").
		Offset(offset).
		Limit(perPage).
		Find(&entries).Error

	if err != nil {
		return nil, 0, err
	}

	return entries, totalCount, nil
}

// GetRoundWithBets возвращает раунд вместе с его ставками
func (r *PostgresRepository) GetRoundWithBets(roundID uint) (*models.HashEntry, error) {
	var entry models.HashEntry
	err := r.db.Preload("Bets").Preload("Bets.User").First(&entry, roundID).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}
