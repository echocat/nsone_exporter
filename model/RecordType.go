package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

type RecordType string

const (
	RT_NONE     RecordType = ""
	RT_A        RecordType = "A"
	RT_AAAA     RecordType = "AAAA"
	RT_ALIAS    RecordType = "ALIAS"
	RT_AFSDB    RecordType = "AFSDB"
	RT_ANY      RecordType = "ANY"
	RT_CNAME    RecordType = "CNAME"
	RT_DNAME    RecordType = "DNAME"
	RT_HINFO    RecordType = "HINFO"
	RT_EBOT     RecordType = "EBOT"
	RT_LINKED   RecordType = "LINKED"
	RT_MX       RecordType = "MX"
	RT_NAPTR    RecordType = "NAPTR"
	RT_NXDOMAIN RecordType = "NXDOMAIN"
	RT_NS       RecordType = "NS"
	RT_PTR      RecordType = "PTR"
	RT_RP       RecordType = "RP"
	RT_SPF      RecordType = "SPF"
	RT_SRV      RecordType = "SRV"
	RT_SOA      RecordType = "SOA"
	RT_TXT      RecordType = "TXT"
)

// AllRecordTypes contains all possible variants of RecordType.
var AllRecordTypes = []RecordType{
	RT_NONE,
	RT_A,
	RT_AAAA,
	RT_ALIAS,
	RT_AFSDB,
	RT_ANY,
	RT_CNAME,
	RT_DNAME,
	RT_HINFO,
	RT_EBOT,
	RT_LINKED,
	RT_MX,
	RT_NAPTR,
	RT_NXDOMAIN,
	RT_NS,
	RT_PTR,
	RT_RP,
	RT_SPF,
	RT_SRV,
	RT_SOA,
	RT_TXT,
}

func (instance RecordType) String() string {
	s, err := instance.CheckedString()
	if err != nil {
		panic(err)
	}
	return s
}

// CheckedString is like String but return also an optional error if there are some
// validation errors.
func (instance RecordType) CheckedString() (string, error) {
	for _, candidate := range AllRecordTypes {
		if candidate == instance {
			return string(instance), nil
		}
	}
	return "", fmt.Errorf("Illegal record type: %d", instance)
}

// Set sets the value and checks for potential errors.
func (instance *RecordType) Set(value string) error {
	upperValue := strings.ToUpper(value)
	for _, candidate := range AllRecordTypes {
		if candidate.String() == upperValue {
			(*instance) = candidate
			return nil
		}
	}
	return fmt.Errorf("Illegal record type: %v" + value)
}

// MarshalJSON is used until json marshalling. Do not call directly.
func (instance RecordType) MarshalJSON() ([]byte, error) {
	s, err := instance.CheckedString()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(s)
}

// UnmarshalJSON is used until json unmarshalling. Do not call directly.
func (instance *RecordType) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	return instance.Set(value)
}
