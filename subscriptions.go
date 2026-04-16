package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/z"
	"github.com/gorilla/websocket"
)

const (
	chanBufSize = 10
)

type SubscriptionState struct {
	// clients is a map from client id to Client structs.
	// It contains all current connected clients. Each client is identified by 64-bit ID.
	Clients      map[uint64]*Client
	ClientsMutex sync.RWMutex

	// Use atomic operation to increase this value.
	NextClientID uint64

	// subscriptions is a map from repo_id to Subscribers struct.
	// It's protected by rw mutex.
	Subscriptions map[string]*Subscribers
	SubMutex      sync.RWMutex
}

// Client contains information about a client.
// Two go routines are associated with each client to handle message reading and writting.
// Messages sent to the client have to be written into WCh, since only one go routine can write to a websocket connection.
type Client struct {
	// The ID of this client
	ID uint64
	// Websocket connection.
	conn *websocket.Conn
	// Connections do not support concurrent writers. Protect write with a mutex.
	connMutex sync.Mutex

	// WCh is used to write messages to a client.
	// The structs written into the channel will be converted to JSON and sent to client.
	WCh chan interface{}

	// Repos is the repos this client subscribed to.
	Repos      map[string]int64
	ReposMutex sync.Mutex
	// Alive is the last time received pong.
	Alive     time.Time
	Semaphore *z.Closer
	// Addr is the address of client.
	Addr string
	// User is the user of client.
	User string
}

// Subscribers contains the clients who subscribe to a repo's notifications.
type Subscribers struct {
	// Clients is a map from client id to Client struct, protected by rw mutex.
	Clients map[uint64]*Client
	Mutex   sync.RWMutex
}

// Init inits clients and subscriptions.
func (state *SubscriptionState) Init() {
	state.Clients = make(map[uint64]*Client)
	state.Subscriptions = make(map[string]*Subscribers)
	state.NextClientID = 1
}

// NewClient creates a new client.
func (state *SubscriptionState) NewClient(conn *websocket.Conn, addr string) *Client {
	client := new(Client)
	client.ID = atomic.AddUint64(&state.NextClientID, 1)
	client.conn = conn
	client.WCh = make(chan interface{}, chanBufSize)
	client.Repos = make(map[string]int64)
	client.Alive = time.Now()
	client.Addr = addr
	client.Semaphore = z.NewCloser(0)

	return client
}

// Register adds the client to the list of clients.
func (state *SubscriptionState) RegisterClient(client *Client) {
	state.ClientsMutex.Lock()
	state.Clients[client.ID] = client
	state.ClientsMutex.Unlock()
}

// Unregister deletes the client from the list of clients.
func (state *SubscriptionState) UnregisterClient(client *Client) {
	state.ClientsMutex.Lock()
	delete(state.Clients, client.ID)
	state.ClientsMutex.Unlock()
}

// subscribe subscribes to notifications of repos.
func (state *AppContext) SubscribeClient(client *Client, repoID, user string, exp int64) {
	client.User = user

	client.ReposMutex.Lock()
	client.Repos[repoID] = exp
	client.ReposMutex.Unlock()

	state.Subscriptions.SubMutex.Lock()
	subscribers, ok := state.Subscriptions.Subscriptions[repoID]
	if !ok {
		subscribers := new(Subscribers)
		subscribers.Clients = make(map[uint64]*Client)
		subscribers.Clients[client.ID] = client
		state.Subscriptions.Subscriptions[repoID] = subscribers
	}
	state.Subscriptions.SubMutex.Unlock()

	subscribers.Mutex.Lock()
	subscribers.Clients[client.ID] = client
	subscribers.Mutex.Unlock()
}

func (state *AppContext) UnsubscribeClient(client *Client, repoID string) {
	client.ReposMutex.Lock()
	delete(client.Repos, repoID)
	client.ReposMutex.Unlock()

	state.Subscriptions.SubMutex.Lock()
	subscribers, ok := state.Subscriptions.Subscriptions[repoID]
	if !ok {
		state.Subscriptions.SubMutex.Unlock()
		return
	}
	state.Subscriptions.SubMutex.Unlock()

	subscribers.Mutex.Lock()
	delete(subscribers.Clients, client.ID)
	subscribers.Mutex.Unlock()
}
