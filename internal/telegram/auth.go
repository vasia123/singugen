package telegram

// IsAuthorized checks if userID is in the allow list.
// An empty or nil allow list permits all users.
func IsAuthorized(userID int64, allowList map[int64]bool) bool {
	if len(allowList) == 0 {
		return true
	}
	return allowList[userID]
}
