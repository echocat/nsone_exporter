package model

import (
	"regexp"
	"fmt"
	"encoding/json"
	"strings"
)

type Regexp struct {
	pattern *regexp.Regexp
}

func NewRegexpOrPanic(plain string) *Regexp {
	trimmed := strings.ToLower(strings.TrimSpace(plain))
	result := &Regexp{}
	if len(trimmed) <= 0 || trimmed == "none" || trimmed == "false" || trimmed == "off" {
		return result
	}
	err := result.Set(plain)
	if err != nil {
		panic(err)
	}
	return result
}

func (instance *Regexp) HasValue() bool {
	return instance != nil && instance.pattern != nil
}

func (instance *Regexp) MatchString(what string) bool {
	return instance != nil && instance.pattern !=  nil && instance.pattern.MatchString(what)
}

func (instance Regexp) String() string {
	if instance.pattern == nil {
		return "off"
	}
	return instance.pattern.String()
}

// Set sets the value and checks for potential errors.
func (instance *Regexp) Set(value string) error {
	pattern, err := regexp.Compile(value)
	if err != nil {
		return fmt.Errorf("Illegal regexp: %v. Got: %v", value, err)
	}
	(*instance).pattern = pattern
	return nil
}

// MarshalJSON is used until json marshalling. Do not call directly.
func (instance Regexp) MarshalJSON() ([]byte, error) {
	s := instance.String()
	return json.Marshal(s)
}

// UnmarshalJSON is used until json unmarshalling. Do not call directly.
func (instance *Regexp) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	return instance.Set(value)
}

