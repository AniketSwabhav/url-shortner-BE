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

func (service *UrlService) CreateUrl(userId uuid.UUID, urlOwner *user.User, newUrl *url.Url) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	foundUser := &user.User{}
	if err := service.repository.GetRecord(uow, foundUser, repository.Filter("id = ?", userId)); err != nil {
		uow.RollBack()
		return err
	}

	if urlOwner.ID != foundUser.ID {
		return errors.NewUnauthorizedError("you are not authorized to create url for this user")
	}

	subscription := &subscription.Subscription{}
	if err := service.repository.GetRecord(uow, &subscription, repository.Order("created_at desc")); err != nil {
		return err
	}

	newUrl.UserID = foundUser.ID

	newUrl.Visits = subscription.FreeVisits
	if foundUser.UrlCount == 0 {
		return errors.NewDatabaseError("maximum url creation limit is reached, purchase more for creating new url")
	}
	newUrl.UserID = foundUser.ID
	newUrl.Visits = subscription.FreeVisits

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

func (service *UrlService) RedirectUrl(shortUrl string) (string, error) {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	url := &url.Url{}
	err := service.repository.GetRecord(uow, url, repository.Filter("short_url = ?", shortUrl))
	if err != nil {
		uow.RollBack()
		return "", err
	}

	if url.Visits == 0 {
		uow.RollBack()
		return "", errors.NewValidationError("no. of visits elapsed")
	}

	url.Visits--

	err = service.repository.UpdateWithMap(uow, url, map[string]interface{}{"Visits": url.Visits})
	if err != nil {
		uow.RollBack()
		return "", err
	}

	uow.Commit()
	return url.LongUrl, nil
}
