package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type aircraftResponse struct {
	Now      float64    `json:"now"`
	Messages int        `json:"messages"`
	Aircraft []Aircraft `json:"aircraft"`
}

type Aircraft struct {
	Hex            string        `json:"hex"`
	AltBaro        int           `json:"alt_baro,omitempty"`
	AltGeom        int           `json:"alt_geom,omitempty"`
	Gs             float64       `json:"gs,omitempty"`
	Track          float64       `json:"track,omitempty"`
	BaroRate       int           `json:"baro_rate,omitempty"`
	Version        int           `json:"version,omitempty"`
	NacP           int           `json:"nac_p,omitempty"`
	NacV           int           `json:"nac_v,omitempty"`
	Sil            int           `json:"sil,omitempty"`
	SilType        string        `json:"sil_type,omitempty"`
	Mlat           []interface{} `json:"mlat"`
	Tisb           []interface{} `json:"tisb"`
	Messages       int           `json:"messages"`
	Seen           float64       `json:"seen"`
	Rssi           float64       `json:"rssi"`
	Lat            float64       `json:"lat,omitempty"`
	Lon            float64       `json:"lon,omitempty"`
	Nic            int           `json:"nic,omitempty"`
	Rc             int           `json:"rc,omitempty"`
	SeenPos        float64       `json:"seen_pos,omitempty"`
	Flight         string        `json:"flight,omitempty"`
	Ias            int           `json:"ias,omitempty"`
	Mach           float64       `json:"mach,omitempty"`
	MagHeading     float64       `json:"mag_heading,omitempty"`
	GeomRate       int           `json:"geom_rate,omitempty"`
	Squawk         string        `json:"squawk,omitempty"`
	Emergency      string        `json:"emergency,omitempty"`
	Category       string        `json:"category,omitempty"`
	NavQnh         float64       `json:"nav_qnh,omitempty"`
	NavAltitudeMcp int           `json:"nav_altitude_mcp,omitempty"`
	NavModes       []string      `json:"nav_modes,omitempty"`
	NicBaro        int           `json:"nic_baro,omitempty"`
	Tas            int           `json:"tas,omitempty"`
	TrackRate      float64       `json:"track_rate,omitempty"`
	Roll           float64       `json:"roll,omitempty"`
	NavHeading     float64       `json:"nav_heading,omitempty"`
	Gva            int           `json:"gva,omitempty"`
	Sda            int           `json:"sda,omitempty"`
	NavAltitudeFms int           `json:"nav_altitude_fms,omitempty"`
}

func GetAirplanes(ctx context.Context, cfg *Config) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://newyork.welch.io/flights/data/aircraft.json", nil)
	if err != nil {
		return 0.0, fmt.Errorf("build request: %w", err)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return 0.0, fmt.Errorf("do request: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.0, fmt.Errorf("body: %w", err)
	}
	cfg.Log.Debugw("got aircraft response", "body", string(body))

	s, err := unmarshalAirplanes(body)
	if err != nil {
		return 0.0, err
	}

	return float64(len(s.Aircraft)), nil
}

func unmarshalAirplanes(body []byte) (*aircraftResponse, error) {
	var s aircraftResponse
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return &s, nil
}
