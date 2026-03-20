package common

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

func GenerateNonce(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func BuildDiscourseConnectURL(baseURL, secret, returnURL string, params map[string]string) (string, string, error) {
	nonce, err := GenerateNonce(16)
	if err != nil {
		return "", "", err
	}

	values := url.Values{}
	values.Set("nonce", nonce)
	values.Set("return_sso_url", returnURL)
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			values.Set(key, value)
		}
	}

	payload := base64.StdEncoding.EncodeToString([]byte(values.Encode()))
	signature := signDiscourseConnectPayload(secret, payload)
	loginURL := strings.TrimRight(baseURL, "/") + "/session/sso_provider?sso=" + url.QueryEscape(payload) + "&sig=" + signature

	return nonce, loginURL, nil
}

func ParseDiscourseConnectPayload(secret, payload, sig string) (url.Values, error) {
	expectedSig := signDiscourseConnectPayload(secret, payload)
	if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
		return nil, fmt.Errorf("invalid discourse connect signature")
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode discourse connect payload: %w", err)
	}

	values, err := url.ParseQuery(string(decoded))
	if err != nil {
		return nil, fmt.Errorf("failed to parse discourse connect payload: %w", err)
	}

	return values, nil
}

func signDiscourseConnectPayload(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
