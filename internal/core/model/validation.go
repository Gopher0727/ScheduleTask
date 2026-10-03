package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/robfig/cron/v3"
)

// ValidateTimer checks the shared HTTP/gRPC create contract.
func ValidateTimer(timer *Timer) error {
	if timer == nil || strings.TrimSpace(timer.App) == "" || strings.TrimSpace(timer.Name) == "" {
		return fmt.Errorf("app and name are required")
	}
	if _, err := cron.ParseStandard(timer.Cron); err != nil {
		return fmt.Errorf("invalid cron: %w", err)
	}
	var param *struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(timer.NotifyHTTPParam), &param); err != nil || param == nil {
		return fmt.Errorf("notify_http_param is required")
	}
	if !strings.EqualFold(param.Method, "POST") {
		return fmt.Errorf("callback method must be POST")
	}
	u, err := url.Parse(param.URL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("callback URL must be an absolute HTTP or HTTPS URL")
	}
	return nil
}

func ValidateEnable(app string, timerID int64, status int) error {
	if strings.TrimSpace(app) == "" || timerID <= 0 || (status != 0 && status != 1) {
		return fmt.Errorf("app and positive timer_id are required; status must be 0 or 1")
	}
	return nil
}
