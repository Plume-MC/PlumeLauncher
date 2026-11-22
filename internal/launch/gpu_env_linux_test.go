//go:build linux

package launch

import (
	"reflect"
	"testing"
)

func TestGPUEnvironmentFor(t *testing.T) {
	nvidiaDiscrete := map[string]string{
		"__NV_PRIME_RENDER_OFFLOAD": "1",
		"__VK_LAYER_NV_optimus":     "NVIDIA_only",
		"__GLX_VENDOR_LIBRARY_NAME": "nvidia",
	}
	tests := []struct {
		name       string
		preference string
		nvidia     bool
		want       map[string]string
	}{
		{"nvidia discrete uses render offload", "discrete", true, nvidiaDiscrete},
		{"amd/intel discrete keeps DRI_PRIME", "discrete", false, map[string]string{"DRI_PRIME": "1"}},
		{"integrated pins mesa to the iGPU", "integrated", true, map[string]string{"DRI_PRIME": "0"}},
		{"auto leaves the environment alone", "auto", true, nil},
		{"empty leaves the environment alone", "", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gpuEnvironmentFor(tt.preference, tt.nvidia); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gpuEnvironmentFor(%q, %v) = %v, want %v", tt.preference, tt.nvidia, got, tt.want)
			}
		})
	}
}

// The NVIDIA blob is never driven through DRI_PRIME: on native Wayland that
// makes Mesa fail to open the NVIDIA node and drop to llvmpipe.
func TestGPUEnvironmentNeverCombinesPrimeAndOffload(t *testing.T) {
	for _, preference := range []string{"discrete", "integrated"} {
		for _, nvidia := range []bool{true, false} {
			env := gpuEnvironmentFor(preference, nvidia)
			if _, offload := env["__NV_PRIME_RENDER_OFFLOAD"]; offload {
				if _, prime := env["DRI_PRIME"]; prime {
					t.Errorf("preference %q nvidia=%v sets both DRI_PRIME and NV offload", preference, nvidia)
				}
			}
		}
	}
}
