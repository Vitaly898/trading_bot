package strategy

import (
	"testing"
	"time"

	"bot/internal/candle"
	"bot/internal/trading"
)

func TestRegimePositionRestorePreservesOwnerAndTimeStop(t *testing.T) {
	s, err := NewRegimeSwitch(Params{})
	if err != nil {
		t.Fatal(err)
	}
	state := trading.PositionState{Dir: 1, Owner: "mr", BarsIn: 9}
	if err := s.RestorePosition(state); err != nil {
		t.Fatal(err)
	}
	if s.PositionState() != state || s.mr.posDir != 1 || s.mr.barsIn != 9 || s.trend.posDir != 0 {
		t.Fatalf("restored state: %+v", s.PositionState())
	}
	if err := s.RestorePosition(trading.PositionState{Dir: 1}); err == nil {
		t.Fatal("unknown owner accepted")
	}
	s.OnPositionChange(0, "stop")
	if s.PositionState().Dir != 0 || s.mr.posDir != 0 {
		t.Fatal("closed module retains position")
	}
}
func TestVWAPRestoredBarsTriggerTimeStop(t *testing.T) {
	for _, bars := range []int{0, 2} {
		s, err := NewVWAPRevert(Params{"rsi_period": 2, "atr_period": 2, "time_stop": 3, "overbought": 101})
		if err != nil {
			t.Fatal(err)
		}
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 6; i++ {
			s.OnCandle(candle.Candle{Time: start.Add(time.Duration(i) * time.Minute), O: 200, H: 201, L: 199, C: 200, V: 1})
		}
		if err := s.RestorePosition(trading.PositionState{Dir: 1, BarsIn: bars}); err != nil {
			t.Fatal(err)
		}
		dir, _, _ := s.OnCandle(candle.Candle{Time: start.Add(6 * time.Minute), O: 100, H: 101, L: 99, C: 100, V: 1})
		want := 1
		if bars == 2 {
			want = 0
		}
		if dir != want || s.barsIn != bars+1 {
			t.Fatalf("bars=%d dir=%d counter=%d", bars, dir, s.barsIn)
		}
	}
}
