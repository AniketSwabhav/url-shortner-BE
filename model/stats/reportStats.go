package stats

type ReportStats struct {
	Month         int
	NewUsers      int
	ActiveUsers   int
	UrlsGenerated int
	UrlsRenewed   int
	TotalRevenue  float64
	PaidUser      int
}


type UserReportStats struct {
	Month         int
	MonthlySpending int 
	UrlsRenewed   int
	VisitsRenewed   int
}
