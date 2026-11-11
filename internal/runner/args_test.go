package runner

import (
	"testing"

	"github.com/kiliczsh/llmconfig/internal/config"
	"gopkg.in/yaml.v3"
)

func TestBuildArgsServerParallel(t *testing.T) {
	for _, tc := range []struct {
		name, backend, serverYAML, want string
	}{
		{"llama omitted", "llama", "server: {}", ""},
		{"llama zero", "llama", "server: {parallel: 0}", ""},
		{"llama one", "llama", "server: {parallel: 1}", "1"},
		{"llama four", "llama", "server: {parallel: 4}", "4"},
		{"ik omitted", "ik_llama", "server: {}", ""},
		{"ik one", "ik_llama", "server: {parallel: 1}", "1"},
		{"prism omitted", "prism_llama", "server: {}", "1"},
		{"prism four", "prism_llama", "server: {parallel: 4}", "4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Backend: tc.backend}
			if err := yaml.Unmarshal([]byte(tc.serverYAML), cfg); err != nil {
				t.Fatal(err)
			}
			config.ApplyDefaults(cfg)
			args := BuildArgs(&config.RunConfig{
				Config:  cfg,
				Backend: tc.backend,
				Profile: cfg.HardwareProfiles.NVIDIA,
			})
			var parallel string
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "--parallel" {
					parallel = args[i+1]
					break
				}
			}
			if parallel != tc.want {
				t.Fatalf("--parallel = %q, want %q (empty means omitted)", parallel, tc.want)
			}
		})
	}
}
