package config

import "os"

// LoadDebugConfigFromEnv overrides debug settings in cfg with values from KONTEKST_DEBUG_* environment variables.
func LoadDebugConfigFromEnv(cfg DebugConfig) DebugConfig {
	if os.Getenv("KONTEKST_DEBUG_LOG_REQUESTS") == "1" {
		cfg.LogRequests = true
	}
	if os.Getenv("KONTEKST_DEBUG_LOG_RESPONSES") == "1" {
		cfg.LogResponses = true
	}
	if os.Getenv("KONTEKST_DEBUG_VALIDATE_ROLES") == "0" {
		cfg.ValidateRoles = false
	}
	return cfg
}
