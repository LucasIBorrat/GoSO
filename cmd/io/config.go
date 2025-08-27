package main

// Estructura de configuración para IO
type IOConfig struct {
	IPIO        string `json:"IP_IO"`
	PortIO      int    `json:"PUERTO_IO"`
	IPKernel    string `json:"IP_KERNEL"`
	PortKernel  int    `json:"PUERTO_KERNEL"`
	LogLevel    string `json:"LOG_LEVEL"`
	RetardoBase int    `json:"RETARDO_BASE"`
}

// Variables globales
var (
	config *IOConfig
)