package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookConfig is stored in tenant_settings under key 'webhook'.
// Push is opt-in: with no URL configured, SendWebhook is a no-op.
type WebhookConfig struct {
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Enabled bool     `json:"enabled"`
	Events  []string `json:"events"` // empty = all events
}

const webhookSettingKey = "webhook"

// LoadWebhookConfig reads the webhook configuration from tenant_settings.
func LoadWebhookConfig(db *gorm.DB, tenantID uuid.UUID) (*WebhookConfig, error) {
	var row struct {
		SettingValue []byte
	}
	err := db.Table("tenant_settings").
		Select("setting_value").
		Where("tenant_id = ? AND setting_key = ?", tenantID, webhookSettingKey).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if len(row.SettingValue) == 0 {
		return &WebhookConfig{}, nil
	}
	var cfg WebhookConfig
	if err := json.Unmarshal(row.SettingValue, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SendWebhook POSTs a JSON payload to the tenant's configured webhook URL,
// signed with HMAC-SHA256 in the X-Smartscan-Signature header. It runs in a
// background goroutine and is strictly best-effort: delivery failures are
// logged and never retried or surfaced to the triggering flow.
func SendWebhook(db *gorm.DB, tenantID uuid.UUID, event string, payload map[string]interface{}) {
	cfg, err := LoadWebhookConfig(db, tenantID)
	if err != nil || cfg == nil || !cfg.Enabled || cfg.URL == "" {
		return
	}
	if len(cfg.Events) > 0 {
		found := false
		for _, e := range cfg.Events {
			if e == event {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	url := cfg.URL
	secret := cfg.Secret
	go func() {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			log.Printf("[WEBHOOK] invalid request for event %s: %v", event, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Smartscan-Event", event)
		req.Header.Set("X-Smartscan-Signature", "sha256="+signature)

		// SSRF-safe client: the webhook URL is tenant-controlled, so connections to
		// internal/loopback/link-local/private addresses (incl. cloud metadata) are
		// refused at dial time, defeating DNS-rebinding and redirect-based bypasses.
		client := SafeHTTPClient(5 * time.Second)
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[WEBHOOK] delivery failed for event %s: %v", event, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			log.Printf("[WEBHOOK] event %s got HTTP %d from %s", event, resp.StatusCode, url)
		}
	}()
}
