package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"file-transfer-client/internal/downloads"
	"file-transfer-client/internal/model"
	"file-transfer-client/internal/settings"
	"file-transfer-client/internal/transfer"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	settingsStore *settings.Store
	downloads     *downloads.Manager

	mu     sync.RWMutex
	client *transfer.Client
}

func NewApp() *App {
	app := &App{}
	app.settingsStore = settings.NewStore("")
	app.downloads = downloads.NewManager(3, app.emitDownloadUpdate)
	return app
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

func (a *App) Bootstrap() (model.AppState, error) {
	currentSettings, err := a.settingsStore.Load()
	if err != nil {
		return model.AppState{}, err
	}

	return model.AppState{
		Settings:  currentSettings,
		Downloads: a.downloads.List(),
	}, nil
}

func (a *App) Connect(serverURL string) (model.ConnectionState, error) {
	client, err := transfer.NewClient(serverURL)
	if err != nil {
		return model.ConnectionState{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		return model.ConnectionState{}, err
	}

	a.mu.Lock()
	a.client = client
	a.mu.Unlock()

	currentSettings, err := a.settingsStore.Update(func(s *model.Settings) {
		s.LastServerURL = client.ServerURL()
		s.RecentServers = prependRecent(s.RecentServers, client.ServerURL(), 6)
	})
	if err != nil {
		return model.ConnectionState{}, err
	}

	return model.ConnectionState{
		ServerURL: client.ServerURL(),
		CheckedAt: time.Now(),
		Settings:  currentSettings,
	}, nil
}

func (a *App) Browse(remotePath string) (model.Directory, error) {
	client := a.currentClient()
	if client == nil {
		return model.Directory{}, errors.New("connect to a server first")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	directory, err := client.Browse(ctx, remotePath)
	if err != nil {
		return model.Directory{}, err
	}

	if _, err := a.settingsStore.Update(func(s *model.Settings) {
		s.LastRemotePath = directory.Path
	}); err != nil {
		return model.Directory{}, err
	}

	return directory, nil
}

func (a *App) PickDownloadLocation(remotePath string) (string, error) {
	currentSettings, err := a.settingsStore.Load()
	if err != nil {
		return "", err
	}

	filename := path.Base(strings.TrimSuffix(strings.TrimSpace(remotePath), "/"))
	if filename == "." || filename == "/" || filename == "" {
		filename = "download.bin"
	}

	ctx := a.currentContext()
	if ctx == nil {
		return "", errors.New("application context is not ready")
	}

	return wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:            "Save file",
		DefaultDirectory: currentSettings.DefaultDownloadDir,
		DefaultFilename:  filename,
	})
}

func (a *App) StartDownload(req model.DownloadRequest) (model.DownloadTask, error) {
	client := a.currentClient()
	if client == nil {
		return model.DownloadTask{}, errors.New("connect to a server first")
	}

	req.RemotePath = strings.TrimSpace(req.RemotePath)
	req.SavePath = strings.TrimSpace(req.SavePath)
	if req.RemotePath == "" {
		return model.DownloadTask{}, errors.New("remote path is required")
	}
	if req.SavePath == "" {
		return model.DownloadTask{}, errors.New("save path is required")
	}

	task, err := a.downloads.Enqueue(req, client)
	if err != nil {
		return model.DownloadTask{}, err
	}

	if _, err := a.settingsStore.Update(func(s *model.Settings) {
		dir := filepath.Dir(req.SavePath)
		if dir != "" {
			s.DefaultDownloadDir = dir
		}
	}); err != nil {
		return task, nil
	}

	return task, nil
}

func (a *App) CancelDownload(taskID string) error {
	return a.downloads.Cancel(strings.TrimSpace(taskID))
}

func (a *App) OpenLocalPath(targetPath string) error {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return errors.New("path is required")
	}

	command, args := openCommand(targetPath)
	cmd := exec.Command(command, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open local path: %w", err)
	}
	return nil
}

func (a *App) PickDefaultDownloadDir() (string, error) {
	ctx := a.currentContext()
	if ctx == nil {
		return "", errors.New("application context is not ready")
	}

	currentSettings, err := a.settingsStore.Load()
	if err != nil {
		return "", err
	}

	return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:            "Choose default download folder",
		DefaultDirectory: currentSettings.DefaultDownloadDir,
	})
}

func (a *App) SetDefaultDownloadDir(dir string) (model.Settings, error) {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return model.Settings{}, errors.New("download directory is required")
	}

	info, err := os.Stat(trimmed)
	if err != nil {
		return model.Settings{}, fmt.Errorf("read download directory: %w", err)
	}
	if !info.IsDir() {
		return model.Settings{}, fmt.Errorf("%q is not a directory", trimmed)
	}

	return a.settingsStore.Update(func(s *model.Settings) {
		s.DefaultDownloadDir = trimmed
	})
}

func (a *App) currentClient() *transfer.Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

func (a *App) currentContext() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ctx
}

func (a *App) emitDownloadUpdate(task model.DownloadTask) {
	ctx := a.currentContext()
	if ctx == nil {
		return
	}

	wailsruntime.EventsEmit(ctx, "download:updated", task)
}

func prependRecent(items []string, value string, limit int) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return items
	}

	next := make([]string, 0, len(items)+1)
	next = append(next, trimmed)
	for _, item := range items {
		if item == "" || item == trimmed {
			continue
		}
		next = append(next, item)
		if limit > 0 && len(next) >= limit {
			break
		}
	}
	return next
}

func openCommand(targetPath string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{targetPath}
	case "windows":
		return "cmd", []string{"/c", "start", "", targetPath}
	default:
		return "xdg-open", []string{targetPath}
	}
}
