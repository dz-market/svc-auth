package outbox

import (
	"fmt"
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		give int
		want time.Duration
	}{
		{give: 1, want: time.Second},
		{give: 2, want: 2 * time.Second},
		{give: 3, want: 4 * time.Second},
		{give: 10, want: maxRetryDelay},
	}

	for _, tt := range tests {
		t.Run(
			fmt.Sprintf("attempt %d", tt.give), func(t *testing.T) {
				t.Parallel()

				if got := retryDelay(tt.give); got != tt.want {
					t.Errorf("retryDelay(%d) = %s, want %s", tt.give, got, tt.want)
				}
			},
		)
	}
}
