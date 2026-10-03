package rpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Gopher0727/ScheduleTask/api/pb"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/timer"
	"github.com/Gopher0727/ScheduleTask/internal/testutil"
)

func TestRPCValidationAndLifecycle(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Timer{}); err != nil {
		t.Fatal(err)
	}
	server := New(&timer.Store{DB: db})
	ctx := context.Background()
	if _, err := server.GetTimer(ctx, &pb.GetTimerReq{TimerId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("zero ID accepted: %v", err)
	}
	if _, err := server.CreateTimer(ctx, &pb.CreateTimerReq{App: "demo", Name: "bad", Cron: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid timer: %v", err)
	}
	created, err := server.CreateTimer(ctx, &pb.CreateTimerReq{App: "demo", Name: "example", Cron: "* * * * *", NotifyHttpParam: &pb.NotifyHTTPParam{Method: "POST", Url: "http://localhost/callback"}})
	if err != nil || created.TimerId <= 0 {
		t.Fatalf("create: %+v, %v", created, err)
	}
	if _, err := server.EnableTimer(ctx, &pb.EnableTimerReq{App: "demo", TimerId: created.TimerId, Status: 2}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid status: %v", err)
	}
	if _, err := server.EnableTimer(ctx, &pb.EnableTimerReq{App: "other", TimerId: created.TimerId, Status: 1}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("app mismatch: %v", err)
	}
	if _, err := server.EnableTimer(ctx, &pb.EnableTimerReq{App: "demo", TimerId: created.TimerId, Status: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := server.GetTimer(ctx, &pb.GetTimerReq{TimerId: created.TimerId})
	if err != nil || got.Status != 1 || got.NotifyHttpParam.Url != "http://localhost/callback" {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := server.GetTimer(ctx, &pb.GetTimerReq{TimerId: 99999}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing timer: %v", err)
	}
}
