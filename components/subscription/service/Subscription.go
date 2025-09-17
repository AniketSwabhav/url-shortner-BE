package service

import (
	"url-shortner-be/components/errors"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/user"
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

func (service *SubscriptionService) SetPrice(subscriptionPrices *subscription.Subscription) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	foundUser := &user.User{}
	if err := service.repository.GetRecord(uow, foundUser, repository.Filter("id = ?", subscriptionPrices.UserId)); err != nil {
		uow.RollBack()
		return err
	}

	if !*foundUser.IsAdmin && !*foundUser.IsActive {
		return errors.NewUnauthorizedError("only admin can set subscription price")
	}

	err := service.repository.Add(uow, &subscriptionPrices)
	if err != nil {
		uow.RollBack()
		return err
	}

	uow.Commit()
	return nil
}

func (service *SubscriptionService) GetPrice(latest *subscription.Subscription) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	err := service.repository.GetRecord(uow, &latest, repository.Order("created_at desc"))
	if err != nil {
		return err
	}

	uow.Commit()
	return nil
}
