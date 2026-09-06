package models

import "time"

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

type IdentifyRequest struct {
	Nama string `json:"nama"`
	NPM  string `json:"npm"`
}