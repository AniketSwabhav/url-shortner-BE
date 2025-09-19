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

	model := &Transaction{}

	err := c.DB.AutoMigrate(model).Error
	if err != nil {
		log.NewLog().Print("Auto Migrating Trnasaction ==> %s", err)
	}

	err = c.DB.Model(&Transaction{}).AddForeignKey("user_id", "users(id)", "CASCADE", "CASCADE").Error
	if err != nil {
		log.GetLogger().Print("Foreign Key Constraints Of Transaction ==> %s", err)
	}

	log.GetLogger().Print("Transaction Module Configured.")
}
