package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// QRGenerationJob represents a QR batch generation job to be processed by a worker
type QRGenerationJob struct {
	ID           string `json:"id"`             // Unique job ID (uuid v7)
	BatchID      string `json:"batch_id"`       // QR batch UUID
	TenantID     string `json:"tenant_id"`      // Tenant UUID (for per-tenant lock)
	TotalQRCount int    `json:"total_qr_count"` // Total QR codes to generate
	Prefix      string `json:"prefix"` // QR code prefix
	Suffix      string `json:"suffix"` // QR code suffix
	RetryCount  int    `json:"retry_count"`
	MaxRetries  int    `json:"max_retries"`
	CreatedAt   time.Time `json:"created_at"`
	LastError   string    `json:"last_error,omitempty"`

	// Redis stream metadata
	StreamID string `json:"stream_id,omitempty"`
}

// NewQRGenerationJob creates a new QR generation job with defaults
func NewQRGenerationJob(batchID, tenantID string, totalQRCount int, prefix, suffix string, maxRetries int) *QRGenerationJob {
	if maxRetries <= 0 {
		maxRetries = 5
	}
	return &QRGenerationJob{
		ID:           uuid.Must(uuid.NewV7()).String(),
		BatchID:      batchID,
		TenantID:     tenantID,
		TotalQRCount: totalQRCount,
		Prefix:       prefix,
		Suffix:       suffix,
		RetryCount:   0,
		MaxRetries:   maxRetries,
		CreatedAt:    time.Now().UTC(),
	}
}

// ToJSON serializes the job to JSON bytes
func (j *QRGenerationJob) ToJSON() ([]byte, error) {
	return json.Marshal(j)
}

// QRGenerationJobFromJSON deserializes JSON bytes into a QRGenerationJob
func QRGenerationJobFromJSON(data []byte) (*QRGenerationJob, error) {
	var job QRGenerationJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// ToMap converts the job to a map for Redis XADD
func (j *QRGenerationJob) ToMap() map[string]interface{} {
	data, _ := j.ToJSON()
	return map[string]interface{}{
		"data": string(data),
	}
}

// QRGenerationJobFromMap creates a QRGenerationJob from a Redis stream message map
func QRGenerationJobFromMap(m map[string]interface{}) (*QRGenerationJob, error) {
	data, ok := m["data"].(string)
	if !ok {
		return nil, ErrInvalidJobData
	}
	return QRGenerationJobFromJSON([]byte(data))
}

// CanRetry checks if the job can be retried
func (j *QRGenerationJob) CanRetry() bool {
	return j.RetryCount < j.MaxRetries
}

// IncrementRetry increments the retry count and records the error
func (j *QRGenerationJob) IncrementRetry(err error) {
	j.RetryCount++
	if err != nil {
		j.LastError = err.Error()
	}
}
