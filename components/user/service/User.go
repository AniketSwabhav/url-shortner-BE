package service

import (
	"fmt"
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	"url-shortner-be/model/credential"
	"url-shortner-be/model/user"
	"url-shortner-be/module/repository"

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

// type RegisterRequest struct {
// 	FirstName string `json:"firstName"`
// 	LastName  string `json:"lastName"`
// 	PhoneNo   string `json:"phoneNo"`
// 	Email     string `json:"email"`
// 	Password  string `json:"password"`
// 	IsAdmin   *bool  `json:"isAdmin"`
// 	IsActive  *bool  `json:"isActive"`
// }

// ---------------- Register ----------------
// func (s *UserService) Register(req RegisterRequest) (*user.User, error) {
// 	if req.Email == "" || req.Password == "" {
// 		return nil, errors.InvalidCredentials()
// 	}

// 	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
// 	if err != nil {
// 		return nil, errors.InvalidCredentials()
// 	}

// 	uow := repository.NewUnitOfWork(s.db, false)
// 	defer uow.RollBack()

// 	newUser := &user.User{
// 		FirstName: req.FirstName,
// 		LastName:  req.LastName,
// 		PhoneNo:   req.PhoneNo,
// 		IsAdmin:   req.IsAdmin,
// 		IsActive:  req.IsActive,
// 		Wallet:    0,
// 		UrlCount:  0,
// 	}

// 	if err := s.repository.Add(uow, newUser); err != nil {
// 		return nil, err
// 	}

// 	cred := &credential.Credential{
// 		Email:    req.Email,
// 		Password: string(hashedPwd),
// 		UserID:   newUser.ID,
// 	}

// 	if err := s.repository.Add(uow, cred); err != nil {
// 		return nil, err
// 	}

// 	uow.Commit()
// 	newUser.Credentials = cred
// 	return newUser, nil
// }

func (service *UserService) CreateAdmin(newUser *user.User) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	err := service.doesEmailExists(newUser.Credentials.Email)
	if err != nil {
		return err
	}

	if err = newUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		uow.RollBack()
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

	err = uow.DB.Create(newUser).Error
	if err != nil {
		return errors.NewDatabaseError("Failed to create user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) CreateUser(newUser *user.User) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	err := service.doesEmailExists(newUser.Credentials.Email)
	if err != nil {
		return err
	}

	if err = newUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		uow.RollBack()
		return err
	}

	// subscriptionPlan := &subscription.Subscription{}
	// if err = service.repository.GetRecord(uow, &subscriptionPlan, repository.Order("created_at desc")); err != nil {
	// 	uow.RollBack()
	// 	return err
	// }
	// newUser.UrlCount = subscriptionPlan.FreeUrlLimit

	if err := newUser.Credentials.Validate(); err != nil {
		return err
	}

	hashedPassword, err := hashPassword(newUser.Credentials.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	newUser.Credentials.Password = string(hashedPassword)

	err = uow.DB.Create(newUser).Error
	if err != nil {
		return errors.NewDatabaseError("Failed to create user")
	}

	uow.Commit()
	return nil
}

func (service *UserService) Login(userCredential *credential.Credential, claim *security.Claims) error {

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	exists, err := repository.DoesEmailExist(service.db, userCredential.Email, credential.Credential{},
		repository.Filter("`email` = ?", userCredential.Email))
	if err != nil {
		return errors.NewDatabaseError("Error checking if email exists")
	}
	if !exists {
		return errors.NewNotFoundError("Email not found")
	}

	foundCredential := credential.Credential{}
	err = uow.DB.Where("email = ?", userCredential.Email).First(&foundCredential).Error
	if err != nil {
		return errors.NewDatabaseError("Could not retrieve credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundCredential.Password), []byte(userCredential.Password))
	if err != nil {
		return errors.NewInValidPasswordError("Incorrect password")
	}

	foundUser := user.User{}
	err = uow.DB.Preload("Credentials").
		Where("id = ?", foundCredential.UserID).First(&foundUser).Error

	if err != nil {
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

// func (s *UserService) doesUserExist(id uuid.UUID) error {
// 	uow := repository.NewUnitOfWork(s.db, true)
// 	defer uow.RollBack()

// 	var u user.User
// 	if err := s.repository.GetRecordByID(uow, id, &u); err != nil {
// 		return errors.NewNotFoundError("user not found")
// 	}
// 	return nil
// }

// // ---------------- Get All ----------------
// func (s *UserService) GetAllUsers() ([]user.User, error) {
// 	var users []user.User
// 	uow := repository.NewUnitOfWork(s.db, true)
// 	defer uow.RollBack()

// 	if err := s.repository.GetAll(uow, &users); err != nil {
// 		return nil, err
// 	}
// 	return users, nil
// }

// // ---------------- Get By ID ----------------
// func (s *UserService) GetUserByID(id uuid.UUID) (*user.User, error) {
// 	var u user.User
// 	uow := repository.NewUnitOfWork(s.db, true)
// 	defer uow.RollBack()

// 	if err := s.repository.GetRecordByID(uow, id, &u); err != nil {
// 		return nil, err
// 	}
// 	return &u, nil
// }

// // ---------------- Update ----------------
// func (s *UserService) UpdateUser(userToUpdate *user.User, email string, password string) error {
// 	if err := s.doesUserExist(userToUpdate.ID); err != nil {
// 		return err
// 	}

// 	uow := repository.NewUnitOfWork(s.db, false)
// 	defer uow.RollBack()

// 	if err := s.repository.UpdateWithMap(uow, &user.User{}, map[string]interface{}{
// 		"first_name": userToUpdate.FirstName,
// 		"last_name":  userToUpdate.LastName,
// 		"phone_no":   userToUpdate.PhoneNo,
// 		"is_admin":   userToUpdate.IsAdmin,
// 		"is_active":  userToUpdate.IsActive,
// 		"updated_by": userToUpdate.UpdatedBy,
// 		"updated_at": time.Now(),
// 	}, repository.Filter("id = ?", userToUpdate.ID)); err != nil {
// 		return err
// 	}

// 	if email != "" || password != "" {
// 		updateData := map[string]interface{}{
// 			"updated_by": userToUpdate.UpdatedBy,
// 			"updated_at": time.Now(),
// 		}
// 		if email != "" {
// 			updateData["email"] = email
// 		}
// 		if password != "" {
// 			hashedPassword, err := claims.HashPassword(password)
// 			if err != nil {
// 				return errors.NewValidationError("failed to hash password")
// 			}
// 			updateData["password"] = hashedPassword
// 		}

// 		if err := s.repository.UpdateWithMap(
// 			uow,
// 			&credential.Credential{},
// 			updateData,
// 			repository.Filter("user_id = ?", userToUpdate.ID),
// 		); err != nil {
// 			return err
// 		}
// 	}

// 	uow.Commit()
// 	return nil
// }

// // ---------------- Delete ----------------
// func (s *UserService) DeleteUser(id uuid.UUID) error {
// 	uow := repository.NewUnitOfWork(s.db, false)
// 	defer uow.RollBack()

// 	if err := uow.DB.Where("id = ?", id).Delete(&user.User{}).Error; err != nil {
// 		return err
// 	}

// 	uow.Commit()
// 	return nil
// }

// // -------- Login --------
// type LoginRequest struct {
// 	Email    string `json:"email"`
// 	Password string `json:"password"`
// }

// type LoginResponse struct {
// 	Message string `json:"message"`
// 	Token   string `json:"token"`
// }

// func (s *UserService) Login(req LoginRequest) (*LoginResponse, error) {
// 	if req.Email == "" || req.Password == "" {
// 		return nil, errors.InvalidCredentials()
// 	}

// 	uow := repository.NewUnitOfWork(s.db, true)
// 	defer uow.RollBack()

// 	var cred credential.Credential
// 	if err := s.repository.GetRecord(uow, &cred, repository.Filter("email = ?", req.Email)); err != nil {
// 		return nil, errors.InvalidCredentials()
// 	}

// 	if !claims.CheckPasswordHash(req.Password, cred.Password) {
// 		return nil, errors.InvalidCredentials()
// 	}

// 	var u user.User
// 	if err := s.repository.GetRecordByID(uow, cred.UserID, &u); err != nil {
// 		return nil, errors.NewNotFoundError("user not found for given credential")
// 	}

// 	isAdmin := u.IsAdmin != nil && *u.IsAdmin
// 	isActive := u.IsActive != nil && *u.IsActive

// 	token, err := claims.GenerateJWT(u.ID.String(), cred.Email, isAdmin, isActive)
// 	if err != nil {
// 		return nil, errors.NewValidationError("failed to generate token")
// 	}

// 	return &LoginResponse{
// 		Message: "login successful",
// 		Token:   token,
// 	}, nil
// }

func (service *UserService) doesEmailExists(Email string) error {
	exists, _ := repository.DoesEmailExist(service.db, Email, credential.Credential{},
		repository.Filter("`email` = ?", Email))
	if exists {
		return errors.NewValidationError("Email is already registered")
	}
	return nil
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), cost)
}
