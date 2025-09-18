package controller

import (
	"net/http"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/security"
	urlService "url-shortner-be/components/url/service"
	"url-shortner-be/components/web"
	"url-shortner-be/model/url"
	"url-shortner-be/model/user"

	"github.com/gorilla/mux"
)

type UrlController struct {
	log        log.Logger
	UrlService *urlService.UrlService
}

func NewUrlController(urlService *urlService.UrlService, log log.Logger) *UrlController {
	return &UrlController{
		log:        log,
		UrlService: urlService,
	}
}

func (urlController *UrlController) RegisterRoutes(router *mux.Router) {

	urlRouter := router.PathPrefix("/user/{userId}").Subrouter()

	urlRouter.HandleFunc("/url", urlController.registerUrl).Methods(http.MethodPost)
	urlRouter.HandleFunc("/url", urlController.getAllUrls).Methods(http.MethodGet)
	urlRouter.HandleFunc("/url/short-url", urlController.getUrlByShortUrl).Methods(http.MethodPost)
	urlRouter.HandleFunc("/url/{urlId}", urlController.getUrl).Methods(http.MethodGet)
	urlRouter.HandleFunc("/url/{urlId}", urlController.updateUrl).Methods(http.MethodPut)
	urlRouter.HandleFunc("/url/{urlId}", urlController.deleteUrl).Methods(http.MethodDelete)
	urlRouter.HandleFunc("/url/{urlId}/renew-visits", urlController.renewVisits).Methods(http.MethodPost)

	urlRouter.Use(security.MiddlewareUrl)

}

func (controller *UrlController) registerUrl(w http.ResponseWriter, r *http.Request) {
	UrlOwner := &user.User{}
	newUrl := &url.Url{}
	parser := web.NewParser(r)

	err := web.UnmarshalJSON(r, &newUrl)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	err = newUrl.Validate(newUrl.LongUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	newUrl.CreatedBy = userIdFromURL

	UrlOwner.ID, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	err = controller.UrlService.CreateUrl(userIdFromURL, UrlOwner, newUrl)
	if err != nil {
		controller.log.Print(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusCreated, newUrl)
}

// for short url redirection---------------------------------------------------
func (controller *UrlController) RegisterRedirectRoute(router *mux.Router) {
	redirectRouter := router.PathPrefix("/").Subrouter()
	redirectRouter.HandleFunc("/{short-url}", controller.redirectUrl)
}

func (controller *UrlController) redirectUrl(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)

	longUrl, err := controller.UrlService.RedirectUrl(params[("short-url")])
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
	}
	http.Redirect(w, r, longUrl, http.StatusSeeOther)

}

//---------------------------------------------------------------------------

func (controller *UrlController) getAllUrls(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) getUrlByShortUrl(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) getUrl(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) updateUrl(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) deleteUrl(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) renewVisits(w http.ResponseWriter, r *http.Request) {}
