package logs

type (
	containerEvent string

	LogEntry struct {
		Owner string `json:"owner"`
		Log   string `json:"log"`
	}

	Client chan LogEntry
)

const (
	containerStart   containerEvent = "start"
	containerDestroy containerEvent = "destroy"
)
