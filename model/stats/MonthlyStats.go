package stats

type MonthlyStat struct {
	Month int         `json:"month"`
	Value interface{} `json:"value"`
}
