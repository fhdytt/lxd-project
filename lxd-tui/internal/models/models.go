package models

// EnvInfo data environment dari API
type EnvInfo struct {
	CourseCode        string `json:"course_code"`
	Module            string `json:"module"`
	Room              string `json:"room"`
	MeetingNumber     int    `json:"meeting_number"`
	SessionDate       string `json:"session_date"`
	Status            string `json:"status"`
	AlreadyIdentified bool   `json:"already_identified"`
	PraktikanID       *int64 `json:"praktikan_id"`
}

// LocalUser untuk daftar user di sistem Linux
type LocalUser struct {
	Username string
	UID      string
	GID      string
	Home     string
	Shell    string
}

// IdentifyRequest payload untuk verifikasi API
type IdentifyRequest struct {
	Nama string `json:"nama"`
	NPM  string `json:"npm"`
}

// APIResponse wrapper untuk semua response API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
