package presence

// Activity is the SET_ACTIVITY activity object. Only secret-free,
// user-gated fields are populated (POST-004 boundaries: no OAuth, no
// runtime image upload, no party/join/voice).
type Activity struct {
	PID        int       `json:"-"`
	Details    string    `json:"details,omitempty"`
	State      string    `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets   `json:"assets,omitempty"`
}

// Timestamps carries the elapsed-time clock.
type Timestamps struct {
	Start int64 `json:"start,omitempty"` // unix seconds
}

// Assets references image keys uploaded to the Discord application's
// Art Assets section. Unknown keys render as empty boxes, so the keys
// must match the uploaded asset names exactly.
type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}
