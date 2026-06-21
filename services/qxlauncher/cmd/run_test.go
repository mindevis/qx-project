package main

import (
	"testing"
)

func TestRunSkipsPollWhenDisabled(t *testing.T) {
	setupLauncherTestRepo(t)
	run()
}
