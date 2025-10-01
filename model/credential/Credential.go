package credential

import (
	"url-shortner-be/components/errors"
	"url-shortner-be/components/util"
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Credential struct {
	model.Base
	Email    string    `json:"email" gorm:"not null;type:varchar(36)"`
	Password string    `json:"password" gorm:"not null;type:varchar(255)"`
	UserID   uuid.UUID `json:"userId" gorm:"not null;type:varchar(36)"`
}

type CredentialDTO struct {
	model.Base
	Email    string    `json:"email" gorm:"unique;not null;type:varchar(36)"`
	Password string    `json:"password" gorm:"not null;type:varchar(255)"`
	UserID   uuid.UUID `json:"userId" gorm:"not null;type:varchar(36)"`
}

func (*CredentialDTO) TableName() string {
	return "credentials"
}

func (user *Credential) Validate() error {

	if util.IsEmpty(user.Email) || !util.ValidateEmail(user.Email) {
		return errors.NewValidationError("User Email must be specified and should be of the type abc@domain.com")
	}

	if util.IsEmpty(user.Password) || len(user.Password) < 8 {
		return errors.NewValidationError("Password should consist of 8 or more characters")
	}
	return nil
}
