package updater

import "github.com/icco/cron/shared"

// Config is a config.
type Config struct {
	shared.Config

	GoogleProject string
}
