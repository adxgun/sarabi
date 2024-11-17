package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sarabi/logger"
	"time"
)

type HttpClient interface {
	Do(ctx context.Context, method, requestUrl string, body, response interface{}) error
}

type caddyHttpClient struct {
	client *http.Client
}

func newCaddyHttpClient() HttpClient {
	return caddyHttpClient{client: &http.Client{Timeout: 30 * time.Second}}
}

func (c caddyHttpClient) Do(ctx context.Context, method, requestUrl string, body, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, requestUrl, nil)
	if err != nil {
		return err
	}
	if body != nil {
		bodyBin, err := json.Marshal(body)
		if err != nil {
			return err
		}
		logger.Info("caddy request", zap.String("req", string(bodyBin)), zap.Any("headers", req.Header))
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBin))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("failed to close response body",
				zap.Error(err))
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	logger.Info("caddy response",
		zap.String("response", string(responseBody)))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("request failed: %s", string(responseBody))
	}

	if response != nil {
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return err
		}
	}
	return nil
}
