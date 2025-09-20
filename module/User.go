package module

import (
	"url-shortner-be/app"
	transactionService "url-shortner-be/components/transaction/service"
	"url-shortner-be/components/user/controller"
	userService "url-shortner-be/components/user/service"
	"url-shortner-be/module/repository"
)

func registerUserRoutes(appObj *app.App, repository repository.Repository) {

	defer appObj.WG.Done()
	userService := userService.NewUserService(appObj.DB, repository, transactionService.NewTransactionService(appObj.DB, repository))

	userController := controller.NewUserController(userService, appObj.Log)

	appObj.RegisterControllerRoutes([]app.Controller{
		userController,
	})
}
