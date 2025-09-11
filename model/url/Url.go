package url

import (
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Url struct {
	model.Base
	LongUrl  string    `json:"longUrl" gorm:"not null;type:text"`
	ShortUrl string    `json:"shortUrl" gorm:"not null;unique;type:varchar(5)"`
	Visits   int       `json:"visits" gorm:"not null;type:int;default:0"`
	UserID   uuid.UUID `json:"userId" gorm:"foreignkey:ID;type:char(36)"`
}
