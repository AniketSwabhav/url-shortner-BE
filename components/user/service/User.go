package service

import (
	"fmt"
	"net/http"
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/security"
	transactionserv "url-shortner-be/components/transaction/service"
	"url-shortner-be/model/credential"
	"url-shortner-be/model/stats"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/transaction"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

	uuid "github.com/satori/go.uuid"

	"github.com/golang-jwt/jwt"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

const cost = 10

type UserService struct {
	db                 *gorm.DB
	repository         repository.Repository
	transactionservice *transactionserv.TransactionService
}

func NewUserService(DB *gorm.DB, repo repository.Repository, txService *transactionserv.TransactionService) *UserService {
	return &UserService{
		db:                 DB,
		repository:         repo,
		transactionservice: txService,
	}
}

func (service *UserService) CreateAdmin(newUser *user.User) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if err := service.doesEmailExists(newUser.Credentials.Email); err != nil {
		return err
	}

	if err := newUser.Credentials.Validate(); err != nil {
		return err
	}

	if newUser.IsAdmin == nil {
		newUser.IsAdmin = new(bool)
	}
	*newUser.IsAdmin = true

	hashedPassword, err := hashPassword(newUser.Credentials.Password)
	if err != nil {
		return errors.NewHTTPError("failed to hash password", http.StatusInternalServerError)
	}
	newUser.Credentials.Password = string(hashedPassword)

	if err := uow.DB.Create(newUser).Error; err != nil {
		return errors.NewDatabaseError("Failed to create user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) CreateUser(newUser *user.User) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if err := service.doesEmailExists(newUser.Credentials.Email); err != nil {
		return err
	}

	if err := newUser.Credentials.Validate(); err != nil {
		return err
	}

	if newUser.IsAdmin == nil {
		newUser.IsAdmin = new(bool)
	}
	*newUser.IsAdmin = false

	hashedPassword, err := hashPassword(newUser.Credentials.Password)
	if err != nil {
		return errors.NewHTTPError("failed to hash password", http.StatusInternalServerError)
	}
	newUser.Credentials.Password = string(hashedPassword)

	subscription := &subscription.Subscription{}
	err = service.repository.GetRecord(uow, &subscription)
	if err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("Admin has not set subscription price yet, please contact admin")
	}
	newUser.UrlCount = subscription.FreeShortUrls

	if err := uow.DB.Create(newUser).Error; err != nil {
		return errors.NewDatabaseError("Failed to create user")
	}

	//transaction--------------------------------------------------------------------------------------------------
	var transactionType = "ACCOUNTCREATION"
	var note = fmt.Sprintf("First %d Free Url's limit is added to account, with %d Free visits per URL", subscription.FreeShortUrls, subscription.FreeVisits)

	if err := service.transactionservice.CreateTransaction(uow, newUser.ID, 0.0, transactionType, note); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}

func (service *UserService) Login(userCredential *credential.Credential, claim *security.Claims) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	exists, err := repository.DoesEmailExist(service.db, userCredential.Email, credential.Credential{},
		repository.Filter("email = ?", userCredential.Email))
	if err != nil {
		return errors.NewDatabaseError("Error checking if email exists")
	}
	if !exists {
		return errors.NewNotFoundError("Email not found")
	}

	var foundCredential credential.Credential
	if err := uow.DB.Where("email = ?", userCredential.Email).First(&foundCredential).Error; err != nil {
		return errors.NewDatabaseError("Could not retrieve credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundCredential.Password), []byte(userCredential.Password)); err != nil {
		return errors.NewInValidPasswordError("Incorrect password")
	}

	var foundUser user.User
	if err := uow.DB.Preload("Credentials").Where("id = ?", foundCredential.UserID).First(&foundUser).Error; err != nil {
		return errors.NewDatabaseError("Could not retrieve user")
	}

	*claim = security.Claims{
		UserID:   foundUser.ID.String(),
		IsAdmin:  *foundUser.IsAdmin,
		IsActive: *foundUser.IsActive,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * time.Hour).Unix(),
		},
	}

	uow.Commit()
	return nil
}

func (service *UserService) GetUserByID(targetUser *user.UserDTO) error {
	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	err := service.repository.GetRecordByID(uow, targetUser.ID, targetUser, repository.PreloadAssociations([]string{"Credentials", "Url", "Transactions"}))
	if err != nil {
		return errors.NewDatabaseError("error getting user data")
	}

	uow.Commit()
	return nil
}

func (service *UserService) GetAllUsers(allUsers *[]user.UserDTO, totalCount *int, limit, offset int) error {
	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	err := service.repository.GetAll(
		uow,
		allUsers,
		repository.PreloadAssociations([]string{"Credentials", "Url", "Transactions"}),
		repository.Paginate(limit, offset, totalCount))
	if err != nil {
		return errors.NewDatabaseError("error getting all users")
	}

	err = service.repository.GetCount(uow, allUsers, totalCount)
	if err != nil {
		return err
	}

	uow.Commit()
	return nil
}

