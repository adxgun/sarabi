package loki

import (
	"context"
	"fmt"
	"net/url"
	"sarabi/internal/integrations"
	"sarabi/internal/types"
	"strconv"
	"time"
)

type Client interface {
	Push(ctx context.Context, payloads map[string][]types.Batch) error
	Ready(ctx context.Context) error
	Query(ctx context.Context, filter types.Filter) ([]types.LogEntry, error)
}

type client struct {
	httpClient integrations.HttpClient
}

const lokiUrl = "http://localhost:3100/loki/api/v1/"

func NewClient() Client {
	return &client{
		httpClient: integrations.NewHttpClient(lokiUrl),
	}
}

func (c client) Push(ctx context.Context, payloads map[string][]types.Batch) error {
	for _, batches := range payloads {
		streams := make([]Stream, 0)
		for _, next := range batches {
			s := Stream{
				Stream: map[string]string{
					"environment": next.Deployment.Environment,
					"app":         next.Deployment.ApplicationID.String(),
				},
				Values: [][2]string{
					{
						next.Log.Ts, next.Log.Line(),
					},
				},
			}
			streams = append(streams, s)
		}

		payload := Payload{Streams: streams}
		err := c.httpClient.Do(ctx, "POST", "push", payload, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c client) Ready(ctx context.Context) error {
	return c.httpClient.Do(ctx, "GET", "http://localhost:3100/ready", nil, nil)
}

func (c client) Query(ctx context.Context, f types.Filter) ([]types.LogEntry, error) {
	u := url.Values{}
	u.Add("limit", "1000")
	u.Add("direction", "forward")
	if f.Since != "" {
		u.Add("since", f.Since)
	}

	if !f.Start.IsZero() {
		u.Add("start", strconv.FormatInt(f.Start.UnixNano(), 10))

		if f.End.IsZero() {
			u.Add("end", strconv.FormatInt(time.Now().UnixNano(), 10))
		} else {
			u.Add("end", strconv.FormatInt(f.End.UnixNano(), 10))
		}
	}

	query := fmt.Sprintf(`{app="%s", environment="%s"}`, f.ApplicationID, f.Environment)
	u.Add("query", query)

	queryUrl := "query_range?" + u.Encode()
	result := &QueryResult{}
	err := c.httpClient.Do(ctx, "GET", queryUrl, nil, result)
	if err != nil {
		return nil, err
	}

	resp := make([]types.LogEntry, 0)
	for _, next := range result.Data.Result {
		for _, value := range next.Values {
			resp = append(resp, types.LogEntry{
				Log: value[1],
				Ts:  value[0],
			})
		}
	}
	return resp, nil
}
