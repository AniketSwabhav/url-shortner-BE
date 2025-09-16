package module

import (
	"url-shortner-be/app"
	"url-shortner-be/components/url/controller"
	urlService "url-shortner-be/components/url/service"
	"url-shortner-be/module/repository"
)

func registerUrlRoutes(appObj *app.App, repository repository.Repository) {

	defer appObj.WG.Done()
	urlService := urlService.NewUrlService(appObj.DB, repository)

	userController := controller.NewUrlController(urlService, appObj.Log)

	appObj.RegisterControllerRoutes([]app.Controller{
		userController,
	})
}
