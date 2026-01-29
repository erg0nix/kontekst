package config

import "time"

type LlamaServerConfig struct {
	Endpoint     string
	BinPath      string
	AutoStart    bool
	InheritStdio bool
	ModelPath    string
	ContextSize  int
	GPULayers    int
	MaxTokens    int
	StartupWait  time.Duration
	HTTPTimeout  time.Duration
}
