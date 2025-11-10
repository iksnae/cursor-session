package internal

import (
	"strings"
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
		// Verify it provides useful information (encryption detection or encoded message)
		if !strings.Contains(decoded, "[Encrypted:") && !strings.Contains(decoded, "[Encoded:") {
			t.Errorf("decodeRedactedReasoning() should provide encryption/encoding info, got: %q", decoded)
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

	// Test with the encrypted string from GitHub
	encryptedStr := "t1UPI2m2UGAGw07kxd1LPNXJF7MG7mX5Oi_vk1TB1a74qPestSubNmvdWSjlGS3SgykAB3aljUEqm9Kz8fSPPadvOyP9dF5h0k7wwJpIC0r3QuTg5hJhQDXs1DxIlFYUbOu5oD5gokjqgHgf-0DY_0hli3nrFl96wmT-oZ350Se59t5X7kSgjifTT0QFDPm1RSWE5Nc-lrr1Nn1WfxTX5oeBRetQXb1VbEJR_nLabUtbptQW1b9JkImHmNno1rsy3f0u37oUcjdxYYeGdfucPG_UYbgcWlsH9q6euD1Wj0vTwe8c_U_EojyX_3bbDp7-9D9PIL5Ohtf3xFDu5yI5JDoPjcifchgAlJtFlwnmradWROTWhFZjEbOXL6k6zpi48K5AryFgZ_7bI1kd2PBH0Ri_KgekjqxkzirFkrR3wj96iaBLiIMcLxkfml9CZTNBzv8Xegu1YMKLPGD1GBY1l1_9vtDxwDs_qiI6ESd7dRb_aA"
	decoded2, wasDecoded2 := decodeRedactedReasoning(encryptedStr)
	if wasDecoded2 {
		t.Errorf("Encrypted string should not decode successfully, got: %q", decoded2)
	}
	if !strings.Contains(decoded2, "[Encrypted:") {
		t.Errorf("Encrypted string should be detected, got: %q", decoded2)
	}
	if !strings.Contains(decoded2, "entropy") {
		t.Errorf("Encrypted string message should include entropy info, got: %q", decoded2)
	}
	t.Logf("Encryption detection working: %s", decoded2)
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
