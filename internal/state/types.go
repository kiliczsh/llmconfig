package state

import "time"

type ModelState struct {
	Name        string    `json:"name"`
	PID         int       `json:"pid"`
	Port        int       `json:"port"`
	Host        string    `json:"host"`
	ConfigPath  string    `json:"config_path"`
	LogFile     string    `json:"log_file"`
	StartedAt   time.Time `json:"started_at"`
	ProfileName string    `json:"profile_name"`
	Status      string    `json:"status"` // "running" | "stopped" | "error"
	BinaryPath  string    `json:"binary_path"`
	Backend     string    `json:"backend"` // "llama" | "sd" | "whisper"
}

type StateFile struct {
	Version int                    `json:"version"`
	Models  map[string]*ModelState `json:"models"`
}
