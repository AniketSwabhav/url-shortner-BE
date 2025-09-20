package controller

import (
	"fmt"
	"net/http"
	"strconv"
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
	urlRouter.HandleFunc("/url", urlController.getAllUrlsByUserId).Methods(http.MethodGet)
	urlRouter.HandleFunc("/url/short-url", urlController.getUrlByShortUrl).Methods(http.MethodPost)
	urlRouter.HandleFunc("/url/{urlId}", urlController.getUrlById).Methods(http.MethodGet)
	urlRouter.HandleFunc("/url/{urlId}", urlController.updateUrlById).Methods(http.MethodPut)
	urlRouter.HandleFunc("/url/{urlId}", urlController.deleteUrlById).Methods(http.MethodDelete)
	urlRouter.HandleFunc("/url/{urlId}/renew-visits", urlController.renewUrlVisits).Methods(http.MethodPost)

	urlRouter.Use(security.MiddlewareUser)

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
	fmt.Println("Redirected to: ", longUrl)
}

// ---------------------------------------------------------------------------
func (controller *UrlController) getAllUrlsByUserId(w http.ResponseWriter, r *http.Request) {
	allUrl := &[]url.UrlDTO{}
	parser := web.NewParser(r)
	var totalCount int
	query := r.URL.Query()

	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid User ID format"))
		return
	}

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	if userIdFromToken != userIdFromURL {
		web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to view URLs of this user"))
		return
	}

	if err = controller.UrlService.GetAllUrls(allUrl, userIdFromURL, &totalCount, limit, offset); err != nil {
		controller.log.Print(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSONWithXTotalCount(w, http.StatusOK, totalCount, allUrl)

}

func (controller *UrlController) getUrlById(w http.ResponseWriter, r *http.Request) {
	var targetURL = &url.UrlDTO{}

	parser := web.NewParser(r)

	userIdFromURL, err := parser.GetUUID("userId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid User ID format"))
		return
	}

	urlIdFromURL, err := parser.GetUUID("urlId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid URL ID format"))
		return
	}
	targetURL.ID = urlIdFromURL

	userIdFromToken, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	targetURL.UserID = userIdFromToken

	if userIdFromToken != userIdFromURL {
		web.RespondError(w, errors.NewUnauthorizedError("you are not authorized to view this URL"))
		return
	}

	err = controller.UrlService.GetUrlByID(targetURL)
	if err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, targetURL)
}

func (controller *UrlController) getUrlByShortUrl(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) updateUrlById(w http.ResponseWriter, r *http.Request) {}

func (controller *UrlController) deleteUrlById(w http.ResponseWriter, r *http.Request) {
	parser := web.NewParser(r)

	urlID, err := parser.GetUUID("urlId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid URL ID format"))
		return
	}

	deletedBy, err := security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	err = controller.UrlService.Delete(urlID, deletedBy)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "URL deleted successfully",
	})
}

func (controller *UrlController) renewUrlVisits(w http.ResponseWriter, r *http.Request) {
	urlToRenew := &url.Url{}
	parser := web.NewParser(r)

	err := web.UnmarshalJSON(r, &urlToRenew)
	if err != nil {
		web.RespondError(w, errors.NewHTTPError("unable to parse requested data", http.StatusBadRequest))
		return
	}

	urlIdFromURL, err := parser.GetUUID("urlId")
	if err != nil {
		web.RespondError(w, errors.NewValidationError("Invalid user ID format"))
		return
	}
	urlToRenew.ID = urlIdFromURL

	urlToRenew.UserID, err = security.ExtractUserIDFromToken(r)
	if err != nil {
		controller.log.Error(err.Error())
		web.RespondError(w, err)
		return
	}
	urlToRenew.UpdatedBy = urlToRenew.UserID

	err = controller.UrlService.RenewUrlVisits(urlToRenew)
	if err != nil {
		web.RespondError(w, err)
		return
	}

	web.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Url Visits Renewed Successfully",
	})

}
