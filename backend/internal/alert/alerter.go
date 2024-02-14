package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Alerter struct {
	webhookURL string
	client     *http.Client
	log        *zap.Logger
}

func NewAlerter(webhookURL string, log *zap.Logger) *Alerter {
	return &Alerter{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
		log:        log,
	}
}

type payload struct {
	Text string `json:"text"`
}

// Send posts a Slack-compatible webhook. If no URL is configured it's a no-op.
func (a *Alerter) Send(ctx context.Context, ev interface{}) {
	if a.webhookURL == "" {
		return
	}

	msg, err := json.Marshal(ev)
	if err != nil {
		a.log.Error("alert marshal failed", zap.Error(err))
		return
	}

	body, _ := json.Marshal(payload{Text: fmt.Sprintf("```%s```", string(msg))})

	// retry up to 3 times before giving up
	for attempt := 1; attempt <= 3; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			a.log.Warn("webhook attempt failed", zap.Int("attempt", attempt), zap.Error(err))
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
		a.log.Warn("webhook non-2xx", zap.Int("status", resp.StatusCode), zap.Int("attempt", attempt))
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	a.log.Error("webhook failed after 3 attempts", zap.String("url", a.webhookURL))
}
