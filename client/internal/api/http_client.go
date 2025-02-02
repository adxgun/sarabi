package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type (
	MultipartFile struct {
		Content io.Reader
		Name    string
	}

	Params struct {
		Method      string
		Path        string
		Body        interface{}
		Response    interface{}
		QueryParams map[string]string
		Headers     map[string]string
	}

	Client interface {
		Do(ctx context.Context, param Params) error
		DoMultipart(ctx context.Context, files []MultipartFile, params Params) (io.ReadCloser, error)
		Download(ctx context.Context, param Params) (io.ReadCloser, error)
		SSE(ctx context.Context, param Params) (io.ReadCloser, error)
	}

	client struct {
		httpClient *http.Client
		baseUrl    string
		accessKey  string
	}
)

const (
	accessKeyHeader = "X-Access-Key"
)

func NewClient(cfg Config) Client {
	host := cfg.Host
	if !strings.HasSuffix(host, "v1") {
		host += "v1/"
	}

	return &client{
		httpClient: &http.Client{},
		baseUrl:    host,
		accessKey:  cfg.AccessKey,
	}
}

func (c client) Do(ctx context.Context, param Params) error {
	requestUrl, err := url.Parse(c.baseUrl + param.Path)
	if err != nil {
		return err
	}

	if len(param.QueryParams) > 0 {
		values := url.Values{}
		for k, v := range param.QueryParams {
			values.Add(k, v)
		}
		requestUrl.RawQuery = values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, param.Method, requestUrl.String(), nil)
	if err != nil {
		return err
	}
	if param.Body != nil {
		bodyBin, err := json.Marshal(param.Body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBin))
	}

	req.Header.Set("Content-Type", "application/json")
	if len(param.Headers) > 0 {
		for k, v := range param.Headers {
			req.Header.Set(k, v)
		}
	}

	if c.accessKey != "" {
		req.Header.Set(accessKeyHeader, c.accessKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			// eat
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return c.parseError(responseBody)
	}

	if param.Response != nil {
		if err := json.Unmarshal(responseBody, &param.Response); err != nil {
			return err
		}
	}
	return nil
}

func (c client) DoMultipart(ctx context.Context, files []MultipartFile, params Params) (io.ReadCloser, error) {
	jsonBytes, err := json.Marshal(params.Body)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	jsonPart, err := writer.CreateFormField("json")
	if err != nil {
		return nil, err
	}
	_, err = jsonPart.Write(jsonBytes)
	if err != nil {
		return nil, err
	}

	for _, nextFile := range files {
		filePart, err := writer.CreateFormFile("files", nextFile.Name)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(filePart, nextFile.Content)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	requestUrl := c.baseUrl + params.Path
	req, err := http.NewRequestWithContext(ctx, params.Method, requestUrl, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if len(params.Headers) > 0 {
		for k, v := range params.Headers {
			req.Header.Set(k, v)
		}
	}

	if c.accessKey != "" {
		req.Header.Set(accessKeyHeader, c.accessKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c client) Download(ctx context.Context, param Params) (io.ReadCloser, error) {
	downloadUrl, err := url.Parse(c.baseUrl + param.Path)
	if err != nil {
		return nil, err
	}

	if len(param.QueryParams) > 0 {
		values := url.Values{}
		for k, v := range param.QueryParams {
			values.Add(k, v)
		}
		downloadUrl.RawQuery = values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, param.Method, downloadUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	if len(param.Headers) > 0 {
		for k, v := range param.Headers {
			req.Header.Set(k, v)
		}
	}

	if c.accessKey != "" {
		req.Header.Set(accessKeyHeader, c.accessKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		return nil, c.parseError([]byte("error"))
	}

	return resp.Body, nil
}

func (c client) SSE(ctx context.Context, param Params) (io.ReadCloser, error) {
	sseUrl, err := url.Parse(c.baseUrl + param.Path)
	if err != nil {
		return nil, err
	}

	if len(param.QueryParams) > 0 {
		values := url.Values{}
		for k, v := range param.QueryParams {
			values.Add(k, v)
		}
		sseUrl.RawQuery = values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, param.Method, sseUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	if len(param.Headers) > 0 {
		for k, v := range param.Headers {
			req.Header.Set(k, v)
		}
	}

	if c.accessKey != "" {
		req.Header.Set(accessKeyHeader, c.accessKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c client) parseError(b []byte) error {
	var errorResponse struct {
		Message string
	}
	if err := json.Unmarshal(b, &errorResponse); err != nil {
		return err
	}
	return errors.New(errorResponse.Message)
}
