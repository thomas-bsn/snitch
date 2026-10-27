package workers

import "regexp"

// Machine contains only bounded, non-secret values from the worker container.
type Machine struct {
	Name           *string `json:"name,omitempty"`
	System         *string `json:"system,omitempty"`
	Processor      *string `json:"processor,omitempty"`
	MemoryBytes    *int64  `json:"memory_bytes,omitempty"`
	GPUName        *string `json:"gpu_name,omitempty"`
	GPUMemoryBytes *int64  `json:"gpu_memory_bytes,omitempty"`
	GPUDriver      *string `json:"gpu_driver,omitempty"`
	WorkerVersion  *string `json:"worker_version,omitempty"`
}

var machineValuePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.,:+()/-]{0,127}$`)

func (m Machine) Valid() bool {
	for _, value := range []*string{m.Name, m.System, m.Processor, m.GPUName, m.GPUDriver, m.WorkerVersion} {
		if value != nil && (!machineValuePattern.MatchString(*value) || secretPattern.MatchString(*value)) {
			return false
		}
	}
	const maxMemory = 109951162777600
	for _, value := range []*int64{m.MemoryBytes, m.GPUMemoryBytes} {
		if value != nil && (*value < 1 || *value > maxMemory) {
			return false
		}
	}
	return true
}
