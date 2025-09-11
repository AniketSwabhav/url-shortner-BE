package transaction

import (
	"url-shortner-be/components/log"

	"github.com/jinzhu/gorm"
)

type TransactionModuleConfig struct {
	DB *gorm.DB
}

func NewTransactionModuleConfig(db *gorm.DB) *TransactionModuleConfig {
	return &TransactionModuleConfig{
		DB: db,
	}
}

func (c *TransactionModuleConfig) MigrateTables() {

	var models = []interface{}{
		&Transaction{},
	}

	for _, model := range models {
		if err := c.DB.AutoMigrate(model).Error; err != nil {
			log.GetLogger().Print("Auto Migration ==> %s", err)
		}
	}

	err := c.DB.Model(&Transaction{}).AddForeignKey("user_id", "users(id)", "CASCADE", "CASCADE").Error
	if err != nil {
		log.GetLogger().Print("Foreign Key Constraints Of Transaction ==> %s", err)
	}

	log.GetLogger().Print("Transaction Module Configured.")
}
