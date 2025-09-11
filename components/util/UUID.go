package util

import (
	"url-shortner-be/components/errors"

	uuid "github.com/satori/go.uuid"
)

func ParseUUID(input string) (uuid.UUID, error) {
	if len(input) == 0 {
		return uuid.Nil, errors.NewValidationError("Empty ID")
	}
	id, err := uuid.FromString(input)
	if err != nil {
		return uuid.Nil, errors.NewValidationError(input + ": Invalid ID")
	}
	return id, nil
}
