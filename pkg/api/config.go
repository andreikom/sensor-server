package api

import (
	"github.com/go-playground/validator/v10"
	"time"
)

const (
	defaultPort    = 8080
	readTimeout    = 5 * time.Second
	maxConnections = 10
	daysToKeep     = 7
)

type Config struct {
	Server struct {
		Port           int `yaml:"port" validate:"numeric,min=1024"`
		MaxConnections int `yaml:"maxConnections" validate:"numeric,min=1"`
	}
}

func validate(cfg *Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}
