package utils

import (
	sharedUtils "github.com/EasterCompany/dex-go-utils/utils"
)

// GetMetrics returns current CPU and Memory usage metrics for the current process.
func GetMetrics() SystemMetrics {
	return sharedUtils.GetMetrics()
}
