package jobs

import (
	"testing"
	"time"
)

func TestRetryDelayIsExponentialAndBounded(t *testing.T) {
	for _, test := range []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 2 * time.Second},
		{attempt: 2, want: 4 * time.Second},
		{attempt: 3, want: 8 * time.Second},
		{attempt: 20, want: 15 * time.Minute},
	} {
		if got := retryDelay(test.attempt); got != test.want {
			t.Errorf("retryDelay(%d) = %s, want %s", test.attempt, got, test.want)
		}
	}
}
