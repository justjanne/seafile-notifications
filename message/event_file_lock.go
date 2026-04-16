package message

type FileLockEvent struct {
	RepoID      string `json:"repo_id"`
	Path        string `json:"path"`
	ChangeEvent string `json:"change_event"`
	LockUser    string `json:"lock_user"`
}
