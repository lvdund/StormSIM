package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	State     string    `json:"state,omitempty"`
}

type BufferedLogger struct {
	*Logger
	buffer       *RingBuffer[LogEntry]
	entityId     string
	entityType   string
	getState     func() string
	delayTracker *DelayTracker // Delay tracking for message timing
}

func NewBufferedLogger(
	bufferSize int,
	entityType string,
	entityId string,
	fields map[string]string,
	getStateFn func() string,
) *BufferedLogger {
	return &BufferedLogger{
		Logger:       InitLogger("", fields),
		buffer:       NewRingBuffer[LogEntry](bufferSize),
		entityId:     entityId,
		entityType:   entityType,
		getState:     getStateFn,
		delayTracker: NewDelayTracker(100), // Default 100 entry capacity
	}
}

func (bl *BufferedLogger) pushLog(level, message string) {
	state := ""
	if bl.getState != nil {
		state = bl.getState()
	}
	bl.buffer.Push(LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		State:     state,
	})
}

func (bl *BufferedLogger) formatMessage(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	if !strings.Contains(format, "%") {
		for _, arg := range args {
			format += fmt.Sprintf(" %v", arg)
		}
		return format
	}
	return fmt.Sprintf(format, args...)
}

func (bl *BufferedLogger) Info(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("INFO", msg)
	bl.Logger.Info(msg)
}

func (bl *BufferedLogger) Warn(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("WARN", msg)
	bl.Logger.Warn(msg)
}

func (bl *BufferedLogger) Error(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("ERROR", msg)
	bl.Logger.Error(msg)
}

func (bl *BufferedLogger) Debug(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("DEBUG", msg)
	bl.Logger.Debug(msg)
}

func (bl *BufferedLogger) Fatal(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("FATAL", msg)
	bl.Logger.Fatal(msg)
}

func (bl *BufferedLogger) Panic(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("PANIC", msg)
	bl.Logger.Panic(msg)
}

func (bl *BufferedLogger) Trace(format string, args ...any) {
	msg := bl.formatMessage(format, args...)
	bl.pushLog("TRACE", msg)
	bl.Logger.Trace(msg)
}

func (bl *BufferedLogger) GetLogs() []LogEntry {
	return bl.buffer.GetAll()
}

func (bl *BufferedLogger) GetLastLogs(n int) []LogEntry {
	return bl.buffer.GetLast(n)
}

func (bl *BufferedLogger) GetLogsByLevel(level string) []LogEntry {
	all := bl.buffer.GetAll()
	filtered := make([]LogEntry, 0)
	level = strings.ToUpper(level)
	for _, entry := range all {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (bl *BufferedLogger) GetLogCount() int {
	return bl.buffer.Count()
}

func (bl *BufferedLogger) GetLogsJSON() (string, error) {
	logs := bl.GetLogs()
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LogSend records the send timestamp for a message and logs the event
// Use this instead of Info() when sending protocol messages for delay tracking
func (bl *BufferedLogger) LogSend(protocol, msgType string) {
	bl.delayTracker.RecordSend(protocol, msgType)
	bl.Info("Send %s", msgType)
}

// LogReceive records the receive timestamp, calculates delay, and logs the event
// Use this instead of Info() when receiving protocol messages for delay tracking
func (bl *BufferedLogger) LogReceive(protocol, msgType string) {
	bl.delayTracker.RecordReceive(protocol, msgType)
	bl.Info("Receive %s", msgType)
}

// GetDelayLogs returns the last n delay entries, or all if n <= 0
func (bl *BufferedLogger) GetDelayLogs(last int) []DelayEntry {
	return bl.delayTracker.GetLogs(last)
}

// GetDelayLogsByProtocol returns delay entries filtered by protocol
func (bl *BufferedLogger) GetDelayLogsByProtocol(protocol string, last int) []DelayEntry {
	return bl.delayTracker.GetLogsByProtocol(protocol, last)
}

// GetDelayStats returns aggregated delay statistics
func (bl *BufferedLogger) GetDelayStats() DelayStats {
	return bl.delayTracker.GetStats()
}

// GetDelayStatsByProtocol returns delay statistics for a specific protocol
func (bl *BufferedLogger) GetDelayStatsByProtocol(protocol string) DelayStats {
	return bl.delayTracker.GetStatsByProtocol(protocol)
}
