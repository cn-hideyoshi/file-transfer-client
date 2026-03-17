package downloads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"file-transfer-client/internal/model"
)

type Downloader interface {
	Download(ctx context.Context, remotePath string, writer io.Writer, progress func(written, total int64)) error
}

type Manager struct {
	mu     sync.RWMutex
	tasks  map[string]*taskState
	slots  chan struct{}
	emit   func(model.DownloadTask)
	nextID atomic.Uint64
}

type taskState struct {
	task      model.DownloadTask
	cancel    context.CancelFunc
	createdAt time.Time
}

func NewManager(concurrency int, emit func(model.DownloadTask)) *Manager {
	if concurrency < 1 {
		concurrency = 1
	}

	return &Manager{
		tasks: make(map[string]*taskState),
		slots: make(chan struct{}, concurrency),
		emit:  emit,
	}
}

func (m *Manager) Enqueue(req model.DownloadRequest, downloader Downloader) (model.DownloadTask, error) {
	if downloader == nil {
		return model.DownloadTask{}, errors.New("downloader is required")
	}

	savePath := strings.TrimSpace(req.SavePath)
	remotePath := strings.TrimSpace(req.RemotePath)
	if savePath == "" || remotePath == "" {
		return model.DownloadTask{}, errors.New("remote path and save path are required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := model.DownloadTask{
		ID:         fmt.Sprintf("task-%d", m.nextID.Add(1)),
		RemotePath: remotePath,
		SavePath:   savePath,
		TempPath:   savePath + ".part",
		State:      model.DownloadStateQueued,
	}

	state := &taskState{
		task:      task,
		cancel:    cancel,
		createdAt: time.Now(),
	}

	m.mu.Lock()
	m.tasks[task.ID] = state
	m.mu.Unlock()
	m.emitTask(task)

	go m.run(ctx, state, req, downloader)

	return task, nil
}

func (m *Manager) Cancel(taskID string) error {
	m.mu.RLock()
	state, ok := m.tasks[taskID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("download task %q not found", taskID)
	}

	state.cancel()
	return nil
}

func (m *Manager) List() []model.DownloadTask {
	m.mu.RLock()
	states := make([]*taskState, 0, len(m.tasks))
	for _, state := range m.tasks {
		states = append(states, state)
	}
	m.mu.RUnlock()

	sort.Slice(states, func(i, j int) bool {
		return states[i].createdAt.After(states[j].createdAt)
	})

	result := make([]model.DownloadTask, 0, len(states))
	for _, state := range states {
		result = append(result, state.task)
	}
	return result
}

func (m *Manager) run(ctx context.Context, state *taskState, req model.DownloadRequest, downloader Downloader) {
	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		m.finishCanceled(state.task.ID)
		return
	}
	defer func() { <-m.slots }()

	m.update(state.task.ID, func(task *model.DownloadTask) {
		task.State = model.DownloadStateRunning
		task.StartedAt = time.Now()
		task.ErrorMessage = ""
	})

	if err := os.MkdirAll(filepath.Dir(req.SavePath), 0o755); err != nil {
		m.finishFailed(state.task.ID, fmt.Errorf("create download directory: %w", err))
		return
	}

	if !req.Overwrite {
		if _, err := os.Stat(req.SavePath); err == nil {
			m.finishFailed(state.task.ID, fmt.Errorf("destination file already exists"))
			return
		}
	}

	tempPath := req.SavePath + ".part"
	_ = os.Remove(tempPath)

	file, err := os.Create(tempPath)
	if err != nil {
		m.finishFailed(state.task.ID, fmt.Errorf("create temp file: %w", err))
		return
	}

	progress := func(written, total int64) {
		m.update(state.task.ID, func(task *model.DownloadTask) {
			task.WrittenBytes = written
			task.TotalBytes = total
		})
	}

	downloadErr := downloader.Download(ctx, req.RemotePath, file, progress)
	closeErr := file.Close()
	if downloadErr == nil && closeErr != nil {
		downloadErr = closeErr
	}

	if downloadErr != nil {
		if errors.Is(downloadErr, context.Canceled) {
			m.finishCanceled(state.task.ID)
			return
		}
		m.finishFailed(state.task.ID, downloadErr)
		return
	}

	if req.Overwrite {
		_ = os.Remove(req.SavePath)
	}
	if err := os.Rename(tempPath, req.SavePath); err != nil {
		m.finishFailed(state.task.ID, fmt.Errorf("finalize download: %w", err))
		return
	}

	m.update(state.task.ID, func(task *model.DownloadTask) {
		task.State = model.DownloadStateDone
		if task.TotalBytes <= 0 {
			task.TotalBytes = task.WrittenBytes
		}
		task.WrittenBytes = task.TotalBytes
		task.FinishedAt = time.Now()
		task.ErrorMessage = ""
	})
}

func (m *Manager) finishCanceled(taskID string) {
	m.update(taskID, func(task *model.DownloadTask) {
		task.State = model.DownloadStateCanceled
		task.ErrorMessage = "download canceled"
		task.FinishedAt = time.Now()
	})
}

func (m *Manager) finishFailed(taskID string, err error) {
	m.update(taskID, func(task *model.DownloadTask) {
		task.State = model.DownloadStateFailed
		task.ErrorMessage = err.Error()
		task.FinishedAt = time.Now()
	})
}

func (m *Manager) update(taskID string, mutator func(*model.DownloadTask)) {
	m.mu.Lock()
	state, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	mutator(&state.task)
	snapshot := state.task
	m.mu.Unlock()

	m.emitTask(snapshot)
}

func (m *Manager) emitTask(task model.DownloadTask) {
	if m.emit != nil {
		m.emit(task)
	}
}
