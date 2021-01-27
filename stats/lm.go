package stats

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/icco/lunchmoney"
)

func GetAssetMix(ctx context.Context) (float64, error) {
	token := os.Getenv("LUNCHMONEY_TOKEN")
	client, err := lunchmoney.NewClient(token)
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
		log.Printf("asset: %+v", v)
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
		log.Printf("account: %+v", v)
	}

	return 0.0, nil
}
