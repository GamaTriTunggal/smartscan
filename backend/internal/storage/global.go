package storage

import "sync"

var (
	globalR2Client *R2Client
	r2Mutex        sync.RWMutex
)

// SetGlobalR2Client sets the global R2 client instance
func SetGlobalR2Client(client *R2Client) {
	r2Mutex.Lock()
	defer r2Mutex.Unlock()
	globalR2Client = client
}

// GetGlobalR2Client returns the global R2 client instance
// Returns nil if R2 is not configured
func GetGlobalR2Client() *R2Client {
	r2Mutex.RLock()
	defer r2Mutex.RUnlock()
	return globalR2Client
}

// IsR2Enabled returns true if R2 storage is configured and available
func IsR2Enabled() bool {
	return GetGlobalR2Client() != nil
}
