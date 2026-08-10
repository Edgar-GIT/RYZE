package config

import "time"

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type JWTConfig struct {
	Secret         string
	AccessTokenTTL time.Duration
}
