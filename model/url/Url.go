package url

import (
	"crypto/rand"
	"net/http"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	model "url-shortner-be/model/general"

	uuid "github.com/satori/go.uuid"
)

type Url struct {
	model.Base
	LongUrl         string    `json:"longUrl" gorm:"not null;type:text"`
	ShortUrl        string    `json:"shortUrl" gorm:"not null;type:varchar(5)"`
	RemainingVisits int       `json:"remainingVisits" gorm:"not null;type:int;default:0"`
	VisitCount      int       `json:"visitCount" gorm:"not null;type:int;default:0"`
	UserID          uuid.UUID `json:"userId" gorm:"type:char(36)"`
}

type UrlDTO struct {
	model.Base
	LongUrl         string    `json:"longUrl" gorm:"not null;type:text"`
	ShortUrl        string    `json:"shortUrl" gorm:"not null;unique;type:varchar(5)"`
	RemainingVisits int       `json:"remainingVisits" gorm:"not null;type:int;default:0"`
	VisitCount      int       `json:"visitCount" gorm:"not null;type:int;default:0"`
	UserID          uuid.UUID `json:"userId" gorm:"foreignkey:ID;type:char(36)"`
}

// ALTER TABLE urls
// MODIFY short_url VARCHAR(255)
// CHARACTER SET utf8mb4
// COLLATE utf8mb4_bin;
// Fire this query for byte-by-byte comparison

func (*UrlDTO) TableName() string {
	return "urls"
}

func (url *Url) Validate(inputUrl string, shortUrl string) error {
	resp, err := http.Get(inputUrl)
	if err != nil {
		log.GetLogger().Print(err)
		return err
	}
	if resp.StatusCode == 404 {
		return errors.NewValidationError("request url not found, please provide a valid Long URL")
	}
	defer resp.Body.Close()

	if len(shortUrl) == 0 || len(shortUrl) != 5 {
		return errors.NewHTTPError("short url must have 5 characters", http.StatusBadRequest)
	}

	return nil
}

func GenerateShortUrl() string {

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, 5)
	rand.Read(b)

	for i := 0; i < 5; i++ {
		b[i] = letterBytes[int(b[i])%len(letterBytes)]
	}

	return string(b)
}
