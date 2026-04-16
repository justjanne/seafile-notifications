package main

import (
	"encoding/json"
	"reflect"
	"runtime/debug"

	"github.com/justjanne/seafile-notifications/message"
	log "github.com/sirupsen/logrus"
)

func (state *AppContext) Notify(msg *message.Message) {
	var repoID string
	// userList is the list of users who need to be notified, if it is nil, all subscribed users will be notified.
	var userList map[string]struct{}

	content := msg.Content
	switch msg.Type {
	case "repo-update":
		var event message.RepoUpdateEvent
		err := json.Unmarshal(content, &event)
		if err != nil {
			log.Warn(err)
			return
		}
		repoID = event.RepoID
	case "file-lock-changed":
		var event message.FileLockEvent
		err := json.Unmarshal(content, &event)
		if err != nil {
			log.Warn(err)
			return
		}
		repoID = event.RepoID
	case "folder-perm-changed":
		var event message.FolderPermEvent
		err := json.Unmarshal(content, &event)
		if err != nil {
			log.Warn(err)
			return
		}
		repoID = event.RepoID
		if event.User != "" {
			userList = make(map[string]struct{})
			userList[event.User] = struct{}{}
		} else if event.Group != -1 {
			userList, err = state.Database.GetGroupMembers(event.Group)
			if err != nil {
				log.Warn(err)
			}
		}
	case "comment-update":
		var event message.CommentEvent
		err := json.Unmarshal(content, &event)
		if err != nil {
			log.Warn(err)
			return
		}
		repoID = event.RepoID
	default:
		return
	}

	clients := make(map[uint64]*Client)

	state.Subscriptions.SubMutex.RLock()
	subscribers := state.Subscriptions.Subscriptions[repoID]
	if subscribers == nil {
		state.Subscriptions.SubMutex.RUnlock()
		return
	}
	state.Subscriptions.SubMutex.RUnlock()

	subscribers.Mutex.RLock()
	for clientID, client := range subscribers.Clients {
		clients[clientID] = client
	}
	subscribers.Mutex.RUnlock()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v\n%s", err, debug.Stack())
			}
		}()
		// In order to avoid being blocked on a Client for a long time, it is necessary to write WCh in a non-blocking way,
		// and the waiting WCh needs to be blocked and processed after other Clients have finished writing.
		value := reflect.ValueOf(msg)
		var branches []reflect.SelectCase
		for _, client := range clients {
			if !needToNotif(userList, client.User) {
				continue
			}
			branch := reflect.SelectCase{Dir: reflect.SelectSend, Chan: reflect.ValueOf(client.WCh), Send: value}
			branches = append(branches, branch)
		}

		for len(branches) != 0 {
			index, _, _ := reflect.Select(branches)
			branches = append(branches[:index], branches[index+1:]...)
		}
	}()
}

func needToNotif(userList map[string]struct{}, user string) bool {
	if userList == nil {
		return true
	}

	_, ok := userList[user]
	return ok
}
