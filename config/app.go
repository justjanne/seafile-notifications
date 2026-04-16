package config

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type AppConfig struct {
	ConfigDir  string
	PrivateKey string
	Host       string
	Port       int
	LogLevel   log.Level
	Database   DatabaseConfig
}

func LoadAppConfig() (AppConfig, error) {
	config := AppConfig{
		Host:     "0.0.0.0",
		Port:     8083,
		LogLevel: log.InfoLevel,
	}

	flag.StringVar(&config.ConfigDir, "c", "", "config directory")
	flag.Parse()

	if config.ConfigDir == "" {
		return config, fmt.Errorf("config directory must be specified")
	}
	if _, err := os.Stat(config.ConfigDir); errors.Is(err, fs.ErrNotExist) {
		return config, fmt.Errorf("config directory %s doesn't exist: %v", config.ConfigDir, err)
	}

	if val, ok := os.LookupEnv("NOTIFICATION_SERVER_HOST"); ok {
		config.Host = val
	}

	if val, ok := os.LookupEnv("NOTIFICATION_SERVER_PORT"); ok {
		if val, err := strconv.Atoi(val); err != nil {
			return config, fmt.Errorf("NOTIFICATION_SERVER_PORT could not be parsed: %w", err)
		} else {
			config.Port = val
		}
	}

	if level, ok := os.LookupEnv("NOTIFICATION_SERVER_LOG_LEVEL"); ok {
		if level, err := log.ParseLevel(level); err != nil {
			return config, fmt.Errorf("NOTIFICATION_SERVER_LOG_LEVEL could not be parsed: %w", err)
		} else {
			config.LogLevel = level
		}
	}

	if key, ok := os.LookupEnv("JWT_PRIVATE_KEY"); !ok {
		return config, fmt.Errorf("JWT_PRIVATE_KEY is missing")
	} else {
		config.PrivateKey = key
	}

	if db, err := LoadDatabaseConfig(); err != nil {
		return config, fmt.Errorf("could not read database config: %w", err)
	} else {
		config.Database = db
	}

	return config, nil
}
