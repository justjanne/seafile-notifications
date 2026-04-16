package config

import (
	"fmt"
	"os"
	"strconv"
)

type DatabaseConfig struct {
	User          string
	Password      string
	Host          string
	Port          int
	CcnetDbName   string
	SeafileDbName string
	UnixSocket    string
	UseTLS        bool
}

func LoadDatabaseConfig() (DatabaseConfig, error) {
	config := DatabaseConfig{
		Port:          3306,
		CcnetDbName:   "ccnet_db",
		SeafileDbName: "seafile_db",
	}

	if val, ok := os.LookupEnv("SEAFILE_DB_HOST"); !ok {
		return config, fmt.Errorf("SEAFILE_DB_HOST is missing")
	} else {
		config.Host = val
	}

	if val, ok := os.LookupEnv("SEAFILE_DB_PORT"); ok {
		if val, err := strconv.Atoi(val); err != nil {
			return config, fmt.Errorf("SEAFILE_DB_PORT could not be parsed: %w", err)
		} else {
			config.Port = val
		}
	}

	if val, ok := os.LookupEnv("SEAFILE_CCNET_DB_NAME"); ok {
		config.CcnetDbName = val
	}

	if val, ok := os.LookupEnv("SEAFILE_SEAFILE_DB_NAME"); ok {
		config.SeafileDbName = val
	}

	if val, ok := os.LookupEnv("SEAFILE_DB_USER"); !ok {
		return config, fmt.Errorf("SEAFILE_DB_USER is missing")
	} else {
		config.Host = val
	}

	if val, ok := os.LookupEnv("SEAFILE_DB_PASSWORD"); !ok {
		return config, fmt.Errorf("SEAFILE_DB_PASSWORD is missing")
	} else {
		config.Host = val
	}

	return config, nil
}
