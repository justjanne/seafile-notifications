package main

import (
	"fmt"
	"net/http"

	"github.com/justjanne/seafile-notifications/config"
	"github.com/justjanne/seafile-notifications/db"
	log "github.com/sirupsen/logrus"
)

type AppContext struct {
	Config        config.AppConfig
	Database      db.Database
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

	if app.Database, err = db.InitDatabase(app.Config.Database); err != nil {
		log.Fatalf("could not init database: %v", err)
	}
	app.Subscriptions = InitSubscriptions()

	router := app.newHTTPRouter()

	log.Info("notification server started.")

	server := new(http.Server)
	server.Addr = fmt.Sprintf("%s:%d", app.Config.Host, app.Config.Port)
	server.Handler = router

	if err := server.ListenAndServe(); err != nil {
		log.Infof("notificationserver exiting: %v", err)
	}
}
