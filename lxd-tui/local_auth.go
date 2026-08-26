package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/amoghe/go-crypt"
)

// LocalUser merepresentasikan satu akun Linux yang bisa dipilih praktikan
// untuk login, dibaca langsung dari /etc/passwd di dalam container itu
// sendiri (bukan dari API — ini murni informasi lokal container).
type LocalUser struct {
	Username string
	UID      int
}

// listLocalUsers membaca /etc/passwd dan mengembalikan daftar user yang
// masuk akal untuk dipilih praktikan: root, plus user apapun yang sudah
// dibuat praktikan sendiri selama praktikum (UID >= 1000, punya shell
// interaktif, bukan akun service/system).
func listLocalUsers() ([]LocalUser, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("tidak bisa membaca /etc/passwd: %w", err)
	}
	defer f.Close()

	var users []LocalUser
	var rootUser *LocalUser

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 7 {
			continue
		}

		username := fields[0]
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		shell := fields[6]

		if strings.HasSuffix(shell, "nologin") || strings.HasSuffix(shell, "/false") {
			continue
		}

		u := LocalUser{Username: username, UID: uid}

		if uid == 0 {
			rootUser = &u
			continue
		}
		if uid >= 1000 && uid < 65534 {
			users = append(users, u)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if rootUser != nil {
		users = append([]LocalUser{*rootUser}, users...)
	}

	return users, nil
}

// verifyLocalPassword mencocokkan password yang diketik praktikan dengan
// hash yang tersimpan di /etc/shadow untuk user tertentu.
//
// PENTING: verifikasi dilakukan lewat fungsi crypt(3) bawaan SISTEM
// (lewat cgo, package amoghe/go-crypt), BUKAN reimplementasi algoritma
// hash di Go murni. Ini disengaja — Ubuntu 22.04+ memakai algoritma
// "yescrypt" (prefix hash "$y$") sebagai default, dan library Go-murni
// (misal GehirnInc/crypt) tidak mengenali algoritma ini sama sekali,
// bahkan bisa panic saat mencoba. Dengan memanggil crypt(3) milik OS
// langsung, verifikasi otomatis kompatibel dengan algoritma APAPUN yang
// dipakai sistem, tanpa TUI perlu tahu detailnya.
func verifyLocalPassword(username, password string) (ok bool, err error) {
	// Pengaman terakhir: kalau ada apapun yang tidak terduga (termasuk dari
	// binding cgo), jangan sampai keseluruhan proses TUI ikut mati dan
	// memutus sesi SSH praktikan begitu saja.
	defer func() {
		if r := recover(); r != nil {
			ok = false
			err = fmt.Errorf("gagal memverifikasi password: %v", r)
		}
	}()

	f, ferr := os.Open("/etc/shadow")
	if ferr != nil {
		return false, fmt.Errorf("tidak bisa membaca /etc/shadow (butuh akses root): %w", ferr)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 2 || fields[0] != username {
			continue
		}

		hash := fields[1]
		if hash == "" || hash == "*" || hash == "!" || strings.HasPrefix(hash, "!") {
			// Akun tanpa password valid (dikunci/disabled) -> tidak boleh lolos.
			return false, nil
		}

		// Trik standar crypt(3): salt yang dipakai adalah HASH ITU SENDIRI
		// (bukan cuma bagian salt-nya) -- crypt() otomatis membaca prefix
		// algoritma & salt dari situ, lalu mengembalikan hash lengkap yang
		// bisa dibandingkan string-equal dengan hash aslinya.
		computed, err := crypt.Crypt(password, hash)
		if err != nil {
			return false, fmt.Errorf("gagal memverifikasi password (algoritma hash tidak didukung sistem ini): %w", err)
		}

		return computed == hash, nil
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, fmt.Errorf("user %q tidak ditemukan di /etc/shadow", username)
}