package services

import (
	"fmt"
	"os"
	"strings"
)

func BuildQuickHeartbeatURL(token string, auto bool) string {
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	link := fmt.Sprintf("%s/api/quick-heartbeat/%s", baseURL, token)
	if auto {
		link += "?auto=1"
	}
	return link
}
