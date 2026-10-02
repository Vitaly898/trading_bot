package strategy

import (
	"fmt"

	"bot/internal/trading"
)

func validPositionState(p trading.PositionState) error {
	if p.Dir < -1 || p.Dir > 1 || p.BarsIn < 0 {
		return fmt.Errorf("invalid position state: %+v", p)
	}
	return nil
}

func (s *Trend) PositionState() trading.PositionState {
	return trading.PositionState{Dir: s.posDir}
}
func (s *Trend) EntryState(dir int) trading.PositionState {
	return trading.PositionState{Dir: dir}
}
func (s *Trend) RestorePosition(p trading.PositionState) error {
	if err := validPositionState(p); err != nil {
		return err
	}
	s.posDir = p.Dir
	return nil
}

func (s *VWAPRevert) PositionState() trading.PositionState {
	return trading.PositionState{Dir: s.posDir, BarsIn: s.barsIn}
}
func (s *VWAPRevert) EntryState(dir int) trading.PositionState {
	return trading.PositionState{Dir: dir}
}
func (s *VWAPRevert) RestorePosition(p trading.PositionState) error {
	if err := validPositionState(p); err != nil {
		return err
	}
	s.posDir, s.barsIn = p.Dir, p.BarsIn
	return nil
}

func (s *RegimeSwitch) PositionState() trading.PositionState {
	p := trading.PositionState{Dir: s.posDir, Owner: s.posFrom}
	if s.posFrom == "mr" {
		p.BarsIn = s.mr.barsIn
	}
	return p
}
func (s *RegimeSwitch) EntryState(dir int) trading.PositionState {
	return trading.PositionState{Dir: dir, Owner: s.nextOwner}
}
func (s *RegimeSwitch) RestorePosition(p trading.PositionState) error {
	if err := validPositionState(p); err != nil {
		return err
	}
	if p.Dir != 0 && p.Owner != "trend" && p.Owner != "mr" {
		return fmt.Errorf("cannot restore regime position without trend/mr owner")
	}
	s.trend.OnPositionChange(0, "restore")
	s.mr.OnPositionChange(0, "restore")
	s.posDir, s.posFrom = p.Dir, p.Owner
	if p.Dir == 0 {
		s.posFrom = ""
		return nil
	}
	if p.Owner == "mr" {
		return s.mr.RestorePosition(p)
	}
	return s.trend.RestorePosition(p)
}
