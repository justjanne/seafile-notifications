package message

type FolderPermEvent struct {
	RepoID      string `json:"repo_id"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	ChangeEvent string `json:"change_event"`
	User        string `json:"user"`
	Group       int    `json:"group"`
	Perm        string `json:"perm"`
}
