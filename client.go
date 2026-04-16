package main

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	"github.com/justjanne/seafile-notifications/message"
	log "github.com/sirupsen/logrus"
)

const (
	writeWait = 1 * time.Second
	pongWait  = 5 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 1 * time.Second

	checkTokenPeriod = 1 * time.Hour
)

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
		state.Subscriptions.UnsubscribeClient(client, id)
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
		var msg message.Message
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

func (state *AppContext) handleMessage(client *Client, msg *message.Message) error {
	content := msg.Content

	if msg.Type == "subscribe" {
		var list message.SubscribeMessage
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
			state.Subscriptions.SubscribeClient(client, repo.RepoID, user, exp)
		}
	} else if msg.Type == "unsubscribe" {
		var list message.UnsubscribeMessage
		err := json.Unmarshal(content, &list)
		if err != nil {
			return err
		}
		for _, r := range list.Repos {
			state.Subscriptions.UnsubscribeClient(client, r.RepoID)
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
			m, _ := msg.(*message.Message)
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
				state.Subscriptions.UnsubscribeClient(client, repoID)
				client.notifJWTExpired(repoID)
			}
		case <-client.Semaphore.HasBeenClosed():
			return
		}
	}
}

func (client *Client) notifJWTExpired(repoID string) {
	client.WCh <- message.Message{
		Type:    "jwt-expired",
		Content: json.RawMessage(fmt.Sprintf("{\"repo_id\":\"%s\"}", repoID)),
	}
}
