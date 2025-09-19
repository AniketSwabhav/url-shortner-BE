package subscription

import (
	model "url-shortner-be/model/general"
)

type Subscription struct {
	model.Base
	FreeShortUrls   int     `json:"freeShortUrls" gorm:"type:int"`
	FreeVisits      int     `json:"freeVisits" gorm:"type:int"`
	NewUrlPrice     float32 `json:"newUrlPrice" gorm:"type:float"`
	ExtraVisitPrice float32 `json:"extraVisitPrice" gorm:"type:float"`
}
