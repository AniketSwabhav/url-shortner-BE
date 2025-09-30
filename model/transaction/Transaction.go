package transaction

import (
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Transaction struct {
	model.Base
	Amount float32   `json:"amount" gorm:"type:decimal(10,2)"`
	Type   string    `json:"type" gorm:"not null;type:varchar(36)" example:"CREDIT/DEBIT/URLRENEWAL/VISITSRENEWAL"`
	Note   string    `json:"note" gorm:"type:varchar(100)"`
	UserID uuid.UUID `json:"userId" gorm:"not null;type:varchar(36)"`
}
