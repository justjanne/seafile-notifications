package message

type RepoUpdateEvent struct {
	RepoID   string `json:"repo_id"`
	CommitID string `json:"commit_id"`
}
