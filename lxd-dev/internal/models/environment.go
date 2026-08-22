package models

import "time"

// EnvironmentDetail adalah gabungan data environment + sesi + ruangan + modul,
// hasil JOIN, persis yang dibutuhkan TUI untuk ditampilkan di dashboard.
type EnvironmentDetail struct {
	ID                string    `json:"-"`
	ContainerName     string    `json:"container_name"`
	CourseCode        string    `json:"course_code"`
	Module            string    `json:"module"`
	Room              string    `json:"room"`
	MeetingNumber     int       `json:"meeting_number"`
	SessionDate       time.Time `json:"session_date"`
	Status            string    `json:"status"`
	AlreadyIdentified bool      `json:"already_identified"`
}

// IdentifyRequest adalah payload yang dikirim TUI saat praktikan submit identitas.
type IdentifyRequest struct {
	Nama string `json:"nama"`
	NPM  string `json:"npm"`
}