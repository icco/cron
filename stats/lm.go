package stats

import (
	"context"
	"fmt"

	"github.com/icco/lunchmoney"
)

// GetAssetMix gets our asset mix from LunchMoney.
func (c *Config) GetAssetMix(ctx context.Context) (float64, error) {
	client, err := lunchmoney.NewClient(c.LunchMoneyToken)
	if err != nil {
		return 0.0, fmt.Errorf("lm client: %w", err)
	}

	as, err := client.GetAssets(ctx)
	if err != nil {
		return 0.0, fmt.Errorf("get assets: %w", err)
	}

	for _, t := range as {
		v, err := t.ParsedAmount()
		if err != nil {
			return 0.0, err
		}

		// .AsMajorUnits()
		c.Log.Debugf("asset: %q - %+v", t.Name, v.Display())
	}

	pas, err := client.GetPlaidAccounts(ctx)
	if err != nil {
		return 0.0, fmt.Errorf("get accounts: %w", err)
	}

	for _, t := range pas {
		v, err := t.ParsedAmount()
		if err != nil {
			return 0.0, err
		}
		c.Log.Debugf("account: %q - %+v", t.Name, v.Display())
	}

	return 0.0, nil
}
