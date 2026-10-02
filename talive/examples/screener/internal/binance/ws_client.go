package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const wsURL = "wss://fstream.binance.com/stream"

type WsClient struct {
	log       *slog.Logger
	conn      *websocket.Conn
	wsMsgIncr *atomic.Uint64
	writeMu   sync.Mutex
	handler   func(data any)
	handlerMu sync.RWMutex
}

func NewWsClient(log *slog.Logger) *WsClient {
	msgCount := atomic.Uint64{}
	msgCount.Store(1)
	return &WsClient{
		log:       log,
		conn:      nil,
		wsMsgIncr: &msgCount,
	}
}

func (wc *WsClient) Connect(errChan chan<- error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	connection, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	wc.conn = connection
	wc.log.Info("connected to binance WS")

	go func() {
		err = wc.startReadLoop()
		if err != nil {
			wc.log.Error("Reading loop error", "error", err)
			errChan <- err
		}
	}()

	return nil
}

func (wc *WsClient) Close() error {
	wc.log.Info("closing websocket")
	return wc.conn.Close()
}

func (wc *WsClient) SetKlineHandler(handler func(kline Kline)) {
	wrappedFunc := func(data any) {
		kline, ok := data.(Kline)
		if !ok {
			wc.log.Error("Kline cast error")
			return
		}
		handler(kline)
	}
	wc.handlerMu.Lock()
	defer wc.handlerMu.Unlock()
	wc.handler = wrappedFunc
}

func (wc *WsClient) startReadLoop() error {
	for {
		_, message, err := wc.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) || errors.Is(err, net.ErrClosed) {
				wc.log.Info("Connection closed normally check")
				return nil // websocket closed normally
			}
			return fmt.Errorf("ws read error: %w", err)
		}

		err = wc.processMessage(message)
		if err != nil {
			wc.log.Error("process message error", "error", err)
		}
	}
}

func (wc *WsClient) SubscribeSymbols(symbols []string) error {
	subscription := make([]string, 0, len(symbols))
	for _, s := range symbols {
		subscription = append(subscription, fmt.Sprintf("%s@kline_1m", strings.ToLower(s)))
	}
	msgSubLastPrice := wsSubscribeParams{
		Method: "SUBSCRIBE",
		Params: subscription,
		ID:     uint(wc.wsMsgIncr.Load()),
	}

	err := wc.wsSubscribe(msgSubLastPrice)
	if err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}
	return nil
}

func (wc *WsClient) wsSubscribe(msg wsSubscribeParams) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to build WS message: %w", err)
	}

	if err = wc.wsSend(websocket.TextMessage, msgBytes); err != nil {
		return fmt.Errorf("failed to send WS message: %w", err)
	}
	wc.wsMsgIncr.Add(1)

	return nil
}

func (wc *WsClient) wsSend(msgType int, msg []byte) error {
	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()

	return wc.conn.WriteMessage(msgType, msg)
}

func (wc *WsClient) processMessage(msg []byte) error {
	var wsMsg wsMessage
	err := json.Unmarshal(msg, &wsMsg)
	if err != nil {
		return fmt.Errorf("unmarshal ws message error: %w", err)
	}
	if wsMsg.Stream != "" { // data message
		var klineMsg wsKlineMessage
		err = json.Unmarshal(wsMsg.Data, &klineMsg)
		if err != nil {
			return fmt.Errorf("unmarshal kline message error: %w", err)
		}
		fOpen, _ := strconv.ParseFloat(klineMsg.KlineData.Open, 64)
		fHigh, _ := strconv.ParseFloat(klineMsg.KlineData.High, 64)
		fLow, _ := strconv.ParseFloat(klineMsg.KlineData.Low, 64)
		fClose, _ := strconv.ParseFloat(klineMsg.KlineData.Close, 64)
		fVolume, _ := strconv.ParseFloat(klineMsg.KlineData.QuoteVolume, 64)
		kline := Kline{
			O:            fOpen,
			H:            fHigh,
			L:            fLow,
			C:            fClose,
			V:            fVolume,
			IsClosed:     klineMsg.KlineData.IsClosed,
			Symbol:       klineMsg.Symbol,
			TimeStart:    time.UnixMilli(klineMsg.KlineData.TimeStart),
			TimeReceived: time.Now(),
		}

		var handlerF func(kline any)
		wc.handlerMu.RLock()
		handlerF = wc.handler
		wc.handlerMu.RUnlock()
		if handlerF != nil {
			handlerF(kline)
		}
		return nil
	}

	var resultMsg wsResultMsg
	err = json.Unmarshal(msg, &resultMsg)
	if err != nil {
		return fmt.Errorf("unmarshal ws result message error: %w", err)
	}
	if resultMsg.ID != 0 { // system msg. Success sub
		return nil
	}
	return errors.New("unknown ws message")
}
