package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/justjanne/seafile-notifications/config"
	log "github.com/sirupsen/logrus"
)

func loadCcnetDB(config config.DatabaseConfig) *sql.DB {
	var dsn string
	if config.UnixSocket == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%t&readTimeout=60s&writeTimeout=60s", config.User, config.Password, config.Host, config.Port, config.CcnetDbName, config.UseTLS)
	} else {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?readTimeout=60s&writeTimeout=60s", config.User, config.Password, config.UnixSocket, config.CcnetDbName)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connected to mysql: %v", err)
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

	return db
}
