package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"url-shortner-be/components/errors"
)

func RespondError(w http.ResponseWriter, err error) {
	switch typedErr := err.(type) {
	case *errors.UnauthorizedError:
		RespondJSON(w, typedErr.HTTPStatus, typedErr)
	case *errors.ValidationError:
		RespondJSON(w, typedErr.HTTPStatus, typedErr)
	case *errors.HTTPError:
		RespondJSON(w, typedErr.HTTPStatus, typedErr)
	default:
		RespondErrorMessage(w, http.StatusInternalServerError, "Unexpected error: "+err.Error())
	}
}

func RespondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		RespondErrorMessage(w, http.StatusInternalServerError, "Failed to marshal JSON: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func RespondErrorMessage(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"type":    "INTERNAL_ERROR",
		"message": msg,
	})
}

func RespondJSONWithXTotalCount(w http.ResponseWriter, code int, count int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	SetNewHeader(w, "X-Total-Count", strconv.Itoa(count))
	w.WriteHeader(code)
	w.Write([]byte(response))
}

func SetNewHeader(w http.ResponseWriter, headerName, value string) {
	w.Header().Add("Access-Control-Expose-Headers", headerName)
	w.Header().Set(headerName, value)
}
