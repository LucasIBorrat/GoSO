package main

// MemoryConfig representa la configuración específica del módulo Memoria
type MemoryConfig struct {
	IPMemory       string `json:"IP_MEMORIA"`
	PortMemory     int    `json:"PUERTO_MEMORIA"`
	LogLevel       string `json:"LOG_LEVEL"`
	MemorySize     int    `json:"TAM_MEMORIA"`        // Tamaño de la memoria en bytes
	PageSize       int    `json:"TAM_PAGINA"`         // Tamaño de página en bytes
	NumberOfLevels int    `json:"CANTIDAD_NIVELES"`   // Número de niveles de tabla de páginas
	EntriesPerPage int    `json:"ENTRADAS_POR_TABLA"` // Entradas por página
	MemoryDelay    int    `json:"RETARDO_MEMORIA"`    // Retardo de acceso a memoria
	SwapDelay      int    `json:"RETARDO_SWAP"`       // Retardo de acceso a swap
	SwapfilePath   string `json:"SWAPFILE_PATH"`      // Ruta al archivo de swap
	DumpPath       string `json:"DUMP_PATH"`          // Ruta para los archivos de dump
	ScriptsPath    string `json:"SCRIPTS_PATH"`
}

var config *MemoryConfig
