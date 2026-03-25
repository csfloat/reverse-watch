package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"reverse-watch/domain/models/constants"
	"reverse-watch/util"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	Database struct {
		Host          string
		Port          string
		User          string
		Password      string
		SSLMode       string
		PrivateDBName string
		PublicDBName  string
	}

	HTTP struct {
		Port                   string
		AllowedOrigins         []string
		AllowFirefoxExtensions bool
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

	// Steam.WebAPIKeys is optional. If set, /api/v1/steam/resolve-vanity can use the Steam Web API
	// (keys are rotated) instead of any fallback resolution.
	Steam struct {
		WebAPIKeys *util.Ring[string]
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
		func(from reflect.Value, to reflect.Value) (interface{}, error) {
			ringType := reflect.TypeOf(&util.Ring[string]{})
			if to.Type() != ringType {
				return from.Interface(), nil
			}

			keys := make([]string, 0)
			switch from.Kind() {
			case reflect.String:
				for _, part := range strings.Split(from.String(), ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						keys = append(keys, part)
					}
				}
			case reflect.Slice, reflect.Array:
				for i := 0; i < from.Len(); i++ {
					val := strings.TrimSpace(fmt.Sprint(from.Index(i).Interface()))
					if val != "" {
						keys = append(keys, val)
					}
				}
			default:
				return util.NewRing[string](nil), nil
			}

			return util.NewRing(keys), nil
		},
	))

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("Database.Host", "localhost")
	v.SetDefault("Database.Port", "5432")
	v.SetDefault("Database.User", "postgres")
	v.SetDefault("Database.Password", "postgres")
	v.SetDefault("Database.SSLMode", "disable")
	v.SetDefault("Database.PrivateDBName", "private")
	v.SetDefault("Database.PublicDBName", "public")
	v.SetDefault("HTTP.Port", "80")
	v.SetDefault("HTTP.AllowFirefoxExtensions", false)
	v.SetDefault("Environment", constants.EnvironmentDevelopment)
	v.SetDefault("TrustProxy", false)
	v.SetDefault("Ingestors.CSFloat.Enable", false)
	v.SetDefault("Ingestors.CSFloat.BaseURL", "https://csfloat.com")

	// Need to register environment variables if defaults aren't set
	v.BindEnv("HTTP.AllowedOrigins")
	v.BindEnv("Ingestors.CSFloat.SecretKey")
	v.BindEnv("Steam.WebAPIKeys")

	// Try to find the root directory, but don't panic if it fails since go.mod doesn't exist in production
	dir, err := GetProjectRootDir()
	if err == nil {
		v.AddConfigPath(dir)
		v.SetConfigName("config")
		v.SetConfigType("json")

		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				panic(err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, opts); err != nil {
		panic(err)
	}

	if cfg.Ingestors.CSFloat.Enable {
		if cfg.Ingestors.CSFloat.BaseURL == "" || cfg.Ingestors.CSFloat.SecretKey == "" {
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
