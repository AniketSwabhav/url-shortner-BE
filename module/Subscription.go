package module

import (
	"url-shortner-be/app"
	"url-shortner-be/components/subscription/controller"
	subscriptionService "url-shortner-be/components/subscription/service"
	"url-shortner-be/module/repository"
)

func registerSubscriptionRoutes(appObj *app.App, repository repository.Repository) {

	defer appObj.WG.Done()
	subscriptionService := subscriptionService.NewSubscriptionService(appObj.DB, repository)

	subscriptionController := controller.NewSubscriptionrController(subscriptionService, appObj.Log)

	appObj.RegisterControllerRoutes([]app.Controller{
		subscriptionController,
	})
}
