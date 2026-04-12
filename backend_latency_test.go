package memogram

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/proto/gen/api/v1/apiv1connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

type testInstanceService struct {
	delay time.Duration
	err   error
}

func (s *testInstanceService) GetInstanceProfile(ctx context.Context, req *connect.Request[v1pb.GetInstanceProfileRequest]) (*connect.Response[v1pb.InstanceProfile], error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&v1pb.InstanceProfile{
		InstanceUrl: "https://example.test",
	}), nil
}

func (s *testInstanceService) GetInstanceSetting(context.Context, *connect.Request[v1pb.GetInstanceSettingRequest]) (*connect.Response[v1pb.InstanceSetting], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (s *testInstanceService) UpdateInstanceSetting(context.Context, *connect.Request[v1pb.UpdateInstanceSettingRequest]) (*connect.Response[v1pb.InstanceSetting], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

type testAuthService struct{}

func (s *testAuthService) GetCurrentUser(context.Context, *connect.Request[v1pb.GetCurrentUserRequest]) (*connect.Response[v1pb.GetCurrentUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("not implemented"))
}

func (s *testAuthService) SignIn(context.Context, *connect.Request[v1pb.SignInRequest]) (*connect.Response[v1pb.SignInResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (s *testAuthService) SignOut(context.Context, *connect.Request[v1pb.SignOutRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (s *testAuthService) RefreshToken(context.Context, *connect.Request[v1pb.RefreshTokenRequest]) (*connect.Response[v1pb.RefreshTokenResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func newTestMemosClient(t *testing.T, instanceHandler apiv1connect.InstanceServiceHandler) *MemosClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewInstanceServiceHandler(instanceHandler))
	mux.Handle(apiv1connect.NewAuthServiceHandler(&testAuthService{}))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return NewMemosClient(server.URL)
}

func TestProbeBackendLatencySuccess(t *testing.T) {
	client := newTestMemosClient(t, &testInstanceService{delay: 15 * time.Millisecond})

	status := ProbeBackendLatency(context.Background(), client)
	if status.Err != nil {
		t.Fatalf("expected no error, got %v", status.Err)
	}
	if status.Latency < 15*time.Millisecond {
		t.Fatalf("expected latency >= 15ms, got %s", status.Latency)
	}

	line := status.StatusLine()
	if !strings.HasPrefix(line, "Backend latency: ") {
		t.Fatalf("unexpected status line: %q", line)
	}
	if strings.Contains(line, "unavailable") {
		t.Fatalf("expected reachable backend status line, got %q", line)
	}
}

func TestProbeBackendLatencyFailure(t *testing.T) {
	client := newTestMemosClient(t, &testInstanceService{
		err: connect.NewError(connect.CodeUnavailable, errors.New("backend offline")),
	})

	status := ProbeBackendLatency(context.Background(), client)
	if status.Err == nil {
		t.Fatal("expected an error")
	}

	line := status.StatusLine()
	if !strings.Contains(line, "unavailable") {
		t.Fatalf("expected unavailable status line, got %q", line)
	}
	if !strings.Contains(line, "backend offline") {
		t.Fatalf("expected error message in status line, got %q", line)
	}
}

func TestProbeBackendLatencyWithNilClient(t *testing.T) {
	status := ProbeBackendLatency(context.Background(), nil)
	if status.Err == nil {
		t.Fatal("expected an error for nil client")
	}
	if got := status.StatusLine(); !strings.Contains(got, "unavailable") {
		t.Fatalf("expected unavailable status, got %q", got)
	}
}
