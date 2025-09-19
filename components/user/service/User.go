package service

import (
	"fmt"
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	"url-shortner-be/model/credential"
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
	db         *gorm.DB
	repository repository.Repository
}

func NewUserService(DB *gorm.DB, repo repository.Repository) *UserService {
	return &UserService{
		db:         DB,
		repository: repo,
	}
}

func (service *UserService) CreateAdmin(newUser *user.User) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if err := service.doesEmailExists(newUser.Credentials.Email); err != nil {
		return err
	}

	if err := newUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
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
		return fmt.Errorf("failed to hash password: %w", err)
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

	if err := newUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		return err
	}

	if err := newUser.Credentials.Validate(); err != nil {
		return err
	}

	hashedPassword, err := hashPassword(newUser.Credentials.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	newUser.Credentials.Password = string(hashedPassword)

	subscription := &subscription.Subscription{}
	err = service.repository.GetRecord(uow, &subscription)
	if err != nil {
		uow.RollBack()
		return err
	}
	newUser.UrlCount = subscription.FreeShortUrls

	if err := uow.DB.Create(newUser).Error; err != nil {
		return errors.NewDatabaseError("Failed to create user")
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
		UserID:   foundUser.ID,
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

	var dbUser user.User
	if err := service.repository.GetRecordByID(uow, targetUser.ID, &dbUser, nil); err != nil {
		return err
	}

	*targetUser = user.ToUserDTO(&dbUser)
	return nil
}

func (service *UserService) GetAllUsers(allUsers *[]user.UserDTO, totalCount *int, limit, offset int) error {
	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	var dbUsers []user.User
	if err := service.repository.GetAll(
		uow,
		&dbUsers,
		repository.PreloadAssociations([]string{"Credentials"}),
		repository.Paginate(limit, offset, totalCount),
	); err != nil {
		return err
	}

	*allUsers = user.ToUserDTOs(dbUsers)
	uow.Commit()
	return nil
}

func (service *UserService) UpdateUser(targetUser *user.User) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	targetUser.UpdatedAt = time.Now()

	if err := service.repository.Update(
		uow,
		targetUser,
		repository.Filter("id = ?", targetUser.ID),
	); err != nil {
		return err
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
		return err
	}

	if err := service.repository.UpdateWithMap(uow, &credential.Credential{}, map[string]interface{}{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}, repository.Filter("user_id = ?", userID)); err != nil {
		return err
	}

	uow.Commit()
	return nil
}

func (service *UserService) AddAmountToWalllet(userID uuid.UUID, userToAddMoney *user.User) error {
	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if userToAddMoney.Wallet <= 0 {
		return errors.NewValidationError("Amount must be greater than zero")
	}

	if err := service.doesUserExist(userID); err != nil {
		return err
	}

	var dbUser user.User
	if err := service.repository.GetRecord(uow, &dbUser, repository.Filter("id = ?", userID)); err != nil {
		return errors.NewNotFoundError("User not found")
	}

	if dbUser.ID != userToAddMoney.ID {
		return errors.NewUnauthorizedError("you are not authorized to add amount to this wallet")
	}

	var amount = userToAddMoney.Wallet

	if amount <= 0 {
		return errors.NewValidationError("Amount must be greater than zero")
	}

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
	if userID == uuid.Nil {
		return errors.NewValidationError("User ID is not valid")
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
		return err
	}

	uow.Commit()
	return nil
}

// func (service *UserService) GetAllUsers(allUsers *[]user.UserDTO, totalCount *int, limit, offset int) error {
// 	uow := repository.NewUnitOfWork(service.db, true)
// 	defer uow.RollBack()

// 	var dbUsers []user.User
// 	if err := service.repository.GetAll(
// 		uow,
// 		&dbUsers,
// 		repository.PreloadAssociations([]string{"Credentials"}),
// 		repository.Paginate(limit, offset, totalCount),
// 	); err != nil {
// 		return err
// 	}

// 	*allUsers = user.ToUserDTOs(dbUsers)
// 	uow.Commit()
// 	return nil
// }

func (service *UserService) GetAllSubscription(subscriptions *[]subscription.Subscription, totalCount *int, page, pageSize int, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return errors.NewValidationError("User ID is not valid")
	}

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	offset := (page - 1) * pageSize

	if err := service.repository.GetAll(
		uow,
		subscriptions,
		repository.Filter("user_id = ?", userID),
		repository.Paginate(pageSize, offset, totalCount),
		repository.Order("created_at desc"),
	); err != nil {
		return err
	}

	uow.Commit()
	return nil
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
