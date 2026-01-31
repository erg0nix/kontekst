package config

import "time"

type LlamaServerConfig struct {
	Endpoint     string
	BinPath      string
	AutoStart    bool
	InheritStdio bool
	ModelDir     string
	ContextSize  int
	GPULayers    int
	StartupWait  time.Duration
	HTTPTimeout  time.Duration
}
