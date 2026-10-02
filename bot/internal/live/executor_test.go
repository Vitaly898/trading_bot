package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bot/internal/backtest"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) (*http.Response, error) {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}
func testExecutor(handler roundTripFunc) *TestnetExecutor {
	e := NewTestnetExecutor(quietLog(), "test-key", "test-secret")
	e.client = &http.Client{Transport: handler}
	e.baseURL = "https://exchange.invalid"
	e.pollInterval = time.Millisecond
	e.steps["TEST"], e.ticks["TEST"] = 0.1, 0.25
	return e
}
func filled(qty, price string) string {
	return fmt.Sprintf(`{"orderId":9007199254740993,"status":"FILLED","avgPrice":%q,"executedQty":%q}`, price, qty)
}
func positionResponse(sym, amt, price string) string {
	return fmt.Sprintf(`[{"symbol":%q,"positionSide":"BOTH","positionAmt":%q,"entryPrice":%q}]`, sym, amt, price)
}
func protectionResponse(sym string) string {
	return fmt.Sprintf(`[{"algoId":10,"clientAlgoId":"tb-stop","symbol":%q,"orderType":"STOP_MARKET","positionSide":"BOTH","side":"SELL","closePosition":true,"triggerPrice":"90"},{"algoId":11,"clientAlgoId":"tb-take","symbol":%q,"orderType":"TAKE_PROFIT_MARKET","positionSide":"BOTH","side":"SELL","closePosition":true,"triggerPrice":"120"}]`, sym, sym)
}
func existingPosition() *Position {
	return &Position{Symbol: "TEST", Dir: 1, Qty: 1, EntryPrice: 100, Stop: 90, Take: 120, StopOrderID: 10, TakeOrderID: 11, EntryTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func TestTestnetUsesConfirmedFillAndQuantity(t *testing.T) {
	gets, posts := 0, 0
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == "/fapi/v1/order" && r.Method == http.MethodPost:
			posts++
			if r.URL.Query().Get("quantity") != "0.3" || r.URL.Query().Get("newOrderRespType") != "RESULT" {
				t.Fatalf("order params: %v", r.URL.Query())
			}
			return response(200, `{"orderId":9007199254740993,"status":"NEW","avgPrice":"0","executedQty":"0"}`)
		case r.URL.Path == "/fapi/v1/order" && r.Method == http.MethodGet:
			gets++
			if !strings.HasPrefix(r.URL.Query().Get("origClientOrderId"), "tb-") {
				t.Fatal("missing stable client order ID")
			}
			return response(200, filled("0.3", "101"))
		case r.URL.Path == "/fapi/v1/algoOrder":
			if r.URL.Query().Get("triggerPrice") != "90.00" {
				t.Fatal("trigger not on tick grid")
			}
			return response(200, `{"algoId":10}`)
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		return nil, nil
	})
	price, err := e.Open(context.Background(), "TEST", 1, 0.39, 90, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, price, 101)
	assertNear(t, e.Positions()["TEST"].Qty, 0.3)
	if gets != 1 || posts != 1 || e.Positions()["TEST"].EntryOrderID != 9007199254740993 {
		t.Fatal("fill confirmation or ID precision lost")
	}
	if got := e.fmtQty("TEST", 0.3); got != "0.3" {
		t.Fatalf("exact lot truncated: %s", got)
	}
	if got := e.fmtPrice("TEST", 90.13); got != "90.25" {
		t.Fatalf("tick rounded incorrectly: %s", got)
	}
}
func TestTestnetConfirmationErrorRetainsIntent(t *testing.T) {
	for _, ambiguousPost := range []bool{false, true} {
		e := testExecutor(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodPost {
				if ambiguousPost {
					return response(503, `{"code":-1007,"msg":"timeout"}`)
				}
				return response(200, `{"orderId":1,"status":"NEW"}`)
			}
			return response(400, `{"code":-2013,"msg":"Order does not exist yet"}`)
		})
		if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err == nil {
			t.Fatal("missing confirmation accepted")
		}
		p := e.Positions()["TEST"]
		if p == nil || !p.Unconfirmed || p.ClientOrderID == "" {
			t.Fatal("uncertain entry intent lost")
		}
	}
}
func TestTestnetDefiniteRejectionDoesNotCreatePosition(t *testing.T) {
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		return response(400, `{"code":-2019,"msg":"Margin is insufficient"}`)
	})
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err == nil {
		t.Fatal("rejection ignored")
	}
	if len(e.Positions()) != 0 {
		t.Fatal("rejected entry cached as position")
	}
}
func TestCloseKeepsProtectionOnFailureAndPartialFill(t *testing.T) {
	for _, partial := range []bool{false, true} {
		requests := 0
		e := testExecutor(func(r *http.Request) (*http.Response, error) {
			requests++
			if r.Method != http.MethodPost || r.URL.Path != "/fapi/v1/order" || r.URL.Query().Get("reduceOnly") != "true" {
				t.Fatal("protection removed before confirmed close")
			}
			if partial {
				return response(200, filled("0.4", "99"))
			}
			return response(400, `{"code":-2019,"msg":"rejected"}`)
		})
		e.positions["TEST"] = existingPosition()
		if _, err := e.Close(context.Background(), "TEST", 100); err == nil {
			t.Fatal("failure or partial close ignored")
		}
		p := e.Positions()["TEST"]
		if p == nil || p.StopOrderID != 10 || requests != 1 {
			t.Fatal("position/protection lost")
		}
		if partial {
			assertNear(t, p.Qty, 0.6)
		} else {
			assertNear(t, p.Qty, 1)
		}
	}
}
func TestCloseCleansOnlyBotProtectionAfterFill(t *testing.T) {
	var calls []string
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			return response(200, filled("1", "99"))
		}
		if r.Method == http.MethodGet {
			return response(200, `[{"algoId":10,"clientAlgoId":"tb-stop","symbol":"TEST"},{"algoId":99,"clientAlgoId":"manual","symbol":"TEST"}]`)
		}
		if r.URL.Query().Get("algoId") != "10" {
			t.Fatal("cancelled unrelated protection")
		}
		return response(200, `{"code":200}`)
	})
	e.positions["TEST"] = existingPosition()
	if _, err := e.Close(context.Background(), "TEST", 100); err != nil {
		t.Fatal(err)
	}
	if len(e.Positions()) != 0 || len(calls) != 3 || calls[0] != "POST /fapi/v1/order" {
		t.Fatalf("close sequence: %v", calls)
	}
}
func TestReconcilePreservesMetadataAndRecoversMissingStop(t *testing.T) {
	for _, missing := range []bool{false, true} {
		placed := 0
		e := testExecutor(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/fapi/v2/positionRisk":
				return response(200, positionResponse("TEST", "0.8", "101"))
			case "/fapi/v1/openAlgoOrders":
				if missing {
					return response(200, `[]`)
				}
				return response(200, protectionResponse("TEST"))
			case "/fapi/v1/algoOrder":
				placed++
				return response(200, fmt.Sprintf(`{"algoId":%d}`, 20+placed))
			}
			t.Fatal("unexpected request")
			return nil, nil
		})
		before := existingPosition()
		e.positions["TEST"] = before
		if err := e.Reconcile(context.Background(), []string{"TEST"}); err != nil {
			t.Fatal(err)
		}
		p := e.Positions()["TEST"]
		assertNear(t, p.Qty, 0.8)
		assertNear(t, p.EntryPrice, 101)
		assertNear(t, p.Stop, 90)
		if !p.EntryTime.Equal(before.EntryTime) {
			t.Fatal("entry time overwritten")
		}
		if missing && placed != 2 {
			t.Fatal("missing protections not restored")
		}
	}
}
func TestReconcileFailureDoesNotPartiallyReplaceCache(t *testing.T) {
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("symbol") == "B" {
			return response(500, `{"msg":"failure"}`)
		}
		if r.URL.Path == "/fapi/v2/positionRisk" {
			return response(200, positionResponse("TEST", "0.8", "101"))
		}
		return response(200, protectionResponse("TEST"))
	})
	e.positions["TEST"] = existingPosition()
	if err := e.Reconcile(context.Background(), []string{"TEST", "B"}); err == nil {
		t.Fatal("failed response ignored")
	}
	assertNear(t, e.Positions()["TEST"].Qty, 1)
	assertNear(t, e.Positions()["TEST"].EntryPrice, 100)
}
func TestRunnerReconcileNotifiesExchangeCloseOnce(t *testing.T) {
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/fapi/v2/positionRisk" {
			return response(200, positionResponse("TEST", "0", "0"))
		}
		return response(200, `[]`)
	})
	e.positions["TEST"] = existingPosition()
	r := NewRunner(quietLog(), runnerConfig(""), e)
	r.positionStates["TEST"] = backtest.PositionState{Dir: 1, Owner: "mr"}
	s := &recordingStrategy{}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.changes) != 1 || s.changes[0] != (change{0, "exchange"}) || s.state.Dir != 0 {
		t.Fatalf("close notifications: %+v", s.changes)
	}
}
func TestEntryIntentIsSavedBeforeSubmission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	e := testExecutor(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/fapi/v1/order" && req.Method == http.MethodPost {
			buf, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var snap stateSnapshot
			if err := json.Unmarshal(buf, &snap); err != nil {
				t.Fatal(err)
			}
			p := snap.Positions["TEST"]
			if p == nil || !p.Unconfirmed || p.ClientOrderID != req.URL.Query().Get("newClientOrderId") || snap.StrategyStates["TEST"].Owner != "mr" {
				t.Fatal("entry intent not persisted before request")
			}
			return response(200, filled("1", "100"))
		}
		if req.URL.Path == "/fapi/v1/algoOrder" {
			return response(200, `{"algoId":10}`)
		}
		t.Fatal("unexpected request")
		return nil, nil
	})
	r := NewRunner(quietLog(), runnerConfig(path), e)
	r.positionStates["TEST"] = backtest.PositionState{Dir: 1, Owner: "mr"}
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err != nil {
		t.Fatal(err)
	}
}
func TestFailedIntentSaveDoesNotSubmitOrder(t *testing.T) {
	e := testExecutor(func(req *http.Request) (*http.Response, error) {
		t.Fatal("submitted without durable intent")
		return nil, nil
	})
	e.beforeSubmit = func(map[string]*Position) error { return errors.New("disk failure") }
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err == nil {
		t.Fatal("save failure ignored")
	}
	if len(e.Positions()) != 0 {
		t.Fatal("unsent entry cached")
	}
}
func TestPendingEntryReconcileUsesAuthoritativeFill(t *testing.T) {
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/fapi/v1/order":
			return response(200, filled("0.8", "102"))
		case "/fapi/v2/positionRisk":
			return response(200, positionResponse("TEST", "0.8", "102"))
		case "/fapi/v1/openAlgoOrders":
			return response(200, protectionResponse("TEST"))
		}
		t.Fatal("unexpected request")
		return nil, nil
	})
	p := existingPosition()
	p.Unconfirmed = true
	p.ClientOrderID = "tb-entry"
	e.positions["TEST"] = p
	if err := e.Reconcile(context.Background(), []string{"TEST"}); err != nil {
		t.Fatal(err)
	}
	p = e.Positions()["TEST"]
	if p.Unconfirmed {
		t.Fatal("fill not confirmed")
	}
	assertNear(t, p.Qty, 0.8)
	assertNear(t, p.EntryPrice, 102)
}
func TestCancelledContextStopsConfirmation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := testExecutor(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			return response(200, `{"status":"NEW"}`)
		}
		cancel()
		return response(200, `{"status":"NEW"}`)
	})
	if _, err := e.Open(ctx, "TEST", 1, 1, 90, 0, 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not returned: %v", err)
	}
}

