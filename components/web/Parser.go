package web

import (
	"net/http"
	"net/url"
	"strconv"
	"url-shortner-be/components/errors"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

type Parser struct {
	Params map[string]string
	Form   url.Values
}

func NewParser(r *http.Request) *Parser {
	r.ParseForm()
	return &Parser{
		Params: mux.Vars(r),
		Form:   r.Form,
	}
}

// GetUUID will get uuid from the given paramName in URL params.
func (p *Parser) GetUUID(paramName string) (uuid.UUID, error) {
	idString := p.Params[paramName]
	id, err := ParseUUID(idString)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func ParseUUID(input string) (uuid.UUID, error) {
	if len(input) == 0 {
		return uuid.Nil, errors.NewValidationError("ID cannot be empty")
	}
	id, err := uuid.FromString(input)
	if err != nil {
		return uuid.Nil, errors.NewValidationError("Invalid UUID format")
	}
	return id, nil
}

// GetString will get a string from the given paramName in URL params.
func (p *Parser) GetString(paramName string) (string, error) {
	value := p.Params[paramName]
	if len(value) == 0 {
		return "", errors.NewValidationError(paramName + " cannot be empty")
	}
	return value, nil
}

// ParseLimitAndOffset will parse limit and offset from query params.
func (p *Parser) ParseLimitAndOffset() (limit, offset int) {
	limitparam := p.Form.Get("limit")
	offsetparam := p.Form.Get("offset")
	var err error
	limit = 5
	if len(limitparam) > 0 {
		limit, err = strconv.Atoi(limitparam)
		if err != nil {
			return
		}
	}
	if len(offsetparam) > 0 {
		offset, err = strconv.Atoi(offsetparam)
		if err != nil {
			return
		}
	}
	return
}
