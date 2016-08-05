package model

type Zone struct {
	Name         string    `json:"zone"`
	Serial       int64     `json:"serial"`
	Id           string    `json:"id"`
	TTL          int       `json:"ttl"`
	NxTTL        int       `json:"nx_ttl"`
	Retry        int       `json:"retry"`
	Refresh      int       `json:"refresh"`
	Expiry       int       `json:"expiry"`
	Hostmaster   string    `json:"hostmaster"`
	Pool         string    `json:"pool"`
	NetworkPools []string  `json:"network_pools"`
	DnsServers   []string  `json:"dns_servers"`
	Records      []*Record `json:"records"`
	Link         string    `json:"link"`
}
