package memogram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

type BackendLatencyStatus struct {
	Latency time.Duration
	Err     error
}

func ProbeBackendLatency(ctx context.Context, client *MemosClient) BackendLatencyStatus {
	if client == nil || client.InstanceService == nil {
		return BackendLatencyStatus{Err: fmt.Errorf("instance service is not configured")}
	}

	start := time.Now()
	_, err := client.InstanceService.GetInstanceProfile(ctx, connect.NewRequest(&v1pb.GetInstanceProfileRequest{}))
	return BackendLatencyStatus{
		Latency: time.Since(start),
		Err:     err,
	}
}

func (s BackendLatencyStatus) StatusLine() string {
	if s.Err != nil {
		return fmt.Sprintf("Backend latency: unavailable (%s)", sanitizeBackendError(s.Err))
	}
	return fmt.Sprintf("Backend latency: %s", formatLatency(s.Latency))
}

func formatLatency(latency time.Duration) string {
	switch {
	case latency < time.Millisecond:
		return fmt.Sprintf("%dµs", latency.Microseconds())
	case latency < time.Second:
		return latency.Round(time.Millisecond).String()
	default:
		return latency.Round(10 * time.Millisecond).String()
	}
}

func sanitizeBackendError(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "unknown error"
	}
	return message
}
