package main

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	writeWait = 1 * time.Second
	pongWait  = 5 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 1 * time.Second

	checkTokenPeriod = 1 * time.Hour
)

// Message is the message communicated between clients and server.
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type SubList struct {
	Repos []Repo `json:"repos"`
}

type UnsubList struct {
	Repos []Repo `json:"repos"`
}

type Repo struct {
	RepoID string `json:"id"`
	Token  string `json:"jwt_token"`
}

type myClaims struct {
	Exp      int64  `json:"exp"`
	RepoID   string `json:"repo_id"`
	UserName string `json:"username"`
	jwt.RegisteredClaims
}

func (*myClaims) Valid() error {
	return nil
}

func (client *Client) Close() {
	client.conn.Close()
}

func RecoverWrapper(f func()) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic: %v\n%s", err, debug.Stack())
		}
	}()

	f()
}

// HandleMessages connects to the client to process message.
func (state *AppContext) HandleMessages(client *Client) {
	// Set keep alive.
	client.conn.SetPongHandler(func(string) error {
		client.Alive = time.Now()
		return nil
	})

	client.Semaphore.AddRunning(4)
	go RecoverWrapper(func() {
		state.ClientReadCoroutine(client)
	})
	go RecoverWrapper(func() {
		client.ClientWriteCoroutine()
	})
	go RecoverWrapper(func() {
		state.ClientTokenExpirationCoroutine(client)
	})
	go RecoverWrapper(func() {
		client.ClientKeepaliveCoroutine()
	})
	client.Semaphore.Wait()

	client.Close()
	state.Subscriptions.UnregisterClient(client)
	for id := range client.Repos {
		state.UnsubscribeClient(client, id)
	}
}

func (state *AppContext) ClientReadCoroutine(client *Client) {
	conn := client.conn
	defer func() {
		client.Semaphore.Done()
	}()

	for {
		select {
		case <-client.Semaphore.HasBeenClosed():
			return
		default:
		}
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			client.Semaphore.Signal()
			log.Debugf("failed to read json data from client: %s: %v", client.Addr, err)
			return
		}

		err = state.handleMessage(client, &msg)

		if err != nil {
			client.Semaphore.Signal()
			log.Debugf("%v", err)
			return
		}
	}
}

func (state *AppContext) handleMessage(client *Client, msg *Message) error {
	content := msg.Content

	if msg.Type == "subscribe" {
		var list SubList
		err := json.Unmarshal(content, &list)
		if err != nil {
			return err
		}
		for _, repo := range list.Repos {
			user, exp, valid := state.checkToken(repo.Token, repo.RepoID)
			if !valid {
				client.notifJWTExpired(repo.RepoID)
				continue
			}
			state.SubscribeClient(client, repo.RepoID, user, exp)
		}
	} else if msg.Type == "unsubscribe" {
		var list UnsubList
		err := json.Unmarshal(content, &list)
		if err != nil {
			return err
		}
		for _, r := range list.Repos {
			state.UnsubscribeClient(client, r.RepoID)
		}
	} else {
		err := fmt.Errorf("recv unexpected type of message: %s", msg.Type)
		return err
	}

	return nil
}

func (client *Client) ClientWriteCoroutine() {
	defer func() {
		client.Semaphore.Done()
	}()

	for {
		select {
		case msg := <-client.WCh:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			client.connMutex.Lock()
			err := client.conn.WriteJSON(msg)
			client.connMutex.Unlock()
			if err != nil {
				client.Semaphore.Signal()
				log.Debugf("failed to send notification to client: %v", err)
				return
			}
			m, _ := msg.(*Message)
			log.Debugf("send %s event to client %s(%d): %s", m.Type, client.User, client.ID, string(m.Content))
		case <-client.Semaphore.HasBeenClosed():
			return
		}
	}
}

func (client *Client) ClientKeepaliveCoroutine() {
	defer func() {
		client.Semaphore.Done()
	}()

	ticker := time.NewTicker(pingPeriod)
	for {
		select {
		case <-ticker.C:
			if time.Since(client.Alive) > pongWait {
				client.Semaphore.Signal()
				log.Debugf("disconnected because no pong was received for more than %v", pongWait)
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			client.connMutex.Lock()
			err := client.conn.WriteMessage(websocket.PingMessage, nil)
			client.connMutex.Unlock()
			if err != nil {
				client.Semaphore.Signal()
				log.Debugf("failed to send ping message to client: %v", err)
				return
			}
		case <-client.Semaphore.HasBeenClosed():
			return
		}
	}
}

func (state *AppContext) ClientTokenExpirationCoroutine(client *Client) {
	defer func() {
		client.Semaphore.Done()
	}()

	ticker := time.NewTicker(checkTokenPeriod)
	for {
		select {
		case <-ticker.C:
			// unsubscribe will delete repo from client.Repos, we'd better unsubscribe repos later.
			pendingRepos := make(map[string]struct{})
			now := time.Now()
			client.ReposMutex.Lock()
			for repoID, exp := range client.Repos {
				if exp >= now.Unix() {
					continue
				}
				pendingRepos[repoID] = struct{}{}
			}
			client.ReposMutex.Unlock()

			for repoID := range pendingRepos {
				state.UnsubscribeClient(client, repoID)
				client.notifJWTExpired(repoID)
			}
		case <-client.Semaphore.HasBeenClosed():
			return
		}
	}
}

func (client *Client) notifJWTExpired(repoID string) {
	msg := new(Message)
	msg.Type = "jwt-expired"
	content := fmt.Sprintf("{\"repo_id\":\"%s\"}", repoID)
	msg.Content = []byte(content)
	client.WCh <- msg
}
