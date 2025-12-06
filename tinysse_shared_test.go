package tinysse

import "testing"

func TestNew(t *testing.T) {
	c := &Config{}
	sse := New(c)
	if sse == nil {
		t.Error("New() returned nil")
	}
	if sse.config != c {
		t.Error("New() did not set config")
	}
}

func TestAutoChannels(t *testing.T) {
	channels := autoChannels("user1", "admin")
	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}
	if channels[0] != "all" {
		t.Errorf("expected channel 'all', got %s", channels[0])
	}
	if channels[1] != "role:admin" {
		t.Errorf("expected channel 'role:admin', got %s", channels[1])
	}
	if channels[2] != "user:user1" {
		t.Errorf("expected channel 'user:user1', got %s", channels[2])
	}
}
