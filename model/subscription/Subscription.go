package subscription

import (
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Subscription struct {
	model.Base
	FreeVisits     int       `json:"freeVisits" gorm:"type:int"`
	FreeUrlLimit   int       `json:"freeUrlLimit" gorm:"type:int"`
	PerUrlPrice    float32   `json:"perUrlPrice" gorm:"type:float"`
	RequiredVisits int       `json:"requiredVisits" gorm:"type:int"`
	TotalRenewCost float32   `json:"totalRenewCost" gorm:"type:float"`
	UserId         uuid.UUID `json:"userId" gorm:"not null;type:varchar(36)"`
}
