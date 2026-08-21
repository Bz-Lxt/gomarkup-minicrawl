package crawler

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusStopped   Status = "stopped"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Strategy struct {
	Seeds     []string `json:"seeds"`
	Workers   int      `json:"workers"`
	GlobalRPS float64  `json:"global_rps"`
	HostRPS   float64  `json:"host_rps"`
	MaxDepth  int      `json:"max_depth"`
	MaxPages  int      `json:"max_pages"`
	UserAgent string   `json:"user_agent"`
}

type TaskView struct {
	ID          string   `json:"id"`
	Status      Status   `json:"status"`
	Strategy    Strategy `json:"strategy"`
	Crawled     int      `json:"crawled"`
	Failures    int      `json:"failures"`
	QueueLength int      `json:"queue_length"`
	Error       string   `json:"error,omitempty"`
	CreatedAt   string   `json:"created_at"`
	StartedAt   string   `json:"started_at,omitempty"`
	FinishedAt  string   `json:"finished_at,omitempty"`
}

type Sample struct {
	Time  string  `json:"t"`
	Pages int     `json:"pages"`
	RPS   float64 `json:"rps"`
}
