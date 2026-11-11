package device

// RegisterDeviceRequest registers (or refreshes) a push token for the caller.
type RegisterDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform,omitempty"`
	Locale   string `json:"locale,omitempty"`
}

// UnregisterDeviceRequest removes a push token (e.g. on logout).
type UnregisterDeviceRequest struct {
	Token string `json:"token"`
}

// DeviceToken is a stored push target.
type DeviceToken struct {
	Token    string
	Platform string
	Locale   string
}
