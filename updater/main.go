package updater

import (
	"go.uber.org/zap"
)

// Config is a config.
type Config struct {
	Log           *zap.SugaredLogger
	GoogleProject string
}
