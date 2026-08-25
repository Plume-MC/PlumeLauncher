//go:build windows

package launch

import "golang.org/x/sys/windows/registry"

// ApplyGPUPreference asks Windows to apply the selected preference to Java.
func ApplyGPUPreference(javaPath, preference string) error {
	keyPath := `Software\Microsoft\DirectX\UserGpuPreferences`
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		key, _, err = registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
		if err != nil {
			return err
		}
	}
	defer key.Close()
	return key.SetStringValue(javaPath, WindowsGPUPreferenceValue(preference))
}
