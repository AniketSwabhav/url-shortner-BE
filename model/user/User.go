package user

import (
	"url-shortner-be/model/credential"
	model "url-shortner-be/model/general"
	"url-shortner-be/model/subscription"
	"url-shortner-be/model/url"
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

//add DTO for get api
