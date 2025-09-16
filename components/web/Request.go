package web

import (
	"encoding/json"
	"io"
	"net/http"
	"url-shortner-be/components/errors"
)

func UnmarshalJSON(request *http.Request, out interface{}) error {
	if request.Body == nil {
		return errors.NewHTTPError(errors.ErrorCodeEmptyRequestBody, http.StatusBadRequest)
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.NewHTTPError(err.Error(), http.StatusBadRequest)
	}

	if len(body) == 0 {
		return errors.NewHTTPError(errors.ErrorCodeEmptyRequestBody, http.StatusBadRequest)
	}

	err = json.Unmarshal(body, out)
	if err != nil {
		return errors.NewHTTPError(err.Error(), http.StatusBadRequest)
	}

	return nil
}
