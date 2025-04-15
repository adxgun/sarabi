package integrations

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

type impl struct {
	client  *http.Client
	baseUrl string
	debug   bool
}

func NewHttpClient(baseUrl string, debug bool) HttpClient {
	return impl{client: &http.Client{Timeout: 30 * time.Second}, baseUrl: baseUrl, debug: debug}
}

func (c impl) Do(ctx context.Context, method, requestUrl string, body, response interface{}) error {
	ctx = logger.With(ctx,
		zap.String(logger.FunctionName, "httpClient.Do"),
		zap.String("url", requestUrl),
		zap.String("method", method),
		zap.Any("body", body))

	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+requestUrl, nil)
	if err != nil {
		return err
	}
	if body != nil {
		bodyBin, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBin))
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
		}
	}()

	if c.debug {
		logger.Info(ctx, "http response")
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, "failed to read response body")
		return err
	}

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
