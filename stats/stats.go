package stats

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Log          *logrus.Logger
	GraphQLToken string
}

func (c *Config) Update(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

// Stat ideas:
// - Steps
// - Planes above
// - Devices on network
// - Blog posts
// - Books read this year
// - Tweets today
// - ETH price
// - Time coding
