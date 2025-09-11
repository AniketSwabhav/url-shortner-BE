package user

import (
	"url-shortner-be/components/log"

	"github.com/jinzhu/gorm"
)

type UserModuleConfig struct {
	DB *gorm.DB
}

func NewUserModuleConfig(db *gorm.DB) *UserModuleConfig {
	return &UserModuleConfig{
		DB: db,
	}
}

func (u *UserModuleConfig) MigrateTables() {

	model := &User{}

	err := u.DB.AutoMigrate(model).Error
	if err != nil {
		log.NewLog().Print("Auto Migrating User ==> %s", err)
	}

}
