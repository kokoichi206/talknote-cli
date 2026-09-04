package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	if Version == "dev" {
		return Version
	}
	return Version + " (" + Commit + ", " + Date + ")"
}
