package live

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (e *TestnetExecutor) publicGet(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (e *TestnetExecutor) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	q := strings.Join(parts, "&")
	mac := hmac.New(sha256.New, []byte(e.apiSecret))
	mac.Write([]byte(q))
	return q + "&signature=" + hex.EncodeToString(mac.Sum(nil))
}

func (e *TestnetExecutor) signedPost(ctx context.Context, path string, params map[string]string, _ map[string]string) (map[string]any, error) {
	if params == nil {
		params = map[string]string{}
	}
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	return e.do(req)
}

func (e *TestnetExecutor) signedGet(ctx context.Context, path string, params map[string]string, out any) error {
	if params == nil {
		params = map[string]string{}
	}
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	resp, err := e.doRaw(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

func (e *TestnetExecutor) signedDelete(ctx context.Context, path string, params map[string]string) error {
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	_, err = e.doRaw(req)
	return err
}

func (e *TestnetExecutor) do(req *http.Request) (map[string]any, error) {
	body, err := e.doRaw(req)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func (e *TestnetExecutor) doRaw(req *http.Request) ([]byte, error) {
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var apiErr exchangeError
		_ = json.Unmarshal(body, &apiErr)
		apiErr.HTTPStatus = resp.StatusCode
		return nil, &apiErr
	}
	return body, nil
}
