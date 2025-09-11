package module

import (
	"url-shortner-be/app"
	"url-shortner-be/model/credential"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/transaction"
	"url-shortner-be/model/url"
	"url-shortner-be/model/user"
)

func Configure(appObj *app.App) {
	appObj.Log.Print("============Configuring-Module-Configs==============")

	userModule := user.NewUserModuleConfig(appObj.DB)
	credentialModule := credential.NewCredentialModuleConfig(appObj.DB)
	urlModule := url.NewUrlModuleConfig(appObj.DB)
	subscriptionModule := subscription.NewSubscriptionModuleConfig(appObj.DB)
	transactionModule := transaction.NewTransactionModuleConfig(appObj.DB)

	appObj.MigrateModuleTables([]app.ModuleConfig{userModule, credentialModule, urlModule, subscriptionModule, transactionModule})
}
