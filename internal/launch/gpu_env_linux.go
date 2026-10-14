//go:build linux

package launch

import "os"

// nvidiaModulePath is present only when the proprietary NVIDIA driver is loaded.
const nvidiaModulePath = "/sys/module/nvidia"

// gpuEnvironment picks the render-offload variables for the requested GPU.
func gpuEnvironment(preference string) map[string]string {
	return gpuEnvironmentFor(preference, nvidiaLoaded())
}

// gpuEnvironmentFor splits the choice from detection so it can be tested.
//
// DRI_PRIME only reaches Mesa drivers. On a native-Wayland session (e.g.
// Hyprland) the game renders through EGL, where DRI_PRIME=1 points Mesa at the
// NVIDIA node, which Mesa cannot drive, so it falls back to llvmpipe (CPU).
// NVIDIA's own blob needs the PRIME render-offload variables instead.
func gpuEnvironmentFor(preference string, nvidia bool) map[string]string {
	switch preference {
	case "discrete":
		if nvidia {
			return map[string]string{
				"__NV_PRIME_RENDER_OFFLOAD": "1",
				"__VK_LAYER_NV_optimus":     "NVIDIA_only",
				"__GLX_VENDOR_LIBRARY_NAME": "nvidia",
			}
		}
		// AMD/Intel discrete GPU: Mesa drives it, DRI_PRIME is the right knob.
		return map[string]string{"DRI_PRIME": "1"}
	case "integrated":
		return map[string]string{"DRI_PRIME": "0"}
	default:
		return nil
	}
}

func nvidiaLoaded() bool {
	_, err := os.Stat(nvidiaModulePath)
	return err == nil
}
