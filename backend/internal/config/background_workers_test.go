package config

import "testing"

func TestBackgroundWorkersDisabled(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"", false}, {"false", false}, {"0", false}, {"invalid", false},
		{"true", true}, {"TRUE", true}, {"1", true}, {" true ", true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("SERVER_DISABLE_BACKGROUND_WORKERS", tc.value)
			if got := BackgroundWorkersDisabled(); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
