package service

import (
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/model/subscription"
	"url-shortner-be/module/repository"

	"github.com/jinzhu/gorm"
)

type SubscriptionService struct {
	db         *gorm.DB
	repository repository.Repository
}

func NewSubscriptionService(DB *gorm.DB, repo repository.Repository) *SubscriptionService {
	return &SubscriptionService{
		db:         DB,
		repository: repo,
	}
}

func (service *SubscriptionService) SetSubscriptionPrice(subscriptionPrices *subscription.Subscription) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	var subscriptionCount int
	var subscriptions []subscription.Subscription

	if err := service.repository.GetCount(uow, &subscriptions, &subscriptionCount); err != nil {
		return errors.NewDatabaseError("Unable to count subscriptions.")
	}

	if subscriptionCount >= 1 {
		return errors.NewValidationError("Subscription price already set. You can update it.")
	}

	if err := service.repository.Add(uow, &subscriptionPrices); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to set subscription price")
	}

	uow.Commit()
	return nil
}

func (service *SubscriptionService) GetPrice(latest *subscription.Subscription) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	err := service.repository.GetAll(uow, &latest)
	if err != nil {
		return errors.NewDatabaseError("unable to fetch subscription details")
	}

	uow.Commit()
	return nil
}

func (service *SubscriptionService) UpdateSubscriptionPrice(prices *subscription.Subscription) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	prices.UpdatedAt = time.Now()

	if err := service.repository.Update(uow, prices); err != nil {
		return errors.NewDatabaseError("unable to update Subscription Price")
	}

	uow.Commit()
	return nil
}
