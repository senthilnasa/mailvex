package worker

import (
	"crypto/rand"
	"math/big"
	"time"
)

var defaultSchedule = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	1 * time.Hour,
	2 * time.Hour,
	6 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
}

type RetryScheduler struct {
	schedule    []time.Duration
	maxAttempts int
	maxAge      time.Duration
	jitterPct   float64
}

func NewRetryScheduler() *RetryScheduler {
	return &RetryScheduler{
		schedule:    defaultSchedule,
		maxAttempts: len(defaultSchedule),
		maxAge:      72 * time.Hour,
		jitterPct:   0.20, // ±20%
	}
}

// CalculateNextAttempt determines the duration until next attempt with ±20% jitter (§15.1)
func (s *RetryScheduler) CalculateNextAttempt(attemptNo int) (time.Duration, bool) {
	if attemptNo >= s.maxAttempts {
		return 0, false
	}

	idx := attemptNo - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.schedule) {
		idx = len(s.schedule) - 1
	}

	baseDuration := s.schedule[idx]

	// Apply jitter: ±20%
	// Generate random factor in [-0.20, +0.20]
	n, _ := rand.Int(rand.Reader, big.NewInt(400)) // 0 to 399
	factor := (float64(n.Int64()) - 200.0) / 1000.0 // -0.20 to +0.199

	jittered := time.Duration(float64(baseDuration) * (1.0 + factor))
	if jittered < 10*time.Second {
		jittered = 10 * time.Second
	}

	return jittered, true
}

func (s *RetryScheduler) MaxAttempts() int {
	return s.maxAttempts
}
