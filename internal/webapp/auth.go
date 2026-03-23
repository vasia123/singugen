package webapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// UserInfo represents a Telegram user extracted from initData.
type UserInfo struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// ValidateInitData verifies the Telegram WebApp initData signature
// using the bot token. Returns user info on success.
func ValidateInitData(initData, botToken string) (UserInfo, error) {
	if initData == "" {
		return UserInfo{}, fmt.Errorf("webapp: empty initData")
	}

	values, err := url.ParseQuery(initData)
	if err != nil {
		return UserInfo{}, fmt.Errorf("webapp: parse initData: %w", err)
	}

	hash := values.Get("hash")
	if hash == "" {
		return UserInfo{}, fmt.Errorf("webapp: missing hash in initData")
	}

	// Build data_check_string: sorted key=value pairs, excluding hash.
	values.Del("hash")
	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(lines, "\n")

	// Compute HMAC.
	secretKey := computeHMAC([]byte("WebAppData"), []byte(botToken))
	computedHash := computeHMAC(secretKey, []byte(dataCheckString))
	computedHashHex := hex.EncodeToString(computedHash)

	if computedHashHex != hash {
		return UserInfo{}, fmt.Errorf("webapp: invalid initData signature")
	}

	// Extract user info.
	userJSON := values.Get("user")
	if userJSON == "" {
		return UserInfo{}, fmt.Errorf("webapp: missing user in initData")
	}

	var user UserInfo
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return UserInfo{}, fmt.Errorf("webapp: parse user: %w", err)
	}

	return user, nil
}

func computeHMAC(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
