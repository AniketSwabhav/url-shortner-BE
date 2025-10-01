package service

import (
	"fmt"
	"net/http"
	urlNet "net/url"
	"time"
	"url-shortner-be/components/errors"
	transactionserv "url-shortner-be/components/transaction/service"
	"url-shortner-be/components/web"
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

func NewUrlService(DB *gorm.DB, repo repository.Repository) *UrlService {

	var transactionService = transactionserv.NewTransactionService(DB, repo)
	return &UrlService{
		db:                 DB,
		repository:         repo,
		transactionservice: transactionService,
	}
}

func (service *UrlService) CreateUrl(userId uuid.UUID, urlOwner *user.User, newUrl *url.Url) error {

	if err := service.doesLongUrlExistsForCurrentUser(newUrl.LongUrl, userId); err != nil {
		return err
	}

	if err := service.doesShortUrlExists(newUrl.ShortUrl); err != nil {
		return err
	}

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

	newUrl.RemainingVisits = subscription.FreeVisits
	if foundUser.UrlCount == 0 {
		return errors.NewDatabaseError("maximum url creation limit is reached, purchase more for creating new url")
	}
	newUrl.UserID = foundUser.ID
	newUrl.RemainingVisits = subscription.FreeVisits

	// for {

	// 	newUrl.ShortUrl = url.GenerateShortUrl()

	// 	foundUrl := &url.Url{}
	// 	service.repository.GetRecord(uow, foundUrl, repository.Filter("short_url = ?", newUrl.ShortUrl))
	// 	if foundUrl.ShortUrl == newUrl.ShortUrl {
	// 		continue
	// 	} else {
	// 		break
	// 	}
	// }

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

// func (service *UrlService) RedirectToUrl(shortUrl string) (string, error) {

// 	uow := repository.NewUnitOfWork(service.db, false)
// 	defer uow.RollBack()

// 	url := &url.Url{}
// 	err := service.repository.GetRecord(uow, url, repository.Filter("short_url = ?", shortUrl))
// 	if err != nil {
// 		uow.RollBack()
// 		return "", err
// 	}

// 	if url.Visits == 0 {
// 		uow.RollBack()
// 		return "", errors.NewValidationError("no. of visits elapsed")
// 	}

// 	url.Visits--

// 	err = service.repository.UpdateWithMap(uow, url, map[string]interface{}{"Visits": url.Visits})
// 	if err != nil {
// 		uow.RollBack()
// 		return "", err
// 	}

// 	uow.Commit()
// 	return url.LongUrl, nil
// }

func (service *UrlService) RedirectToUrl(urlToRedirect *url.Url) error {
	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetRecord(uow, urlToRedirect, repository.Filter("short_url = ?", urlToRedirect.ShortUrl)); err != nil {
		return errors.NewDatabaseError("no short url matches the given short url")
	}

	if urlToRedirect.RemainingVisits == 0 {
		uow.RollBack()
		return errors.NewHTTPError("no. of visits reacheed it's limit, please renew the visits", http.StatusForbidden)
	}

	urlToRedirect.RemainingVisits--
	urlToRedirect.VisitCount++

	if err := service.repository.UpdateWithMap(uow, urlToRedirect, map[string]interface{}{
		"remainingVisits": urlToRedirect.RemainingVisits,
		"visitCount":      urlToRedirect.VisitCount}); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to update visits count")
	}

	// uow.Commit()
	return nil
}

func (service *UrlService) RenewUrlVisits(urlToRenew *url.Url) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	// if urlToRenew.RemainingVisits <= 0 {
	// 	return errors.NewValidationError("number of visits should be a positive integer")
	// }

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

	totalPriceToRenew := float32(urlToRenew.RemainingVisits) * subscription.ExtraVisitPrice

	if urlOwner.Wallet < totalPriceToRenew {
		return errors.NewValidationError("insufficient balance in wallet, please add money to wallet")
	}

	urlOwner.Wallet -= totalPriceToRenew

	newVisitCount := existingUrl.RemainingVisits + urlToRenew.RemainingVisits

	if err := service.repository.UpdateWithMap(uow, urlOwner, map[string]interface{}{
		"wallet": urlOwner.Wallet,
	}); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to update wallet balance")
	}

	if err := service.repository.UpdateWithMap(uow, existingUrl, map[string]interface{}{
		"remaining_visits": newVisitCount,
		"updated_by":       urlToRenew.UserID,
	}); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to renew url visits")
	}

	// //transaction--------------------------------------------------------------------------------------------------
	var transactionType = "VISITSRENEWAL"
	var note = fmt.Sprintf("%d visits renewed for %0.2f per visit price", urlToRenew.RemainingVisits, subscription.ExtraVisitPrice)

	if err := service.transactionservice.CreateTransaction(uow, urlOwner.ID, totalPriceToRenew, transactionType, note); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}

