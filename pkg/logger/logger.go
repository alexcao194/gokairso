package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DefaultDir    = "logs"
	DefaultPrefix = "app"
)

type DailyFileWriter struct {
	dir         string
	prefix      string
	mu          sync.Mutex
	currentDate string
	file        *os.File
}

func InitDefault() (*DailyFileWriter, error) {
	writer := NewDailyFileWriter(DefaultDir, DefaultPrefix)
	if err := writer.Open(); err != nil {
		return nil, err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	return writer, nil
}

func NewDailyFileWriter(dir string, prefix string) *DailyFileWriter {
	return &DailyFileWriter{
		dir:    dir,
		prefix: prefix,
	}
}

func (w *DailyFileWriter) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.fileForDateLocked(time.Now())
	return err
}

func (w *DailyFileWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := w.fileForDateLocked(time.Now())
	if err != nil {
		return 0, err
	}
	return file.Write(data)
}

func (w *DailyFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *DailyFileWriter) fileForDateLocked(now time.Time) (*os.File, error) {
	date := now.Format("2006-01-02")
	if w.file != nil && w.currentDate == date {
		return w.file, nil
	}

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return nil, err
	}

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return nil, err
		}
		w.file = nil
	}

	file, err := os.OpenFile(filepath.Join(w.dir, fmt.Sprintf("%s-%s.txt", w.prefix, date)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	w.currentDate = date
	w.file = file
	return w.file, nil
}
