package giteego

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexTimeEpochMillis(t *testing.T) {
	var ft FlexTime
	// Backend serializes java.util.Date as epoch millis (number).
	if err := json.Unmarshal([]byte(`1700000000000`), &ft); err != nil {
		t.Fatalf("unmarshal epoch millis: %v", err)
	}
	want := time.Unix(1700000000, 0)
	if !ft.Time.Equal(want) {
		t.Errorf("got %v, want %v", ft.Time, want)
	}
}

func TestFlexTimeNull(t *testing.T) {
	var ft FlexTime
	if err := json.Unmarshal([]byte(`null`), &ft); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !ft.IsZero() {
		t.Errorf("expected zero time for null, got %v", ft.Time)
	}
}

func TestFlexTimeRFC3339(t *testing.T) {
	var ft FlexTime
	if err := json.Unmarshal([]byte(`"2024-11-15T21:35:12+08:00"`), &ft); err != nil {
		t.Fatalf("unmarshal rfc3339: %v", err)
	}
	if ft.IsZero() {
		t.Fatal("expected parsed time, got zero")
	}
}

func TestFlexTimeDatetimeLayout(t *testing.T) {
	var ft FlexTime
	if err := json.Unmarshal([]byte(`"2024-11-15 21:35:12"`), &ft); err != nil {
		t.Fatalf("unmarshal datetime: %v", err)
	}
	if ft.IsZero() {
		t.Fatal("expected parsed time, got zero")
	}
}

// TestPipelineBuildStatusDecodeEpochMillis verifies the full status VO decodes
// when the backend sends epoch-millis number timestamps (the bug reported).
func TestPipelineBuildStatusDecodeEpochMillis(t *testing.T) {
	body := `{
		"id": 1079,
		"belongType": "GIT_OPS",
		"belongIdentifier": "1~@~master~@~ci.yml",
		"startTime": 1700000000000,
		"endTime": 1700000100000,
		"status": "SUCC",
		"stages": [
			{
				"id": 11,
				"name": "build",
				"startTime": 1700000000000,
				"endTime": 1700000050000,
				"status": "SUCC",
				"jobs": [[
					{"id": 21, "name": "compile", "startTime": 1700000000000, "endTime": 1700000020000, "status": "SUCC"}
				]]
			}
		]
	}`
	var out ResultVO[PipelineBuildStatusVO]
	if err := json.Unmarshal([]byte(`{"code":0,"msg":"","data":`+body+`}`), &out); err != nil {
		t.Fatalf("decode status VO: %v", err)
	}
	if out.Data.ID != 1079 {
		t.Errorf("got id %d, want 1079", out.Data.ID)
	}
	if out.Data.StartTime == nil || out.Data.StartTime.Year() != 2023 {
		t.Errorf("unexpected startTime: %v", out.Data.StartTime)
	}
	if len(out.Data.Stages) != 1 || out.Data.Stages[0].StartTime == nil {
		t.Fatalf("unexpected stages: %+v", out.Data.Stages)
	}
	if len(out.Data.Stages[0].Jobs) != 1 || len(out.Data.Stages[0].Jobs[0]) != 1 {
		t.Fatalf("unexpected jobs: %+v", out.Data.Stages[0].Jobs)
	}
	job := out.Data.Stages[0].Jobs[0][0]
	if job.StartTime == nil || job.StartTime.Year() != 2023 {
		t.Errorf("unexpected job startTime: %v", job.StartTime)
	}
}
