package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Run struct {
	PromptTPS   float64   `json:"prompt_tps"`
	GenerateTPS float64   `json:"generate_tps"`
	Timestamp   time.Time `json:"timestamp"`
}

type Result struct {
	Name           string  `json:"name"`
	Runs           []Run   `json:"runs"`
	AvgPromptTPS   float64 `json:"avg_prompt_tps"`
	AvgGenerateTPS float64 `json:"avg_generate_tps"`
}

func Save(benchDir string, r *Result) error {
	if err := os.MkdirAll(benchDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(benchDir, r.Name+".json"), data, 0644)
}

func Load(benchDir, name string) (*Result, error) {
	data, err := os.ReadFile(filepath.Join(benchDir, name+".json"))
	if err != nil {
		return nil, err
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func Avg(runs []Run) (promptTPS, generateTPS float64) {
	if len(runs) == 0 {
		return
	}
	for _, r := range runs {
		promptTPS += r.PromptTPS
		generateTPS += r.GenerateTPS
	}
	n := float64(len(runs))
	return promptTPS / n, generateTPS / n
}
