package global

import "time"

const (
	// LinkWaitTimeout is the maximum time to wait for a QR code scan during device linking.
	LinkWaitTimeout = 10 * time.Minute
)
