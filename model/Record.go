package model

type Record struct {
	Name         string     `json:"domain"`
	Type         RecordType `json:"Type"`
	Id           string     `json:"id"`
	ShortAnswers []string   `json:"short_answers"`
	Link         string     `json:"link"`
	TTL          int        `json:"ttl"`
	Tier         int        `json:"tier"`
}
