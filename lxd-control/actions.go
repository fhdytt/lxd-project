package main

import (
	"os"
	"os/exec"
)

// RunKelolaScript menjalankan kelola-lxd.sh dengan argumen tertentu,
// meneruskan PGPASSWORD ke environment subprocess (dibutuhkan script itu
// untuk konek ke PostgreSQL), dan mengembalikan gabungan stdout+stderr
// sebagai satu string untuk ditampilkan di layar hasil.
//
// Dijalankan secara BLOCKING (bukan streaming baris-per-baris) — untuk versi
// pertama ini cukup, karena provisioning/reset di skala testing (5 container)
// selesai dalam hitungan detik. Kalau nanti skalanya jauh lebih besar dan
// prosesnya makan waktu lama, ini kandidat kuat untuk diubah ke streaming
// (kirim tea.Msg per baris output).
func RunKelolaScript(cfg *Config, args ...string) (output string, exitErr error) {
	cmd := exec.Command(cfg.ScriptPath, args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.PGPassword)

	out, err := cmd.CombinedOutput()
	return string(out), err
}