func (service *UrlService) GetAllUrls(allUrl *[]url.UrlDTO, totalCount *int, parser *web.Parser, userIdFromUrl, userIdFromToken uuid.UUID) error {

	var queryProcessors []repository.QueryProcessor
	limit, offset := parser.ParseLimitAndOffset()

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	actualUser := user.User{}
	if err := service.repository.GetRecordByID(uow, userIdFromUrl, &actualUser); err != nil {
		return errors.NewUnauthorizedError("invalid user making the request")
	}

	tokenUser := user.User{}
	if err := service.repository.GetRecordByID(uow, userIdFromToken, &tokenUser); err != nil {
		return errors.NewUnauthorizedError("invalid user making the request")
	}

	isAdmin := tokenUser.IsAdmin != nil && *tokenUser.IsAdmin
	isSameUser := actualUser.ID == tokenUser.ID

	if !isSameUser && !isAdmin {
		return errors.NewUnauthorizedError("you are not authorized to view this user data")
	}

	queryProcessors = append(queryProcessors, repository.Filter("user_id = ?", actualUser.ID),
		service.addSearchQueries(parser.Form),
		repository.Paginate(limit, offset, totalCount))

	if err := service.repository.GetAll(uow, allUrl, queryProcessors...); err != nil {
		return errors.NewDatabaseError("error in fetching urls of user")
	}

	// uow.Commit()
	return nil
}

// func (service *UserService) addSearchQueries(requestForm url.Values) repository.QueryProcessor {
// 	searchTerm := requestForm.Get("search")
// 	if searchTerm == "" {
// 		return repository.QueryProcessor(func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
// 			return db.Find(out), nil
// 		})
// 	}

// 	return repository.QueryProcessor(func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
// 		return db.Joins("JOIN credentials ON credentials.user_id = users.id").
// 			Where("users.first_name LIKE ? OR users.last_name LIKE ? OR credentials.email LIKE ?",
// 				"%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%").
// 			Find(out), nil
// 	})
// }

func (service *UrlService) addSearchQueries(requestForm urlNet.Values) repository.QueryProcessor {
	searchTerm := requestForm.Get("search")
	if searchTerm == "" {
		return repository.QueryProcessor(func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
			return db.Find(out), nil
		})
	}

	var queryProcessors []repository.QueryProcessor

	if len(requestForm) == 0 {
		return nil
	}

	queryProcessors = append(queryProcessors,
		repository.Filter("(long_url LIKE ? OR short_url LIKE ?)", "%"+searchTerm+"%", "%"+searchTerm+"%"),
	)

	return repository.CombineQueries(queryProcessors)
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

func (service *UrlService) GetUrlByShortUrl(originalUrl *url.UrlDTO) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if err := service.repository.GetRecord(uow, &originalUrl, repository.Filter("short_url = ? AND user_id = ?", originalUrl.ShortUrl, originalUrl.UserID)); err != nil {
		return errors.NewDatabaseError("no url found for this user with given short url")
	}

	uow.Commit()
	return nil
}

func (service *UrlService) UpdateUrl(targetUrl *url.Url) error {

	if err := service.doesUrlExist(targetUrl.ID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	targetUrl.UpdatedAt = time.Now()

	if err := service.repository.Update(uow, targetUrl, repository.Filter("id = ? AND user_id = ?", targetUrl.ID, targetUrl.UserID)); err != nil {
		return errors.NewDatabaseError("unable to update url")
	}

	uow.Commit()
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

//--------------------------------------------------------------------------------------------

func (service *UrlService) doesUrlExist(urlID uuid.UUID) error {
	var u url.Url
	if err := service.db.First(&u, "id = ?", urlID).Error; err != nil {
		return errors.NewValidationError("URL ID is invalid")
	}
	return nil
}

func (service *UrlService) doesLongUrlExistsForCurrentUser(longUrl string, userId uuid.UUID) error {
	exists, _ := repository.DoesLongUrlExist(service.db, longUrl, userId, url.Url{},
		repository.Filter("long_url = ?", longUrl))
	if exists {
		return errors.NewValidationError("Requested URL is already registered")
	}
	return nil
}

func (service *UrlService) doesShortUrlExists(shortUrl string) error {
	exists, _ := repository.DoesShortUrlExist(service.db, shortUrl, url.Url{},
		repository.Filter("short_url = ?", shortUrl))
	if exists {
		return errors.NewValidationError("This Short URL is already registered, try another pattern")
	}
	return nil
}
