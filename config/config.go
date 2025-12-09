package config

import (
	"sync"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	cfg  = &Config{}
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

func Get() Config {
	once.Do(func() {
		load()
	})
	return *cfg
}

func load() {
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

	if err := v.Unmarshal(cfg); err != nil {
		panic(err)
	}
}
