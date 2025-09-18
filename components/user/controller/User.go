package controller

import (
	"net/http"
	"strconv"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	"url-shortner-be/components/web"
	"url-shortner-be/model/credential"
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
	// router.HandleFunc("/login", userController.Login).Methods(http.MethodPost)

	userRouter := router.PathPrefix("/user").Subrouter()
	guardedRouter := userRouter.PathPrefix("/").Subrouter()
	unguardedRouter := userRouter.PathPrefix("/").Subrouter()
	commonRouter := userRouter.PathPrefix("/").Subrouter()

	unguardedRouter.HandleFunc("/login", userController.login).Methods(http.MethodPost)

	unguardedRouter.HandleFunc("/register-user", userController.registerUser).Methods(http.MethodPost)
	unguardedRouter.HandleFunc("/register-admin", userController.registerAdmin).Methods(http.MethodPost)

	guardedRouter.HandleFunc("/", userController.GetAllUsers).Methods(http.MethodGet)
	guardedRouter.HandleFunc("/{userId}", userController.GetUserByID).Methods(http.MethodGet)
	guardedRouter.HandleFunc("/{userId}", userController.UpdateUser).Methods(http.MethodPut)
	guardedRouter.HandleFunc("/{userId}", userController.deleteUserById).Methods(http.MethodDelete)

	guardedRouter.HandleFunc("/{userId}/wallet/add", userController.AddAmountToWallet).Methods(http.MethodPost)
	guardedRouter.HandleFunc("/{userId}/wallet/withdraw", userController.WithdrawAmountFromWallet).Methods(http.MethodPost)
	// guardedRouter.HandleFunc("/{userId}/amount", userController.GetAmount).Methods(http.MethodGet)
	// guardedRouter.HandleFunc("/{userId}/transactions", userController.GetAllTransactions).Methods(http.MethodGet)
	// guardedRouter.HandleFunc("/{userId}/subcription", userController.getSubscription).Methods(http.MethodGet)

	// guardedRouter.HandleFunc("/{userId}/renew-urls", userController.RenewUrlsByUser).Methods(http.MethodPost)

	commonRouter.Use(security.MiddlewareUser)
}

func (controller *UserController) registerAdmin(w http.ResponseWriter, r *http.Request) {
	newAdmin := user.User{}

	err := web.UnmarshalJSON(r, &newAdmin)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	err = controller.UserService.CreateAdmin(&newAdmin)
	if err != nil {
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

	err = controller.UserService.CreateUser(&newUser)
	if err != nil {
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

	err = controller.UserService.Login(&userCredentials, &claim)
	if err != nil {
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

func (controller *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
	var targetUser = &user.UserDTO{}

	parser := web.NewParser(r)

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	targetUser.ID = userIdFromURL

	err = controller.UserService.GetUserByID(targetUser)
	if err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, targetUser)
}

func (controller *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var targetUser = &user.User{}

	parser := web.NewParser(r)

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	targetUser.ID = userIdFromURL

	err = web.UnmarshalJSON(r, &targetUser)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	targetUser.UpdatedBy, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	err = controller.UserService.UpdateUser(targetUser)
	if err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "User updated successfully",
	})
}

func (controller *UserController) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	allUsers := &[]user.UserDTO{}
	var totalCount int
	query := r.URL.Query()

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

	err = controller.UserService.GetAllUsers(allUsers, &totalCount, limit, offset)
	if err != nil {
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

	err = controller.UserService.Delete(userID, deletedBy)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

func (controller *UserController) AddAmountToWallet(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	userId, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	var req user.User
	err = web.UnmarshalJSON(r, &req)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if req.Wallet <= 0 {
		web.RespondError(w, errors.NewValidationError("Amount must be greater than zero"))
		return
	}

	err = controller.UserService.AddAmountToWalllet(userId, req.Wallet)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Amount added successfully",
	})
}

func (controller *UserController) WithdrawAmountFromWallet(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	userId, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}

	var req user.User
	err = web.UnmarshalJSON(r, &req)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if req.Wallet <= 0 {
		web.RespondError(w, errors.NewValidationError("Amount must be greater than zero"))
		return
	}

	err = controller.UserService.WithdrawAmountFromWallet(userId, req.Wallet)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Amount withdrawn successfully",
	})
}
