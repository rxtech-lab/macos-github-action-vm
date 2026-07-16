package buildinfo

// These values are replaced by linker flags in release builds.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	TeamID    = ""
)
