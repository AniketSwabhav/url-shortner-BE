package module

import (
	"url-shortner-be/app"
	transactionService "url-shortner-be/components/transaction/service"
	"url-shortner-be/module/repository"
)

func registerTransactionRoutes(appObj *app.App, repository repository.Repository) {

	defer appObj.WG.Done()
	transactionService.NewTransactionService(appObj.DB, repository)

	// userController := controller.NewUserController(userService, appObj.Log)

	// appObj.RegisterControllerRoutes([]app.Controller{
	// 	userController,
	// })
}
