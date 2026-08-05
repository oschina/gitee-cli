package build

import "strings"

var (
	Version   = "dev"
	CommitSHA = "unknown"
	Date      = "unknown"
)

// UserAgent identifies network requests made by this Gitee CLI build.
func UserAgent() string {
	version := strings.TrimSpace(strings.TrimPrefix(Version, "v"))
	if version == "" {
		version = "dev"
	}
	return "gitee-cli@" + version
}
