package main

import (
	"sync"
)

// EntradaTabla representa una entrada en una tabla de páginas
type EntradaTabla struct {
	Marco     int  // Número de marco asignado (para el último nivel)
	Presente  bool // Indica si la página está en memoria principal
	Valido    bool // Indica si la entrada es válida
	Direccion int  // Dirección de la siguiente tabla o frame
}

// TablaPaginas representa una tabla de páginas en cualquier nivel
type TablaPaginas struct {
	Entradas []EntradaTabla
	Nivel    int // Nivel de la tabla (1 para primer nivel, etc.)
}

// EntradaSwap representa una entrada en el archivo de SWAP
type EntradaSwap struct {
	PID     int
	Pagina  int
	Offset  int64 // Posición en el archivo SWAP
	Tamanio int
	EnUso   bool
}

// MetricasProceso almacena estadísticas sobre el uso de memoria de un proceso
type MetricasProceso struct {
	AccesosTablasPaginas     int
	InstruccionesSolicitadas int
	BajadasSwap              int
	SubidasMemoria           int
	LecturasMemoria          int
	EscriturasMemoria        int
}

// Variables globales
var memoriaPrincipal []byte
var instruccionesPorProceso map[int][]string
var instruccionesMutex sync.RWMutex
var tablasPaginas map[int]*TablaPaginas     // Mapa de PID a tabla de páginas de primer nivel
var marcosLibres []bool                     // true = libre, false = ocupado
var marcosAsignadosPorProceso map[int][]int // PID -> lista de marcos asignados
var mapaSwap map[string]EntradaSwap         // key: "PID-Pagina"
var swapMutex sync.Mutex                    // Para sincronizar accesos al archivo SWAP
var metricasPorProceso map[int]*MetricasProceso

// Estructuras como EntradaTabla, TablaPaginas, EntradaSwap
// Variables globales como memoriaPrincipal, tablasPaginas.
