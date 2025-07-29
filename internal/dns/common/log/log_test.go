package log

import (
	"testing"
)

type testLogger struct {
	entries []string
}

func (l *testLogger) Info(_ map[string]any, msg string)  { l.entries = append(l.entries, "INFO:"+msg) }
func (l *testLogger) Error(_ map[string]any, msg string) { l.entries = append(l.entries, "ERROR:"+msg) }
func (l *testLogger) Debug(_ map[string]any, msg string) { l.entries = append(l.entries, "DEBUG:"+msg) }
func (l *testLogger) Warn(_ map[string]any, msg string)  { l.entries = append(l.entries, "WARN:"+msg) }
func (l *testLogger) Panic(_ map[string]any, msg string) {}
func (l *testLogger) Fatal(_ map[string]any, msg string) {}

func TestActualZapLogger(t *testing.T) {
	// test with fields and message
	Debug(map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}, "test debug")
	// test with just a message
	Info(nil, "test info")
	Warn(nil, "test warn")
	Error(nil, "test error")
	// recover handler for panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, but none occurred")
		}
	}()
	// test panic
	Panic(nil, "test panic") // This should panic
	// Note: Fatal will stop the test, so we don't call it here.
}

func TestSetLoggerAndGlobalLogging(t *testing.T) {
	// set up test fixtures
	orig := GetLogger()
	defer func() {
		SetLogger(orig) // Restore original logger after test
	}()
	tlog := &testLogger{}
	SetLogger(tlog)

	// Test code

	Info(nil, "info msg")
	Error(nil, "error msg")
	Debug(nil, "debug msg")
	Warn(nil, "warn msg")

	expected := []string{
		"INFO:info msg",
		"ERROR:error msg",
		"DEBUG:debug msg",
		"WARN:warn msg",
	}

	if len(tlog.entries) != len(expected) {
		t.Fatalf("expected %d log entries, got %d", len(expected), len(tlog.entries))
	}
	for i, msg := range expected {
		if tlog.entries[i] != msg {
			t.Errorf("expected log[%d] = %q, got %q", i, msg, tlog.entries[i])
		}
	}
}

func TestConfigure_ValidLevels(t *testing.T) {
	// set up test fixtures
	orig := GetLogger()
	defer func() {
		SetLogger(orig) // Restore original logger after test
	}()
	tlog := &testLogger{}
	SetLogger(tlog)

	// Test code
	err := Configure("dev", "debug")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = Configure("prod", "info")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigure_InvalidLevel(t *testing.T) {
	// set up test fixtures
	orig := GetLogger()
	defer func() {
		SetLogger(orig) // Restore original logger after test
	}()
	tlog := &testLogger{}
	SetLogger(tlog)

	// Test code
	err := Configure("dev", "notalevel")
	if err == nil {
		t.Fatal("expected error for invalid log level, got nil")
	}
}

func TestNoopLogger_TestAllLevels(t *testing.T) {
	// set up test fixtures
	orig := GetLogger()
	defer func() {
		SetLogger(orig) // Restore original logger after test
	}()
	tlog := &noopLogger{}
	SetLogger(tlog)

	// Test code
	Debug(nil, "debug message")
	Info(nil, "info message")
	Warn(nil, "warn message")
	Error(nil, "error message")
	Panic(nil, "panic message")
	Fatal(nil, "fatal message")
}
