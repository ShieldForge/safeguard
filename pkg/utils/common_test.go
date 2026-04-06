package utils

import "testing"

func TestParseIntDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{"empty string", "", 42, 42},
		{"valid int", "100", 0, 100},
		{"negative int", "-5", 0, -5},
		{"zero", "0", 99, 0},
		{"invalid string", "abc", 10, 10},
		{"float string", "3.14", 10, 10},
		{"whitespace", " 50 ", 0, 50},
		{"leading whitespace", "  7", 0, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIntDefault(tt.value, tt.fallback)
			if got != tt.want {
				t.Errorf("ParseIntDefault(%q, %d) = %d, want %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestParseBoolDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback bool
		want     bool
	}{
		{"empty string", "", false, false},
		{"empty with true fallback", "", true, true},
		{"true", "true", false, true},
		{"True", "True", false, true},
		{"TRUE", "TRUE", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"y", "y", false, true},
		{"Y", "Y", false, true},
		{"on", "on", false, true},
		{"false", "false", true, false},
		{"False", "False", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"n", "n", true, false},
		{"off", "off", true, false},
		{"invalid", "maybe", false, false},
		{"invalid with true fallback", "maybe", true, true},
		{"whitespace", " true ", false, true},
		{"whitespace false", " false ", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBoolDefault(tt.value, tt.fallback)
			if got != tt.want {
				t.Errorf("ParseBoolDefault(%q, %v) = %v, want %v", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}
