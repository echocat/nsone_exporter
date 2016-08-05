package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StatsPeriod string

const (
	P_HOURLY  StatsPeriod = "1h"
	P_DAILY   StatsPeriod = "24h"
	P_MONTHLY StatsPeriod = "30d"
)

// AllStatsPeriods contains all possible variants of StatsPeriod.
var AllStatsPeriods = []StatsPeriod{
	P_HOURLY,
	P_DAILY,
	P_MONTHLY,
}

func (instance StatsPeriod) String() string {
	s, err := instance.CheckedString()
	if err != nil {
		panic(err)
	}
	return s
}

// CheckedString is like String but return also an optional error if there are some
// validation errors.
func (instance StatsPeriod) CheckedString() (string, error) {
	for _, candidate := range AllStatsPeriods {
		if candidate == instance {
			return string(instance), nil
		}
	}
	return "", fmt.Errorf("Illegal period: %s", string(instance))
}

// Set sets the value and checks for potential errors.
func (instance *StatsPeriod) Set(value string) error {
	lowerValue := strings.ToLower(value)
	for _, candidate := range AllStatsPeriods {
		if candidate.String() == lowerValue {
			(*instance) = candidate
			return nil
		}
	}
	return fmt.Errorf("Illegal period: %s" + value)
}

// MarshalJSON is used until json marshalling. Do not call directly.
func (instance StatsPeriod) MarshalJSON() ([]byte, error) {
	s, err := instance.CheckedString()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(s)
}

// UnmarshalJSON is used until json unmarshalling. Do not call directly.
func (instance *StatsPeriod) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	return instance.Set(value)
}
