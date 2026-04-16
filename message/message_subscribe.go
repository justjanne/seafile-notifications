package message

type SubscribeMessage struct {
	Repos []Repo `json:"repos"`
}