func (service *UserService) UpdateUser(targetUser *user.User) error {

	if err := service.doesUserExist(targetUser.ID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	targetUser.UpdatedAt = time.Now()

	if err := service.repository.Update(uow, targetUser, repository.Filter("id = ?", targetUser.ID)); err != nil {
		return errors.NewDatabaseError("unable to update user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) Delete(userID uuid.UUID, deletedBy uuid.UUID) error {
	if err := service.doesUserExist(userID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	now := time.Now()

	if err := service.repository.UpdateWithMap(uow, &user.User{}, map[string]interface{}{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}, repository.Filter("id = ?", userID)); err != nil {
		return errors.NewDatabaseError("unable to delete user")
	}

	if err := service.repository.UpdateWithMap(uow, &credential.Credential{}, map[string]interface{}{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}, repository.Filter("user_id = ?", userID)); err != nil {
		return errors.NewDatabaseError("unable to delete credentials")
	}

	uow.Commit()
	return nil
}

func (service *UserService) AddAmountToWalllet(userID uuid.UUID, userToAddMoney *user.User) error {

	if userToAddMoney.Wallet <= 0 {
		return errors.NewValidationError("Credit Amount must be greater than zero")
	}

	if err := service.doesUserExist(userID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	var dbUser user.User
	if err := service.repository.GetRecord(uow, &dbUser, repository.Filter("id = ?", userID)); err != nil {
		return errors.NewNotFoundError("User not found")
	}

	if dbUser.ID != userToAddMoney.ID {
		return errors.NewUnauthorizedError("you are not authorized to add amount to this wallet")
	}

	var amount = userToAddMoney.Wallet

	dbUser.Wallet += amount

	if err := service.repository.UpdateWithMap(uow, &dbUser,
		map[string]interface{}{
			"wallet":     dbUser.Wallet,
			"updated_at": time.Now(),
		},
		repository.Filter("id = ?", userID),
	); err != nil {
		return errors.NewDatabaseError("Failed to update wallet")
	}

	//transaction--------------------------------------------------------------------------------------------------
	var transactionType = "CREDIT"
	var note = fmt.Sprintf("%0.2f added in the wallet", amount)

	if err := service.transactionservice.CreateTransaction(uow, userToAddMoney.ID, amount, transactionType, note); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}
func (service *UserService) WithdrawMoneyFromWallet(userID uuid.UUID, userToWthdrawMoney *user.User) error {

	if userToWthdrawMoney.Wallet <= 0 {
		return errors.NewValidationError("Wihdraw amount must be greater than zero")
	}

	if err := service.doesUserExist(userID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	var dbUser user.User
	if err := service.repository.GetRecord(uow, &dbUser, repository.Filter("id = ?", userID)); err != nil {
		return errors.NewNotFoundError("User not found")
	}

	if dbUser.ID != userToWthdrawMoney.ID {
		return errors.NewUnauthorizedError("you are not authorized to withdraw amount from wallet")
	}

	var amount = userToWthdrawMoney.Wallet

	dbUser.Wallet -= amount

	if err := service.repository.UpdateWithMap(uow, &dbUser,
		map[string]interface{}{
			"wallet":     dbUser.Wallet,
			"updated_at": time.Now(),
		},
		repository.Filter("id = ?", userID),
	); err != nil {
		return errors.NewDatabaseError("Failed to update wallet")
	}

	//transaction--------------------------------------------------------------------------------------------------
	var transactionType = "DEBIT"
	var note = fmt.Sprintf("%0.2f removed from the wallet", amount)

	if err := service.transactionservice.CreateTransaction(uow, userToWthdrawMoney.ID, amount, transactionType, note); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}

func (service *UserService) WithdrawAmountFromWallet(userID uuid.UUID, amount float32) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if err := service.doesUserExist(userID); err != nil {
		return err
	}

	var dbUser user.User
	if err := service.repository.GetRecord(
		uow,
		&dbUser,
		repository.Filter("id = ?", userID),
	); err != nil {
		return errors.NewNotFoundError("User not found")
	}

	if dbUser.Wallet < amount {
		return errors.NewValidationError("Insufficient balance")
	}

	dbUser.Wallet -= amount

	if err := service.repository.UpdateWithMap(
		uow,
		&dbUser,
		map[string]interface{}{
			"wallet":     dbUser.Wallet,
			"updated_at": time.Now(),
		},
		repository.Filter("id = ?", userID),
	); err != nil {
		return errors.NewDatabaseError("Failed to update wallet")
	}

	uow.Commit()
	return nil
}

func (service *UserService) GetAllTransactions(transactions *[]transaction.Transaction, totalCount *int, limit, offset int, userID uuid.UUID) error {

	if err := service.doesUserExist(userID); err != nil {
		return errors.NewDatabaseError("user not found with given id")
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetAll(
		uow,
		transactions,
		repository.Filter("user_id = ?", userID),
		repository.Paginate(limit, offset, totalCount),
		repository.Order("created_at desc"),
	); err != nil {
		return errors.NewDatabaseError("unable to fetch transactions for this user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) GetWalletAmount(user *user.UserDTO) error {

	if err := service.doesUserExist(user.ID); err != nil {
		return errors.NewDatabaseError("user not found with given id")
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetRecordByID(uow, user.ID, user); err != nil {
		return errors.NewDatabaseError("Unable to fetch wallet amount for this user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) GetSubscription(subscriptions *[]subscription.Subscription, totalCount *int, limit, offset int, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return errors.NewValidationError("User ID is not valid")
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if err := service.repository.GetAll(uow, subscriptions, repository.Filter("user_id = ?", userID), repository.Paginate(limit, offset, totalCount), repository.Order("created_at desc")); err != nil {
		return errors.NewDatabaseError("unable to fetch subscriptions for this user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) RenewUrls(userToUpdate *user.User) error {

	if err := service.doesUserExist(userToUpdate.ID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	if userToUpdate.UpdatedBy != userToUpdate.ID {
		return errors.NewUnauthorizedError("you are not authorized to renew urls for this user")
	}

	if userToUpdate.UrlCount <= 0 {
		return errors.NewValidationError("number of url renews should be a positive integer")
	}

	existingUser := &user.User{}
	if err := service.repository.GetRecordByID(uow, userToUpdate.ID, &existingUser); err != nil {
		return errors.NewDatabaseError("unable to find user")
	}

	subscription := &subscription.Subscription{}
	if err := service.repository.GetRecord(uow, &subscription, repository.Order("created_at desc")); err != nil {
		return errors.NewDatabaseError("unable to fetch subscription details")
	}

	totalPriceToRenew := float32(userToUpdate.UrlCount) * subscription.NewUrlPrice

	if existingUser.Wallet < totalPriceToRenew {
		return errors.NewValidationError("insufficient balance in wallet, please add money to wallet")
	}

	existingUser.Wallet -= totalPriceToRenew

	newUrlCount := userToUpdate.UrlCount + existingUser.UrlCount

	if err := service.repository.UpdateWithMap(uow, existingUser, map[string]interface{}{
		"wallet":     existingUser.Wallet,
		"url_count":  newUrlCount,
		"updated_by": userToUpdate.UpdatedBy,
	}); err != nil {
		uow.RollBack()
		return err
	}

	//transaction--------------------------------------------------------------------------------------------------
	var transactionType = "URLRENEWAL"
	var note = fmt.Sprintf("%d url renewed for %0.2f per url renewal price", userToUpdate.UrlCount, subscription.NewUrlPrice)

	if err := service.transactionservice.CreateTransaction(uow, existingUser.ID, totalPriceToRenew, transactionType, note); err != nil {
		uow.RollBack()
		return errors.NewDatabaseError("unable to create transaction")
	}

	uow.Commit()
	return nil
}

func (s *UserService) GetMonthlyStats(table string, column string, year int, extraFilter string) ([]stats.MonthlyStat, error) {
	var stats []stats.MonthlyStat

	query := fmt.Sprintf(`
		SELECT MONTH(%s) as month, COUNT(*) as value
		FROM %s
		WHERE YEAR(%s) = ? %s
		GROUP BY MONTH(%s)
		ORDER BY MONTH(%s)
	`, column, table, column, extraFilter, column, column)

	err := s.db.Raw(query, year).Scan(&stats).Error
	return stats, err
}

func (s *UserService) GetMonthlyRevenue(year int) ([]stats.MonthlyStat, error) {
	var stats []stats.MonthlyStat

	query := `
		SELECT MONTH(created_at) as month, SUM(amount) as value
		FROM transactions
		WHERE YEAR(created_at) = ?
		GROUP BY MONTH(created_at)
		ORDER BY MONTH(created_at)
	`

	err := s.db.Raw(query, year).Scan(&stats).Error
	return stats, err
}

// ---------------- Helpers ----------------
func (service *UserService) doesEmailExists(email string) error {
	exists, _ := repository.DoesEmailExist(service.db, email, credential.Credential{},
		repository.Filter("email = ?", email))
	if exists {
		return errors.NewValidationError("Email is already registered")
	}
	return nil
}

func (service *UserService) doesUserExist(ID uuid.UUID) error {
	var u user.User
	if err := service.db.First(&u, "id = ?", ID).Error; err != nil {
		return errors.NewValidationError("User ID is invalid")
	}
	return nil
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), cost)
}
