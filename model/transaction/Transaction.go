package transaction

import (
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Transaction struct {
	model.Base
	Amount float32   `json:"amount" gorm:"type:float"`
	UserId uuid.UUID `json:"userId" gorm:"not null;type:varchar(36)"`
}
