package url

import (
	"url-shortner-be/components/log"

	"github.com/jinzhu/gorm"
)

type UrlModuleConfig struct {
	DB *gorm.DB
}

func NewUrlModuleConfig(db *gorm.DB) *UrlModuleConfig {
	return &UrlModuleConfig{
		DB: db,
	}
}

func (c *UrlModuleConfig) MigrateTables() {

	var models []interface{} = []interface{}{
		&Url{},
	}

	for _, model := range models {
		err := c.DB.AutoMigrate(model).Error
		if err != nil {
			log.GetLogger().Print("Auto Migration ==> %s", err)
		}
	}

	err := c.DB.Model(&Url{}).AddForeignKey("user_id", "users(id)", "CASCADE", "CASCADE").Error
	if err != nil {
		log.GetLogger().Print("Foreign Key Constraints Of Url ==> %s", err)
	}

	log.GetLogger().Print("Url Module Configured.")

}
