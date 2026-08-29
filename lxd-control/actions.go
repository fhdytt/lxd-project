package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
)

// GenerateToken membuat token acak 32-byte (64 karakter hex) untuk
// otentikasi TUI praktikan ke API backend, plus hash SHA-256-nya. Token asli
// TIDAK PERNAH disimpan ke database — cuma hash-nya (lihat InsertEnvironment
// di db.go) — token asli hanya "hidup" sesaat di sini dan di dalam container
// (sebagai env var).
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

// ProvisionContainer menjalankan "kelola-lxd.sh provision" untuk SATU
// container. Semua keputusan (nama container, port, token) sudah
// ditentukan lxd-control SEBELUM memanggil fungsi ini — script cuma
// eksekutor perintah LXD teknis, tidak tahu apa-apa soal ruangan/sesi/database.
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