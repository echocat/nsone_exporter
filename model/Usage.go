package model

type Usage struct {
	Zone    string      `json:"zone"`
	Domain  string      `json:"domain"`
	Type    RecordType  `json:"rectype"`
	Queries float64     `json:"queries"`
	Period  StatsPeriod `json:"period"`
	Graph   [][]float64 `json:"graph"`
	Records float64     `json:"records"`
}
