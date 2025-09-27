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

	subscriptionRouter := router.PathPrefix("/url").Subrouter()
	guardedRouter := subscriptionRouter.PathPrefix("/").Subrouter()
	commonRouter := subscriptionRouter.PathPrefix("/").Subrouter()

	commonRouter.HandleFunc("/subscription", SubscriptionController.getSubscriptionPrice).Methods(http.MethodGet)

	guardedRouter.HandleFunc("/subscription", SubscriptionController.setSubscriptionPrice).Methods(http.MethodPost)
	guardedRouter.HandleFunc("/subscription/update", SubscriptionController.updateSubscriptionPrice).Methods(http.MethodPut)

	guardedRouter.Use(security.MiddlewareAdmin)
	commonRouter.Use(security.MiddlewareCommon)
}

func (controller *SubscriptionController) setSubscriptionPrice(w http.ResponseWriter, r *http.Request) {

	// parser := web.NewParser(r)
	subscriptionPrices := subscription.Subscription{}

	err := web.UnmarshalJSON(r, &subscriptionPrices)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	if err := subscriptionPrices.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		web.RespondError(w, err)
		return
	}

	// userIdFromURL, err := parser.GetUUID("userId")
	// if err != nil {
	// 	web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
	// 	return
	// }

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	subscriptionPrices.CreatedBy = userIdFromToken

	// if userIdFromURL != userIdFromToken {
	// 	web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to set subscription prizes"))
	// 	return
	// }

	if err = controller.SubscriptionService.SetSubscriptionPrice(&subscriptionPrices); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	web.RespondJSON(w, http.StatusCreated, subscriptionPrices)
}

func (controller *SubscriptionController) getSubscriptionPrice(w http.ResponseWriter, r *http.Request) {

	// parser := web.NewParser(r)
	subscriptionPrices := subscription.Subscription{}

	// userIdFromURL, err := parser.GetUUID("userId")
	// if err != nil {
	// 	web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
	// 	return
	// }

	// userIdFromToken, err := security.ExtractUserIDFromToken(r)
	// if err != nil {
	// 	controller.log.Error(err.Error())
	// 	web.RespondError(w, err)
	// 	return
	// }

	// if userIdFromURL != userIdFromToken {
	// 	web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to view subscription prizes"))
	// 	return
	// }

	if err := controller.SubscriptionService.GetPrice(&subscriptionPrices); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	web.RespondJSON(w, http.StatusCreated, subscriptionPrices)
}

func (controller *SubscriptionController) updateSubscriptionPrice(w http.ResponseWriter, r *http.Request) {
	// parser := web.NewParser(r)
	subscriptionPrices := subscription.Subscription{}

	err := web.UnmarshalJSON(r, &subscriptionPrices)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("Unable to parse request body", http.StatusBadRequest))
		return
	}

	if err := subscriptionPrices.Validate(); err != nil {
		log.GetLogger().Error(err.Error())
		web.RespondError(w, err)
		return
	}

	// userIdFromURL, err := parser.GetUUID("userId")
	// if err != nil {
	// 	web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
	// 	return
	// }

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	subscriptionPrices.UpdatedBy = userIdFromToken

	// if userIdFromURL != userIdFromToken {
	// 	web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to update subscription prizes"))
	// 	return
	// }

	if err = controller.SubscriptionService.UpdateSubscriptionPrice(&subscriptionPrices); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription Prices updated successfully",
	})
}
