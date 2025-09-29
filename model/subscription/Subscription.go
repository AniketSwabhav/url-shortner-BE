package subscription

import (
	"url-shortner-be/components/errors"
	model "url-shortner-be/model/general"
)

type Subscription struct {
	model.Base
	FreeShortUrls   int     `json:"freeShortUrls" gorm:"type:int"`
	FreeVisits      int     `json:"freeVisits" gorm:"type:int"`
	NewUrlPrice     float32 `json:"newUrlPrice" gorm:"type:float"`
	ExtraVisitPrice float32 `json:"extraVisitPrice" gorm:"type:float"`

	ExtraVisitPriceNew float32 `json:"extraVisitPriceNew" gorm:"-"`
}

func (s *Subscription) Validate() error {
	if s.FreeShortUrls < 0 {
		return errors.NewValidationError("Free short URLs cannot be negative")
	}
	if s.FreeVisits < 0 {
		return errors.NewValidationError("Free visits cannot be negative")
	}
	if s.NewUrlPrice < 0 {
		return errors.NewValidationError("New URL price cannot be negative")
	}
	if s.ExtraVisitPrice < 0 {
		return errors.NewValidationError("Extra visit price cannot be negative")
	}
	return nil
}
