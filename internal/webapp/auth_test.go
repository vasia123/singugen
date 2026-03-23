package webapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// buildTestInitData creates a valid initData string signed with the given bot token.
func buildTestInitData(botToken string, params map[string]string) string {
	// Build data_check_string.
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, params[k]))
	}
	dataCheckString := strings.Join(lines, "\n")

	// Compute hash.
	secretKey := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	hash := hmacSHA256(secretKey, []byte(dataCheckString))
	hashHex := hex.EncodeToString(hash)

	// Build query string.
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("hash", hashHex)

	return values.Encode()
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func TestValidateInitData_Valid(t *testing.T) {
	token := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	params := map[string]string{
		"user":       `{"id":12345,"first_name":"Test","username":"testuser"}`,
		"auth_date":  "1700000000",
		"query_id":   "AAHdF6IQAAAAAN0XohDDDDD",
	}
	initData := buildTestInitData(token, params)

	user, err := ValidateInitData(initData, token)
	if err != nil {
		t.Fatalf("ValidateInitData() error: %v", err)
	}
	if user.ID != 12345 {
		t.Errorf("user.ID = %d, want 12345", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("user.Username = %q", user.Username)
	}
}

func TestValidateInitData_TamperedData(t *testing.T) {
	token := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	params := map[string]string{
		"user":      `{"id":12345,"first_name":"Test","username":"testuser"}`,
		"auth_date": "1700000000",
	}
	initData := buildTestInitData(token, params)

	// Tamper with the data.
	initData = strings.Replace(initData, "12345", "99999", 1)

	_, err := ValidateInitData(initData, token)
	if err == nil {
		t.Error("tampered data should fail validation")
	}
}

func TestValidateInitData_MissingHash(t *testing.T) {
	_, err := ValidateInitData("user=test&auth_date=123", "token")
	if err == nil {
		t.Error("missing hash should fail")
	}
}

func TestValidateInitData_Empty(t *testing.T) {
	_, err := ValidateInitData("", "token")
	if err == nil {
		t.Error("empty initData should fail")
	}
}
