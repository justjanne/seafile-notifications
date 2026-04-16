package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func (state *AppContext) newHTTPRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", appHandler(func(writer http.ResponseWriter, request *http.Request) *appError {
		return state.messageCB(writer, request)
	}))
	r.Handle("/events{slash:\\/?}", appHandler(func(writer http.ResponseWriter, request *http.Request) *appError {
		return state.eventCB(writer, request)
	}))
	r.Handle("/ping{slash:\\/?}", appHandler(pingCB))

	return r
}

// Any http request will be automatically upgraded to websocket.
func (state *AppContext) messageCB(rsp http.ResponseWriter, r *http.Request) *appError {
	upgrader := newUpgrader()
	conn, err := upgrader.Upgrade(rsp, r, nil)
	if err != nil {
		log.Warnf("failed to upgrade http to websocket: %v", err)
		// Don't return eror here, because the upgrade fails, then Upgrade replies to the client with an HTTP error response.
		return nil
	}

	addr := r.Header.Get("x-forwarded-for")
	if addr == "" {
		addr = conn.RemoteAddr().String()
	}
	client := state.Subscriptions.NewClient(conn, addr)
	state.Subscriptions.RegisterClient(client)
	state.HandleMessages(client)

	return nil
}

func (state *AppContext) eventCB(rsp http.ResponseWriter, r *http.Request) *appError {
	msg := Message{}

	token := getAuthorizationToken(r.Header)
	if !state.checkAuthToken(token) {
		return &appError{Error: nil,
			Message: "Notification token not match",
			Code:    http.StatusBadRequest,
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return &appError{Error: err,
			Message: "",
			Code:    http.StatusInternalServerError,
		}
	}

	if err := json.Unmarshal(body, &msg); err != nil {
		return &appError{Error: err,
			Message: "",
			Code:    http.StatusInternalServerError,
		}
	}

	state.Notify(&msg)

	return nil
}

func newUpgrader() *websocket.Upgrader {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return upgrader
}

func pingCB(rsp http.ResponseWriter, r *http.Request) *appError {
	fmt.Fprintln(rsp, "{\"ret\": \"pong\"}")
	return nil
}

type appError struct {
	Error   error
	Message string
	Code    int
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e := fn(w, r)
	if e != nil {
		if e.Error != nil && e.Code == http.StatusInternalServerError {
			log.Infof("path %s internal server error: %v\n", r.URL.Path, e.Error)
		}
		http.Error(w, e.Message, e.Code)
	}
}
