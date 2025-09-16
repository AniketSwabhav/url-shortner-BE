package service

import (
	"url-shortner-be/components/errors"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/url"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type UrlService struct {
	db         *gorm.DB
	repository repository.Repository
}

func NewUrlService(DB *gorm.DB, repo repository.Repository) *UrlService {
	return &UrlService{
		db:         DB,
		repository: repo,
	}
}

func (service *UrlService) CreateUrl(userId uuid.UUID, newUrl *url.Url) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	foundUser := &user.User{}
	if err := service.repository.GetRecord(uow, foundUser, repository.Filter("id = ?", userId)); err != nil {
		uow.RollBack()
		return err
	}

	latestPriceRecord := &subscription.Subscription{}
	if err := service.repository.GetRecord(uow, &latestPriceRecord, repository.Order("created_at desc")); err != nil {
		return err
	}

	newUrl.UserID = foundUser.ID
	newUrl.Visits = latestPriceRecord.FreeVisits

	if foundUser.UrlCount == 0 {
		return errors.NewDatabaseError("maximum url creation limit is reached, purchase more for creating new url")
	}

	for {

		newUrl.ShortUrl = url.GenerateShortUrl()

		foundUrl := &url.Url{}
		service.repository.GetRecord(uow, foundUrl, repository.Filter("short_url = ?", newUrl.ShortUrl))
		if foundUrl.ShortUrl == newUrl.ShortUrl {
			continue
		} else {
			break
		}
	}

	if err := service.repository.Add(uow, &newUrl); err != nil {
		uow.RollBack()
		return err
	}

	foundUser.UrlCount--

	if err := service.repository.UpdateWithMap(uow, foundUser, map[string]interface{}{"url_count": foundUser.UrlCount}); err != nil {
		return err
	}

	uow.Commit()
	return nil
}
