package credential

import (
	"url-shortner-be/components/log"

	"github.com/jinzhu/gorm"
)

type CredentialModuleConfig struct {
	DB *gorm.DB
}

func NewCredentialModuleConfig(db *gorm.DB) *CredentialModuleConfig {
	return &CredentialModuleConfig{
		DB: db,
	}
}

func (c *CredentialModuleConfig) MigrateTables() {

	model := &Credential{}

	err := c.DB.AutoMigrate(model).Error
	if err != nil {
		log.NewLog().Print("Auto Migrating Credential ==> %s", err)
	}

	err = c.DB.Model(model).AddForeignKey("user_id", "users(id)", "CASCADE", "CASCADE").Error
	if err != nil {
		log.NewLog().Print("Foreign Key Constraints Of credentials ==> %s", err)
	}

}
