package engine

import (
	"os"
	"sync"
	"time"
)

const (
	defaultLogLines      = 1000
	defaultBatchSize     = 10
	defaultFlushInterval = 100 * time.Millisecond
)

type Logger struct {
	mu       sync.Mutex
	lines    []string
	capacity int
	head     int
	count    int

	filePath string
	file     *os.File
	ch       chan string
	closed   bool
}

func NewLogger(filePath string, capacity int) *Logger {
	if capacity <= 0 {
		capacity = defaultLogLines
	}

	l := &Logger{
		lines:    make([]string, capacity),
		capacity: capacity,
		filePath: filePath,
		ch:       make(chan string, 100),
	}

	if err := l.openFile(); err != nil {
		return l
	}

	go l.writer()

	return l
}

func (l *Logger) openFile() error {
	if l.filePath == "" {
		return nil
	}

	dir := ""
	// Extract directory from file path
	for i := len(l.filePath) - 1; i >= 0; i-- {
		if l.filePath[i] == '/' || l.filePath[i] == '\\' {
			dir = l.filePath[:i]
			break
		}
	}
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	l.file = f
	return nil
}

func (l *Logger) Write(msg string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}

	l.lines[l.head] = msg
	l.head = (l.head + 1) % l.capacity
	if l.count < l.capacity {
		l.count++
	}

	select {
	case l.ch <- msg:
	default:
	}
}

func (l *Logger) ReadAll() string {
	if l == nil {
		return ""
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.count == 0 {
		return ""
	}

	start := 0
	if l.count >= l.capacity {
		start = l.head
	}

	var result []byte
	for i := 0; i < l.count; i++ {
		idx := (start + i) % l.capacity
		if l.lines[idx] != "" {
			result = append(result, l.lines[idx]...)
			result = append(result, '\n')
		}
	}

	return string(result)
}

func (l *Logger) Chan() <-chan string {
	if l == nil {
		return nil
	}
	return l.ch
}

func (l *Logger) writer() {
	batch := make([]string, 0, defaultBatchSize)
	ticker := time.NewTicker(defaultFlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 || l.file == nil {
			return
		}

		l.mu.Lock()
		defer l.mu.Unlock()

		for _, msg := range batch {
			l.file.WriteString(msg + "\n")
		}
		batch = batch[:0]
	}

	for {
		select {
		case msg, ok := <-l.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, msg)
			if len(batch) >= defaultBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (l *Logger) Close() {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}
	l.closed = true

	close(l.ch)

	if l.file != nil {
		l.file.Close()
	}
}
