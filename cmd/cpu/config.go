package main

type CPUConfig struct {
	PortCPU          int    `json:"PUERTO_CPU"`
	IPCPU            string `json:"IP_CPU"`
	IPMemory         string `json:"IP_MEMORIA"`
	PortMemory       int    `json:"PUERTO_MEMORIA"`
	IPKernel         string `json:"IP_KERNEL"`
	PortKernel       int    `json:"PUERTO_KERNEL"`
	TLBEntries       int    `json:"ENTRADAS_TLB"`
	TLBReplacement   string `json:"REEMPLAZO_TLB"`
	CacheEntries     int    `json:"ENTRADAS_CACHE"`
	CacheReplacement string `json:"REEMPLAZO_CACHE"`
	CacheDelay       int    `json:"RETARDO_CACHE"`
	LogLevel         string `json:"LOG_LEVEL"`
}

var config *CPUConfig
