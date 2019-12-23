package bl3_auto_vip

import (
	"github.com/thedevsaddam/gojsonq"
)

type StringSet map[string]struct{}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func JsonFromString(s string) *gojsonq.JSONQ {
	return gojsonq.New().JSONString(s)
}

func JsonFromBytes(bytes []byte) *gojsonq.JSONQ {
	return JsonFromString(string(bytes))
}

// Bl3Config contains the information needed to fetch the correct data
type Bl3Config struct {
	Version string    `json:"version"`
	VIP     VipConfig `json:"vipConfig"`
	// Shift   ShiftConfig `json:"shiftConfig"`
}
