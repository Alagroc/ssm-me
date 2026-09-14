package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var Dir = "/tmp/ssm-me"

type Execution struct {
	ID          string    `json:"id"`
	CommandID   string    `json:"command_id"`
	NodeNames   []string  `json:"node_names"`
	InstanceIDs []string  `json:"instance_ids"`
	Command     string    `json:"command"`
	Comment     string    `json:"comment"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

func Init() error {
	return os.MkdirAll(Dir, 0755)
}

func indexPath() string {
	return filepath.Join(Dir, "executions.json")
}

func Load() ([]Execution, error) {
	data, err := os.ReadFile(indexPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Execution
	return out, json.Unmarshal(data, &out)
}

func Save(e Execution) error {
	execs, err := Load()
	if err != nil {
		return err
	}
	for i, ex := range execs {
		if ex.ID == e.ID {
			execs[i] = e
			return write(execs)
		}
	}
	return write(append(execs, e))
}

func Delete(id string) error {
	execs, err := Load()
	if err != nil {
		return err
	}
	filtered := execs[:0]
	for _, e := range execs {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	if err := write(filtered); err != nil {
		return err
	}
	_ = os.Remove(OutputPath(id))
	return nil
}

func write(execs []Execution) error {
	data, err := json.MarshalIndent(execs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath(), data, 0644)
}

// Settings holds user-configurable preferences, persisted alongside
// execution history so they survive restarts.
type Settings struct {
	Theme       string `json:"theme"`
	DebugLog    bool   `json:"debug_log"`
	AutoRefresh bool   `json:"auto_refresh"`
}

func settingsPath() string {
	return filepath.Join(Dir, "settings.json")
}

func LoadSettings() (Settings, error) {
	data, err := os.ReadFile(settingsPath())
	if os.IsNotExist(err) {
		return Settings{Theme: "default"}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}
	if s.Theme == "" {
		s.Theme = "default"
	}
	return s, nil
}

func SaveSettings(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0644)
}

func OutputPath(id string) string {
	return filepath.Join(Dir, fmt.Sprintf("%s.txt", id))
}

func SaveOutput(id, content string) error {
	return os.WriteFile(OutputPath(id), []byte(content), 0644)
}

func LoadOutput(id string) (string, error) {
	data, err := os.ReadFile(OutputPath(id))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
