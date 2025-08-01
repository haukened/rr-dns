package clock

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (c RealClock) Now() time.Time {
	return time.Now()
}

type MockClock struct {
	currentTime time.Time
}

func (c *MockClock) Now() time.Time {
	return c.currentTime
}

func (c *MockClock) Advance(d time.Duration) {
	c.currentTime = c.currentTime.Add(d)
}
