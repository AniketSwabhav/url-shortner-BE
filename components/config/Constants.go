package config

const (
	Local Environment = "local"
)

const (
	// For DB
	DBName EnvKey = "DB_NAME"
	DBUser EnvKey = "DB_USER"
	DBPass EnvKey = "DB_PASS"
	DBPort EnvKey = "DB_PORT"
	DBHost EnvKey = "DB_HOST"

	// For Server
	PORT   EnvKey = "PORT"
	JWTKey EnvKey = "JWT_KEY"
)
