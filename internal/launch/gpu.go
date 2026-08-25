package launch

// WindowsGPUPreferenceValue returns the DirectX per-app preference value.
func WindowsGPUPreferenceValue(preference string) string {
	switch preference {
	case "integrated":
		return "GpuPreference=1;"
	case "discrete":
		return "GpuPreference=2;"
	default:
		return "GpuPreference=0;"
	}
}