func TestProtectionFailureClosesEntryAndKeepsUnknownCloseForReconcile(t *testing.T) {
	for _, closeFails := range []bool{false, true} {
		markets := 0
		e := testExecutor(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/fapi/v1/order" {
				markets++
				if markets == 2 && closeFails {
					return response(503, `{"code":-1007,"msg":"timeout"}`)
				}
				return response(200, filled("1", "100"))
			}
			if r.URL.Path == "/fapi/v1/algoOrder" {
				return response(400, `{"code":-2021,"msg":"would immediately trigger"}`)
			}
			return response(200, `[]`)
		})
		if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err == nil {
			t.Fatal("protection failure ignored")
		}
		p := e.Positions()["TEST"]
		if closeFails {
			if p == nil || p.PendingClose == "" {
				t.Fatal("uncertain emergency close lost")
			}
		} else if p != nil {
			t.Fatal("confirmed emergency close retained")
		}
		if markets != 2 {
			t.Fatalf("market attempts: %d", markets)
		}
	}
}
func TestRestartResolvesPersistedEntryWithoutResubmission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	markets := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/fapi/v1/order":
			if r.Method == http.MethodPost {
				markets++
			}
			return response(200, filled("1", "101"))
		case "/fapi/v1/algoOrder":
			return response(200, `{"algoId":10}`)
		case "/fapi/v2/positionRisk":
			return response(200, positionResponse("TEST", "1", "101"))
		case "/fapi/v1/openAlgoOrders":
			return response(200, protectionResponse("TEST"))
		}
		t.Fatal("unexpected request")
		return nil, nil
	})
	e := testExecutor(transport)
	r := NewRunner(quietLog(), runnerConfig(path), e)
	r.positionStates["TEST"] = backtest.PositionState{Dir: 1, Owner: "mr"}
	r.entryBars["TEST"] = event(0, 100, true).Candle.Time
	r.lastClosed["TEST"] = r.entryBars["TEST"]
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err != nil {
		t.Fatal(err)
	}
	// The disk still contains the pre-request intent: simulate losing the final checkpoint.
	nextExec := testExecutor(transport)
	next := NewRunner(quietLog(), runnerConfig(path), nextExec)
	if err := next.LoadState(); err != nil {
		t.Fatal(err)
	}
	if !nextExec.Positions()["TEST"].Unconfirmed {
		t.Fatal("test did not restore pre-request intent")
	}
	if err := next.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := &recordingStrategy{}
	if err := next.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	if markets != 1 || nextExec.Positions()["TEST"].Unconfirmed || s.state.Owner != "mr" || s.state.Dir != 1 {
		t.Fatal("restart duplicated entry or lost ownership")
	}
}
