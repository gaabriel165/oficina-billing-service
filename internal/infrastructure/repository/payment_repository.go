package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	domainrepo "github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/database/model"
	"gorm.io/gorm"
)

type GormPaymentRepository struct {
	db *gorm.DB
}

func NewGormPaymentRepository(db *gorm.DB) *GormPaymentRepository {
	return &GormPaymentRepository{db: db}
}

func (r *GormPaymentRepository) Create(ctx context.Context, payment *entity.Payment, outgoing ...messaging.OutgoingMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model.PaymentModelFromDomain(payment)).Error; err != nil {
			return err
		}
		return appendToOutbox(ctx, tx, outgoing)
	})
}

func (r *GormPaymentRepository) Update(ctx context.Context, payment *entity.Payment, outgoing ...messaging.OutgoingMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(model.PaymentModelFromDomain(payment)).Error; err != nil {
			return err
		}
		return appendToOutbox(ctx, tx, outgoing)
	})
}

func (r *GormPaymentRepository) FindByID(ctx context.Context, id string) (*entity.Payment, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *GormPaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*entity.Payment, error) {
	return r.findOne(ctx, "order_id = ?", orderID)
}

func (r *GormPaymentRepository) findOne(ctx context.Context, condition, value string) (*entity.Payment, error) {
	var m model.PaymentModel
	err := r.db.WithContext(ctx).First(&m, condition, value).Error
	if isNotFound(err) {
		return nil, domainrepo.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	return m.ToDomain(), nil
}
