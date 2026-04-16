package message

type UnsubscribeMessage struct {
	Repos []Repo `json:"repos"`
}
