package version

// Current is injected at build time via:
//   go build -ldflags "-X youtube-downloader/internal/version.Current=1.2.3"
// Falls back to "dev" for local builds.
var Current = "dev"
