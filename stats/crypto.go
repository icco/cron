package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type coinbaseResponse struct {
	Data struct {
		Currency string            `json:"currency"`
		Rates    map[string]string `json:"rates"`
	} `json:"data"`
}

// GetChiaPrice gets the price of XCH in USD.
func GetChiaPrice(ctx context.Context, cfg *Config) (float64, error) {
	return GetCryptoPrice(ctx, "XCH")
}

// GetETHPrice gets the price of eth in USD.
func GetETHPrice(ctx context.Context, cfg *Config) (float64, error) {
	return GetCryptoPrice(ctx, "ETH")
}

// GetBTCPrice gets the price of BTC in USD.
func GetBTCPrice(ctx context.Context, cfg *Config) (float64, error) {
	return GetCryptoPrice(ctx, "BTC")
}

// GetCryptoPrice gets a crypto in USD from coinbase.
func GetCryptoPrice(ctx context.Context, crypto string) (float64, error) {
	url := fmt.Sprintf("https://api.coinbase.com/v2/exchange-rates?currency=%s", crypto)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0.0, fmt.Errorf("build request: %w", err)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return 0.0, fmt.Errorf("do request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0.0, fmt.Errorf("body: %w", err)
	}

	var s = new(coinbaseResponse)
	if err := json.Unmarshal(body, &s); err != nil {
		return 0.0, fmt.Errorf("parse: %w", err)
	}

	return strconv.ParseFloat(s.Data.Rates["USD"], 64)
}
