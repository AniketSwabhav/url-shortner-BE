package module

import (
	"url-shortner-be/app"
	"url-shortner-be/module/repository"
)

func RegisterModuleRoutes(app *app.App, repository repository.Repository) {
	log := app.Log
	log.Print("============Registering-Module-Routes==============")

	app.WG.Add(1)

	app.WG.Done()
}
