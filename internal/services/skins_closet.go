package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"plumelauncher/internal/storage"
)

// closet layout under the data root. Atomic index writes reuse the shared
// storage writer, so a crash leaves either the old or the new index.
const (
	closetDirName      = "closet"
	closetIndexName    = "index.json"
	closetSkinsDirName = "skins"
)

// ClosetSkin is one saved skin in the local closet.
type ClosetSkin struct {
	TextureKey string    `json:"textureKey"`
	Variant    string    `json:"variant"`
	SavedAt    time.Time `json:"savedAt"`
	FileName   string    `json:"fileName"`
}

type closetIndex struct {
	storage.Document
	Skins []ClosetSkin `json:"skins"`
}

// closetPaths resolves the closet layout under a data root.
type closetPaths struct {
	root      string
	indexFile string
	skinsDir  string
}

func newClosetPaths(dataRoot string) closetPaths {
	root := filepath.Join(dataRoot, closetDirName)
	return closetPaths{
		root:      root,
		indexFile: filepath.Join(root, closetIndexName),
		skinsDir:  filepath.Join(root, closetSkinsDirName),
	}
}

// closetLoadIndex reads the closet index. A missing or empty file is an empty
// closet, not a failure.
func closetLoadIndex(paths closetPaths) (closetIndex, error) {
	var index closetIndex
	err := storage.ReadJSON(paths.indexFile, &index)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "empty file") {
			return closetIndex{Document: storage.Document{SchemaVersion: 1}}, nil
		}
		return closetIndex{}, err
	}
	return index, nil
}

// closetSaveIndex writes the closet index atomically.
func closetSaveIndex(paths closetPaths, index closetIndex) error {
	if err := os.MkdirAll(paths.root, 0o755); err != nil {
		return err
	}
	return storage.WriteJSON(paths.indexFile, index)
}

// closetTextureKey hashes normalized PNG bytes; the same content never saves
// twice.
func closetTextureKey(pngData []byte) string {
	digest := sha256.Sum256(pngData)
	return hex.EncodeToString(digest[:])
}

// closetSave stores normalized PNG bytes in the closet and returns the
// texture key. Same-content saves are idempotent. The closet belongs to the
// launcher rather than one account, so entries survive account switches.
func closetSave(dataRoot string, pngData []byte, variant string) (string, error) {
	paths := newClosetPaths(dataRoot)
	index, err := closetLoadIndex(paths)
	if err != nil {
		return "", err
	}

	key := closetTextureKey(pngData)
	fileName := key + ".png"
	if err := os.MkdirAll(paths.skinsDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(paths.skinsDir, fileName), pngData, 0o644); err != nil {
		return "", err
	}

	for _, existing := range index.Skins {
		if existing.TextureKey == key {
			return key, nil
		}
	}
	index.Skins = append(index.Skins, ClosetSkin{
		TextureKey: key,
		Variant:    variant,
		SavedAt:    time.Now(),
		FileName:   fileName,
	})
	if err := closetSaveIndex(paths, index); err != nil {
		return "", err
	}
	return key, nil
}

// closetList returns every saved skin, newest first.
func closetList(dataRoot string) ([]ClosetSkin, error) {
	paths := newClosetPaths(dataRoot)
	index, err := closetLoadIndex(paths)
	if err != nil {
		return nil, err
	}
	skins := index.Skins
	for i, j := 0, len(skins)-1; i < j; i, j = i+1, j-1 {
		skins[i], skins[j] = skins[j], skins[i]
	}
	return skins, nil
}

// closetLoad reads one saved skin's normalized bytes.
func closetLoad(dataRoot, textureKey string) ([]byte, error) {
	if textureKey == "" {
		return nil, os.ErrNotExist
	}
	paths := newClosetPaths(dataRoot)
	index, err := closetLoadIndex(paths)
	if err != nil {
		return nil, err
	}
	for _, entry := range index.Skins {
		if entry.TextureKey == textureKey {
			return os.ReadFile(filepath.Join(paths.skinsDir, textureKey+".png"))
		}
	}
	return nil, os.ErrNotExist
}

// closetDelete removes one closet entry and its PNG.
func closetDelete(dataRoot, textureKey string) error {
	paths := newClosetPaths(dataRoot)
	index, err := closetLoadIndex(paths)
	if err != nil {
		return err
	}
	kept := index.Skins[:0]
	found := false
	for _, entry := range index.Skins {
		if entry.TextureKey == textureKey {
			found = true
			continue
		}
		kept = append(kept, entry)
	}
	if !found {
		return os.ErrNotExist
	}
	index.Skins = kept
	if err := closetSaveIndex(paths, index); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(paths.skinsDir, textureKey+".png"))
	return nil
}

// SkinDebouncer guards Mojang write operations. Mojang auto-suspends
// accounts that trigger repeated 429s on skin uploads, so rapid clicking
// must be rejected locally before it ever reaches the API.
type SkinDebouncer struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

func NewSkinDebouncer(interval time.Duration) *SkinDebouncer {
	return &SkinDebouncer{interval: interval}
}

// Acquire reports whether enough time has passed since the last accepted
// write. Rejections do not move the window forward.
func (d *SkinDebouncer) Acquire() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if time.Since(d.last) < d.interval {
		return false
	}
	d.last = time.Now()
	return true
}

// Reset clears the window, used by tests.
func (d *SkinDebouncer) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.last = time.Time{}
}