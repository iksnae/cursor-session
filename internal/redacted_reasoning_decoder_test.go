package internal

import (
	"testing"
)

func TestDecodeRedactedReasoning(t *testing.T) {
	// Test with a real redacted reasoning string from the export
	testStr := "Ul-oirObhdvNGUMxfM-7R-9SDspzUJNdwk-dQPStKNHHKTDPmKTm1PityyKuPUiL4U4TwH8o31bBu5fSljvD8IxA5swAfL_ZbNsHYjdr3C4aK35VQNGtZnLlPX8GEUIo2Lrm-Uf-ftRcrr6PLh1LK9J0jwZGnirLWtbxYI0KstKT-AV8bryUggOiBwL3T1zzfoAFqxz0mdneXCNa3AtXCarrHsB4CRUn7KSiv4GK4cQWJykQrpgC_pO5gS2288597GesReLghCwMQJs0_OC9Kf9I"

	decoded, wasDecoded := decodeRedactedReasoning(testStr)

	if !wasDecoded {
		t.Logf("Decoding failed, result: %s", decoded)
		// This is OK - the string might be encrypted or in a format we can't decode
		// But we should at least verify it's not the original string
		if decoded == testStr {
			t.Error("decodeRedactedReasoning() returned original string unchanged")
		}
	} else {
		// If decoding succeeded, verify it's readable text
		if !isReadableText(decoded) {
			t.Errorf("decodeRedactedReasoning() returned non-readable text: %q", decoded)
		}
		if len(decoded) < 10 {
			t.Errorf("decodeRedactedReasoning() returned too short text: %q", decoded)
		}
		t.Logf("Successfully decoded: %s", decoded[:min(200, len(decoded))])
	}
}

func TestDecodeBase64URL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid base64url", "SGVsbG8gV29ybGQ", false},
		{"valid base64url with padding", "SGVsbG8=", false},
		{"invalid", "not-base64!!!", true}, // Invalid base64 characters
		{"empty", "", false},               // Empty string decodes to empty bytes, not an error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := decodeBase64URL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBase64URL() error = %v, wantErr %v", err, tt.wantErr)
			}
			// For valid inputs, verify we got some data (unless it's empty input)
			if !tt.wantErr && tt.input != "" && len(decoded) == 0 {
				t.Errorf("decodeBase64URL() returned empty for non-empty input")
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
