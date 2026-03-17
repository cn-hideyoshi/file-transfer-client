package downloads

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"file-transfer-client/internal/model"
)

type stubDownloader struct {
	content []byte
	block   chan struct{}
}

func (s stubDownloader) Download(ctx context.Context, _ string, writer io.Writer, progress func(written, total int64)) error {
	if s.block != nil {
		select {
		case <-s.block:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if _, err := writer.Write(s.content); err != nil {
		return err
	}
	if progress != nil {
		progress(int64(len(s.content)), int64(len(s.content)))
	}
	return nil
}

func TestManagerCompletesDownload(t *testing.T) {
	manager := NewManager(1, nil)
	target := filepath.Join(t.TempDir(), "hello.txt")

	task, err := manager.Enqueue(model.DownloadRequest{
		RemotePath: "/hello.txt",
		SavePath:   target,
	}, stubDownloader{content: []byte("hello")})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	waitForState(t, manager, task.ID, model.DownloadStateDone)

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("downloaded file = %q, want hello", string(data))
	}
}

func TestManagerCancelsDownload(t *testing.T) {
	manager := NewManager(1, nil)
	block := make(chan struct{})

	task, err := manager.Enqueue(model.DownloadRequest{
		RemotePath: "/hello.txt",
		SavePath:   filepath.Join(t.TempDir(), "hello.txt"),
	}, stubDownloader{content: []byte("hello"), block: block})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if err := manager.Cancel(task.ID); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	close(block)

	waitForState(t, manager, task.ID, model.DownloadStateCanceled)
}

func waitForState(t *testing.T, manager *Manager, taskID, want string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, task := range manager.List() {
			if task.ID != taskID {
				continue
			}
			if task.State == want {
				return
			}
			if task.State == model.DownloadStateFailed {
				t.Fatalf("task failed unexpectedly: %s", task.ErrorMessage)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("task %s did not reach state %s", taskID, want)
}
