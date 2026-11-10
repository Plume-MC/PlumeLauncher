package skins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Mojang appearance API endpoints, all under api.minecraftservices.com and
// authenticated with the account's Minecraft bearer token.
const (
	mojangBaseURL        = "https://api.minecraftservices.com"
	mojangProfilePath    = "/minecraft/profile"
	mojangSkinsPath      = "/minecraft/profile/skins"
	mojangActiveSkinPath = "/minecraft/profile/skins/active"
	mojangCapesPath      = "/minecraft/profile/capes/active"
	mojangUserAgent      = "PlumeLauncher/1.2 (+https://github.com/Plume-MC/PlumeLauncher)"
)

// SkinVariant is the arm model a skin is built for.
type SkinVariant string

const (
	VariantClassic SkinVariant = "classic"
	VariantSlim    SkinVariant = "slim"
)

// SkinVariantFromProfile converts the profile's CLASSIC/SLIM casing into the
// upload API's lowercase form. Unknown values map to classic, matching how
// the game treats a missing model metadata as Steve.
func SkinVariantFromProfile(variant string) SkinVariant {
	switch variant {
	case "SLIM":
		return VariantSlim
	default:
		return VariantClassic
	}
}

// ProfileSkin is one entry of the profile's skins list.
type ProfileSkin struct {
	ID      string `json:"id"`
	State   string `json:"state"` // ACTIVE or INACTIVE
	URL     string `json:"url"`
	Variant string `json:"variant"` // CLASSIC or SLIM
}

// ProfileCape is one entry of the profile's capes list.
type ProfileCape struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url"`
	Alias string `json:"alias"`
}

// Profile is the player appearance profile Mojang returns for every
// successful skin or cape operation, so callers can refresh in one step.
type Profile struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Skins []ProfileSkin `json:"skins"`
	Capes []ProfileCape `json:"capes"`
}

// ActiveSkin returns the profile's equipped skin, or nil when none.
func (p *Profile) ActiveSkin() *ProfileSkin {
	for i := range p.Skins {
		if p.Skins[i].State == "ACTIVE" {
			return &p.Skins[i]
		}
	}
	return nil
}

// MojangError carries the structured error body Mojang answers with.
type MojangError struct {
	Status       int    `json:"-"`
	Type         string `json:"error"`
	ErrorMessage string `json:"errorMessage"`
}

func (e *MojangError) Error() string {
	return fmt.Sprintf("mojang %s: %s", e.Type, e.ErrorMessage)
}

// IsInvalidImage reports whether Mojang rejected the texture itself.
func IsInvalidImage(err error) bool {
	var mojangErr *MojangError
	if !asMojangError(err, &mojangErr) {
		return false
	}
	return mojangErr.Status == 415
}

// IsTokenExpired reports whether the bearer token was rejected, which the
// caller answers by refreshing once and retrying.
func IsTokenExpired(err error) bool {
	var mojangErr *MojangError
	if !asMojangError(err, &mojangErr) {
		return false
	}
	return mojangErr.Status == 401 || mojangErr.Status == 403
}

func asMojangError(err error, target **MojangError) bool {
	mojangErr, ok := err.(*MojangError)
	if ok {
		*target = mojangErr
		return true
	}
	return false
}

// Client calls the Mojang appearance API. HTTPClient is injectable so tests
// point it at an httptest server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

func newClientHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return newClientHTTPClient()
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return mojangBaseURL
}

func (c *Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return mojangUserAgent
}

// do sends one authenticated request and decodes either the profile response
// or the structured error body.
func (c *Client) do(method, path, token string, body io.Reader, contentType string) (*Profile, error) {
	request, err := http.NewRequest(method, c.baseURL()+path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", c.userAgent())
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient().Do(request)
	if err != nil {
		return nil, fmt.Errorf("mojang request: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, MaxPNGBytes*4))
	if err != nil {
		return nil, fmt.Errorf("read mojang response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		mojangErr := &MojangError{Status: response.StatusCode}
		if jsonErr := json.Unmarshal(payload, mojangErr); jsonErr != nil || mojangErr.Type == "" {
			return nil, &MojangError{
				Status:       response.StatusCode,
				Type:         "HTTPError",
				ErrorMessage: fmt.Sprintf("HTTP %d", response.StatusCode),
			}
		}
		return nil, mojangErr
	}

	var profile Profile
	if err := json.Unmarshal(payload, &profile); err != nil {
		// Some successes return an empty body; treat those as unknown.
		return nil, fmt.Errorf("decode mojang profile: %w", err)
	}
	return &profile, nil
}

// FetchProfile returns the full appearance profile.
func (c *Client) FetchProfile(token string) (*Profile, error) {
	return c.do(http.MethodGet, mojangProfilePath, token, nil, "")
}

// UploadSkin posts a normalized 64x64 PNG with the chosen arm variant and
// returns the refreshed profile.
func (c *Client) UploadSkin(token string, pngData []byte, variant SkinVariant) (*Profile, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("variant", string(variant)); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("file", "skin.png")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(pngData); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return c.do(http.MethodPost, mojangSkinsPath, token, &body, writer.FormDataContentType())
}

// ResetSkin removes the active custom skin, restoring the default look.
func (c *Client) ResetSkin(token string) (*Profile, error) {
	return c.do(http.MethodDelete, mojangActiveSkinPath, token, nil, "")
}

// EquipCape activates one owned cape.
func (c *Client) EquipCape(token, capeID string) (*Profile, error) {
	payload, err := json.Marshal(map[string]string{"capeId": capeID})
	if err != nil {
		return nil, err
	}
	return c.do(http.MethodPut, mojangCapesPath, token, bytes.NewReader(payload), "application/json")
}

// HideCape removes the active cape.
func (c *Client) HideCape(token string) (*Profile, error) {
	return c.do(http.MethodDelete, mojangCapesPath, token, nil, "")
}
