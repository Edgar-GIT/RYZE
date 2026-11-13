package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// AdminAuditFilter narrows the audit log listing. Empty values are ignored.
type AdminAuditFilter struct {
	AdminID string
	Outcome string
}

// AdminAuditRepository stores and lists immutable administrative audit records.
// It deliberately exposes no update or delete: the table rejects both.
type AdminAuditRepository interface {
	Create(ctx context.Context, entry *models.AdminAuditLog) error
	List(ctx context.Context, filter AdminAuditFilter, page, limit int) ([]models.AdminAuditLog, int64, error)
}

type adminAuditRepository struct {
	db *gorm.DB
}

func NewAdminAuditRepository(db *gorm.DB) AdminAuditRepository {
	return &adminAuditRepository{db: db}
}

func (r *adminAuditRepository) Create(ctx context.Context, entry *models.AdminAuditLog) error {
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("failed to record admin audit entry: %w", err)
	}
	return nil
}

// List returns one page of audit records, newest first.
func (r *adminAuditRepository) List(ctx context.Context, filter AdminAuditFilter, page, limit int) ([]models.AdminAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AdminAuditLog{})
	if filter.AdminID != "" {
		query = query.Where("admin_id = ?", filter.AdminID)
	}
	if filter.Outcome != "" {
		query = query.Where("outcome = ?", filter.Outcome)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count admin audit entries: %w", err)
	}

	var entries []models.AdminAuditLog
	if err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list admin audit entries: %w", err)
	}
	return entries, total, nil
}
