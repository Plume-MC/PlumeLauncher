package services

import (
	"errors"
	"fmt"
	"os"
	"time"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/skins"
)

// skinChangeWindow is how fast appearance writes are allowed to repeat.
// Mojang auto-suspends accounts that hammer the skin endpoints with 429s,
// so writes are debounced locally for the same reason Modrinth debounces.
const skinChangeWindow = 10 * time.Second

// SkinService owns profile appearance operations for Microsoft accounts:
// the Mojang client, the local closet, and the write debouncer.
type SkinService struct {
	DataRoot  string
	Keyring   auth.Keyring
	Mojang    *skins.Client
	Debouncer *SkinDebouncer
}

func NewSkinService(dataRoot string) *SkinService {
	return &SkinService{
		DataRoot:  dataRoot,
		Mojang:    &skins.Client{},
		Debouncer: NewSkinDebouncer(skinChangeWindow),
	}
}

// Appearance is the merged view shown in the dialog: the Mojang profile
// plus the local closet.
type Appearance struct {
	Profile    *skins.Profile `json:"profile"`
	LocalSkins []ClosetSkin   `json:"localSkins"`
}

// accountService builds an AccountService sharing this service's data root
// and keyring, so token resolution follows the same path as launch.
func (s *SkinService) accountService() *AccountService {
	return &AccountService{DataRoot: s.DataRoot, Keyring: s.Keyring}
}

// accountToken resolves the Minecraft bearer token for one account,
// refreshing through the standard launch token path.
func (s *SkinService) accountToken(accountUUID string) (string, error) {
	svc := s.accountService()
	account, err := svc.microsoftAccount(accountUUID)
	if err != nil {
		return "", err
	}
	_, token, err := svc.microsoftLaunchToken(account)
	return token, err
}

func (s *SkinService) requireMicrosoft(accountUUID string) error {
	svc := s.accountService()
	account, err := svc.microsoftAccount(accountUUID)
	if err != nil {
		return NewValidationError("Skin and cape editing needs a Microsoft account.")
	}
	if account.Type != AccountTypeMicrosoft {
		return NewValidationError("Skin and cape editing needs a Microsoft account.")
	}
	return nil
}

// GetProfileAppearance returns the merged Mojang profile and local closet.
func (s *SkinService) GetProfileAppearance(accountUUID string) (*Appearance, error) {
	if err := s.requireMicrosoft(accountUUID); err != nil {
		return nil, err
	}
	token, err := s.accountToken(accountUUID)
	if err != nil {
		return nil, err
	}
	profile, err := s.Mojang.FetchProfile(token)
	if err != nil {
		return nil, appearanceError(err)
	}
	local, err := s.ListLocalSkins()
	if err != nil {
		return nil, err
	}
	return &Appearance{Profile: profile, LocalSkins: local}, nil
}
// UploadSkin validates and normalizes the PNG, saves it to the closet, then
// posts it to Mojang. The closet save happens first so a Mojang failure does
// not lose the user's work (the Modrinth pattern).
func (s *SkinService) UploadSkin(accountUUID string, pngData []byte, variant skins.SkinVariant) (*Appearance, error) {
	if err := s.requireMicrosoft(accountUUID); err != nil {
		return nil, err
	}
	if !s.Debouncer.Acquire() {
		return nil, NewConflictError(fmt.Sprintf(
			"Please wait %d seconds between skin changes; Minecraft rate-limits these.",
			int(skinChangeWindow.Seconds()),
		))
	}

	normalized, err := skins.Normalize(pngData)
	if err != nil {
		return nil, NewValidationError(validationMessage(err))
	}
	if _, err := s.SaveLocalSkin(normalized, variant); err != nil {
		return nil, err
	}

	token, err := s.accountToken(accountUUID)
	if err != nil {
		return nil, err
	}
	profile, err := s.Mojang.UploadSkin(token, normalized, variant)
	if err != nil {
		if skins.IsTokenExpired(err) {
			return nil, NewUpstreamError("Minecraft session expired. Sign in again, then try once more.")
		}
		return nil, appearanceError(err)
	}
	return s.merge(profile), nil
}

// ResetSkin restores the default look.
func (s *SkinService) ResetSkin(accountUUID string) (*Appearance, error) {
	if err := s.requireMicrosoft(accountUUID); err != nil {
		return nil, err
	}
	if !s.Debouncer.Acquire() {
		return nil, NewConflictError(fmt.Sprintf(
			"Please wait %d seconds between skin changes; Minecraft rate-limits these.",
			int(skinChangeWindow.Seconds()),
		))
	}
	token, err := s.accountToken(accountUUID)
	if err != nil {
		return nil, err
	}
	profile, err := s.Mojang.ResetSkin(token)
	if err != nil {
		if skins.IsTokenExpired(err) {
			return nil, NewUpstreamError("Minecraft session expired. Sign in again, then try once more.")
		}
		return nil, appearanceError(err)
	}
	return s.merge(profile), nil
}

