package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
)

func GenerateToken() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(buf)
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])
	return token, hash, nil
}

// Menjalankan fungsi dari file script
func ProvisionContainer(cfg *Config, masterContainer, profile, containerName string, port int, apiURL, token string) error {
	cmd := exec.Command(cfg.ScriptPath, "provision", masterContainer, profile, containerName, fmt.Sprintf("%d", port), apiURL, token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

func DeprovisionContainer(cfg *Config, containerName string) error {
	cmd := exec.Command(cfg.ScriptPath, "deprovision", containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

func ResetContainer(cfg *Config, containerName string) error {
	cmd := exec.Command(cfg.ScriptPath, "reset", containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}
