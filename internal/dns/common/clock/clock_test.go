package clock

import (
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	clock := RealClock{}

	// Capture time before and after the clock call
	before := time.Now()
	now := clock.Now()
	after := time.Now()

	// The clock's time should be between our before/after measurements
	if now.Before(before) {
		t.Errorf("Clock time %v is before measurement time %v", now, before)
	}
	if now.After(after) {
		t.Errorf("Clock time %v is after measurement time %v", now, after)
	}
}

func TestRealClock_Now_Multiple_Calls(t *testing.T) {
	clock := RealClock{}

	first := clock.Now()
	time.Sleep(1 * time.Millisecond) // Small delay to ensure time difference
	second := clock.Now()

	if !second.After(first) {
		t.Errorf("Second call %v should be after first call %v", second, first)
	}
}

func TestMockClock_Now(t *testing.T) {
	fixedTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: fixedTime}

	now := clock.Now()

	if !now.Equal(fixedTime) {
		t.Errorf("Expected %v, got %v", fixedTime, now)
	}
}

func TestMockClock_Now_Consistent(t *testing.T) {
	fixedTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: fixedTime}

	first := clock.Now()
	second := clock.Now()

	if !first.Equal(second) {
		t.Errorf("Mock clock should return consistent time: first=%v, second=%v", first, second)
	}
}

func TestMockClock_Advance(t *testing.T) {
	initialTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: initialTime}

	// Test advancing by various durations
	testCases := []struct {
		name     string
		duration time.Duration
		expected time.Time
	}{
		{
			name:     "advance by 1 hour",
			duration: 1 * time.Hour,
			expected: initialTime.Add(1 * time.Hour),
		},
		{
			name:     "advance by 30 minutes more",
			duration: 30 * time.Minute,
			expected: initialTime.Add(1*time.Hour + 30*time.Minute),
		},
		{
			name:     "advance by 1 microsecond",
			duration: 1 * time.Microsecond,
			expected: initialTime.Add(1*time.Hour + 30*time.Minute + 1*time.Microsecond),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock.Advance(tc.duration)
			now := clock.Now()

			if !now.Equal(tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, now)
			}
		})
	}
}

func TestMockClock_Advance_Negative_Duration(t *testing.T) {
	initialTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: initialTime}

	// Advance backwards
	clock.Advance(-1 * time.Hour)
	now := clock.Now()
	expected := initialTime.Add(-1 * time.Hour)

	if !now.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, now)
	}
}

func TestMockClock_Advance_Zero_Duration(t *testing.T) {
	initialTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: initialTime}

	// Advance by zero
	clock.Advance(0)
	now := clock.Now()

	if !now.Equal(initialTime) {
		t.Errorf("Expected %v, got %v", initialTime, now)
	}
}

func TestClock_Interface_Compliance(t *testing.T) {
	// Test that both implementations satisfy the Clock interface
	var _ Clock = RealClock{}
	var _ Clock = &MockClock{}
}

func TestMockClock_Simulation(t *testing.T) {
	// Simulate a realistic scenario where we need to test time-dependent behavior
	startTime := time.Date(2025, 8, 1, 9, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: startTime}

	// Simulate a day's worth of operations
	events := []struct {
		description  string
		advance      time.Duration
		expectedHour int
	}{
		{"Start of day", 0, 9},
		{"Mid-morning", 2 * time.Hour, 11},
		{"Lunch time", 2 * time.Hour, 13},
		{"Afternoon", 3 * time.Hour, 16},
		{"End of day", 2 * time.Hour, 18},
	}

	for _, event := range events {
		t.Run(event.description, func(t *testing.T) {
			if event.advance > 0 {
				clock.Advance(event.advance)
			}

			now := clock.Now()
			if now.Hour() != event.expectedHour {
				t.Errorf("Expected hour %d, got %d (time: %v)", event.expectedHour, now.Hour(), now)
			}
		})
	}
}

func TestMockClock_TTL_Simulation(t *testing.T) {
	// Simulate DNS record TTL expiration testing
	startTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: startTime}

	// Simulate a DNS record with 300 second TTL
	recordCreated := clock.Now()
	ttl := 300 * time.Second
	expiresAt := recordCreated.Add(ttl)

	// Test at various points in time
	testPoints := []struct {
		name    string
		advance time.Duration
		expired bool
	}{
		{"immediately", 0, false},
		{"halfway through TTL", 150 * time.Second, false},
		{"just before expiry", 299 * time.Second, false},
		{"at expiry", 300 * time.Second, true},
		{"after expiry", 301 * time.Second, true},
		{"long after expiry", 600 * time.Second, true},
	}

	for _, tp := range testPoints {
		t.Run(tp.name, func(t *testing.T) {
			// Reset clock and advance to test point
			clock.CurrentTime = startTime
			clock.Advance(tp.advance)

			now := clock.Now()
			isExpired := now.After(expiresAt) || now.Equal(expiresAt)

			if isExpired != tp.expired {
				t.Errorf("At %v (advanced %v), expected expired=%v, got expired=%v",
					now, tp.advance, tp.expired, isExpired)
			}
		})
	}
}

func TestMockClock_Concurrent_Access(t *testing.T) {
	// Test that MockClock can be safely used concurrently for reads
	// Note: This doesn't test concurrent writes (Advance) as that would require synchronization
	initialTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	clock := &MockClock{CurrentTime: initialTime}

	done := make(chan bool, 10)

	// Start 10 goroutines that read the time
	for i := 0; i < 10; i++ {
		go func() {
			now := clock.Now()
			if !now.Equal(initialTime) {
				t.Errorf("Expected %v, got %v", initialTime, now)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
