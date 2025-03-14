package repository

import "roulette/internal/models"

// SaveHashEntry зберігає запис хешу в базу даних
func (r *PostgresRepository) SaveHashEntry(entry *models.HashEntry) error {
	return r.db.Create(entry).Error
}

// GetHashEntries отримує записи хешів з бази даних з пагінацією
func (r *PostgresRepository) GetHashEntries(offset, limit int) ([]models.HashEntry, error) {
	var entries []models.HashEntry
	err := r.db.Order("id desc").Offset(offset).Limit(limit).Find(&entries).Error
	return entries, err
}

// CountHashEntries підраховує загальну кількість записів хешів
func (r *PostgresRepository) CountHashEntries() (int64, error) {
	var count int64
	err := r.db.Model(&models.HashEntry{}).Count(&count).Error
	return count, err
}
