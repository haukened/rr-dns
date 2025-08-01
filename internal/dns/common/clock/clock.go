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
	CurrentTime time.Time
}

func (c *MockClock) Now() time.Time {
	return c.CurrentTime
}

func (c *MockClock) Advance(d time.Duration) {
	c.CurrentTime = c.CurrentTime.Add(d)
}
