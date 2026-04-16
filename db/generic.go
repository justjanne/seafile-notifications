package db

import (
	"fmt"

	"github.com/justjanne/seafile-notifications/config"
)

type Database interface {
	GetGroupMembers(group int) (map[string]struct{}, error)
}

func InitDatabase(config config.DatabaseConfig) (Database, error) {
	switch config.Type {
	case "mysql":
		return InitMysqlDatabase(config)
	default:
		return nil, fmt.Errorf("failed to open database: unknown type %s", config.Type)
	}
}
