package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/justjanne/seafile-notifications/config"
	log "github.com/sirupsen/logrus"
)

type AppContext struct {
	Config        config.AppConfig
	CcnetDB       *sql.DB
	Subscriptions SubscriptionState
}

func main() {
	log.SetFormatter(&LogFormatter{})

	var app AppContext
	var err error

	if app.Config, err = config.LoadAppConfig(); err != nil {
		log.Fatalf("could not load config: %v", err)
	}
	log.SetLevel(app.Config.LogLevel)

	app.CcnetDB = loadCcnetDB(app.Config.Database)
	app.Subscriptions.Init()

	router := app.newHTTPRouter()

	log.Info("notification server started.")

	server := new(http.Server)
	server.Addr = fmt.Sprintf("%s:%d", app.Config.Host, app.Config.Port)
	server.Handler = router

	if err := server.ListenAndServe(); err != nil {
		log.Infof("notificationserver exiting: %v", err)
	}
}
