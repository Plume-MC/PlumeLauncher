package skins

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// mustPNG encodes an NRGBA atlas of the given size.
func mustPNG(t *testing.T, width, height int, fill color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateAcceptsModernAtlas(t *testing.T) {
	pngData := mustPNG(t, 64, 64, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	got, err := Validate(pngData)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !bytes.Equal(got, pngData) {
		t.Error("Validate must return the raw bytes untouched")
	}
	if !IsModernAtlas(pngData) {
		t.Error("IsModernAtlas = false for a 64x64 input")
	}
}

func TestValidateAcceptsLegacyAtlas(t *testing.T) {
	pngData := mustPNG(t, 64, 32, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	if _, err := Validate(pngData); err != nil {
		t.Fatalf("Validate legacy 64x32: %v", err)
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":              {},
		"not png":            []byte("hello world"),
		"wrong size 32x32":   mustPNG(t, 32, 32, color.NRGBA{A: 255}),
		"wrong size 128x128": mustPNG(t, 128, 128, color.NRGBA{A: 255}),
		"oversize png":       bytes.Repeat([]byte{0xFF}, MaxPNGBytes+1),
	}
	for name, pngData := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Validate(pngData); err == nil {
				t.Fatalf("Validate(%s) = nil error, want rejection", name)
			}
		})
	}
}

func TestValidateRejectsCorruptIHDR(t *testing.T) {
	pngData := mustPNG(t, 64, 64, color.NRGBA{A: 255})
	pngData[12] = 'X' // not IHDR
	if _, err := Validate(pngData); err == nil {
		t.Fatal("Validate corrupt IHDR = nil error, want rejection")
	}
}

func TestNormalizeLegacyMirrorsRightLimb(t *testing.T) {
	// Legacy 64x32 atlas with a unique color in the left arm region
	// (4,16). After normalization the mirrored pixel must appear at the
	// right arm position (36+11, 16) and the atlas must be 64x64.
	legacy := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	armColor := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	legacy.SetNRGBA(4, 16, armColor) // top-left of the left arm face

	legacyPNG, err := pngEncode(legacy)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := Normalize(legacyPNG)
	if err != nil {
		t.Fatalf("Normalize legacy: %v", err)
	}

	decoded, err := png.Decode(bytes.NewReader(normalized))
	if err != nil {
		t.Fatalf("decode normalized: %v", err)
	}
	if got := decoded.Bounds().Dy(); got != AtlasHeight {
		t.Fatalf("normalized height = %d, want %d", got, AtlasHeight)
	}

	// The mirrored right-arm pixel sits at x=36+12-1-0, y=16.
	mirrored := decoded.At(47, 16)
	if mirrored != armColor {
		t.Errorf("mirrored arm pixel = %v, want %v", mirrored, armColor)
	}
}

func TestNormalizeModernIsRGBA64x64(t *testing.T) {
	src := mustPNG(t, 64, 64, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	normalized, err := Normalize(src)
	if err != nil {
		t.Fatalf("Normalize modern: %v", err)
	}
	decoded, err := png.Decode(bytes.NewReader(normalized))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Bounds().Dx() != AtlasWidth || decoded.Bounds().Dy() != AtlasHeight {
		t.Fatalf("size = %dx%d, want 64x64", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
	if _, _, _, a := decoded.At(0, 0).RGBA(); a != 0xFFFF {
		t.Errorf("alpha = %d, want fully opaque", a)
	}
}

func TestNormalizeRejectsFullyTransparentAtlas(t *testing.T) {
	blank := mustPNG(t, 64, 64, color.NRGBA{A: 0})
	if _, err := Normalize(blank); err != ErrEmptyAtlas {
		t.Fatalf("Normalize blank = %v, want ErrEmptyAtlas", err)
	}
}

func TestNormalizeRejectsTruncatedPNG(t *testing.T) {
	pngData := mustPNG(t, 64, 64, color.NRGBA{A: 255})
	truncated := pngData[:len(pngData)-6] // chop the IEND chunk
	if _, err := Normalize(truncated); err == nil {
		t.Fatal("Normalize truncated = nil error, want rejection")
	}
}

func pngEncode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
