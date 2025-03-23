package types

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"regexp"
	"time"
)

type (
	Log struct {
		ID            uuid.UUID `gorm:"primaryKey" json:"id"`
		DeploymentID  uuid.UUID `json:"deployment_id"`
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		Location      string    `json:"location"`
		StorageType   string    `json:"storage_type"`
		ContainerID   string    `json:"container_id"`
		Timestamp     time.Time `json:"created_at"`
	}

	FilterParams struct {
		Environment string
		Start       *string
		End         *string
		Since       *string
		Limit       *int64
	}

	Filter struct {
		Environment   string
		Start         time.Time
		End           time.Time
		Since         string
		ApplicationID uuid.UUID
		Limit         int64

		Identifier string
	}

	LogEntry struct {
		Owner string `json:"owner"`
		Log   string `json:"log"`
		Ts    string `json:"ts"`
	}

	Batch struct {
		Log        LogEntry
		Deployment *Deployment
	}
)

func (l LogEntry) Line() string {
	return fmt.Sprintf("%s %s", l.Owner, l.Log)
}

func (f FilterParams) Validate() (*Filter, error) {
	if f.Environment == "" {
		return nil, errors.New("'environment' is blank")
	}

	if f.Since != nil && *f.Since != "" {
		ok := regexp.MustCompile(`^[1-9]\d*[smhdw]$`).MatchString(*f.Since)
		if !ok {
			return nil, fmt.Errorf("invalid 'since' format %s", *f.Since)
		}
	}

	r := &Filter{}
	if f.Start != nil && *f.Start != "" {
		start, err := f.parseTime(*f.Start)
		if err != nil {
			return nil, err
		}
		r.Start = start
	}

	if f.End != nil && *f.End != "" {
		end, err := f.parseTime(*f.End)
		if err != nil {
			return nil, err
		}
		r.End = end
	}

	r.Environment = f.Environment
	r.Since = *f.Since

	if !r.Start.IsZero() && r.Since != "" {
		return nil, errors.New("you cannot use both 'start' and 'since' to query logs")
	}

	if f.Limit != nil {
		if *f.Limit <= 0 {
			r.Limit = 50
		} else {
			r.Limit = *f.Limit
		}
	} else {
		r.Limit = 50
	}

	return r, nil
}

func (f FilterParams) parseTime(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as either date+time or date-only: %v", s, err)
}
