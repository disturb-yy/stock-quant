package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// OutputConfig controls where application logs are written.
type OutputConfig struct {
	Destination string
	Directory   string
	Service     string
}

// OpenOutput opens the configured log destination.
func OpenOutput(config OutputConfig) (io.WriteCloser, error) {
	switch strings.ToLower(strings.TrimSpace(config.Destination)) {
	case "", "console":
		return writeCloser{Writer: os.Stdout}, nil
	case "file":
		return openFileOutput(config)
	default:
		return nil, fmt.Errorf("unsupported log output %q", config.Destination)
	}
}

func openFileOutput(config OutputConfig) (io.WriteCloser, error) {
	directory := strings.TrimSpace(config.Directory)
	if directory == "" {
		return nil, fmt.Errorf("log directory is required for file output")
	}

	service := strings.TrimSpace(config.Service)
	if !isSafeLogFileName(service) {
		return nil, fmt.Errorf("log service name %q cannot be used as a file name", config.Service)
	}

	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("create log directory %q: %w", directory, err)
	}

	path := filepath.Join(directory, service+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	return file, nil
}

func isSafeLogFileName(service string) bool {
	return service != "" && service != "." && service != ".." && !strings.ContainsAny(service, `/\`)
}

type writeCloser struct {
	io.Writer
}

func (writeCloser) Close() error {
	return nil
}
