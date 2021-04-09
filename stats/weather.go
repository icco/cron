package stats

import (
	"context"

	"github.com/briandowns/openweathermap"
)

// GetCurrentWeather gets the weather for a place from OWM.
func GetCurrentWeather(location string) KeyFunc {
	return func(ctx context.Context, cfg *Config) (float64, error) {
		if err := openweathermap.ValidAPIKey(cfg.OWMKey); err != nil {
			return 0.0, err
		}

		wc := openweathermap.Config{
			Mode:   "json",
			Unit:   "F",
			Lang:   "EN",
			APIKey: cfg.OWMKey,
		}

		w, err := openweathermap.NewCurrent(wc.Unit, wc.Lang, wc.APIKey)
		if err != nil {
			return 0.0, err
		}

		if err := w.CurrentByName(location); err != nil {
			return 0.0, err
		}

		return w.Main.Temp, nil
	}
}
