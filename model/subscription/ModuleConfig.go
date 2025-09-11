package subscription

import (
	"url-shortner-be/components/log"

	"github.com/jinzhu/gorm"
)

type SubscriptionModuleConfig struct {
	DB *gorm.DB
}

func NewSubscriptionModuleConfig(db *gorm.DB) *SubscriptionModuleConfig {
	return &SubscriptionModuleConfig{
		DB: db,
	}
}

func (c *SubscriptionModuleConfig) MigrateTables() {

	var models = []interface{}{
		&Subscription{},
	}

	for _, model := range models {
		if err := c.DB.AutoMigrate(model).Error; err != nil {
			log.GetLogger().Print("Auto Migration ==> %s", err)
		}
	}

	err := c.DB.Model(&Subscription{}).AddForeignKey("user_id", "users(id)", "CASCADE", "CASCADE").Error
	if err != nil {
		log.GetLogger().Print("Foreign Key Constraints Of Subscription ==> %s", err)
	}

	log.GetLogger().Print("Subscription Module Configured.")
}
