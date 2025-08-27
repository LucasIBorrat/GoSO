package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

func inicializarMemoria() {
	utils.InfoLog.Info("Inicializando memoria",
		"tamaño_total", config.MemorySize,
		"tamaño_página", config.PageSize,
		"niveles_tabla", config.NumberOfLevels,
		"entradas_por_tabla", config.EntriesPerPage)

	// Inicializar la memoria principal
	memoriaPrincipal = make([]byte, config.MemorySize)
	utils.InfoLog.Info("Memoria principal inicializada", "tamaño_bytes", len(memoriaPrincipal))

	// Inicializar tabla de páginas
	tablasPaginas = make(map[int]*TablaPaginas)
	utils.InfoLog.Info("Mapa de tablas de páginas inicializado")

	// Inicializar array para rastrear marcos libres
	totalMarcos := config.MemorySize / config.PageSize
	marcosLibres = make([]bool, totalMarcos)
	for i := range marcosLibres {
		marcosLibres[i] = true // Inicialmente, todos los marcos están libres
	}
	utils.InfoLog.Info("Array de marcos libres inicializado", "total_marcos", totalMarcos)

	// Inicializar el mapeo de marcos por proceso
	marcosAsignadosPorProceso = make(map[int][]int)
	utils.InfoLog.Info("Mapa de marcos por proceso inicializado")

	// Inicializar área de swap
	utils.InfoLog.Info("Inicializando área de swap", "ruta", config.SwapfilePath)
	err := inicializarAreaSwap()
	if err != nil {
		utils.ErrorLog.Error("Error al inicializar el área de swap", "error", err)
		os.Exit(1)
	}

	utils.InfoLog.Info("Memoria completamente inicializada")
}

// Función para inicializar el área de swap
func inicializarAreaSwap() error {
	utils.InfoLog.Info("Configurando área de swap", "archivo", config.SwapfilePath)

	// Crear directorio si no existe
	dir := filepath.Dir(config.SwapfilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.ErrorLog.Error("Error creando directorio para swap", "directorio", dir, "error", err)
		return fmt.Errorf("error al crear directorio para swap: %v", err)
	}

	utils.InfoLog.Info("Directorio de swap verificado", "directorio", dir)

	// Crear o truncar el archivo SWAP
	swapFile, err := os.Create(config.SwapfilePath)
	if err != nil {
		utils.ErrorLog.Error("Error creando archivo SWAP", "archivo", config.SwapfilePath, "error", err)
		return fmt.Errorf("error al crear archivo SWAP: %v", err)
	}
	defer swapFile.Close()

	utils.InfoLog.Info("Archivo SWAP creado", "archivo", config.SwapfilePath)

	// Inicializar el mapa de SWAP
	mapaSwap = make(map[string]EntradaSwap)
	utils.InfoLog.Info("Mapa de SWAP inicializado")

	utils.InfoLog.Info("Área de SWAP inicializada correctamente", "archivo", config.SwapfilePath)
	return nil
}

// Inicializar métricas
func inicializarMetricas() {
	metricasPorProceso = make(map[int]*MetricasProceso)
	utils.InfoLog.Info("Sistema de métricas inicializado")
}
