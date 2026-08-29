package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Init (bagian dari interface tea.Model) — men-trigger command async pertama
// kali dijalankan: cuma spinner tick, karena layar awal (Menu Utama) tidak
// butuh data apapun dari database.
func (m model) Init() tea.Cmd {
	return m.spin.Tick
}

// errSomeFailed adalah sentinel error dipakai commandDoneMsg untuk menandai
// "sebagian item gagal" pada operasi yang memproses banyak container
// sekaligus — detail lengkapnya selalu ada di teks output, bukan di pesan
// error ini sendiri.
var errSomeFailed = errors.New("sebagian gagal, lihat detail di atas")

func errFromBool(failed bool) error {
	if failed {
		return errSomeFailed
	}
	return nil
}

// ==================== COMMANDS: loader data (read-only) ====================

func loadRoomsCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		rooms, err := ListRooms(context.Background(), db)
		return roomsLoadedMsg{rooms: rooms, err: err}
	}
}

func loadModulesCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		modules, err := ListModules(context.Background(), db)
		return modulesLoadedMsg{modules: modules, err: err}
	}
}

func loadEnvironmentsCmd(db *pgxpool.Pool, room string) tea.Cmd {
	return func() tea.Msg {
		rows, err := ListEnvironmentsForRoom(context.Background(), db, room)
		return environmentsLoadedMsg{rows: rows, err: err}
	}
}

func loadContainersCmd(db *pgxpool.Pool, room string) tea.Cmd {
	return func() tea.Msg {
		names, err := ListContainerNamesForRoom(context.Background(), db, room)
		return containersLoadedMsg{names: names, err: err}
	}
}

func loadRoomsDetailedCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		rooms, err := ListRoomsDetailed(context.Background(), db)
		return roomsDetailedLoadedMsg{rooms: rooms, err: err}
	}
}

func loadSessionsCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		sessions, err := ListSessions(context.Background(), db)
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

// loadSessionsForProvisionCmd HANYA mengambil sesi berstatus scheduled/active
// untuk kombinasi ruangan+modul tertentu — dipakai supaya admin "menempel"
// provisioning ke sesi yang sudah dibuat lewat Kelola Sesi, bukan menciptakan
// sesi baru asal-asalan tiap kali provisioning (perilaku lama yang sudah dibuang).
func loadSessionsForProvisionCmd(db *pgxpool.Pool, room, module string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := ListSessionsForProvision(context.Background(), db, room, module)
		return sessionsForProvisionLoadedMsg{sessions: sessions, err: err}
	}
}

// ==================== COMMANDS: CRUD ruangan & sesi ====================

func createRoomCmd(db *pgxpool.Pool, nama, portPrefix string, capacity int) tea.Cmd {
	return func() tea.Msg {
		err := CreateRoom(context.Background(), db, nama, portPrefix, capacity)
		return roomMutatedMsg{err: err}
	}
}

func updateRoomCmd(db *pgxpool.Pool, originalNama, nama, portPrefix string, capacity int) tea.Cmd {
	return func() tea.Msg {
		err := UpdateRoom(context.Background(), db, originalNama, nama, portPrefix, capacity)
		return roomMutatedMsg{err: err}
	}
}

func deleteRoomCmd(db *pgxpool.Pool, nama string) tea.Cmd {
	return func() tea.Msg {
		err := DeleteRoom(context.Background(), db, nama)
		return roomMutatedMsg{err: err}
	}
}

func createSessionCmd(db *pgxpool.Pool, courseCode, room, module string, meetingNumber int, date, status string) tea.Cmd {
	return func() tea.Msg {
		err := CreateSession(context.Background(), db, courseCode, room, module, meetingNumber, date, status)
		return sessionMutatedMsg{err: err}
	}
}

func updateSessionCmd(db *pgxpool.Pool, id, courseCode string, meetingNumber int, date, status string) tea.Cmd {
	return func() tea.Msg {
		err := UpdateSession(context.Background(), db, id, courseCode, meetingNumber, date, status)
		return sessionMutatedMsg{err: err}
	}
}

func deleteSessionCmd(db *pgxpool.Pool, id string) tea.Cmd {
	return func() tea.Msg {
		err := DeleteSession(context.Background(), db, id)
		return sessionMutatedMsg{err: err}
	}
}

// ==================== COMMANDS: orkestrasi provisioning (BARU) ====================
// Ini menggantikan pemanggilan "kelola-lxd.sh start/stop/reset-room" yang
// lama. Sekarang lxd-control sendiri yang memutuskan nama container, port,
// generate token, DAN menulis ke database — kelola-lxd.sh cuma dipanggil
// per-container untuk operasi LXD teknis murni.

