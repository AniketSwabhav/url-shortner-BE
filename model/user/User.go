package user

import (
	"url-shortner-be/components/errors"
	"url-shortner-be/components/util"
	"url-shortner-be/model/credential"
	model "url-shortner-be/model/general"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/url"

	uuid "github.com/satori/go.uuid"
)

type User struct {
	model.Base
	FirstName    string                       `json:"firstName" example:"Ravi" gorm:"type:varchar(50)"`
	LastName     string                       `json:"lastName" example:"Sharma" gorm:"type:varchar(50)"`
	PhoneNo      string                       `sql:"index" json:"phoneNo" example:"9700795509" gorm:"type:varchar(15)"`
	IsAdmin      *bool                        `json:"isAdmin" gorm:"type:tinyint(1);default:false"`
	IsActive     *bool                        `json:"isActive" gorm:"type:tinyint(1);default:true"`
	Wallet       float32                      `json:"wallet" gorm:"type:float"`
	UrlCount     int                          `json:"urlCount" gorm:"type:int"`
	Credentials  *credential.Credential       `json:"credential"`
	Url          []url.Url                    `json:"url" gorm:"foreignKey:urlID"`
	Subscription []*subscription.Subscription `json:"Subscription" gorm:"foreignKey:subscriptionID"`
}

func (user *User) Validate() error {

	if util.IsEmpty(user.FirstName) || !util.ValidateString(user.FirstName) {
		return errors.NewValidationError("User FirstName must be specified and must have characters only")
	}
	if util.IsEmpty(user.LastName) || !util.ValidateString(user.LastName) {
		return errors.NewValidationError("User LastName must be specified and must have characters only")
	}
	if util.IsEmpty(user.PhoneNo) || !util.ValidateContact(user.PhoneNo) {
		return errors.NewValidationError("User Contact must be specified and have 10 digits")
	}
	return nil
}

type UserDTO struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	PhoneNo   string    `json:"phoneNo"`
	IsAdmin   *bool     `json:"isAdmin"`
	IsActive  *bool     `json:"isActive"`
	Wallet    float32   `json:"wallet"`
	UrlCount  int       `json:"urlCount"`
}

func ToUserDTO(u *User) UserDTO {
	return UserDTO{
		ID:        u.Base.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		PhoneNo:   u.PhoneNo,
		IsAdmin:   u.IsAdmin,
		IsActive:  u.IsActive,
		Wallet:    u.Wallet,
		UrlCount:  u.UrlCount,
	}
}

func ToUserDTOs(users []User) []UserDTO {
	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = ToUserDTO(&u)
	}
	return dtos
}
