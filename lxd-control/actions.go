package main

import (
	"os"
	"os/exec"
)

func RunKelolaScript(cfg *Config, args ...string) (output string, exitErr error) {
	cmd := exec.Command(cfg.ScriptPath, args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.PGPassword)

	out, err := cmd.CombinedOutput()
	return string(out), err
}