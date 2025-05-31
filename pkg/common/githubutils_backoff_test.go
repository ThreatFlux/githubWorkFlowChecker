package common

import (
	"math"
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	tests := []struct {
		name            string
		attempt         int
		baseDelay       time.Duration
		maxDelay        time.Duration
		wantMinDuration time.Duration
		wantMaxDuration time.Duration
	}{
		{
			name:            "attempt 0 returns base delay",
			attempt:         0,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: baseDelay,
			wantMaxDuration: baseDelay,
		},
		{
			name:            "negative attempt returns base delay",
			attempt:         -1,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: baseDelay,
			wantMaxDuration: baseDelay,
		},
		{
			name:            "attempt 1 doubles delay with jitter",
			attempt:         1,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: time.Duration(float64(2*baseDelay) * 0.75), // 2s * 0.75 = 1.5s
			wantMaxDuration: time.Duration(float64(2*baseDelay) * 1.25), // 2s * 1.25 = 2.5s
		},
		{
			name:            "attempt 2 quadruples delay with jitter",
			attempt:         2,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: time.Duration(float64(4*baseDelay) * 0.75), // 4s * 0.75 = 3s
			wantMaxDuration: time.Duration(float64(4*baseDelay) * 1.25), // 4s * 1.25 = 5s
		},
		{
			name:            "attempt 3 is 8x delay with jitter",
			attempt:         3,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: time.Duration(float64(8*baseDelay) * 0.75), // 8s * 0.75 = 6s
			wantMaxDuration: time.Duration(float64(8*baseDelay) * 1.25), // 8s * 1.25 = 10s
		},
		{
			name:            "high attempt hits max delay cap",
			attempt:         10,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: time.Duration(float64(maxDelay) * 0.75), // 60s * 0.75 = 45s
			wantMaxDuration: time.Duration(float64(maxDelay) * 1.25), // 60s * 1.25 = 75s (capped at maxDelay + jitter)
		},
		{
			name:            "very high attempt still caps at max",
			attempt:         50,
			baseDelay:       baseDelay,
			maxDelay:        maxDelay,
			wantMinDuration: time.Duration(float64(maxDelay) * 0.75),
			wantMaxDuration: time.Duration(float64(maxDelay) * 1.25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the test multiple times to account for jitter randomness
			for i := 0; i < 10; i++ {
				got := CalculateBackoff(tt.attempt, tt.baseDelay, tt.maxDelay)

				// For attempt 0 or negative, should always return exact base delay
				if tt.attempt <= 0 {
					if got != tt.wantMinDuration {
						t.Errorf("CalculateBackoff() = %v, want %v for attempt %d", got, tt.wantMinDuration, tt.attempt)
					}
					continue
				}

				// For other attempts, check jitter range
				if got < tt.wantMinDuration || got > tt.wantMaxDuration {
					t.Errorf("CalculateBackoff() = %v, want between %v and %v for attempt %d",
						got, tt.wantMinDuration, tt.wantMaxDuration, tt.attempt)
				}
			}
		})
	}
}

func TestCalculateBackoffExponentialProgression(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 120 * time.Second

	// Test that the progression is roughly exponential (ignoring jitter)
	var prevMidpoint time.Duration

	for attempt := 1; attempt <= 6; attempt++ {
		// Calculate multiple samples to get an average (reducing jitter effect)
		var totalDuration time.Duration
		samples := 100

		for i := 0; i < samples; i++ {
			totalDuration += CalculateBackoff(attempt, baseDelay, maxDelay)
		}
		avgDuration := totalDuration / time.Duration(samples)

		if attempt > 1 {
			// The average should be roughly double the previous (within 30% due to jitter)
			expectedRatio := 2.0
			actualRatio := float64(avgDuration) / float64(prevMidpoint)

			if actualRatio < expectedRatio*0.7 || actualRatio > expectedRatio*1.3 {
				t.Errorf("Exponential progression broken at attempt %d: ratio %f, expected ~%f",
					attempt, actualRatio, expectedRatio)
			}
		}

		prevMidpoint = avgDuration
	}
}

func TestCalculateBackoffMaxDelayEnforcement(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 5 * time.Second

	// High attempt should never exceed maxDelay + jitter
	maxAllowedDelay := time.Duration(float64(maxDelay) * 1.25) // Max jitter is 25%

	for attempt := 10; attempt <= 20; attempt++ {
		for i := 0; i < 50; i++ {
			got := CalculateBackoff(attempt, baseDelay, maxDelay)
			if got > maxAllowedDelay {
				t.Errorf("CalculateBackoff() = %v, exceeds max allowed %v for attempt %d",
					got, maxAllowedDelay, attempt)
			}
		}
	}
}

func TestCalculateBackoffJitterDistribution(t *testing.T) {
	baseDelay := 4 * time.Second // Use 4s so we can see jitter clearly
	maxDelay := 60 * time.Second
	attempt := 2 // Should give us 16s base with ±4s jitter (12s-20s range)

	samples := 1000
	var totalDuration time.Duration
	minSeen := time.Duration(math.MaxInt64)
	maxSeen := time.Duration(0)

	for i := 0; i < samples; i++ {
		duration := CalculateBackoff(attempt, baseDelay, maxDelay)
		totalDuration += duration
		if duration < minSeen {
			minSeen = duration
		}
		if duration > maxSeen {
			maxSeen = duration
		}
	}

	avgDuration := totalDuration / time.Duration(samples)
	expectedAvg := 16 * time.Second // 4s * 2^2 = 16s

	// Average should be close to expected (within 10%)
	if avgDuration < time.Duration(float64(expectedAvg)*0.9) ||
		avgDuration > time.Duration(float64(expectedAvg)*1.1) {
		t.Errorf("Average duration %v not close to expected %v", avgDuration, expectedAvg)
	}

	// Should see both sides of jitter range
	expectedMin := time.Duration(float64(expectedAvg) * 0.75) // 12s
	expectedMax := time.Duration(float64(expectedAvg) * 1.25) // 20s

	if minSeen > time.Duration(float64(expectedMin)*1.1) {
		t.Errorf("Minimum seen %v too high, expected around %v", minSeen, expectedMin)
	}
	if maxSeen < time.Duration(float64(expectedMax)*0.9) {
		t.Errorf("Maximum seen %v too low, expected around %v", maxSeen, expectedMax)
	}
}

func TestCalculateBackoffOverflowProtection(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	// Test with extremely high attempt values that could cause overflow
	for _, attempt := range []int{50, 100, 1000} {
		got := CalculateBackoff(attempt, baseDelay, maxDelay)
		maxAllowed := time.Duration(float64(maxDelay) * 1.25)

		if got > maxAllowed || got < 0 {
			t.Errorf("CalculateBackoff() = %v, should be positive and <= %v for attempt %d",
				got, maxAllowed, attempt)
		}
	}
}

func TestCalculateBackoffEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "zero base delay",
			attempt:   3,
			baseDelay: 0,
			maxDelay:  60 * time.Second,
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "zero max delay",
			attempt:   3,
			baseDelay: 1 * time.Second,
			maxDelay:  0,
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "max delay smaller than base",
			attempt:   1,
			baseDelay: 10 * time.Second,
			maxDelay:  5 * time.Second,
			wantMin:   time.Duration(float64(5*time.Second) * 0.75),
			wantMax:   time.Duration(float64(5*time.Second) * 1.25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 10; i++ {
				got := CalculateBackoff(tt.attempt, tt.baseDelay, tt.maxDelay)
				if got < tt.wantMin || got > tt.wantMax {
					t.Errorf("CalculateBackoff() = %v, want between %v and %v",
						got, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}
