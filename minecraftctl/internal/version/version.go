package version

// Version is set via ldflags during build:
// -ldflags "-X github.com/paul/minecraftctl/internal/version.Version=$(git describe --tags --always --dirty)"
var Version = "dev"


