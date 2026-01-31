package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = ""
)

func Full() string {
	result := fmt.Sprintf("tt %s, commit %s, built at %s", Version, Commit, Date)
	if BuiltBy != "" {
		result += fmt.Sprintf(" by %s", BuiltBy)
	}
	return result
}
