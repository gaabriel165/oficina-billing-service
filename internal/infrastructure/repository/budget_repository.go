package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	domainrepo "github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/database/model"
	"gorm.io/gorm"
)

type GormBudgetRepository struct {
	db *gorm.DB
}

func NewGormBudgetRepository(db *gorm.DB) *GormBudgetRepository {
	return &GormBudgetRepository{db: db}
}

func (r *GormBudgetRepository) Create(ctx context.Context, budget *entity.Budget, outgoing ...messaging.OutgoingMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model.BudgetModelFromDomain(budget)).Error; err != nil {
			return err
		}
		return appendToOutbox(ctx, tx, outgoing)
	})
}

func (r *GormBudgetRepository) Update(ctx context.Context, budget *entity.Budget, outgoing ...messaging.OutgoingMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Items").Save(model.BudgetModelFromDomain(budget)).Error; err != nil {
			return err
		}
		return appendToOutbox(ctx, tx, outgoing)
	})
}

func (r *GormBudgetRepository) FindByID(ctx context.Context, id string) (*entity.Budget, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *GormBudgetRepository) FindByOrderID(ctx context.Context, orderID string) (*entity.Budget, error) {
	return r.findOne(ctx, "order_id = ?", orderID)
}

func (r *GormBudgetRepository) FindAll(ctx context.Context) ([]*entity.Budget, error) {
	return r.findMany(r.db.WithContext(ctx))
}

func (r *GormBudgetRepository) FindByCustomerID(ctx context.Context, customerID string) ([]*entity.Budget, error) {
	return r.findMany(r.db.WithContext(ctx).Where("customer_id = ?", customerID))
}

func (r *GormBudgetRepository) findOne(ctx context.Context, condition, value string) (*entity.Budget, error) {
	var m model.BudgetModel
	err := r.db.WithContext(ctx).Preload("Items").First(&m, condition, value).Error
	if isNotFound(err) {
		return nil, domainrepo.ErrBudgetNotFound
	}
	if err != nil {
		return nil, err
	}
	return m.ToDomain(), nil
}

func (r *GormBudgetRepository) findMany(query *gorm.DB) ([]*entity.Budget, error) {
	var models []model.BudgetModel
	if err := query.Preload("Items").Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	budgets := make([]*entity.Budget, len(models))
	for index := range models {
		budgets[index] = models[index].ToDomain()
	}
	return budgets, nil
}
