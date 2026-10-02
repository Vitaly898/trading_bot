package backtest

// PositionState contains trading state only; indicators are warmed separately.
type PositionState struct {
	Dir    int    `json:"dir"`
	Owner  string `json:"owner,omitempty"`
	BarsIn int    `json:"bars_in,omitempty"`
}

// PositionRestorer allows live restarts without resetting ownership or time stops.
type PositionRestorer interface {
	PositionState() PositionState
	EntryState(dir int) PositionState
	RestorePosition(PositionState) error
}
