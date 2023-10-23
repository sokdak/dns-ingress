package cloudflare

import "net/http"

type RetryPolicy struct {
	MaxRetryCount int
	MinDelay      int
	MaxDelay      int
}

var DefaultRetryPolicy RetryPolicy = RetryPolicy{
	MaxRetryCount: 3,
	MinDelay:      1,
	MaxDelay:      30,
}

var DefaultDebugConfig bool = false

var DefaultRateLimit float64 = 4.0

var DefaultHttpClient = http.DefaultClient
