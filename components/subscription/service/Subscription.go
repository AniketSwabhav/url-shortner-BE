package service

import (
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
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

func (service *SubscriptionService) SetSubscriptionPrice(subscriptionPrices *subscription.Subscription, userIdFromToken uuid.UUID) error {

	if err := service.doesUserExist(userIdFromToken); err != nil {
		return err
	}

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

func (service *SubscriptionService) GetPrice(latest *subscription.Subscription, userIdFromToken uuid.UUID) error {

	if err := service.doesUserExist(userIdFromToken); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	err := service.repository.GetAll(uow, &latest)
	if err != nil {
		return errors.NewDatabaseError("unable to fetch subscription details")
	}

	// uow.Commit()
	return nil
}

func (service *SubscriptionService) UpdateSubscriptionPrice(prices *subscription.Subscription, userIdFromToken uuid.UUID) error {

	if err := service.doesUserExist(userIdFromToken); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	tempUser := user.User{}
	if err := service.repository.GetRecordByID(uow, userIdFromToken, &tempUser); err != nil {
		return errors.NewDatabaseError("Unable to fetch Admin details")
	}

	if !*tempUser.IsActive {
		return errors.NewUnauthorizedError("Inactive users cannot update subscription")
	}

	prices.UpdatedAt = time.Now()

	if err := service.repository.Update(uow, prices); err != nil {
		return errors.NewDatabaseError("unable to update Subscription Price")
	}

	uow.Commit()
	return nil
}

// ---------------- Helpers ----------------

func (service *SubscriptionService) doesUserExist(ID uuid.UUID) error {
	var u user.User
	if err := service.db.First(&u, "id = ?", ID).Error; err != nil {
		return errors.NewValidationError("Admin Doesn't exists")
	}
	return nil
}
