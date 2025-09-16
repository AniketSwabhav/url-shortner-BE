package module

import (
	"url-shortner-be/app"
	"url-shortner-be/module/repository"
)

func RegisterModuleRoutes(app *app.App, repository repository.Repository) {
	log := app.Log
	log.Print("============Registering-Module-Routes==============")

	app.WG.Add(3)
	registerUserRoutes(app, repository)
	registerUrlRoutes(app, repository)
	app.WG.Done()
}
