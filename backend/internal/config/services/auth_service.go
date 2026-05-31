package services

import (
	"os"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/config/common"
)

type AuthModule struct{}

func (AuthModule) Name() string { return "AuthModule" }
func (AuthModule) Section() string {
	return "auth"
}

func init() {
	common.Register(AuthModule{})
}

type AuthSection struct {
	SessionTTLHours   int
	RefreshTTLHours   int
	AllowRegistration bool
	MasterPassword    string
	CookieSecureMode  string
}

func (AuthModule) LoadAndValidate() (AuthSection, error) {
	cookieMode := strings.ToLower(common.GetenvTrim("AUTH_COOKIE_SECURE_MODE"))
	switch cookieMode {
	case "always", "never":
	default:
		cookieMode = ""
	}

	return AuthSection{
		SessionTTLHours:   common.GetPositiveInt("AUTH_SESSION_TTL_HOURS", common.DefaultSessionTTLHours),
		RefreshTTLHours:   common.GetPositiveInt("AUTH_REFRESH_TTL_HOURS", common.DefaultRefreshTTLHours),
		AllowRegistration: os.Getenv("ALLOW_REGISTRATION") == "true",
		MasterPassword:    os.Getenv("MASTER_PASSWORD"),
		CookieSecureMode:  cookieMode,
	}, nil
}
