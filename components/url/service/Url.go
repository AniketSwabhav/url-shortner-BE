package service

import (
	"time"
	"url-shortner-be/components/errors"
	transactionserv "url-shortner-be/components/transaction/service"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/url"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type UrlService struct {
	db                 *gorm.DB
	repository         repository.Repository
	transactionservice *transactionserv.TransactionService
}

func NewUrlService(DB *gorm.DB, repo repository.Repository, txService *transactionserv.TransactionService) *UrlService {
	return &UrlService{
		db:                 DB,
		repository:         repo,
		transactionservice: txService,
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
		return errors.NewDatabaseError("unable to fetch subscription details")
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
		return errors.NewDatabaseError("unable to create new url")
	}

	foundUser.UrlCount--

	if err := service.repository.UpdateWithMap(uow, foundUser, map[string]interface{}{"url_count": foundUser.UrlCount}); err != nil {
		return errors.NewDatabaseError("unable to update user url count")
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

func (service *UrlService) RenewUrlVisits(urlToRenew *url.Url) error {

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if urlToRenew.Visits <= 0 {
		return errors.NewValidationError("number of visits should be a positive integer")
	}

	urlOwner := &user.User{}
	if err := service.repository.GetRecordByID(uow, urlToRenew.UserID, urlOwner); err != nil {
		return errors.NewDatabaseError("unable to find url owner")
	}

	if urlOwner.ID != urlToRenew.UserID {
		return errors.NewUnauthorizedError("you are not authorized to renew url for this user")
	}

	existingUrl := &url.Url{}
	if err := service.repository.GetRecord(uow, existingUrl, repository.Filter("id = ? And user_id = ?", urlToRenew.ID, urlToRenew.UserID)); err != nil {
		return errors.NewValidationError("no url found for this user with given url id")
	}

	subscription := &subscription.Subscription{}
	if err := service.repository.GetRecord(uow, &subscription, repository.Order("created_at desc")); err != nil {
		return errors.NewDatabaseError("unable to fetch subscription details")
	}

	totalPriceToRenew := float32(urlToRenew.Visits) * subscription.ExtraVisitPrice

	if urlOwner.Wallet < totalPriceToRenew {
		return errors.NewValidationError("insufficient balance in wallet, please add money to wallet")
	}

	urlOwner.Wallet -= totalPriceToRenew

	newVisitCount := existingUrl.Visits + urlToRenew.Visits

	if err := service.repository.UpdateWithMap(uow, urlOwner, map[string]interface{}{
		"wallet": urlOwner.Wallet,
	}); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to update wallet balance")
	}

	if err := service.repository.UpdateWithMap(uow, existingUrl, map[string]interface{}{
		"visits":     newVisitCount,
		"updated_by": urlToRenew.UserID,
	}); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to renew url visits")
	}

	if err := service.transactionservice.CreateTransaction(uow, urlOwner.ID, totalPriceToRenew); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}

func (service *UrlService) GetAllUrls(allUrl *[]url.UrlDTO, userId uuid.UUID, totalCount *int, limit, offset int) error {

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetAll(uow, allUrl, repository.Filter("user_id = ?", userId), repository.Paginate(limit, offset, totalCount)); err != nil {
		return errors.NewDatabaseError("error in fetching urls of user")
	}

	if err := service.repository.GetCount(uow, allUrl, totalCount, repository.Filter("user_id = ?", userId)); err != nil {
		return err
	}

	uow.Commit()
	return nil
}

func (service *UrlService) GetUrlByID(targetURL *url.UrlDTO) error {

	if err := service.doesUrlExist(targetURL.ID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetRecord(uow, targetURL, repository.Filter("id = ? And user_id = ?", targetURL.ID, targetURL.UserID)); err != nil {
		return errors.NewDatabaseError("no url found for this user with given url id")
	}

	return nil
}

func (service *UrlService) Delete(urlID uuid.UUID, deletedBy uuid.UUID) error {
	if err := service.doesUrlExist(urlID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	now := time.Now()

	if err := service.repository.UpdateWithMap(uow, &url.Url{}, map[string]interface{}{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}, repository.Filter("id = ?", urlID)); err != nil {
		return errors.NewDatabaseError("unable to delete url")
	}

	uow.Commit()
	return nil
}

func (service *UrlService) doesUrlExist(urlID uuid.UUID) error {
	var u url.Url
	if err := service.db.First(&u, "id = ?", urlID).Error; err != nil {
		return errors.NewValidationError("URL ID is invalid")
	}
	return nil
}
