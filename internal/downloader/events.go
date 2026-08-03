package downloader

// DownloadEvent is the typed payload emitted for download progress via Wails events.
type DownloadEvent struct {
	OperationID   string  `json:"operationId"`
	InstanceID    string  `json:"instanceId"`
	Status        string  `json:"status"` // "downloading", "completed", "failed", "cancelled"
	FileProgress  int     `json:"fileProgress"`
	TotalFiles    int     `json:"totalFiles"`
	ByteProgress  int64   `json:"byteProgress"`
	TotalBytes    int64   `json:"totalBytes"`
	Speed         float64 `json:"speed"`
	ETA           float64 `json:"eta"` // seconds
	Error         string  `json:"error,omitempty"`
}

// Event types
const (
	EventStatusDownloading = "downloading"
	EventStatusCompleted   = "completed"
	EventStatusFailed      = "failed"
	EventStatusCancelled   = "cancelled"
)
