package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"reverse-watch/domain/models/constants"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	// StaticDir is the directory containing the database files
	// This is relative to the project's root directory
	StaticDir string

	HTTP struct {
		Port string
	}

	Environment constants.Environment
	TrustProxy  bool

	Ingestors struct {
		CSFloat struct {
			Enable    bool
			BaseURL   string
			SecretKey string
		}
	}
}

func Load() Config {
	return load()
}

func load() Config {
	v := viper.New()

	opts := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		func(from reflect.Value, to reflect.Value) (interface{}, error) {
			if from.Kind() != reflect.String || to.Type() != reflect.TypeOf(constants.Environment("")) {
				return from.Interface(), nil
			}

			env := constants.Environment(from.String())
			switch env {
			case constants.EnvironmentDevelopment, constants.EnvironmentProduction:
				return env, nil
			default:
				return nil, fmt.Errorf("invalid environment")
			}
		},
	))

	v.SetDefault("StaticDir", "./static")
	v.SetDefault("HTTP.Port", "8080")
	v.SetDefault("Environment", constants.EnvironmentDevelopment)
	v.SetDefault("TrustProxy", false)
	v.SetDefault("Ingestors.CSFloat.Enable", false)
	v.SetDefault("Ingestors.CSFloat.BaseURL", "https://csfloat.com")

	dir, err := GetProjectRootDir()
	if err != nil {
		panic(err)
	}

	v.AddConfigPath(dir)
	v.SetConfigName("config")
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, opts); err != nil {
		panic(err)
	}

	if cfg.Ingestors.CSFloat.Enable {
		if cfg.Ingestors.CSFloat.BaseURL == "" && cfg.Ingestors.CSFloat.SecretKey == "" {
			panic("csfloat ingestor configuration is required when enabled")
		}
	}

	return cfg
}

func GetProjectRootDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for {
		targetPath := filepath.Join(dir, "go.mod")

		if _, err := os.Stat(targetPath); err == nil {
			return strings.TrimSuffix(targetPath, "go.mod"), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		// Get parent directory for next iteration
		parent := filepath.Dir(dir)

		// Check if we're at the root of the volume
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod")
		}
		dir = parent
	}
}
