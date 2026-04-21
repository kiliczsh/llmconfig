package hardware

type Class int

const (
	ClassCPU Class = iota
	ClassAppleSilicon
	ClassNVIDIA
	ClassAMD
	ClassIntelGPU
)

func (c Class) String() string {
	switch c {
	case ClassAppleSilicon:
		return "apple_silicon"
	case ClassNVIDIA:
		return "nvidia"
	case ClassAMD:
		return "amd"
	case ClassIntelGPU:
		return "intel_gpu"
	default:
		return "cpu"
	}
}

type DetectionResult struct {
	Class       Class
	GPUName     string
	VRAMBytes   uint64
	RAMBytes    uint64
	CPUCores    int
	IsARM       bool
	CUDAVersion string
	ROCmVersion string
}
