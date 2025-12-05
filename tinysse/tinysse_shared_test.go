package tinysse

import (
	"testing"
)

func TestAutoChannels(t *testing.T) {
	channels := autoChannels("user123", "admin")
	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}

	expectedChannels := map[string]bool{
		"all":          true,
		"role:admin":   true,
		"user:user123": true,
	}

	for _, ch := range channels {
		if !expectedChannels[ch] {
			t.Errorf("unexpected channel: %s", ch)
		}
	}
}
