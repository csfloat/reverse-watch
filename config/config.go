package config

import (
	"github.com/spf13/viper"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	ReversalsDB struct {
		Filename string
	}

	KeysDB struct {
		Filename string
	}

	HTTP struct {
		Port string
	}

	Admin struct {
		APIKey string
	}

	Environment Environment
}

func Load() Config {
	return load()
}

func load() Config {
	v := viper.New()

	v.SetDefault("ReversalsDB.Filename", "./data/reversals.db")
	v.SetDefault("KeysDB.Filename", "./data/keys.db")
	v.SetDefault("HTTP.Port", "8080")
	v.SetDefault("Environment", Development)

	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return cfg
}