// provisionRoomCmd melakukan provisioning satu ruangan penuh: loop tiap slot
// (1..capacity ruangan), generate token, jalankan kelola-lxd.sh, lalu catat
// ke database. Container yang namanya SUDAH ada di database dilewati
// (idempotent) — sama seperti perilaku kelola-lxd.sh versi lama.
func provisionRoomCmd(cfg *Config, db *pgxpool.Pool, roomNama, moduleCode, sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var output strings.Builder
		failed := false

		mod, err := GetModuleByCode(ctx, db, moduleCode)
		if err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}
		room, err := GetRoomByNama(ctx, db, roomNama)
		if err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}

		existing, err := ListContainerNamesForRoom(ctx, db, roomNama)
		if err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}
		existingSet := make(map[string]bool, len(existing))
		for _, name := range existing {
			existingSet[name] = true
		}

		for slot := 1; slot <= room.Capacity; slot++ {
			containerName := fmt.Sprintf("%s-%02d", roomNama, slot)

			if existingSet[containerName] {
				output.WriteString(fmt.Sprintf("Lewati %s, sudah ada di database.\n", containerName))
				continue
			}

			portStr := fmt.Sprintf("%s%02d", room.PortPrefix, slot)
			port, convErr := strconv.Atoi(portStr)
			if convErr != nil {
				output.WriteString(fmt.Sprintf("%s: port_prefix ruangan tidak valid (%q)\n", containerName, room.PortPrefix))
				failed = true
				continue
			}

			token, hash, tokErr := GenerateToken()
			if tokErr != nil {
				output.WriteString(fmt.Sprintf("%s: gagal generate token: %v\n", containerName, tokErr))
				failed = true
				continue
			}

			if provErr := ProvisionContainer(cfg, mod.MasterContainer, mod.LxdProfile, containerName, port, cfg.APIURL, token); provErr != nil {
				output.WriteString(fmt.Sprintf("%s: GAGAL provisioning LXD — %v\n", containerName, provErr))
				failed = true
				continue
			}

			if dbErr := InsertEnvironment(ctx, db, sessionID, containerName, slot, port, hash); dbErr != nil {
				output.WriteString(fmt.Sprintf("%s: container LXD JADI tapi GAGAL disimpan ke database — %v\n", containerName, dbErr))
				failed = true
				continue
			}

			_ = MarkSnapshotReady(ctx, db, containerName)
			output.WriteString(fmt.Sprintf("%s (port %d): berhasil\n", containerName, port))
		}

		return commandDoneMsg{output: output.String(), err: errFromBool(failed)}
	}
}

// stopRoomCmd menghapus SEMUA container di satu ruangan (LXD + database).
func stopRoomCmd(cfg *Config, db *pgxpool.Pool, roomNama string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		names, err := ListContainerNamesForRoom(ctx, db, roomNama)
		if err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}

		var output strings.Builder
		failed := false

		if len(names) == 0 {
			output.WriteString("Tidak ada container di ruangan ini.\n")
		}

		for _, name := range names {
			if err := DeprovisionContainer(cfg, name); err != nil {
				output.WriteString(fmt.Sprintf("%s: GAGAL dihapus dari LXD — %v\n", name, err))
				failed = true
				continue
			}
			if err := DeleteEnvironmentByContainerName(ctx, db, name); err != nil {
				output.WriteString(fmt.Sprintf("%s: dihapus dari LXD TAPI gagal dihapus dari database — %v\n", name, err))
				failed = true
				continue
			}
			output.WriteString(fmt.Sprintf("%s: berhasil dihapus\n", name))
		}

		return commandDoneMsg{output: output.String(), err: errFromBool(failed)}
	}
}

// resetContainerCmd me-reset SATU container ke snapshot 'clean', DAN
// melepas kaitan identitas praktikan yang sebelumnya login di sini (lihat
// UnlinkPraktikan di db.go) — supaya praktikan berikutnya bisa
// mengidentifikasi diri dari nol, bukan ketolak terus-menerus karena
// identitas lama masih "nempel".
func resetContainerCmd(cfg *Config, db *pgxpool.Pool, containerName string) tea.Cmd {
	return func() tea.Msg {
		if err := ResetContainer(cfg, containerName); err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}

		if err := UnlinkPraktikan(context.Background(), db, containerName); err != nil {
			return commandDoneMsg{
				output: fmt.Sprintf("%s: container berhasil direset TAPI gagal melepas identitas praktikan lama — %v", containerName, err),
				err:    err,
			}
		}

		return commandDoneMsg{output: containerName + ": berhasil direset (identitas praktikan lama juga sudah dilepas).", err: nil}
	}
}

// resetRoomCmd me-reset SEMUA container di satu ruangan, DAN melepas
// kaitan identitas praktikan di masing-masing container (lihat catatan di
// resetContainerCmd di atas).
func resetRoomCmd(cfg *Config, db *pgxpool.Pool, roomNama string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		names, err := ListContainerNamesForRoom(ctx, db, roomNama)
		if err != nil {
			return commandDoneMsg{output: err.Error(), err: err}
		}

		var output strings.Builder
		failed := false

		if len(names) == 0 {
			output.WriteString("Tidak ada container di ruangan ini.\n")
		}

		for _, name := range names {
			if err := ResetContainer(cfg, name); err != nil {
				output.WriteString(fmt.Sprintf("%s: GAGAL — %v\n", name, err))
				failed = true
				continue
			}
			if err := UnlinkPraktikan(ctx, db, name); err != nil {
				output.WriteString(fmt.Sprintf("%s: direset TAPI gagal melepas identitas lama — %v\n", name, err))
				failed = true
				continue
			}
			output.WriteString(fmt.Sprintf("%s: berhasil direset, identitas lama dilepas\n", name))
		}

		return commandDoneMsg{output: output.String(), err: errFromBool(failed)}
	}
}