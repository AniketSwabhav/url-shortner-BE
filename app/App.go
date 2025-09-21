package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
	"url-shortner-be/components/config"
	"url-shortner-be/components/log"
	"url-shortner-be/components/url/controller"
	"url-shortner-be/components/url/service"
	"url-shortner-be/module/repository"

	_ "github.com/jinzhu/gorm/dialects/mysql"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

type App struct {
	sync.Mutex
	Name                 string
	Router               *mux.Router
	myAppRouter          *mux.Router
	urlRedirectionRouter *mux.Router
	DB                   *gorm.DB
	Log                  log.Logger
	Server               *http.Server
	WG                   *sync.WaitGroup
	Repository           repository.Repository
}

type Controller interface {
	RegisterRoutes(router *mux.Router)
}

type ModuleConfig interface {
	MigrateTables()
}

func NewApp(name string, db *gorm.DB, log log.Logger,
	wg *sync.WaitGroup, repo repository.Repository) *App {
	return &App{
		Name:       name,
		DB:         db,
		Log:        log,
		WG:         wg,
		Repository: repo,
	}
}

func (app *App) getPort() string {
	return config.PORT.GetStringValue()
}

func getConnectionString() string {
	conn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		config.DBUser.GetStringValue(),
		config.DBPass.GetStringValue(),
		config.DBHost.GetStringValue(),
		config.DBPort.GetStringValue(),
		config.DBName.GetStringValue())

	displayConnection := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		config.DBUser.GetStringValue(),
		"******",
		config.DBHost.GetStringValue(),
		config.DBPort.GetStringValue(),
		config.DBName.GetStringValue())
	log.GetLogger().Info("HERE IS THE OPEN URL:", displayConnection)
	return conn
}

func NewDBConnection(log log.Logger) *gorm.DB {
	// const url = "root:12345@tcp(127.0.0.1:3306)/contact_app_db?charset=utf8&parseTime=True&loc=Local"

	url := getConnectionString()

	db, err := gorm.Open("mysql", url)
	if err != nil {
		log.Print(err.Error())
		return nil
	}

	sqlDB := db.DB()
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(500)
	sqlDB.SetConnMaxLifetime(3 * time.Minute)

	db.LogMode(true)

	return db
}

func (a *App) Init() {
	a.initializeRouter()
	a.initializeServer()
}

func (a *App) initializeRouter() {
	a.Log.Print("Initializing " + a.Name + " Route")
	a.Router = mux.NewRouter().StrictSlash(true)
	a.myAppRouter = a.Router.PathPrefix("/api/v1/url-shortner").Subrouter()
	a.urlRedirectionRouter = a.Router.PathPrefix("/").Subrouter()
}

func (a *App) initializeServer() {
	headersOk := handlers.AllowedHeaders([]string{
		"Content-Type", "Authorization",
	})
	originsOk := handlers.AllowedOrigins([]string{
		"http://localhost:4200",
	})
	methodsOk := handlers.AllowedMethods([]string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions,
	})
	credentialsOk := handlers.AllowCredentials()
	apiPort := a.getPort()
	a.Server = &http.Server{
		// Addr:         "localhost:2611",
		Addr:         "0.0.0.0:" + apiPort,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(originsOk, methodsOk, headersOk, credentialsOk)(a.Router),
	}
	a.Log.Printf("Server Exposed On %s", apiPort)
}

func (a *App) StartServer() error {

	a.Log.Print("Server Time: ", time.Now())
	a.Log.Print("Server Running on port:", a.getPort())

	err := a.Server.ListenAndServe()
	if err != nil {
		a.Log.Print("Listen and serve error: ", err)
		return err
	}
	return nil
}

func (a *App) RegisterControllerRoutes(controllers []Controller) {

	a.Lock()
	defer a.Unlock()

	controller := controller.NewUrlController(service.NewUrlService(a.DB, a.Repository), a.Log)
	controller.RegisterRedirectRoute(a.urlRedirectionRouter)

	for _, controller := range controllers {
		controller.RegisterRoutes(a.myAppRouter)
	}

}

func (a *App) MigrateModuleTables(moduleConfigs []ModuleConfig) {

	a.Lock()
	defer a.Unlock()

	for _, moduleConfig := range moduleConfigs {
		moduleConfig.MigrateTables()
	}

}

func (app *App) Stop() {

	context, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	app.DB.Close()
	app.Log.Print("Db closed")

	err := app.Server.Shutdown(context)
	if err != nil {
		app.Log.Print("Failed to Stop Server")
		return
	}
	app.Log.Print("Server Shutdown")
}
