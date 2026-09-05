package config

import (
	"os"
	"strconv"
	"strings"
)

// BackgroundWorkersDisabled keeps a production database clone from running
// scheduled account refreshes, payments, probes, backups, and cleanup tasks.
// Normal installations leave this environment variable unset.
func BackgroundWorkersDisabled() bool {
	disabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("SERVER_DISABLE_BACKGROUND_WORKERS")))
	return disabled
}
