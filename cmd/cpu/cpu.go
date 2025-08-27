package main

import (
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Estructuras para ciclo de CPU
type TLBEntry struct {
	PageNumber  int
	FrameNumber int
	PID         int
	LoadTime    int64 // Para FIFO
	LastUsed    int64 // Para LRU
}

type CacheEntry struct {
	PageNumber  int
	FrameNumber int
	Content     string
	PID         int
	Modified    bool // Para algoritmo CLOCK-M
	Referenced  bool // Para algoritmo CLOCK
}

// Variables globales para CPU
var (
	tlbEntries            []TLBEntry
	cacheEntries          []CacheEntry
	tlbCounter            int64 = 0
	clockPointer          int   = 0
	mutex                 sync.Mutex
	interrupcionPendiente bool
	pidInterrumpido       int
	procesoEnEjecucion    int = -1 // PID del proceso actualmente en ejecución
)

// Inicializar componentes de la CPU
func inicializarCPU() {
	// Cargar configuración de memoria desde archivo
	err := cargarConfigMemoria()
	if err != nil {
		utils.ErrorLog.Error("Error al cargar configuración de memoria", "error", err)
		// Usar valores por defecto
		tamanoPagina = 64
		entradasPorTabla = 4
		numeroDeNiveles = 5
		utils.InfoLog.Info("Usando configuración por defecto",
			"page_size", tamanoPagina,
			"entries_per_page", entradasPorTabla,
			"number_of_levels", numeroDeNiveles)
	}

	// Inicializar TLB y Cache
	inicializarTLB()
	inicializarCache()

	utils.InfoLog.Info("CPU inicializada correctamente")
}

// Inicializar TLB según configuración
func inicializarTLB() {
	if config.TLBEntries > 0 {
		tlbEntries = make([]TLBEntry, config.TLBEntries)
		for i := range tlbEntries {
			tlbEntries[i] = TLBEntry{
				PageNumber:  -1,
				FrameNumber: -1,
				PID:         -1,
			}
		}
		utils.InfoLog.Info("TLB inicializada", "entradas", config.TLBEntries, "algoritmo", config.TLBReplacement)
	} else {
		utils.InfoLog.Info("TLB deshabilitada")
	}
}

// Inicializar Cache según configuración
func inicializarCache() {
	if config.CacheEntries > 0 {
		cacheEntries = make([]CacheEntry, config.CacheEntries)
		for i := range cacheEntries {
			cacheEntries[i] = CacheEntry{
				PageNumber: -1,
				Content:    "",
				PID:        -1,
				Modified:   false,
				Referenced: false,
			}
		}
		utils.InfoLog.Info("Cache inicializada", "entradas", config.CacheEntries, "algoritmo", config.CacheReplacement)
	} else {
		utils.InfoLog.Info("Cache deshabilitada")
	}
}

// Implementar ciclo de instrucción completo
func ejecutarCiclo(pid, pc int) (int, string, map[string]interface{}) {
	procesoEnEjecucion = pid

	// Fetch
	instruccion := fetch(pid, pc)
	if instruccion == "" {
		return pc, "ERROR", nil
	}

	// Decode y Execute
	siguientePC, motivo, parametrosSyscall := decodeAndExecute(pid, pc, instruccion)

	// Check Interrupt
	if checkInterrupt(pid) {
		limpiarEstructurasPorPID(pid)
		procesoEnEjecucion = -1
		return siguientePC, "INTERRUPTED", nil
	}

	// Si el PC no fue modificado por GOTO, incrementar
	if siguientePC == pc && motivo == "" {
		siguientePC = pc + 1
	}

	// Si hay motivo de retorno, el proceso debe salir de la CPU
	if motivo != "" {
		procesoEnEjecucion = -1
	}

	return siguientePC, motivo, parametrosSyscall
}

// Verificar interrupciones
func checkInterrupt(pid int) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if interrupcionPendiente && pidInterrumpido == pid {
		utils.InfoLog.Info("Interrupción recibida al puerto Interrupt")
		interrupcionPendiente = false
		pidInterrumpido = -1
		return true
	}
	return false
}

// Limpiar estructuras al desalojar proceso
func limpiarEstructurasPorPID(pid int) {
	mutex.Lock()
	defer mutex.Unlock()

	// Limpiar TLB
	for i := range tlbEntries {
		if tlbEntries[i].PID == pid {
			tlbEntries[i] = TLBEntry{
				PageNumber:  -1,
				FrameNumber: -1,
				PID:         -1,
			}
		}
	}

	// Limpiar cache y actualizar páginas modificadas
	for i := range cacheEntries {
		if cacheEntries[i].PID == pid {
			if cacheEntries[i].Modified {
				actualizarMemoria(pid, cacheEntries[i].PageNumber)
			}
			cacheEntries[i] = CacheEntry{
				PageNumber: -1,
				Content:    "",
				PID:        -1,
				Modified:   false,
				Referenced: false,
			}
		}
	}

	utils.InfoLog.Info("Estructuras TLB y Cache limpiadas", "pid", pid)
}
