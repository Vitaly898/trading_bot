package binance

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const httpURL = "https://fapi.binance.com"

type HTTPClient struct {
	log        *slog.Logger
	httpClient *http.Client
}

func NewHTTPClient(log *slog.Logger) *HTTPClient {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &HTTPClient{
		log.With("client", "http"),
		httpClient,
	}
}

func (hc *HTTPClient) TopVolumeSymbols(ctx context.Context, topNumber int) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)
	var exchangeInfo *ExchangeInfoResp
	var dayTickerData DayTickerResp

	g.Go(func() error {
		var err error
		dayTickerData, err = hc.DayTicker(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		exchangeInfo, err = hc.ExchangeInfo(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	activeSymbols := make(map[string]bool)
	for _, symbolInfo := range exchangeInfo.Symbols {
		if symbolInfo.Status == "TRADING" {
			activeSymbols[symbolInfo.Symbol] = true
		}
	}

	slices.SortFunc(dayTickerData, func(a, b DayTicker) int {
		bVol, parseErr := strconv.ParseFloat(b.QuoteVolume, 64)
		if parseErr != nil {
			return 1
		}
		aVol, parseErr := strconv.ParseFloat(a.QuoteVolume, 64)
		if parseErr != nil {
			return -1
		}
		return cmp.Compare(bVol, aVol) // desc
	})

	symbols := make([]string, 0, topNumber)
	for _, dayTicker := range dayTickerData {
		if _, ok := activeSymbols[dayTicker.Symbol]; ok {
			symbols = append(symbols, dayTicker.Symbol)
			if len(symbols) >= topNumber {
				break
			}
		}
	}
	return symbols, nil
}

func (hc *HTTPClient) DayTicker(ctx context.Context) ([]DayTicker, error) {
	params := RequestParams{
		Method: http.MethodGet,
		Path:   "/fapi/v1/ticker/24hr",
	}
	var parsedResponse DayTickerResp
	err := hc.makeTypedRequest(ctx, &params, &parsedResponse)
	if err != nil {
		return nil, err
	}
	return parsedResponse, nil
}

func (hc *HTTPClient) ExchangeInfo(ctx context.Context) (*ExchangeInfoResp, error) {
	params := RequestParams{
		Method: http.MethodGet,
		Path:   "/fapi/v1/exchangeInfo",
	}
	var parsedResponse ExchangeInfoResp
	err := hc.makeTypedRequest(ctx, &params, &parsedResponse)
	if err != nil {
		return nil, err
	}
	return &parsedResponse, nil
}

func (hc *HTTPClient) LastKlines(ctx context.Context, symbol string, limit int) ([]Kline, error) {
	params := RequestParams{
		Method: http.MethodGet,
		Path:   "/fapi/v1/klines",
		QueryParams: map[string]string{
			"limit":    strconv.Itoa(limit),
			"symbol":   strings.ToLower(symbol),
			"interval": "1m",
		},
	}
	var parsedResponse KlinesResp
	err := hc.makeTypedRequest(ctx, &params, &parsedResponse)
	if err != nil {
		return nil, err
	}
	receivedAt := time.Now()

	result := make([]Kline, 0, len(parsedResponse))
	if len(parsedResponse) == 0 {
		return result, nil
	}
	for _, klineData := range parsedResponse {
		openTimeF, ok := klineData[0].(float64)
		if !ok {
			hc.log.Warn("Invalid kline data at index 0", "data", klineData[0], "type", fmt.Sprintf("%T", klineData[0]))
			continue
		}
		openTime := time.UnixMilli(int64(openTimeF))
		if receivedAt.Truncate(time.Minute).Equal(openTime) {
			continue
		}

		openStr, ok := klineData[1].(string)
		if !ok {
			hc.log.Warn("Invalid kline data at index 1", "data", klineData[1], "type", fmt.Sprintf("%T", klineData[1]))
			continue
		}
		openF, _ := strconv.ParseFloat(openStr, 64)

		highStr, ok := klineData[2].(string)
		if !ok {
			hc.log.Warn("Invalid kline data at index 2", "data", klineData[2], "type", fmt.Sprintf("%T", klineData[2]))
			continue
		}
		highF, _ := strconv.ParseFloat(highStr, 64)

		lowStr, ok := klineData[3].(string)
		if !ok {
			hc.log.Warn("Invalid kline data at index 3", "data", klineData[3], "type", fmt.Sprintf("%T", klineData[3]))
			continue
		}
		lowF, _ := strconv.ParseFloat(lowStr, 64)

		closeStr, ok := klineData[4].(string)
		if !ok {
			hc.log.Warn("Invalid kline data at index 4", "data", klineData[4], "type", fmt.Sprintf("%T", klineData[4]))
			continue
		}
		closeF, _ := strconv.ParseFloat(closeStr, 64)

		volumeStr, ok := klineData[5].(string)
		if !ok {
			hc.log.Warn("Invalid kline data at index 5", "data", klineData[5], "type", fmt.Sprintf("%T", klineData[5]))
			continue
		}
		volumeF, _ := strconv.ParseFloat(volumeStr, 64)

		result = append(result, Kline{
			O:            openF,
			H:            highF,
			L:            lowF,
			C:            closeF,
			V:            volumeF,
			IsClosed:     true,
			Symbol:       symbol,
			TimeStart:    openTime,
			TimeReceived: receivedAt,
		})
	}
	return result, nil
}

func (hc *HTTPClient) makeTypedRequest(ctx context.Context, params *RequestParams, respType any) error {
	response, err := hc.makeRequest(ctx, params)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := response.Body.Close()
		if closeErr != nil {
			hc.log.Error("response body close err", "err", closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("response error: %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(respType)
}

func (hc *HTTPClient) makeRequest(ctx context.Context, params *RequestParams) (*http.Response, error) {
	switch params.Method {
	case http.MethodPost:
		var jsonBody []byte
		var err error

		if len(params.BodyParams) > 0 {
			jsonBody, err = json.Marshal(params.BodyParams)
			if err != nil {
				return nil, fmt.Errorf("error marshaling request body: %w", err)
			}
		}
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			httpURL+params.Path,
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		return hc.httpClient.Do(req)
	case http.MethodGet:
		u, err := url.Parse(httpURL + params.Path)
		if err != nil {
			return nil, err
		}

		if len(params.QueryParams) > 0 {
			query := u.Query()
			for key, value := range params.QueryParams {
				query.Add(key, value)
			}
			u.RawQuery = query.Encode()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		hc.log.Debug(u.String())
		req.Header.Set("Content-Type", "application/json")
		return hc.httpClient.Do(req)
	default:
		return nil, errors.New("unexpected request method")
	}
}
