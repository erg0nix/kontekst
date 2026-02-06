package config

import "os"

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
	if os.Getenv("KONTEKST_DEBUG_DUMP_ON_ERROR") == "0" {
		cfg.DumpOnError = false
	}
	return cfg
}
