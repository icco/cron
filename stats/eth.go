package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type ethResponse struct {
	Data struct {
		Base     string `json:"base"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	} `json:"data"`
}

func GetETHPrice(ctx context.Context, cfg *Config) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.coinbase.com/v2/prices/ETH-USD/buy", nil)
	if err != nil {
		return 0.0, fmt.Errorf("build eth request: %w", err)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return 0.0, fmt.Errorf("do eth request: %w", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0.0, fmt.Errorf("eth body: %w", err)
	}

	var s = new(ethResponse)
	if err := json.Unmarshal(body, &s); err != nil {
		return 0.0, fmt.Errorf("eth parse:", err)
	}

	return strconv.ParseFloat(s.Amount, 64)
}
