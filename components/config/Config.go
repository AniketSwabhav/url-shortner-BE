package config

import (
	"fmt"
	"os"
	"strconv"
	"url-shortner-be/components/log"

	"github.com/spf13/viper"
)

var GlobalConfig *Config

type Config struct {
	viper       *viper.Viper
	Environment Environment
}

func InitializeGlobalConfig(environment Environment) {

	if GlobalConfig != nil {
		log.GetLogger().Warn("Global Config already Initialized")
	}

	vp := viper.New()

	switch environment {
	case Local:
		vp.SetConfigName("config-local")
	default:
		vp.SetConfigName("config-local")
	}

	vp.SetConfigType("env")
	vp.AddConfigPath(".")
	vp.AutomaticEnv()

	config := Config{
		viper:       vp,
		Environment: environment,
	}

	if err := vp.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.GetLogger().Warn("File Not Found")
		} else {
			log.GetLogger().Fatalf("Something Wrong in File Reading Error:[%s]", err.Error())
		}
	}

	GlobalConfig = &config
}

func (config *Config) GetString(key EnvKey) string {
	if config.Environment == Local {
		return config.viper.GetString(string(key))
	}
	return os.Getenv(string(key))
}

// IsSet checks if environment variable is set.
func (config *Config) IsSet(key EnvKey) bool {
	if config.Environment == Local {
		return config.viper.IsSet(string(key))
	}
	value := os.Getenv(string(key))
	return value != ""
}

func (config *Config) GetInt64(key EnvKey) int64 {
	if config.Environment == Local {
		return config.viper.GetInt64(string(key))
	}
	// Use os.Getenv for non-local environments
	value := os.Getenv(string(key))
	if value == "" {
		log.GetLogger().Error(fmt.Sprintf("Key %s is not set", key))
		return 0
	}

	// Parse the value as int64
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.GetLogger().Error(err.Error())
		return 0
	}
	return intValue
}
