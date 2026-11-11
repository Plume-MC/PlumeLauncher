// Package skins implements Minecraft profile appearance operations
// (POST-003): PNG validation and normalization, the Mojang appearance API
// client, and the local skin closet.
package skins

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

const (
	// MaxPNGBytes caps an uploaded skin. A 64x64 PNG is a few KB; 1 MiB is
	// a generous ceiling that still rejects garbage.
	MaxPNGBytes = 1 << 20
	// AtlasWidth is the only texture width the game accepts. Height is
	// AtlasHeight for the modern layout or 32 for the legacy layout, which
	// is normalized to 64x64 before upload.
	AtlasWidth   = 64
	AtlasHeight  = 64
	legacyHeight = 32
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

// Skin validation errors.
var (
	ErrNotPNG       = errors.New("file is not a PNG image")
	ErrTooLarge     = errors.New("PNG is too large")
	ErrWrongSize    = errors.New("skin must be 64x64 (or legacy 64x32)")
	ErrTruncatedPNG = errors.New("PNG data is truncated or corrupt")
	ErrEmptyAtlas   = errors.New("skin has no opaque pixels")
)

// skinDimensions reads width and height from the IHDR chunk without
// decoding the whole image, so an obviously wrong atlas is rejected early.
func skinDimensions(pngData []byte) (int, int, error) {
	if len(pngData) < 33 || !bytes.HasPrefix(pngData, pngMagic) {
		return 0, 0, ErrNotPNG
	}
	// PNG signature (8) + IHDR length (4) + "IHDR" (4) = 16, then W and H.
	if string(pngData[12:16]) != "IHDR" {
		return 0, 0, ErrNotPNG
	}
	width := int(binary.BigEndian.Uint32(pngData[16:20]))
	height := int(binary.BigEndian.Uint32(pngData[20:24]))
	return width, height, nil
}

// IsModernAtlas reports whether the PNG has the modern 64x64 atlas size.
func IsModernAtlas(pngData []byte) bool {
	width, height, err := skinDimensions(pngData)
	return err == nil && width == AtlasWidth && height == AtlasHeight
}

// Validate checks a skin PNG is a size the game accepts and is small enough
// to upload. Raw bytes are returned untouched; normalization is separate.
func Validate(pngData []byte) ([]byte, error) {
	if len(pngData) == 0 {
		return nil, ErrNotPNG
	}
	if len(pngData) > MaxPNGBytes {
		return nil, ErrTooLarge
	}
	width, height, err := skinDimensions(pngData)
	if err != nil {
		return nil, err
	}
	if width != AtlasWidth || (height != AtlasHeight && height != legacyHeight) {
		return nil, ErrWrongSize
	}
	return pngData, nil
}

// Normalize converts any accepted skin atlas into the modern 64x64 RGBA
// layout. Legacy 64x32 skins get their right arm and right leg mirrored
// from the left ones, which is the transformation the game itself applies.
func Normalize(pngData []byte) ([]byte, error) {
	validated, err := Validate(pngData)
	if err != nil {
		return nil, err
	}
	if !pngComplete(validated) {
		return nil, ErrTruncatedPNG
	}

	decoded, err := png.Decode(bytes.NewReader(validated))
	if err != nil {
		return nil, fmt.Errorf("decode skin: %w", err)
	}

	bounds := decoded.Bounds()
	if bounds.Dx() != AtlasWidth || (bounds.Dy() != AtlasHeight && bounds.Dy() != legacyHeight) {
		return nil, ErrWrongSize
	}

	out := image.NewNRGBA(image.Rect(0, 0, AtlasWidth, AtlasHeight))
	opaque := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			if a != 0 {
				opaque = true
			}
			out.SetNRGBA(x, y, colorFrom(r, g, b, a))
		}
	}
	if !opaque {
		return nil, ErrEmptyAtlas
	}

	if bounds.Dy() == legacyHeight {
		normalizeLegacy(out)
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("encode skin: %w", err)
	}
	return buf.Bytes(), nil
}

// normalizeLegacy mirrors the left arm and left leg regions onto the right
// ones inside a normalized 64x64 atlas, per the classic-to-modern mapping.
// Head and body regions stay untouched.
func normalizeLegacy(atlas *image.NRGBA) {
	// Left arm occupies (4,16)-(15,31); the game maps it to (36,16).
	copyRegionFlipped(atlas, 4, 16, 12, 36, 16)
	// Left leg occupies (4,20)-(19,35) in the legacy strip; maps to (0,32).
	copyRegionFlipped(atlas, 4, 20, 16, 0, 32)
}

// copyRegionFlipped copies a width x height rect from srcX, srcY to dstX,
// dstY with a horizontal flip, matching how the game interprets legacy
// skins. Columns beyond the atlas height are clamped by the image bounds.
func copyRegionFlipped(atlas *image.NRGBA, srcX, srcY, width, dstX, dstY int) {
	height := 16
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			src := atlas.NRGBAAt(srcX+col, srcY+row)
			atlas.SetNRGBA(dstX+width-1-col, dstY+row, src)
		}
	}
}

func colorFrom(r, g, b, a uint32) color.NRGBA {
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// pngComplete checks the PNG ends with an IEND chunk, so truncated uploads
// are rejected before they are shipped to Mojang.
func pngComplete(pngData []byte) bool {
	const iendMagic = "IEND"
	if len(pngData) < 12 {
		return false
	}
	return bytes.Contains(pngData[len(pngData)-12:], []byte(iendMagic))
}
