package logs

import "sarabi/internal/types"

type (
	containerEvent string

	Client chan types.LogEntry
)

const (
	containerStart   containerEvent = "start"
	containerDestroy containerEvent = "destroy"
)
