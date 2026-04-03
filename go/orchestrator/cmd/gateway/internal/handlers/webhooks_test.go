package handlers

import (
	"testing"
)

// Test secret validation
func TestValidateSecret_MinLength(t *testing.T) {
	tests := []struct {
		name    string
		secret  *string
		wantErr bool
	}{
		{"nil secret - skip update", nil, false},
		{"empty string - clear secret", strPtr(""), false}, // Allow clearing
		{"too short - 10 chars", strPtr("abcdefghij"), true},
		{"exactly 32 chars", strPtr("abcdefghijklmnopqrstuvwxyz123456"), false},
		{"longer than 32", strPtr("abcdefghijklmnopqrstuvwxyz1234567890"), false},
		{"exactly 255 chars", strPtr(string(make([]byte, 255))), false},
		{"too long - 256 chars", strPtr(string(make([]byte, 256))), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test metadata validation
func TestValidateMetadata_SizeAndStructure(t *testing.T) {
	tests := []struct {
		name    string
		meta    []byte
		wantErr bool
	}{
		{"nil metadata", nil, false},
		{"empty metadata", []byte{}, false},
		{"valid object", []byte(`{"key": "value"}`), false},
		{"empty object", []byte(`{}`), false},
		{"array - invalid", []byte(`[1,2,3]`), true},
		{"string - invalid", []byte(`"hello"`), true},
		{"number - invalid", []byte(`42`), true},
		{"too large - over 16KB", makeLargeMetadata(17000), true},
		{"at limit - 16KB", makeLargeMetadata(16000), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetadata(tt.meta)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func makeLargeMetadata(size int) []byte {
	// Build a JSON object that's approximately `size` bytes
	// e.g. {"padding": "aaaa...aaa"}
	padding := make([]byte, size-15) // account for {"padding":""}
	for i := range padding {
		padding[i] = 'a'
	}
	return append(append([]byte(`{"padding":"`), padding...), []byte(`"}`)...)
}
