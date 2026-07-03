package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQRBatchStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   QRBatchStatus
		terminal bool
	}{
		{QRBatchStatusPendingQueue, false},
		{QRBatchStatusQueued, false},
		{QRBatchStatusProcessing, false},
		{QRBatchStatusCompleted, true},
		{QRBatchStatusFailed, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.terminal, tc.status.IsTerminal())
		})
	}
}

func TestQRBatchStatus_IsInProgress(t *testing.T) {
	tests := []struct {
		status     QRBatchStatus
		inProgress bool
	}{
		{QRBatchStatusPendingQueue, true},
		{QRBatchStatusQueued, true},
		{QRBatchStatusProcessing, true},
		{QRBatchStatusCompleted, false},
		{QRBatchStatusFailed, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.inProgress, tc.status.IsInProgress())
		})
	}
}

func TestQRBatchStatus_MutuallyExclusive(t *testing.T) {
	// A status cannot be both terminal and in-progress
	statuses := []QRBatchStatus{
		QRBatchStatusPendingQueue,
		QRBatchStatusQueued,
		QRBatchStatusProcessing,
		QRBatchStatusCompleted,
		QRBatchStatusFailed,
	}

	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			assert.False(t, s.IsTerminal() && s.IsInProgress(),
				"status %s cannot be both terminal and in-progress", s)
		})
	}
}

func TestQRBatchStatus_StringValues(t *testing.T) {
	// Verify enum string values match what's used in the DB / JSON
	assert.Equal(t, "pending_queue", string(QRBatchStatusPendingQueue))
	assert.Equal(t, "queued", string(QRBatchStatusQueued))
	assert.Equal(t, "processing", string(QRBatchStatusProcessing))
	assert.Equal(t, "completed", string(QRBatchStatusCompleted))
	assert.Equal(t, "failed", string(QRBatchStatusFailed))
}
