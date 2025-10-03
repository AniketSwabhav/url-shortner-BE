package service

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/security"
	transactionserv "url-shortner-be/components/transaction/service"
	"url-shortner-be/components/web"
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

type Values map[string][]string
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
	// var transactionType = "ACCOUNTCREATION"
	// var note = fmt.Sprintf("First %d Free Url's limit is added to account, with %d Free visits per URL", subscription.FreeShortUrls, subscription.FreeVisits)

	// if err := service.transactionservice.CreateTransaction(uow, newUser.ID, 0.0, transactionType, note); err != nil {
	// 	uow.RollBack()
	// 	return errors.NewDatabaseError("unable to create transaction")
	// }

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
		return errors.NewValidationError("Invalid credentials")
	}

	foundCredentials := credential.Credential{}
	if err := uow.DB.Where("email = ?", userCredential.Email).First(&foundCredentials).Error; err != nil {
		return errors.NewDatabaseError("Could not retrieve credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundCredentials.Password), []byte(userCredential.Password)); err != nil {
		return errors.NewHTTPError("Invalid credentials", http.StatusUnauthorized)
	}

	var foundUser user.User
	if err := uow.DB.Preload("Credentials").Where("id = ?", foundCredentials.UserID).First(&foundUser).Error; err != nil {
		return errors.NewDatabaseError("Could not retrieve user")
	}

	if !*foundUser.IsActive {
		errors.NewValidationError("user is inactive ")
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

func (service *UserService) GetUserByID(targetUser *user.UserDTO, tokenUserId uuid.UUID) error {
	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	// Step 1: Get token user (who is making the request)
	var requestingUser user.User
	if err := service.repository.GetRecordByID(uow, tokenUserId, &requestingUser); err != nil {
		return errors.NewUnauthorizedError("invalid user making the request")
	}

	// Step 2: Access control logic
	isAdmin := requestingUser.IsAdmin != nil && *requestingUser.IsAdmin
	isSameUser := targetUser.ID == tokenUserId

	if !isSameUser && !isAdmin {
		return errors.NewUnauthorizedError("you are not authorized to view this user data")
	}

	// Step 3: Fetch target user data
	err := service.repository.GetRecordByID(uow, targetUser.ID, targetUser,
		repository.PreloadAssociations([]string{"Credentials", "Url", "Transactions"}))
	if err != nil {
		return errors.NewDatabaseError("error getting user data")
	}

	// uow.Commit()
	return nil
}

func (service *UserService) GetAllUsers(allUsers *[]user.UserDTO, parser *web.Parser, totalCount *int) error {
	var queryProcessors []repository.QueryProcessor
	limit, offset := parser.ParseLimitAndOffset()

	uow := repository.NewUnitOfWork(service.db, true)
	defer uow.RollBack()

	// repository.PreloadAssociations([]string{"Credentials", "Url", "Transactions"}),
	queryProcessors = append(queryProcessors,
		service.addSearchQueries(parser.Form),
		repository.Paginate(limit, offset, totalCount))

	err := service.repository.GetAll(uow, allUsers, queryProcessors...)
	if err != nil {
		return errors.NewDatabaseError("error getting all users")
	}

	// uow.Commit()
	return nil
}

// func (service *UserService) addSearchQueries(requestForm url.Values) repository.QueryProcessor {

// 	var queryProcessors []repository.QueryProcessor

// 	if len(requestForm) == 0 {
// 		return nil
// 	}

// 	// var columnNames []string
// 	// var conditions []string
// 	// var operators []string
// 	// var values []interface{}

// 	// if _, ok := requestForm["firstName"]; ok {
// 	// 	util.AddToSlice("`first_name`", "LIKE ?", "OR", "%"+requestForm.Get("firstName")+"%", &columnNames, &conditions, &operators, &values)
// 	// }

// 	queryProcessors = append(queryProcessors, repository.Filter("first_name LIKE (?)", "%"+requestForm.Get("firstName")+"%"))

// 	return repository.CombineQueries(queryProcessors)
// }

func (service *UserService) addSearchQueries(requestForm url.Values) repository.QueryProcessor {
	searchTerm := requestForm.Get("search")
	if searchTerm == "" {
		return repository.QueryProcessor(func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
			return db.Find(out), nil
		})
	}

	return repository.QueryProcessor(func(db *gorm.DB, out interface{}) (*gorm.DB, error) {
		return db.Joins("JOIN credentials ON credentials.user_id = users.id").
			Where("users.first_name LIKE ? OR users.last_name LIKE ? OR credentials.email LIKE ?",
				"%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%").
			Find(out), nil
	})
}

func (service *UserService) UpdateUser(targetUser *user.User) error {

	if err := service.doesUserExist(targetUser.ID); err != nil {
		return err
	}

	// if err := service.doesEmailExists(targetUser.Email); err != nil {
	// 	return err
	// }

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	// targetUser.Credentials = &credential.Credential{}
	// if err := service.repository.GetRecord(uow, &targetUser.Credentials, repository.Filter("user_id = ?", targetUser.ID)); err != nil {
	// 	return errors.NewDatabaseError("unable to get credentials")
	// }
	// targetUser.Credentials.Email = targetUser.Email

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

	if dbUser.Wallet > 1000000000.00 {
		return errors.NewHTTPError("wallet balance must not exceed 1000000000.00", http.StatusInternalServerError)
	}

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
	// var transactionType = "CREDIT"
	// var note = fmt.Sprintf("%0.2f added in the wallet", amount)

	// if err := service.transactionservice.CreateTransaction(uow, userToAddMoney.ID, amount, transactionType, note); err != nil {
	// 	uow.RollBack()
	// 	return errors.NewDatabaseError("unable to create transaction")
	// }

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

	if userToWthdrawMoney.Wallet >= dbUser.Wallet {
		return errors.NewValidationError("insufficient balance to withdraw")
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
	// var transactionType = "DEBIT"
	// var note = fmt.Sprintf("%0.2f removed from the wallet", amount)

	// if err := service.transactionservice.CreateTransaction(uow, userToWthdrawMoney.ID, amount, transactionType, note); err != nil {
	// 	uow.RollBack()
	// 	return errors.NewDatabaseError("unable to create transaction")
	// }

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

func (service *UserService) GetAllTransactions(transactions *[]transaction.Transaction, totalCount *int, parser *web.Parser, userIdFromUrl, userIdFromToken uuid.UUID) error {

	if err := service.doesUserExist(userIdFromUrl); err != nil {
		return errors.NewDatabaseError("user not found with given id")
	}

	// var queryProcessors []repository.QueryProcessor
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

	if err := service.repository.GetAll(
		uow,
		transactions,
		repository.Filter("user_id = ?", actualUser.ID),
		repository.Paginate(limit, offset, totalCount),
		repository.Order("created_at desc"),
	); err != nil {
		return errors.NewDatabaseError("unable to fetch transactions for this user")
	}

	// uow.Commit()
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

	// uow.Commit()
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

	// uow.Commit()
	return nil
}

func (service *UserService) RenewUrls(userToUpdate *user.User) error {

	if err := service.doesUserExist(userToUpdate.ID); err != nil {
		return err
	}

	uow := repository.NewUnitOfWork(service.db, false)
	defer uow.RollBack()

	if userToUpdate.UpdatedBy != userToUpdate.ID {
		return errors.NewUnauthorizedError("you are not authorized to renew urls for this user")
	}
	// --1`
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
		 AND type In('URLRENEWAL','VISITSRENEWAL')
		GROUP BY MONTH(created_at)
		ORDER BY MONTH(created_at)
	`
	err := s.db.Raw(query, year).Scan(&stats).Error
	return stats, err
}

// func (s *UserService) GetReportStats(year int) ([]stats.ReportStats, error) {

// 	uow := repository.NewUnitOfWork(s.db, true)
// 	defer uow.RollBack()


// 	sql := `
// 		SELECT
// 	    MONTH(u.created_at) AS month,
// 	    COUNT(DISTINCT u.id) AS new_users,
// 	    COUNT(DISTINCT CASE WHEN u.is_active = 1 THEN u.id END) AS active_users,
// 	    COUNT(DISTINCT ur.id) AS urls_generated,
// 	    COUNT(DISTINCT t.id) AS urls_renewed,
// 	    COALESCE(SUM(CASE WHEN t.type IN ('URLRENEWAL','VISITSRENEWAL') THEN t.amount ELSE 0 END), 0) AS total_revenue
// 	FROM
// 	    users u
// 	LEFT JOIN urls ur ON MONTH(ur.created_at) = MONTH(u.created_at)
// 	LEFT JOIN transactions t ON MONTH(t.created_at) = MONTH(u.created_at)
// 	WHERE YEAR(u.created_at) = ?
// 	GROUP BY MONTH(u.created_at)
// 	ORDER BY MONTH(u.created_at);

//     `

// 	proc := repository.RawQuery(sql, year)

// 	var result []stats.ReportStats

// 	if err := s.repository.GetRaw(uow, &result, proc); err != nil {
// 		return nil, err
// 	}

// 	// Fill missing months
// 	m := make(map[int]stats.ReportStats)
// 	for _, r := range result {
// 		m[r.Month] = r
// 	}

// 	final := make([]stats.ReportStats, 0, 12)
// 	for i := 1; i <= 12; i++ {
// 		if r, ok := m[i]; ok {
// 			final = append(final, r)
// 		} else {
// 			final = append(final, stats.ReportStats{
// 				Month:         i,
// 				NewUsers:      0,
// 				ActiveUsers:   0,
// 				UrlsGenerated: 0,
// 				UrlsRenewed:   0,
// 				TotalRevenue:  0,
// 			})
// 		}
// 	}

// 	return final, nil
// }

func (s *UserService) GetReportStats(year int) ([]stats.ReportStats, error) {
	// Get stats from individual functions
	newUsers, err := s.GetMonthlyStats("users", "created_at", year, "")
	if err != nil {
		return nil, err
	}

	activeUsers, err := s.GetMonthlyStats("users", "created_at", year, "AND is_active = 1")
	if err != nil {
		return nil, err
	}

	urlsGenerated, err := s.GetMonthlyStats("urls", "created_at", year, "")
	if err != nil {
		return nil, err
	}

	urlsRenewed, err := s.GetMonthlyStats("transactions", "created_at", year, "AND type IN ('URLRENEWAL','VISITSRENEWAL')")
	if err != nil {
		return nil, err
	}

	revenue, err := s.GetMonthlyRevenue(year)
	if err != nil {
		return nil, err
	}

	// Combine results
	statsMap := make(map[int]stats.ReportStats)
	
	// Initialize all months
	for i := 1; i <= 12; i++ {
		statsMap[i] = stats.ReportStats{
			Month:         i,
			NewUsers:      0,
			ActiveUsers:   0,
			UrlsGenerated: 0,
			UrlsRenewed:   0,
			TotalRevenue:  0,
		}
	}

	// Populate with actual data
	for _, nu := range newUsers {
		if stat, exists := statsMap[nu.Month]; exists {
			stat.NewUsers = int(nu.Value)
			statsMap[nu.Month] = stat
		}
	}

	for _, au := range activeUsers {
		if stat, exists := statsMap[au.Month]; exists {
			stat.ActiveUsers = int(au.Value)
			statsMap[au.Month] = stat
		}
	}

	for _, ug := range urlsGenerated {
		if stat, exists := statsMap[ug.Month]; exists {
			stat.UrlsGenerated = int(ug.Value)
			statsMap[ug.Month] = stat
		}
	}

	for _, ur := range urlsRenewed {
		if stat, exists := statsMap[ur.Month]; exists {
			stat.UrlsRenewed = int(ur.Value)
			statsMap[ur.Month] = stat
		}
	}

	for _, rev := range revenue {
		if stat, exists := statsMap[rev.Month]; exists {
			stat.TotalRevenue = rev.Value
			statsMap[rev.Month] = stat
		}
	}

	// Convert to slice
	final := make([]stats.ReportStats, 0, 12)
	for i := 1; i <= 12; i++ {
		final = append(final, statsMap[i])
	}

	return final, nil
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
