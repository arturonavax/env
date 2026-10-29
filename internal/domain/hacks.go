package domain

// HardwareMatcher defines hardware, GPU, OS, and desktop environment predicates for machine-specific hacks.
type HardwareMatcher struct {
	OS            []string `json:"os,omitempty" yaml:"os,omitempty"`
	Distros       []string `json:"distros,omitempty" yaml:"distros,omitempty"`
	DMIProduct    string   `json:"dmi_product,omitempty" yaml:"dmi_product,omitempty"`
	DMIVendor     string   `json:"dmi_vendor,omitempty" yaml:"dmi_vendor,omitempty"`
	GPUVendor     string   `json:"gpu_vendor,omitempty" yaml:"gpu_vendor,omitempty"`
	GPUDriver     string   `json:"gpu_driver,omitempty" yaml:"gpu_driver,omitempty"`
	GPUModel      string   `json:"gpu_model,omitempty" yaml:"gpu_model,omitempty"`
	DesktopEnv    []string `json:"desktop_env,omitempty" yaml:"desktop_env,omitempty"`
	DisplayServer []string `json:"display_server,omitempty" yaml:"display_server,omitempty"`
}

// HackStep describes one discrete optimization or workaround inside a HackProfile.
type HackStep struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Command     []string `json:"command,omitempty" yaml:"command,omitempty"`
	Script      string   `json:"script,omitempty" yaml:"script,omitempty"`
	Args        []string `json:"args,omitempty" yaml:"args,omitempty"`
	Touches     []string `json:"touches,omitempty" yaml:"touches,omitempty"`
	Privileged  bool     `json:"privileged,omitempty" yaml:"privileged,omitempty"`
}

// HackProfile represents a declarative machine- or hardware-specific optimization bundle under hacks/<id>/.
type HackProfile struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Category    string          `json:"category" yaml:"category"`
	Description string          `json:"description" yaml:"description"`
	SafeByOptIn bool            `json:"safe_by_opt_in" yaml:"safe_by_opt_in"`
	Match       HardwareMatcher `json:"match" yaml:"match"`
	BackupPaths []string        `json:"backup_paths,omitempty" yaml:"backup_paths,omitempty"`
	Steps       []HackStep      `json:"steps" yaml:"steps"`
	Dir         string          `json:"dir,omitempty" yaml:"-"`
}

// HardwareProfile captures probed host hardware and display metadata used to match hacks and run diagnostics.
type HardwareProfile struct {
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
	Distro        string   `json:"distro"`
	Kernel        string   `json:"kernel"`
	DMIVendor     string   `json:"dmi_vendor"`
	DMIProduct    string   `json:"dmi_product"`
	CPUModel      string   `json:"cpu_model"`
	CPUCores      int      `json:"cpu_cores"`
	MemoryTotalMB int64    `json:"memory_total_mb"`
	GPUModels     []string `json:"gpu_models"`
	GPUDrivers    []string `json:"gpu_drivers"`
	DesktopEnv    string   `json:"desktop_env"`
	DisplayServer string   `json:"display_server"`
}
