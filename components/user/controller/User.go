package controller

import (
	"net/http"
	"strconv"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	"url-shortner-be/components/web"
	"url-shortner-be/model/credential"
	"url-shortner-be/model/stats"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/transaction"
	"url-shortner-be/model/user"

	userService "url-shortner-be/components/user/service"

	"github.com/gorilla/mux"
)

// brijesh
type UserController struct {
	log         log.Logger
	UserService *userService.UserService
}

func NewUserController(userService *userService.UserService, log log.Logger) *UserController {
	return &UserController{
		log:         log,
		UserService: userService,
	}
}

func (userController *UserController) RegisterRoutes(router *mux.Router) {

	userRouter := router.PathPrefix("/users").Subrouter()
	unguardedRouter := userRouter.PathPrefix("/").Subrouter()
	userguardedRouter := userRouter.PathPrefix("/").Subrouter()
	adminguardedRouter := userRouter.PathPrefix("/").Subrouter()

	commonRouter := userRouter.PathPrefix("/").Subrouter()

	unguardedRouter.HandleFunc("/login", userController.login).Methods(http.MethodPost)
	unguardedRouter.HandleFunc("/register-user", userController.registerUser).Methods(http.MethodPost)
	unguardedRouter.HandleFunc("/register-admin", userController.registerAdmin).Methods(http.MethodPost)

	userguardedRouter.HandleFunc("/{userId}/wallet/add", userController.addAmountToWallet).Methods(http.MethodPost)
	userguardedRouter.HandleFunc("/{userId}/wallet/withdraw", userController.withdrawAmountFromWallet).Methods(http.MethodPost)
	userguardedRouter.HandleFunc("/{userId}/renew-urls", userController.renewUrlsByUserId).Methods(http.MethodPost)
	userguardedRouter.HandleFunc("/{userId}/amount", userController.getwalletAmount).Methods(http.MethodGet)

	commonRouter.HandleFunc("/{userId}", userController.getUserByID).Methods(http.MethodGet)
	commonRouter.HandleFunc("/{userId}/transactions", userController.getTransactionByUserId).Methods(http.MethodGet)
	commonRouter.HandleFunc("/{userId}", userController.updateUserById).Methods(http.MethodPut)

	adminguardedRouter.HandleFunc("/", userController.getAllUsers).Methods(http.MethodGet)
	adminguardedRouter.HandleFunc("/monthwise-records", userController.getMonthWiseRecords).Methods(http.MethodGet)
	adminguardedRouter.HandleFunc("/report", userController.getReportStats).Methods(http.MethodGet)
	adminguardedRouter.HandleFunc("/{userId}", userController.deleteUserById).Methods(http.MethodDelete)
	adminguardedRouter.HandleFunc("/{userId}/subcription", userController.getSubscription).Methods(http.MethodGet)
	// adminguardedRouter.HandleFunc("/{userId}/all-user-transactions", userController.getAllUserTransactions).Methods(http.MethodGet)

	userguardedRouter.Use(security.MiddlewareUser)
	adminguardedRouter.Use(security.MiddlewareAdmin)
	commonRouter.Use(security.MiddlewareCommon)

}

func (controller *UserController) registerAdmin(w http.ResponseWriter, r *http.Request) {
	newAdmin := user.User{}

	err := web.UnmarshalJSON(r, &newAdmin)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	if err := newAdmin.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.CreateAdmin(&newAdmin); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusCreated, newAdmin)
}

func (controller *UserController) registerUser(w http.ResponseWriter, r *http.Request) {
	newUser := user.User{}

	err := web.UnmarshalJSON(r, &newUser)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	if err := newUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.CreateUser(&newUser); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusCreated, newUser)
}

func (controller *UserController) login(w http.ResponseWriter, r *http.Request) {

	userCredentials := credential.Credential{}
	claim := security.Claims{}

	err := web.UnmarshalJSON(r, &userCredentials)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	if err = controller.UserService.Login(&userCredentials, &claim); err != nil {
		web.RespondError(w, err)
		return
	}

	token, err := claim.GenerateToken()
	if err != nil {
		web.RespondError(w, err)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)

	web.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Login successful",
		"token":   token,
	})
}

func (controller *UserController) getUserByID(w http.ResponseWriter, r *http.Request) {
	var targetUser = &user.UserDTO{}

	parser := web.NewParser(r)

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	targetUser.ID = userIdFromURL

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.GetUserByID(targetUser, userIdFromToken); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, targetUser)
}

func (controller *UserController) updateUserById(w http.ResponseWriter, r *http.Request) {
	var targetUser = &user.User{}
	parser := web.NewParser(r)

	err := web.UnmarshalJSON(r, &targetUser)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if err := targetUser.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
	}

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	targetUser.ID = userIdFromURL

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	targetUser.UpdatedBy = userIdFromToken

	// if userIdFromURL != userIdFromToken {
	// 	web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to update the user"))
	// 	return
	// }

	if err = controller.UserService.UpdateUser(targetUser); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "User updated successfully",
	})
}

