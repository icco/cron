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

type FlexInt int

type Aircraft struct {
	AltBaro        FlexInt       `json:"alt_baro,omitempty"`
	AltGeom        int           `json:"alt_geom,omitempty"`
	BaroRate       int           `json:"baro_rate,omitempty"`
	Category       string        `json:"category,omitempty"`
	Emergency      string        `json:"emergency,omitempty"`
	Flight         string        `json:"flight,omitempty"`
	GeomRate       int           `json:"geom_rate,omitempty"`
	Gs             float64       `json:"gs,omitempty"`
	Gva            int           `json:"gva,omitempty"`
	Hex            string        `json:"hex"`
	Lat            float64       `json:"lat,omitempty"`
	Lon            float64       `json:"lon,omitempty"`
	Messages       int           `json:"messages"`
	Mlat           []interface{} `json:"mlat"`
	NacP           int           `json:"nac_p,omitempty"`
	NacV           int           `json:"nac_v,omitempty"`
	NavAltitudeMcp int           `json:"nav_altitude_mcp,omitempty"`
	NavHeading     float64       `json:"nav_heading,omitempty"`
	NavModes       []string      `json:"nav_modes,omitempty"`
	NavQnh         float64       `json:"nav_qnh,omitempty"`
	Nic            int           `json:"nic,omitempty"`
	NicBaro        int           `json:"nic_baro,omitempty"`
	Rc             int           `json:"rc,omitempty"`
	Rssi           float64       `json:"rssi"`
	Sda            int           `json:"sda,omitempty"`
	Seen           float64       `json:"seen"`
	SeenPos        float64       `json:"seen_pos,omitempty"`
	Sil            int           `json:"sil,omitempty"`
	SilType        string        `json:"sil_type,omitempty"`
	Squawk         string        `json:"squawk,omitempty"`
	Tisb           []interface{} `json:"tisb"`
	Track          float64       `json:"track,omitempty"`
	Type           string        `json:"type,omitempty"`
	Version        int           `json:"version,omitempty"`
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

// UnmarshalJSON implements the json.Unmarshaler interface, which
// allows us to ingest values of any json type as an int and run our custom conversion
func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		return json.Unmarshal(b, (*int)(fi))
	}

	*fi = FlexInt(0)
	return nil
}
