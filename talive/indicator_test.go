package talive

import (
	"testing"
	"time"
)

func TestAnchorChanged(t *testing.T) {
	mustParse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("bad timestamp %q: %v", s, err)
		}
		return ts
	}

	tests := []struct {
		name string
		mode Anchor
		prev string
		curr string
		want bool
	}{
		// AnchorNone — never reports a change.
		{"None/same", AnchorNone, "2026-04-25T10:00:00Z", "2026-04-25T11:00:00Z", false},
		{"None/different", AnchorNone, "2026-04-25T10:00:00Z", "2027-01-01T00:00:00Z", false},

		// AnchorDaily.
		{"Daily/sameDay", AnchorDaily, "2026-04-25T08:00:00Z", "2026-04-25T23:59:59Z", false},
		{"Daily/nextDay", AnchorDaily, "2026-04-25T23:30:00Z", "2026-04-26T00:30:00Z", true},
		{"Daily/yearRollover", AnchorDaily, "2026-12-31T23:00:00Z", "2027-01-01T00:30:00Z", true},
		{"Daily/sameDOYDifferentYear", AnchorDaily, "2026-04-25T10:00:00Z", "2027-04-25T10:00:00Z", true},

		// AnchorWeekly (ISO week, Mon-start).
		{"Weekly/sameWeekMidweek", AnchorWeekly, "2026-04-21T10:00:00Z", "2026-04-23T10:00:00Z", false},
		{"Weekly/sunToMon", AnchorWeekly, "2026-04-26T23:00:00Z", "2026-04-27T00:30:00Z", true}, // Sun→Mon
		{"Weekly/sameWeekSatToSun", AnchorWeekly, "2026-04-25T10:00:00Z", "2026-04-26T10:00:00Z", false},
		// ISO year edge: 2024-12-30 (Mon) is ISO week 1 of 2025.
		{"Weekly/isoYearEdgeSameWeek", AnchorWeekly, "2024-12-30T10:00:00Z", "2025-01-01T10:00:00Z", false},
		// Dec 29 2025 is ISO week 1 of 2026, Dec 28 2025 is week 52 of 2025 → boundary.
		{"Weekly/isoYearTransition", AnchorWeekly, "2025-12-28T10:00:00Z", "2025-12-29T10:00:00Z", true},

		// AnchorMonthly.
		{"Monthly/sameMonth", AnchorMonthly, "2026-04-01T00:00:00Z", "2026-04-30T23:59:00Z", false},
		{"Monthly/nextMonth", AnchorMonthly, "2026-04-30T23:00:00Z", "2026-05-01T00:30:00Z", true},
		{"Monthly/sameMonthDifferentYear", AnchorMonthly, "2026-04-15T10:00:00Z", "2027-04-15T10:00:00Z", true},

		// AnchorQuarterly. Quarters: Q1=Jan-Mar, Q2=Apr-Jun, Q3=Jul-Sep, Q4=Oct-Dec.
		{"Quarterly/sameQuarter", AnchorQuarterly, "2026-04-01T00:00:00Z", "2026-06-30T23:59:00Z", false},
		{"Quarterly/Q1toQ2", AnchorQuarterly, "2026-03-31T23:00:00Z", "2026-04-01T00:30:00Z", true},
		{"Quarterly/Q4toQ1nextYear", AnchorQuarterly, "2026-12-31T23:00:00Z", "2027-01-01T00:30:00Z", true},
		{"Quarterly/sameQuarterDifferentYear", AnchorQuarterly, "2026-02-15T10:00:00Z", "2027-02-15T10:00:00Z", true},

		// AnchorYearly.
		{"Yearly/sameYear", AnchorYearly, "2026-01-01T00:00:00Z", "2026-12-31T23:59:00Z", false},
		{"Yearly/yearRollover", AnchorYearly, "2026-12-31T23:00:00Z", "2027-01-01T00:30:00Z", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := anchorChanged(mustParse(tc.prev), mustParse(tc.curr), tc.mode)
			if got != tc.want {
				t.Fatalf("anchorChanged(%s, %s, %s) = %v, want %v",
					tc.prev, tc.curr, tc.mode, got, tc.want)
			}
		})
	}
}

// Verify embedded location of timestamps governs the boundary check (not UTC).
func TestAnchorChangedRespectsTimestampLocation(t *testing.T) {
	sydney := time.FixedZone("Sydney", 10*3600)

	// Same instant: 2026-04-25 23:30 UTC = 2026-04-26 09:30 Sydney.
	utcLate := time.Date(2026, 4, 25, 23, 30, 0, 0, time.UTC)
	utcNextHour := time.Date(2026, 4, 26, 0, 30, 0, 0, time.UTC)

	// In UTC: 23:30 → 00:30 next day = boundary.
	if !anchorChanged(utcLate, utcNextHour, AnchorDaily) {
		t.Error("UTC: expected daily boundary across midnight")
	}

	// Same two instants, viewed from Sydney: 09:30 → 10:30 same day = no boundary.
	syLate := utcLate.In(sydney)
	syNextHour := utcNextHour.In(sydney)
	if anchorChanged(syLate, syNextHour, AnchorDaily) {
		t.Error("Sydney: expected no daily boundary (same Sydney day)")
	}
}
