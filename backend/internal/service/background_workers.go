package service

import "github.com/Wei-Shaw/sub2api/internal/config"

func startBackgroundWorker(start func()) {
	if !config.BackgroundWorkersDisabled() {
		start()
	}
}
