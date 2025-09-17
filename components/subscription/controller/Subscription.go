package controller

import (
	"fmt"
	"net/http"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	subscriptionService "url-shortner-be/components/subscription/service"
	"url-shortner-be/components/web"
	"url-shortner-be/model/subscription"

	"github.com/gorilla/mux"
)

type SubscriptionController struct {
	log                 log.Logger
	SubscriptionService *subscriptionService.SubscriptionService
}

func NewSubscriptionrController(userService *subscriptionService.SubscriptionService, log log.Logger) *SubscriptionController {
	return &SubscriptionController{
		log:                 log,
		SubscriptionService: userService,
	}
}

func (SubscriptionController *SubscriptionController) RegisterRoutes(router *mux.Router) {

	userRouter := router.PathPrefix("/user/{userId}").Subrouter()
	guardedRouter := userRouter.PathPrefix("/").Subrouter()

	guardedRouter.HandleFunc("/subscriptions", SubscriptionController.setSubscriptionPrice).Methods(http.MethodPost)
	guardedRouter.HandleFunc("/subscriptions", SubscriptionController.getSubscriptionPrice).Methods(http.MethodGet)

	guardedRouter.Use(security.MiddlewareAdmin)
}

func (controller *SubscriptionController) setSubscriptionPrice(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	subscriptionPrices := subscription.Subscription{}
	err := web.UnmarshalJSON(r, &subscriptionPrices)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	subscriptionPrices.CreatedBy, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	subscriptionPrices.UserId = userIdFromURL

	err = controller.SubscriptionService.SetPrice(&subscriptionPrices)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	web.RespondJSON(w, http.StatusCreated, subscriptionPrices)
}

func (controller *SubscriptionController) getSubscriptionPrice(w http.ResponseWriter, r *http.Request) {

	subscriptionPrices := subscription.Subscription{}

	err := controller.SubscriptionService.GetPrice(&subscriptionPrices)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	web.RespondJSON(w, http.StatusCreated, subscriptionPrices)
}
