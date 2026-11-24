// Package logx configures logging for t3-devices.
//
// Logging is off unless Init(true) is called (the --debug flag). Then records
// go to $XDG_STATE_HOME/t3-devices/debug.log, or
// ~/.local/state/t3-devices/debug.log when XDG_STATE_HOME is unset, empty or
// not absolute. Every message and attribute value passes through Redact
// before it is written, so pair links, pairing codes and tokens never reach
// disk.
package logx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	appDir  = "t3-devices"
	logName = "debug.log"
	dirMode = 0o700
	logMode = 0o600
)

var (
	logger atomic.Pointer[slog.Logger]

	mu      sync.Mutex
	logFile *os.File
)

func init() {
	logger.Store(discardLogger())
}

// Init configures logging, enabling the redacted debug log when debug is true.
// With debug false nothing is written and no file or directory is created.
// If the log file cannot be opened, logging is discarded.
func Init(debug bool) {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	if !debug {
		logger.Store(discardLogger())
		return
	}

	f, err := openLogFile()
	if err != nil {
		logger.Store(discardLogger())
		return
	}
	logFile = f
	logger.Store(newLogger(f))
}

// Logger returns the current logger. It is safe for concurrent use.
func Logger() *slog.Logger { return logger.Load() }

// Debug logs at debug level through the redacting logger.
func Debug(msg string, args ...any) { Logger().Debug(msg, args...) }

// Info logs at info level through the redacting logger.
func Info(msg string, args ...any) { Logger().Info(msg, args...) }

// Warn logs at warn level through the redacting logger.
func Warn(msg string, args ...any) { Logger().Warn(msg, args...) }

// Error logs at error level through the redacting logger.
func Error(msg string, args ...any) { Logger().Error(msg, args...) }

// LogPath returns the debug log path, honoring XDG_STATE_HOME.
func LogPath() (string, error) {
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" || !filepath.IsAbs(state) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		state = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(state, appDir, logName), nil
}

func openLogFile() (*os.File, error) {
	path, err := LogPath()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, dirMode); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, logMode)
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(logMode); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newLogger builds a debug-level logger writing redacted text records to w.
func newLogger(w io.Writer) *slog.Logger {
	inner := slog.NewTextHandler(redactWriter{w}, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(redactHandler{inner: inner})
}

// redactWriter redacts each formatted record again before it reaches w, as a
// second layer behind redactHandler.
type redactWriter struct{ w io.Writer }

func (r redactWriter) Write(p []byte) (int, error) {
	if _, err := io.WriteString(r.w, Redact(string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}

// redactHandler masks secrets in the message and every attribute value before
// passing the record to inner.
type redactHandler struct{ inner slog.Handler }

func (h redactHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h redactHandler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, Redact(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, out)
}

func (h redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	red := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		red[i] = redactAttr(a)
	}
	return redactHandler{inner: h.inner.WithAttrs(red)}
}

func (h redactHandler) WithGroup(name string) slog.Handler {
	return redactHandler{inner: h.inner.WithGroup(Redact(name))}
}

// redactAttr masks a sensitive key's whole value and redacts text in all
// other values, descending into groups.
func redactAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	a.Key = Redact(a.Key)
	if a.Value.Kind() == slog.KindGroup {
		group := a.Value.Group()
		red := make([]slog.Attr, len(group))
		for i, g := range group {
			red[i] = redactAttr(g)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(red...)}
	}
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, Redact(a.Value.String()))
	case slog.KindAny:
		return slog.String(a.Key, Redact(anyString(a.Value.Any())))
	default:
		return a
	}
}

// anyString formats an arbitrary value as text so it can be redacted. A
// panicking Error, String or MarshalJSON method (such as a typed nil pointer)
// falls back to fmt formatting instead of crashing the caller.
func anyString(v any) (s string) {
	defer func() {
		if recover() != nil {
			s = fmt.Sprintf("%+v", v)
		}
	}()
	switch t := v.(type) {
	case []byte:
		return string(t)
	case error:
		return t.Error()
	case fmt.Stringer:
		return t.String()
	}
	// Prefer JSON so struct and map field names stay visible to Redact. HTML
	// escaping is off so '&' in links is not hidden as \u0026.
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err == nil {
		return strings.TrimSuffix(b.String(), "\n")
	}
	return fmt.Sprintf("%+v", v)
}
