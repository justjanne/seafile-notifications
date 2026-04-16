package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/justjanne/seafile-notifications/config"
	"github.com/lib/pq"
)

type PgsqlDatabase struct {
	connection *sql.DB
}

func InitPgsqlDatabase(config config.DatabaseConfig) (*PgsqlDatabase, error) {
	var sslMode pq.SSLMode
	if config.UseTLS {
		sslMode = pq.SSLModeVerifyFull
	} else {
		sslMode = pq.SSLModeDisable
	}
	connector, err := pq.NewConnectorConfig(pq.Config{
		Host:           config.Host,
		Port:           config.Port,
		User:           config.User,
		Password:       config.Password,
		Database:       config.CcnetDbName,
		ConnectTimeout: 5 * time.Second,
		SSLMode:        sslMode,
	})
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

	return &PgsqlDatabase{
		connection: db,
	}, nil
}

func (db *PgsqlDatabase) GetGroupMembers(group int) (map[string]struct{}, error) {
	query := `SELECT user_name FROM "groupuser" WHERE group_id = ?`
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	stmt, err := db.connection.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare sql: %s：%v", query, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to query sql: %v", err)
	}
	defer rows.Close()

	userList := make(map[string]struct{})
	var userName string

	for rows.Next() {
		if err := rows.Scan(&userName); err == nil {
			userList[userName] = struct{}{}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan sql rows: %v", err)
	}

	return userList, nil
}
