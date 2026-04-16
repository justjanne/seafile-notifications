package message

type CommentEvent struct {
	RepoID   string `json:"repo_id"`
	Type     string `json:"type"`
	FileUUID string `json:"file_uuid"`
	FilePath string `json:"file_path"`
}
