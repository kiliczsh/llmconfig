package cmd

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/hardware"
	"github.com/spf13/cobra"
)

func newHardwareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hardware",
		Short: "Detect and display system hardware info",
		RunE: func(cmd *cobra.Command, args []string) error {
			hw := hardware.Detect()

			bold := lipgloss.NewStyle().Bold(true)
			label := func(k, v string) {
				fmt.Printf("  %-18s %s\n", bold.Render(k+":"), v)
			}

			fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("System Hardware"))
			fmt.Println()

			label("OS", runtime.GOOS+"/"+runtime.GOARCH)
			label("CPU cores", fmt.Sprintf("%d", hw.CPUCores))

			if hw.GPUName != "" {
				label("GPU", hw.GPUName)
			}
			if hw.VRAMBytes > 0 {
				label("VRAM", humanize.Bytes(hw.VRAMBytes))
			}
			if hw.CUDAVersion != "" {
				label("CUDA", hw.CUDAVersion)
			}
			if hw.ROCmVersion != "" {
				label("ROCm", hw.ROCmVersion)
			}

			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Profile Selection"))
			fmt.Println()

			profileColor := map[string]string{
				"apple_silicon": "5",
				"nvidia":        "2",
				"amd":           "1",
				"intel_gpu":     "6",
				"cpu":           "3",
			}

			profileName := hw.Class.String()
			color := profileColor[profileName]
			if color == "" {
				color = "7"
			}
			styled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(profileName)
			label("Selected profile", styled)
			label("GPU layers default", fmt.Sprintf("%d (auto)", defaultGPULayers(hw)))
			return nil
		},
	}
}

func defaultGPULayers(hw *hardware.DetectionResult) int {
	if hw.Class == hardware.ClassCPU {
		return 0
	}
	return 99
}
