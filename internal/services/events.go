package services

// Event types emitted via Wails app.Event.Emit
const (
	EventDownloadProgress = "download-progress"
	EventInstanceState    = "instance-state"
	EventLaunchState      = "launch-state"
	EventLogLine          = "log-line"
)

// DownloadProgressEvent is emitted during download operations.
type DownloadProgressEvent struct {
	OperationID   string  `json:"operationId"`
	InstanceID    string  `json:"instanceId"`
	Status        string  `json:"status"`
	FileProgress  int     `json:"fileProgress"`
	TotalFiles    int     `json:"totalFiles"`
	ByteProgress  int64   `json:"byteProgress"`
	TotalBytes    int64   `json:"totalBytes"`
	Speed         float64 `json:"speed"`
	ETA           float64 `json:"eta"`
	Error         string  `json:"error,omitempty"`
}

// InstanceStateEvent is emitted when instance state changes.
type InstanceStateEvent struct {
	InstanceID string `json:"instanceId"`
	OldState   string `json:"oldState"`
	NewState   string `json:"newState"`
}

// LaunchStateEvent is emitted during launch lifecycle.
type LaunchStateEvent struct {
	InstanceID string `json:"instanceId"`
	State      string `json:"state"`
	ExitCode   *int   `json:"exitCode,omitempty"`
}

// LogLineEvent is emitted for console/log output.
type LogLineEvent struct {
	Level       string `json:"level"`
	Message     string `json:"message"`
	OperationID string `json:"operationId,omitempty"`
	InstanceID  string `json:"instanceId,omitempty"`
}
