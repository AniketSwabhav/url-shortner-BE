package config

type Environment string

type EnvKey string

func (e EnvKey) GetStringValue() string {
	return GlobalConfig.GetString(e)
}

func (e EnvKey) GetInt64Value() int64 {
	return GlobalConfig.GetInt64(e)
}