// EquipCape activates one owned cape.
func (s *SkinService) EquipCape(accountUUID, capeID string) (*Appearance, error) {
	if err := s.requireMicrosoft(accountUUID); err != nil {
		return nil, err
	}
	if capeID == "" {
		return nil, NewValidationError("Pick a cape to equip.")
	}
	if !s.Debouncer.Acquire() {
		return nil, NewConflictError(fmt.Sprintf(
			"Please wait %d seconds between skin changes; Minecraft rate-limits these.",
			int(skinChangeWindow.Seconds()),
		))
	}
	token, err := s.accountToken(accountUUID)
	if err != nil {
		return nil, err
	}
	profile, err := s.Mojang.EquipCape(token, capeID)
	if err != nil {
		if skins.IsTokenExpired(err) {
			return nil, NewUpstreamError("Minecraft session expired. Sign in again, then try once more.")
		}
		return nil, appearanceError(err)
	}
	return s.merge(profile), nil
}

// HideCape removes the active cape.
func (s *SkinService) HideCape(accountUUID string) (*Appearance, error) {
	if err := s.requireMicrosoft(accountUUID); err != nil {
		return nil, err
	}
	if !s.Debouncer.Acquire() {
		return nil, NewConflictError(fmt.Sprintf(
			"Please wait %d seconds between skin changes; Minecraft rate-limits these.",
			int(skinChangeWindow.Seconds()),
		))
	}
	token, err := s.accountToken(accountUUID)
	if err != nil {
		return nil, err
	}
	profile, err := s.Mojang.HideCape(token)
	if err != nil {
		if skins.IsTokenExpired(err) {
			return nil, NewUpstreamError("Minecraft session expired. Sign in again, then try once more.")
		}
		return nil, appearanceError(err)
	}
	return s.merge(profile), nil
}

// ListLocalSkins exposes the closet to the dialog.
func (s *SkinService) ListLocalSkins() ([]ClosetSkin, error) {
	return closetList(s.DataRoot)
}

// LoadLocalSkinPNG exposes one closet entry's normalized bytes.
func (s *SkinService) LoadLocalSkinPNG(textureKey string) ([]byte, error) {
	pngData, err := closetLoad(s.DataRoot, textureKey)
	if err != nil {
		return nil, NewNotFoundError("Skin not found.")
	}
	return pngData, nil
}

// DeleteLocalSkin removes one closet entry.
func (s *SkinService) DeleteLocalSkin(textureKey string) error {
	if err := closetDelete(s.DataRoot, textureKey); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewNotFoundError("Skin not found.")
		}
		return NewInternalError("Unable to update the skin closet.")
	}
	return nil
}

// SaveLocalSkin stores normalized PNG bytes in the closet.
func (s *SkinService) SaveLocalSkin(pngData []byte, variant skins.SkinVariant) (string, error) {
	key, err := closetSave(s.DataRoot, pngData, string(variant))
	if err != nil {
		return "", NewInternalError("Unable to update the skin closet.")
	}
	return key, nil
}

// merge builds the Appearance response for a profile Mojang just returned.
func (s *SkinService) merge(profile *skins.Profile) *Appearance {
	local, err := s.ListLocalSkins()
	if err != nil {
		local = []ClosetSkin{}
	}
	return &Appearance{Profile: profile, LocalSkins: local}
}

// appearanceError maps Mojang errors to the launcher's user-facing taxonomy.
func appearanceError(err error) *ServiceError {
	if skins.IsInvalidImage(err) {
		return NewValidationError("Minecraft rejected that image. Use a 64x64 skin PNG.")
	}
	if skins.IsTokenExpired(err) {
		return NewUpstreamError("Minecraft session expired. Sign in again, then try once more.")
	}
	return NewUpstreamError("Minecraft did not accept the change. Try again in a moment.")
}

// validationMessage converts PNG validation errors to user-facing copy.
func validationMessage(err error) string {
	switch err {
	case skins.ErrNotPNG:
		return "That file is not a PNG image."
	case skins.ErrTooLarge:
		return "That PNG is too large. Skin files are small; re-export the image."
	case skins.ErrWrongSize:
		return "Skins must be 64x64 pixels (or the legacy 64x32 layout)."
	case skins.ErrTruncatedPNG:
		return "That PNG is truncated or corrupt."
	case skins.ErrEmptyAtlas:
		return "That skin is fully transparent; draw something first."
	default:
		return "That skin could not be processed. Try another PNG."
	}
}