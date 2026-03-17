package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"file-transfer-client/internal/model"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(serverURL string) (*Client, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return nil, err
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}

	return &Client{
		baseURL:    parsed,
		httpClient: &http.Client{},
	}, nil
}

func (c *Client) ServerURL() string {
	return c.baseURL.String()
}

func NormalizeServerURL(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", fmt.Errorf("server URL is required")
	}

	if !strings.Contains(value, "://") {
		value = "http://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("server URL must include a valid host")
	}

	normalized := &url.URL{
		Scheme: parsed.Scheme,
		User:   parsed.User,
		Host:   parsed.Host,
		Path:   strings.TrimRight(parsed.Path, "/"),
	}
	return normalized.String(), nil
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/healthz", nil), nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseResponseError(resp)
	}

	return nil
}

func (c *Client) Browse(ctx context.Context, remotePath string) (model.Directory, error) {
	query := url.Values{}
	query.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(remoteRequestPath(remotePath), query), nil)
	if err != nil {
		return model.Directory{}, fmt.Errorf("create browse request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.Directory{}, fmt.Errorf("browse remote path: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.Directory{}, parseResponseError(resp)
	}

	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return model.Directory{}, fmt.Errorf("remote path is a file; download it instead")
	}

	var payload struct {
		Path    string `json:"path"`
		Entries []struct {
			Name         string    `json:"name"`
			Path         string    `json:"path"`
			IsDir        bool      `json:"is_dir"`
			Size         int64     `json:"size"`
			LastModified time.Time `json:"last_modified"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return model.Directory{}, fmt.Errorf("decode directory response: %w", err)
	}

	directory := model.Directory{
		Path:    stripFilesPrefix(payload.Path),
		Entries: make([]model.Entry, 0, len(payload.Entries)),
	}
	for _, entry := range payload.Entries {
		directory.Entries = append(directory.Entries, model.Entry{
			Name:         entry.Name,
			Path:         stripFilesPrefix(entry.Path),
			IsDir:        entry.IsDir,
			Size:         entry.Size,
			LastModified: entry.LastModified,
		})
	}

	if directory.Path == "" {
		directory.Path = "/"
	}

	return directory, nil
}

func (c *Client) Download(ctx context.Context, remotePath string, writer io.Writer, progress func(written, total int64)) error {
	query := url.Values{}
	query.Set("download", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(remoteRequestPath(remotePath), query), nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseResponseError(resp)
	}

	var written int64
	buffer := make([]byte, 32*1024)
	total := resp.ContentLength
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := writer.Write(buffer[:n]); err != nil {
				return fmt.Errorf("write file data: %w", err)
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}

		if readErr == nil {
			continue
		}
		if readErr == io.EOF {
			break
		}
		return fmt.Errorf("read file response: %w", readErr)
	}

	return nil
}

func (c *Client) endpoint(requestPath string, query url.Values) string {
	endpoint := *c.baseURL
	endpoint.Path = joinURLPath(c.baseURL.Path, requestPath)
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

func joinURLPath(basePath, requestPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" || basePath == "/" {
		return requestPath
	}
	return basePath + requestPath
}

func remoteRequestPath(remotePath string) string {
	cleaned := normalizeRemotePath(remotePath)
	if cleaned == "/" {
		return "/files/"
	}
	return "/files" + cleaned
}

func normalizeRemotePath(remotePath string) string {
	trimmed := strings.TrimSpace(remotePath)
	if trimmed == "" {
		return "/"
	}

	cleaned := path.Clean("/" + strings.TrimPrefix(trimmed, "/"))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func stripFilesPrefix(value string) string {
	if value == "" || value == "/files" || value == "/files/" {
		return "/"
	}

	trimmed := strings.TrimPrefix(value, "/files")
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func parseResponseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("server returned %s: %s", resp.Status, message)
}