func (controller *UserController) getAllUsers(w http.ResponseWriter, r *http.Request) {
	allUsers := []user.UserDTO{}
	var totalCount int
	parser := web.NewParser(r)

	if err := controller.UserService.GetAllUsers(&allUsers, parser, &totalCount); err != nil {
		controller.log.Print(err.Error())
		web.RespondError(w, err)
		return
	}
	web.RespondJSONWithXTotalCount(w, http.StatusOK, totalCount, allUsers)
}

func (controller *UserController) deleteUserById(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	userID, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	deletedBy, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.Delete(userID, deletedBy); err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

func (controller *UserController) addAmountToWallet(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	userIdFromUrl, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	userToAddMoney := &user.User{}
	err = web.UnmarshalJSON(r, &userToAddMoney)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if userToAddMoney.Wallet > 1000000.00 {
		web.RespondErrorMessage(w, http.StatusInternalServerError, "add amount must not be greater than 1000000")
		return
	}

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	userToAddMoney.ID = userIdFromToken

	if err = controller.UserService.AddAmountToWalllet(userIdFromUrl, userToAddMoney); err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Amount added successfully",
	})
}

func (controller *UserController) withdrawAmountFromWallet(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	userIdFromUrl, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	userToWithdrawMoney := &user.User{}
	err = web.UnmarshalJSON(r, &userToWithdrawMoney)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if userToWithdrawMoney.Wallet > 1000000.00 {
		web.RespondErrorMessage(w, http.StatusInternalServerError, "withdrawal amount must not be greater than 1000000")
		return
	}

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	userToWithdrawMoney.ID = userIdFromToken

	if err = controller.UserService.WithdrawMoneyFromWallet(userIdFromUrl, userToWithdrawMoney); err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Amount removed successfully",
	})
}

func (controller *UserController) getTransactionByUserId(w http.ResponseWriter, r *http.Request) {
	transactions := []transaction.Transaction{}
	var totalCount int
	parser := web.NewParser(r)

	userIdFromUrl, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.GetAllTransactions(&transactions, &totalCount, parser, userIdFromUrl, userIdFromToken); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSONWithXTotalCount(w, http.StatusOK, totalCount, transactions)
}

func (controller *UserController) getwalletAmount(w http.ResponseWriter, r *http.Request) {
	user := user.UserDTO{}
	parser := web.NewParser(r)

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if userIdFromToken != userIdFromURL {
		web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to view transactions for this user"))
	}

	user.ID = userIdFromURL

	if err := controller.UserService.GetWalletAmount(&user); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, user.Wallet)
}

func (controller *UserController) getSubscription(w http.ResponseWriter, r *http.Request) {

	subscriptions := []subscription.Subscription{}

	var totalCount int
	parser := web.NewParser(r)
	query := r.URL.Query()

	userId, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	if err = controller.UserService.GetSubscription(&subscriptions, &totalCount, limit, offset, userId); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSONWithXTotalCount(w, http.StatusOK, totalCount, subscriptions)
}

func (controller *UserController) renewUrlsByUserId(w http.ResponseWriter, r *http.Request) {
	userToUpdate := user.User{}
	parser := web.NewParser(r)

	err := web.UnmarshalJSON(r, &userToUpdate)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	userToUpdate.ID, err = parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	userToUpdate.UpdatedBy, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if err = controller.UserService.RenewUrls(&userToUpdate); err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Urls Renewed Successfully",
	})
}

func (controller *UserController) getMonthWiseRecords(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	value := query.Get("value")
	yearStr := query.Get("year")
	monthStr := query.Get("month")

	// Validate year
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	// Validate month
	// strconv.Atoi ->string to int convert
	monthInt, err := strconv.Atoi(monthStr)
	if err != nil || monthInt < 1 || monthInt > 12 {
		http.Error(w, "Invalid month", http.StatusBadRequest)
		return
	}

	// Call appropriate service
	var stats []stats.MonthlyStat
	switch value {
	case "new-users":
		stats, err = controller.UserService.GetMonthlyStats("users", "created_at", year, "")
	case "active-users":
		stats, err = controller.UserService.GetMonthlyStats("users", "created_at", year, "AND is_active = 1")
	case "urls-generated":
		stats, err = controller.UserService.GetMonthlyStats("urls", "created_at", year, "")
	case "urls-renewed":
		stats, err = controller.UserService.GetMonthlyStats("transactions", "created_at", year, "AND amount > 0")
	case "total-revenue":
		stats, err = controller.UserService.GetMonthlyRevenue(year)
	default:
		http.Error(w, "Invalid value type", http.StatusBadRequest)
		return
	}

	if err != nil {
		web.RespondErrorMessage(w, http.StatusInternalServerError, "Error fetching stats")
		return
	}

	// Month name mapping
	months := [...]string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}

	monthName := months[monthInt-1]
	var valueForMonth float64 = 0.0

	for _, stat := range stats {
		if stat.Month == monthInt {
			valueForMonth = stat.Value
			break
		}
	}

	// Final response
	response := map[string]interface{}{
		"month": monthName,
		"value": valueForMonth,
	}

	web.RespondJSON(w, http.StatusOK, response)
}

func (c *UserController) getReportStats(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	stats, err := c.UserService.GetReportStats(year)
	if err != nil {
		web.RespondErrorMessage(w, http.StatusInternalServerError, "Error fetching stats")
		return
	}

	web.RespondJSON(w, http.StatusOK, stats)
}
