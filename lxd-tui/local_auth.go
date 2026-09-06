package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/amoghe/go-crypt"
)

type LocalUser struct {
	Username string
	UID      int
}

// listLocalUsers membaca /etc/passwd dan mengembalikan daftar user 
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

func verifyLocalPassword(username, password string) (ok bool, err error) {
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
			return false, nil
		}

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