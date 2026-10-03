package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/provider"
)

type HTTPHandler struct {
	timerStore provider.TimerStore
	taskStore  provider.TaskStore
	ready      func(context.Context) error
}

type CreateTimerReq struct {
	App             string           `json:"app"`
	Name            string           `json:"name"`
	Cron            string           `json:"cron"`
	NotifyHTTPParam *NotifyHTTPParam `json:"notify_http_param"`
}

type NotifyHTTPParam struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type EnableTimerReq struct {
	App     string `json:"app"`
	TimerID int64  `json:"timer_id"`
	Status  int    `json:"status"`
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func New(timerStore provider.TimerStore, taskStore provider.TaskStore) *HTTPHandler {
	return &HTTPHandler{timerStore: timerStore, taskStore: taskStore}
}

func (h *HTTPHandler) SetReadinessCheck(check func(context.Context) error) { h.ready = check }

func (h *HTTPHandler) RegisterRoutes(e *echo.Echo) {
	e.Use(middleware.Recover(), middleware.BodyLimit("1M"))
	e.GET("/healthz", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })
	e.GET("/readyz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if h.ready != nil {
			if err := h.ready(ctx); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	g := e.Group("/xtimer")
	g.POST("/createTimer", h.CreateTimer)
	g.POST("/enableTimer", h.EnableTimer)
	g.GET("/getTimer", h.GetTimer)
	g.GET("/getTasks", h.GetTasks)
}

func (h *HTTPHandler) GetTimer(c echo.Context) error {
	id, err := strconv.ParseInt(c.QueryParam("timer_id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, Response{Code: 90003, Message: "positive timer_id is required"})
	}
	timer, err := h.timerStore.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Code: 90002, Message: "query failed"})
	}
	if timer == nil {
		return c.JSON(http.StatusNotFound, Response{Code: 90003, Message: "timer not found"})
	}
	var callback any
	if err := json.Unmarshal([]byte(timer.NotifyHTTPParam), &callback); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Code: 90002, Message: "invalid stored callback"})
	}
	return c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: map[string]any{
		"timer_id": timer.TimerID, "app": timer.App, "name": timer.Name, "status": timer.Status, "cron": timer.Cron, "notify_http_param": callback,
	}})
}

func (h *HTTPHandler) GetTasks(c echo.Context) error {
	id, err := strconv.ParseInt(c.QueryParam("timer_id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, Response{Code: 90003, Message: "positive timer_id is required"})
	}
	limit := 20
	if raw := c.QueryParam("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return c.JSON(http.StatusBadRequest, Response{Code: 90003, Message: "limit must be between 1 and 100"})
		}
	}
	tasks, err := h.taskStore.ListByTimerID(c.Request().Context(), id, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Code: 90002, Message: "query failed"})
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	return c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: tasks})
}

func (h *HTTPHandler) CreateTimer(c echo.Context) error {
	var req CreateTimerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Code:    90003,
			Message: "invalid request",
			Data:    nil,
		})
	}

	paramBytes, err := json.Marshal(req.NotifyHTTPParam)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Code:    90002,
			Message: "serialization failed",
			Data:    nil,
		})
	}

	timer := &model.Timer{
		App:             req.App,
		Name:            req.Name,
		Status:          0,
		Cron:            req.Cron,
		NotifyHTTPParam: string(paramBytes),
	}
	if err := model.ValidateTimer(timer); err != nil {
		return c.JSON(http.StatusBadRequest, Response{Code: 90003, Message: err.Error()})
	}

	ctx := c.Request().Context()
	if err := h.timerStore.Save(ctx, timer); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Code:    90002,
			Message: "save failed",
			Data:    nil,
		})
	}

	return c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]any{
			"timer_id": timer.TimerID,
		},
	})
}

func (h *HTTPHandler) EnableTimer(c echo.Context) error {
	var req EnableTimerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Code:    90003,
			Message: "invalid request",
			Data:    nil,
		})
	}

	if err := model.ValidateEnable(req.App, req.TimerID, req.Status); err != nil {
		return c.JSON(http.StatusBadRequest, Response{Code: 90003, Message: err.Error()})
	}
	ctx := c.Request().Context()
	timer, err := h.timerStore.GetByID(ctx, req.TimerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Code:    90002,
			Message: "query failed",
			Data:    nil,
		})
	}
	if timer == nil {
		return c.JSON(http.StatusBadRequest, Response{
			Code:    90003,
			Message: "timer not found",
			Data:    nil,
		})
	}

	if timer.App != req.App {
		return c.JSON(http.StatusForbidden, Response{Code: 90003, Message: "app does not match timer"})
	}
	timer.Status = req.Status
	if err := h.timerStore.Update(ctx, timer); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Code:    90002,
			Message: "update failed",
			Data:    nil,
		})
	}

	return c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    map[string]any{},
	})
}
