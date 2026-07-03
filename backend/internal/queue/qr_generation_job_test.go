package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQRGenerationJob_DefaultsAndFields(t *testing.T) {
	job := NewQRGenerationJob("batch-id-1", "tenant-id-1", 1000, "PRE-", "-POST", 5)

	assert.NotEmpty(t, job.ID, "job ID should be generated")
	assert.Equal(t, "batch-id-1", job.BatchID)
	assert.Equal(t, "tenant-id-1", job.TenantID)
	assert.Equal(t, 1000, job.TotalQRCount)
	assert.Equal(t, "PRE-", job.Prefix)
	assert.Equal(t, "-POST", job.Suffix)
	assert.Equal(t, 0, job.RetryCount)
	assert.Equal(t, 5, job.MaxRetries)
	assert.False(t, job.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.WithinDuration(t, time.Now().UTC(), job.CreatedAt, 2*time.Second)
}

func TestNewQRGenerationJob_DefaultMaxRetries(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		expected   int
	}{
		{"zero becomes default 5", 0, 5},
		{"negative becomes default 5", -1, 5},
		{"positive value respected", 3, 3},
		{"large value respected", 100, 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job := NewQRGenerationJob("b", "t", 10, "", "", tc.maxRetries)
			assert.Equal(t, tc.expected, job.MaxRetries)
		})
	}
}

func TestQRGenerationJob_SerializeRoundtrip(t *testing.T) {
	original := &QRGenerationJob{
		ID:           "job-123",
		BatchID:      "batch-456",
		TenantID:     "tenant-789",
		TotalQRCount: 500000,
		Prefix:       "PROD-",
		Suffix:       "-2026",
		RetryCount:   2,
		MaxRetries:   5,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
		LastError:    "previous attempt failed: timeout",
	}

	// Serialize to JSON
	data, err := original.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	decoded, err := QRGenerationJobFromJSON(data)
	require.NoError(t, err)

	// All fields should match
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.BatchID, decoded.BatchID)
	assert.Equal(t, original.TenantID, decoded.TenantID)
	assert.Equal(t, original.TotalQRCount, decoded.TotalQRCount)
	assert.Equal(t, original.Prefix, decoded.Prefix)
	assert.Equal(t, original.Suffix, decoded.Suffix)
	assert.Equal(t, original.RetryCount, decoded.RetryCount)
	assert.Equal(t, original.MaxRetries, decoded.MaxRetries)
	assert.Equal(t, original.LastError, decoded.LastError)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
}

func TestQRGenerationJob_MapRoundtrip(t *testing.T) {
	original := NewQRGenerationJob("b1", "t1", 100, "", "", 5)

	// Convert to map (for Redis XADD)
	m := original.ToMap()
	assert.NotNil(t, m)
	assert.Contains(t, m, "data", "map should have a 'data' key")

	// Convert back from map
	decoded, err := QRGenerationJobFromMap(m)
	require.NoError(t, err)
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.BatchID, decoded.BatchID)
}

func TestQRGenerationJob_FromMap_InvalidData(t *testing.T) {
	// Missing data key
	_, err := QRGenerationJobFromMap(map[string]interface{}{"foo": "bar"})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidJobData, err)

	// Invalid JSON
	_, err = QRGenerationJobFromMap(map[string]interface{}{"data": "not-json-{{{"})
	assert.Error(t, err)
}

func TestQRGenerationJob_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		maxRetries int
		canRetry   bool
	}{
		{"fresh job", 0, 5, true},
		{"mid-retries", 2, 5, true},
		{"last retry allowed", 4, 5, true},
		{"at max", 5, 5, false},
		{"exceeded max", 6, 5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job := &QRGenerationJob{RetryCount: tc.retryCount, MaxRetries: tc.maxRetries}
			assert.Equal(t, tc.canRetry, job.CanRetry())
		})
	}
}

func TestQRGenerationJob_IncrementRetry(t *testing.T) {
	job := NewQRGenerationJob("b", "t", 10, "", "", 5)
	assert.Equal(t, 0, job.RetryCount)
	assert.Empty(t, job.LastError)

	testErr := assertError{"connection refused"}
	job.IncrementRetry(testErr)

	assert.Equal(t, 1, job.RetryCount)
	assert.Equal(t, "connection refused", job.LastError)

	job.IncrementRetry(nil)
	assert.Equal(t, 2, job.RetryCount)
	// LastError should remain unchanged when err is nil
	assert.Equal(t, "connection refused", job.LastError)
}

// assertError is a minimal error type for testing
type assertError struct{ msg string }

func (e assertError) Error() string { return e.msg }
