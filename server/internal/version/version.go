package version

// Version is the server release version. Override at link time, e.g.:
// go build -ldflags "-X github.com/ekhrunov/messenger/server/internal/version.Version=0.1.0" ./cmd/messenger
var Version = "0.1.0"
