package logicCommon

import (
	"testing"
	"time"

	"github.com/drop/GoServer/server/enum"
)

func mustParseLocalMilli(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		t.Fatalf("parse time %q failed: %v", value, err)
	}
	return parsed.UnixMilli()
}

func TestGetRankSettleTaskSettleDatesDayUsesCompletedDate(t *testing.T) {
	currentTime := mustParseLocalMilli(t, "2026-06-03 00:15:03")

	got := GetRankSettleTaskSettleDates(
		int32(enum.RANK_BOARD_SCORE_TYPE_LEVEL),
		int32(enum.RANK_BOARD_SETTLE_TYPE_DAY),
		[]int32{int32(enum.RANK_BOARD_SETTLE_TYPE_DAY)},
		"",
		currentTime,
	)

	want := []int64{20260602}
	if len(got) != len(want) {
		t.Fatalf("settle dates length mismatch, got:%v want:%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("settle dates mismatch, got:%v want:%v", got, want)
		}
	}
}

func TestGetRankSettleTaskSettleDatesArenaCapsAtWeekEnd(t *testing.T) {
	currentTime := mustParseLocalMilli(t, "2026-06-03 00:15:03")
	settleTypes := []int32{
		int32(enum.RANK_BOARD_SETTLE_TYPE_DAY),
		int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK),
	}

	gotDay := GetRankSettleTaskSettleDates(
		int32(enum.RANK_BOARD_SCORE_TYPE_ARENA),
		int32(enum.RANK_BOARD_SETTLE_TYPE_DAY),
		settleTypes,
		"s0:t20260525",
		currentTime,
	)
	wantDay := []int64{20260525, 20260526, 20260527, 20260528, 20260529, 20260530}
	if len(gotDay) != len(wantDay) {
		t.Fatalf("day settle dates length mismatch, got:%v want:%v", gotDay, wantDay)
	}
	for i := range wantDay {
		if gotDay[i] != wantDay[i] {
			t.Fatalf("day settle dates mismatch, got:%v want:%v", gotDay, wantDay)
		}
	}

	gotWeek := GetRankSettleTaskSettleDates(
		int32(enum.RANK_BOARD_SCORE_TYPE_ARENA),
		int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK),
		settleTypes,
		"s0:t20260525",
		currentTime,
	)
	wantWeek := []int64{20260531}
	if len(gotWeek) != len(wantWeek) {
		t.Fatalf("week settle dates length mismatch, got:%v want:%v", gotWeek, wantWeek)
	}
	for i := range wantWeek {
		if gotWeek[i] != wantWeek[i] {
			t.Fatalf("week settle dates mismatch, got:%v want:%v", gotWeek, wantWeek)
		}
	}
}
