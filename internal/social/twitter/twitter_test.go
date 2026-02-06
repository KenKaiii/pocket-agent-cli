package twitter

import (
	"strings"
	"testing"
)

func TestPercentEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello%20world"},
		{"a+b", "a%2Bb"},
		{"test@example.com", "test%40example.com"},
		{"100%", "100%25"},
		{"", ""},
		{"simple", "simple"},
		{"key=value&foo=bar", "key%3Dvalue%26foo%3Dbar"},
		{"https://example.com/path", "https%3A%2F%2Fexample.com%2Fpath"},
	}

	for _, tt := range tests {
		got := percentEncode(tt.input)
		if got != tt.want {
			t.Errorf("percentEncode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateNonce(t *testing.T) {
	c := &oauthClient{}

	nonce := c.generateNonce()

	// Nonce should be 32 hex characters (16 bytes * 2)
	if len(nonce) != 32 {
		t.Errorf("expected nonce length 32, got %d", len(nonce))
	}

	// Should be valid hex
	for _, ch := range nonce {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("nonce contains non-hex character: %c", ch)
		}
	}

	// Two nonces should be different (probabilistic)
	nonce2 := c.generateNonce()
	if nonce == nonce2 {
		t.Error("two consecutive nonces should differ")
	}
}

func TestGenerateSignature(t *testing.T) {
	c := &oauthClient{
		consumerKey:    "xvz1evFS4wEEPTGEFPHBog",
		consumerSecret: "kAcSOqF21Fu85e7zjz7ZN2U4ZRhfV3WpwPAoE3Z7kBw",
		accessToken:    "370773112-GmHxMAgYyLbNEtIKZeRNFsMKPR9EyMZeS9weJAEb",
		accessSecret:   "LswwdoUaIvS8ltyTt5jkRh4J50vUPVVHtR2YPi5kE",
	}

	params := map[string]string{
		"oauth_consumer_key":     c.consumerKey,
		"oauth_nonce":            "kYjzVBB8Y0ZFabxSWbWovY3uYSQ2pTgmZeNu2VS4cg",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1318622958",
		"oauth_token":            c.accessToken,
		"oauth_version":          "1.0",
	}

	sig := c.generateSignature("POST", "https://api.x.com/2/tweets", params)

	// Signature should be non-empty base64
	if sig == "" {
		t.Error("signature should not be empty")
	}
	// Should be valid base64 (contains only base64 chars)
	for _, ch := range sig {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=') {
			t.Errorf("signature contains invalid base64 character: %c", ch)
		}
	}
}

func TestBuildAuthHeader(t *testing.T) {
	c := &oauthClient{
		consumerKey:    "test_consumer_key",
		consumerSecret: "test_consumer_secret",
		accessToken:    "test_access_token",
		accessSecret:   "test_access_secret",
	}

	header := c.buildAuthHeader("POST", "https://api.x.com/2/tweets")

	// Should start with "OAuth "
	if !strings.HasPrefix(header, "OAuth ") {
		t.Errorf("header should start with 'OAuth ', got: %s", header[:20])
	}

	// Should contain required OAuth parameters
	required := []string{
		"oauth_consumer_key=",
		"oauth_nonce=",
		"oauth_signature_method=",
		"oauth_timestamp=",
		"oauth_token=",
		"oauth_version=",
		"oauth_signature=",
	}

	for _, param := range required {
		if !strings.Contains(header, param) {
			t.Errorf("header missing parameter: %s", param)
		}
	}
}

func TestGenerateSignatureDeterministic(t *testing.T) {
	c := &oauthClient{
		consumerKey:    "key",
		consumerSecret: "secret",
		accessToken:    "token",
		accessSecret:   "tokensecret",
	}

	params := map[string]string{
		"oauth_consumer_key":     "key",
		"oauth_nonce":            "fixednonce",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1000000",
		"oauth_token":            "token",
		"oauth_version":          "1.0",
	}

	sig1 := c.generateSignature("GET", "https://api.example.com/test", params)
	sig2 := c.generateSignature("GET", "https://api.example.com/test", params)

	if sig1 != sig2 {
		t.Error("same inputs should produce same signature")
	}

	// Different method should produce different signature
	sig3 := c.generateSignature("POST", "https://api.example.com/test", params)
	if sig1 == sig3 {
		t.Error("different methods should produce different signatures")
	}
}
