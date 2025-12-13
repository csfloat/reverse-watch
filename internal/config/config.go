package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	// DataDir Directory containing the database files
	// Relative to the project's root directory
	DataDir string

	PublicDB struct {
		Filename string
	}

	PrivateDB struct {
		Filename string
	}

	HTTP struct {
		Port string
	}

	Admin struct {
		APIKey string
		Salt   string
	}

	Environment Environment
}

func Load() Config {
	return load()
}

func load() Config {
	v := viper.New()

	v.SetDefault("DataDir", "./data")
	v.SetDefault("PublicDB.Filename", "public.db")
	v.SetDefault("PrivateDB.Filename", "private.db")
	v.SetDefault("HTTP.Port", "8080")
	v.SetDefault("Environment", Development)

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
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return cfg
}

func GetProjectRootDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(wd)
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
