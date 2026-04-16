package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/justjanne/seafile-notifications/config"
)

type MysqlDatabase struct {
	connection *sql.DB
}

func InitMysqlDatabase(config config.DatabaseConfig) (*MysqlDatabase, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%t&readTimeout=60s&writeTimeout=60s", config.User, config.Password, config.Host, config.Port, config.CcnetDbName, config.UseTLS)
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

	return &MysqlDatabase{
		connection: db,
	}, nil
}

func (db *MysqlDatabase) GetGroupMembers(group int) (map[string]struct{}, error) {
	query := `SELECT user_name FROM GroupUser WHERE group_id = ?`
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
