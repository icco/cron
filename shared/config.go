package shared

import "go.uber.org/zap"

type Config struct {
	Log *zap.SugaredLogger
}
