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
	}

	Client interface {
		Do(ctx context.Context, param Params) error
		DoMultipart(ctx context.Context, files []MultipartFile, params Params) error
	}

	client struct {
		httpClient *http.Client
		baseUrl    string
	}
)

func NewClient() (Client, error) {
	return &client{
		httpClient: &http.Client{},
		baseUrl:    "http://localhost:3646/v1/",
	}, nil
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

	if err := json.Unmarshal(responseBody, &param.Response); err != nil {
		return err
	}
	return nil
}

func (c client) DoMultipart(ctx context.Context, files []MultipartFile, params Params) error {
	jsonBytes, err := json.Marshal(params.Body)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	jsonPart, err := writer.CreateFormField("json")
	if err != nil {
		return err
	}
	_, err = jsonPart.Write(jsonBytes)
	if err != nil {
		return err
	}

	for _, nextFile := range files {
		filePart, err := writer.CreateFormFile("files", nextFile.Name)
		if err != nil {
			return err
		}
		_, err = io.Copy(filePart, nextFile.Content)
		if err != nil {
			return err
		}
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	url := c.baseUrl + params.Path
	req, err := http.NewRequestWithContext(ctx, params.Method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		return c.parseError(responseBody)
	}

	if err := json.Unmarshal(responseBody, &params.Response); err != nil {
		return err
	}
	return nil
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
