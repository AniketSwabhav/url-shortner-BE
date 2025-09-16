package controller

import (
	"net/http"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	"url-shortner-be/components/web"
	"url-shortner-be/model/credential"
	"url-shortner-be/model/user"

	userService "url-shortner-be/components/user/service"

	"github.com/gorilla/mux"
)

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
	// guardedRouter := userRouter.PathPrefix("/").Subrouter()
	unguardedRouter := userRouter.PathPrefix("/").Subrouter()
	commonRouter := userRouter.PathPrefix("/").Subrouter()

	unguardedRouter.HandleFunc("/login", userController.login).Methods(http.MethodPost)
	unguardedRouter.HandleFunc("/register-user", userController.registerUser).Methods(http.MethodPost)
	unguardedRouter.HandleFunc("/register-admin", userController.registerAdmin).Methods(http.MethodPost)

	// router.HandleFunc("/register", userController.RegisterUser).Methods(http.MethodPost)
	// router.HandleFunc("/register/admin", userController.RegisterAdmin).Methods(http.MethodPost)
	// router.HandleFunc("/users", userController.GetAllUsers).Methods(http.MethodGet)
	// router.HandleFunc("/users/{id}", userController.GetUserByID).Methods(http.MethodGet)
	// router.HandleFunc("/users/{id}", userController.UpdateUser).Methods(http.MethodPut)
	// router.HandleFunc("/users/{id}", userController.DeleteUser).Methods(http.MethodDelete)

	commonRouter.Use(security.MiddlewareUser)
}

// -------- Register User --------
// func (uc *UserController) RegisterAdmin(w http.ResponseWriter, r *http.Request) {
// 	var req userService.RegisterRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	isAdmin := true
// 	isActive := true
// 	req.IsAdmin = &isAdmin
// 	req.IsActive = &isActive

// 	user, err := uc.UserService.Register(req)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(user)
// }

// func (uc *UserController) RegisterUser(w http.ResponseWriter, r *http.Request) {
// 	var req userService.RegisterRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	isAdmin := false
// 	isActive := true
// 	req.IsAdmin = &isAdmin
// 	req.IsActive = &isActive

// 	user, err := uc.UserService.Register(req)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(user)
// }

func (controller *UserController) registerAdmin(w http.ResponseWriter, r *http.Request) {
	newAdmin := user.User{}

	err := web.UnmarshalJSON(r, &newAdmin)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	newAdmin.CreatedBy, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	newAdmin.Credentials.CreatedBy = newAdmin.CreatedBy

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

	newUser.CreatedBy, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	newUser.Credentials.CreatedBy = newUser.CreatedBy

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

// -------- Get All --------
// func (uc *UserController) GetAllUsers(w http.ResponseWriter, r *http.Request) {
// 	users, err := uc.UserService.GetAllUsers()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	json.NewEncoder(w).Encode(users)
// }

// // -------- Get By ID --------
// func (uc *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
// 	idStr := mux.Vars(r)["id"]
// 	id, err := uuid.FromString(idStr)
// 	if err != nil {
// 		http.Error(w, "invalid user ID", http.StatusBadRequest)
// 		return
// 	}

// 	user, err := uc.UserService.GetUserByID(id)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusNotFound)
// 		return
// 	}
// 	json.NewEncoder(w).Encode(user)
// }

// // -------- Update --------
// func (uc *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
// 	idStr := mux.Vars(r)["id"]
// 	id, err := uuid.FromString(idStr)
// 	if err != nil {
// 		http.Error(w, "invalid user ID", http.StatusBadRequest)
// 		return
// 	}

// 	var u userService.RegisterRequest
// 	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
// 		http.Error(w, "invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	existing, err := uc.UserService.GetUserByID(id)
// 	if err != nil {
// 		http.Error(w, "user not found", http.StatusNotFound)
// 		return
// 	}

// 	existing.FirstName = u.FirstName
// 	existing.LastName = u.LastName
// 	existing.PhoneNo = u.PhoneNo
// 	existing.IsAdmin = u.IsAdmin
// 	existing.IsActive = u.IsActive

// 	if err := uc.UserService.UpdateUser(existing, u.Email, u.Password); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	json.NewEncoder(w).Encode(existing)
// }

// // -------- Delete --------
// func (uc *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
// 	idStr := mux.Vars(r)["id"]
// 	id, err := uuid.FromString(idStr)
// 	if err != nil {
// 		http.Error(w, "invalid user ID", http.StatusBadRequest)
// 		return
// 	}

// 	if err := uc.UserService.DeleteUser(id); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusNoContent)
// }

// // -------- Login --------
// func (uc *UserController) Login(w http.ResponseWriter, r *http.Request) {
// 	var req userService.LoginRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	resp, err := uc.UserService.Login(req)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusUnauthorized)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(resp)
// }
