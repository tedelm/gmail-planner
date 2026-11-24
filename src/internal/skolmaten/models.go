package skolmaten

// DayMenu is one school day's lunch dishes (names only).
type DayMenu struct {
	Date  string   // YYYY-MM-DD in Europe/Stockholm calendar sense from API
	Meals []string // dish names for that day
}

// SchoolWeek is one school's menu for a single ISO week.
type SchoolWeek struct {
	Slug       string
	SchoolName string
	Year       int
	Week       int
	ChildNames []string // configured children mapped to this slug
	Days       []DayMenu
}

// YearWeek is an ISO-8601 year/week pair.
type YearWeek struct {
	Year int
	Week int
}
