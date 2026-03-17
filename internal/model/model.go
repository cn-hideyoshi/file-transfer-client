package model

import "time"

const (
	DownloadStateQueued   = "queued"
	DownloadStateRunning  = "running"
	DownloadStateDone     = "done"
	DownloadStateFailed   = "failed"
	DownloadStateCanceled = "canceled"
)

type Settings struct {
	LastServerURL      string   `json:"last_server_url"`
	LastRemotePath     string   `json:"last_remote_path"`
	DefaultDownloadDir string   `json:"default_download_dir"`
	RecentServers      []string `json:"recent_servers"`
}

type AppState struct {
	Settings  Settings       `json:"settings"`
	Downloads []DownloadTask `json:"downloads"`
}

type ConnectionState struct {
	ServerURL string    `json:"server_url"`
	CheckedAt time.Time `json:"checked_at"`
	Settings  Settings  `json:"settings"`
}

type Directory struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

type DownloadRequest struct {
	RemotePath string `json:"remote_path"`
	SavePath   string `json:"save_path"`
	Overwrite  bool   `json:"overwrite"`
}

type DownloadTask struct {
	ID           string    `json:"id"`
	RemotePath   string    `json:"remote_path"`
	SavePath     string    `json:"save_path"`
	TempPath     string    `json:"temp_path,omitempty"`
	TotalBytes   int64     `json:"total_bytes"`
	WrittenBytes int64     `json:"written_bytes"`
	State        string    `json:"state"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
}
