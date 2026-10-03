package rpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Gopher0727/ScheduleTask/api/pb"
	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
)

type XTimerServer struct {
	pb.UnimplementedXTimerServer
	timerStore provider.TimerStore
}

func New(timerStore provider.TimerStore) *XTimerServer {
	return &XTimerServer{timerStore: timerStore}
}

func (s *XTimerServer) CreateTimer(ctx context.Context, req *pb.CreateTimerReq) (*pb.CreateTimerResp, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	paramBytes, err := json.Marshal(req.NotifyHttpParam)
	if err != nil {
		return nil, err
	}

	timer := &model.Timer{
		App:             req.App,
		Name:            req.Name,
		Status:          0,
		Cron:            req.Cron,
		NotifyHTTPParam: string(paramBytes),
	}
	if err := model.ValidateTimer(timer); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.timerStore.Save(ctx, timer); err != nil {
		return nil, err
	}

	return &pb.CreateTimerResp{TimerId: timer.TimerID}, nil
}

func (s *XTimerServer) EnableTimer(ctx context.Context, req *pb.EnableTimerReq) (*pb.EnableTimerResp, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if err := model.ValidateEnable(req.App, req.TimerId, int(req.Status)); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	timer, err := s.timerStore.GetByID(ctx, req.TimerId)
	if err != nil {
		return nil, err
	}
	if timer == nil {
		return nil, status.Error(codes.NotFound, "timer not found")
	}

	if timer.App != req.App {
		return nil, status.Error(codes.PermissionDenied, "app does not match timer")
	}
	timer.Status = int(req.Status)
	if err := s.timerStore.Update(ctx, timer); err != nil {
		return nil, err
	}

	return &pb.EnableTimerResp{}, nil
}

func (s *XTimerServer) GetTimer(ctx context.Context, req *pb.GetTimerReq) (*pb.GetTimerResp, error) {
	if req == nil || req.TimerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "positive timer_id is required")
	}
	timer, err := s.timerStore.GetByID(ctx, req.TimerId)
	if err != nil {
		return nil, err
	}
	if timer == nil {
		return nil, status.Error(codes.NotFound, "timer not found")
	}

	var notifyParam pb.NotifyHTTPParam
	if err := json.Unmarshal([]byte(timer.NotifyHTTPParam), &notifyParam); err != nil {
		return nil, err
	}

	return &pb.GetTimerResp{
		TimerId:         timer.TimerID,
		App:             timer.App,
		Name:            timer.Name,
		Status:          int32(timer.Status),
		Cron:            timer.Cron,
		NotifyHttpParam: &notifyParam,
	}, nil
}
