package twitter

import (
	"testing"
)

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier() error: %v", err)
	}

	// Base64-URL encoding of 32 bytes = 43 chars
	if len(verifier) != 43 {
		t.Errorf("expected verifier length 43, got %d", len(verifier))
	}

	// Two verifiers should differ
	verifier2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier() error: %v", err)
	}
	if verifier == verifier2 {
		t.Error("two consecutive verifiers should differ")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test_verifier_string_for_testing"
	challenge := generateCodeChallenge(verifier)

	if challenge == "" {
		t.Error("challenge should not be empty")
	}

	// Same input should produce same output
	challenge2 := generateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Error("same verifier should produce same challenge")
	}

	// Different input should produce different output
	challenge3 := generateCodeChallenge("different_verifier")
	if challenge == challenge3 {
		t.Error("different verifiers should produce different challenges")
	}
}

func TestGenerateState(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}

	if state == "" {
		t.Error("state should not be empty")
	}

	// Two states should differ
	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}
	if state == state2 {
		t.Error("two consecutive states should differ")
	}
}
