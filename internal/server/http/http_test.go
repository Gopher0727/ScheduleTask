package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/Gopher0727/ScheduleTask/internal/core/model"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/task"
	"github.com/Gopher0727/ScheduleTask/internal/core/service/timer"
	"github.com/Gopher0727/ScheduleTask/internal/testutil"
)

func TestTimerAPIValidationAndLifecycle(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.Timer{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	New(&timer.Store{DB: db}, &task.Store{DB: db}).RegisterRoutes(e)
	post := func(path string, payload any) *httptest.ResponseRecorder {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	valid := CreateTimerReq{App: "demo", Name: "example", Cron: "* * * * *", NotifyHTTPParam: &NotifyHTTPParam{Method: "POST", URL: "http://localhost:9090/callback"}}
	oversized := valid
	oversized.NotifyHTTPParam = &NotifyHTTPParam{Method: "POST", URL: "http://localhost/callback", Body: strings.Repeat("x", 1<<20)}
	if rec := post("/xtimer/createTimer", oversized); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request returned %d", rec.Code)
	}
	for _, tc := range []struct {
		name   string
		change func(*CreateTimerReq)
	}{
		{"empty app", func(r *CreateTimerReq) { r.App = "" }},
		{"invalid cron", func(r *CreateTimerReq) { r.Cron = "bad" }},
		{"missing callback", func(r *CreateTimerReq) { r.NotifyHTTPParam = nil }},
		{"unsupported method", func(r *CreateTimerReq) {
			r.NotifyHTTPParam = &NotifyHTTPParam{Method: "GET", URL: "http://localhost/callback"}
		}},
		{"relative URL", func(r *CreateTimerReq) { r.NotifyHTTPParam = &NotifyHTTPParam{Method: "POST", URL: "/callback"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := valid
			tc.change(&r)
			rec := post("/xtimer/createTimer", r)
			if rec.Code != 400 {
				t.Fatalf("invalid request returned %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
	var before int64
	db.Model(&model.Timer{}).Count(&before)
	if before != 0 {
		t.Fatalf("invalid requests saved %d timers", before)
	}
	rec := post("/xtimer/createTimer", valid)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var result struct {
		Data struct {
			TimerID int64 `json:"timer_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id := result.Data.TimerID
	if id <= 0 {
		t.Fatal("API returned no timer ID")
	}
	if rec := post("/xtimer/enableTimer", EnableTimerReq{App: "demo", TimerID: id, Status: 2}); rec.Code != 400 {
		t.Fatalf("invalid status returned %d", rec.Code)
	}
	if rec := post("/xtimer/enableTimer", EnableTimerReq{App: "other", TimerID: id, Status: 1}); rec.Code != 403 {
		t.Fatalf("app mismatch returned %d", rec.Code)
	}
	if rec := post("/xtimer/enableTimer", EnableTimerReq{App: "demo", TimerID: id, Status: 1}); rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var stored model.Timer
	if err := db.First(&stored, id).Error; err != nil || stored.Status != 1 {
		t.Fatalf("enable failed: %+v, %v", stored, err)
	}
	for _, path := range []string{"/xtimer/getTimer?timer_id=1", "/xtimer/getTasks?timer_id=1&limit=5", "/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 {
			t.Errorf("%s returned %d: %s", path, rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{"/xtimer/getTimer?timer_id=bad", "/xtimer/getTasks?timer_id=1&limit=1000"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 400 {
			t.Errorf("%s returned %d", path, rec.Code)
		}
	}
}
