package main

import (
	"fmt"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Traer página desde SWAP a memoria principal
func traerPaginaDeSwap(pid int, numPagina int, marco int) error {
	utils.InfoLog.Info("Intentando traer página desde SWAP", "pid", pid, "pagina", numPagina, "marco", marco)

	// Verificar si la página está en SWAP
	key := fmt.Sprintf("%d-%d", pid, numPagina)

	swapMutex.Lock()
	entrada, existe := mapaSwap[key]
	swapMutex.Unlock()

	if existe && entrada.EnUso {
		utils.InfoLog.Info("Página encontrada en SWAP", "pid", pid, "pagina", numPagina)

		// Recuperar la página desde SWAP
		err := recuperarDeSwap(pid, numPagina, marco)
		if err != nil {
			utils.ErrorLog.Error("Error recuperando desde SWAP", "pid", pid, "pagina", numPagina, "error", err)
			return err
		}

		// Marcar como liberada en SWAP (opcional, podemos mantenerla como copia)
		swapMutex.Lock()
		entrada.EnUso = false
		mapaSwap[key] = entrada
		swapMutex.Unlock()

		utils.InfoLog.Info("Página recuperada desde SWAP", "pid", pid, "pagina", numPagina, "marco", marco)
		return nil
	}

	// La página no está en SWAP, debe ser una página nueva o limpia
	utils.InfoLog.Info("Página nueva inicializada", "pid", pid, "pagina", numPagina, "marco", marco)

	// Inicializar la página con ceros
	dirFisica := marco * config.PageSize
	for i := 0; i < config.PageSize; i++ {
		memoriaPrincipal[dirFisica+i] = 0
	}

	return nil
}

// moverASwap mueve una página de memoria principal a SWAP
func moverASwap(pid int, numPagina int, marco int) (int64, error) {
	utils.InfoLog.Info("Moviendo página a SWAP", "pid", pid, "pagina", numPagina, "marco", marco)

	// Aplicar retardo de SWAP
	utils.AplicarRetardo("swap", config.SwapDelay)

	swapMutex.Lock()
	defer swapMutex.Unlock()

	// Generar clave única para esta página
	key := fmt.Sprintf("%d-%d", pid, numPagina)

	// Verificar si ya existe en SWAP (sobreescribir)
	var offset int64
	if entrada, existe := mapaSwap[key]; existe {
		offset = entrada.Offset
		utils.InfoLog.Info("Sobreescribiendo entrada existente en SWAP", "pid", pid, "pagina", numPagina, "offset", offset)
	} else {
		// Calcular nueva posición en archivo de SWAP
		offset = calcularNuevoOffsetSwap()
		utils.InfoLog.Info("Nueva posición en SWAP calculada", "pid", pid, "pagina", numPagina, "offset", offset)
	}

	// Abrir archivo de SWAP
	swapFile, err := os.OpenFile(config.SwapfilePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		utils.ErrorLog.Error("Error abriendo archivo SWAP", "archivo", config.SwapfilePath, "error", err)
		return 0, fmt.Errorf("error al abrir archivo SWAP: %v", err)
	}
	defer swapFile.Close()

	// Obtener los datos del marco de memoria
	dirFisica := marco * config.PageSize
	datos := make([]byte, config.PageSize)
	copy(datos, memoriaPrincipal[dirFisica:dirFisica+config.PageSize])

	// Escribir datos en la posición asignada
	_, err = swapFile.WriteAt(datos, offset)
	if err != nil {
		utils.ErrorLog.Error("Error escribiendo en SWAP", "archivo", config.SwapfilePath, "offset", offset, "error", err)
		return 0, fmt.Errorf("error al escribir en SWAP: %v", err)
	}

	// Actualizar mapa de SWAP
	mapaSwap[key] = EntradaSwap{
		PID:     pid,
		Pagina:  numPagina,
		Offset:  offset,
		Tamanio: config.PageSize,
		EnUso:   true,
	}

	// Actualizar métricas
	actualizarMetricasBajadaSwap(pid)

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Datos movidos a SWAP - Página: %d", pid, numPagina))

	utils.InfoLog.Info("Página movida a SWAP exitosamente", "pid", pid, "pagina", numPagina, "offset", offset)

	return offset, nil
}

// calcularNuevoOffsetSwap encuentra la próxima posición libre en el archivo SWAP
func calcularNuevoOffsetSwap() int64 {
	var maxOffset int64 = 0

	// Encontrar la posición más alta usada actualmente
	for _, entrada := range mapaSwap {
		if entrada.EnUso && entrada.Offset+int64(entrada.Tamanio) > maxOffset {
			maxOffset = entrada.Offset + int64(entrada.Tamanio)
		}
	}

	utils.InfoLog.Info("Nuevo offset calculado", "offset", maxOffset)
	return maxOffset
}

// recuperarDeSwap trae una página desde SWAP a memoria principal
func recuperarDeSwap(pid int, numPagina int, marco int) error {
	utils.InfoLog.Info("Recuperando página desde SWAP", "pid", pid, "pagina", numPagina, "marco", marco)

	// Aplicar retardo de SWAP
	utils.AplicarRetardo("swap", config.SwapDelay)

	swapMutex.Lock()
	defer swapMutex.Unlock()

	// Buscar entrada en el mapa de SWAP
	key := fmt.Sprintf("%d-%d", pid, numPagina)
	entrada, existe := mapaSwap[key]
	if !existe || !entrada.EnUso {
		utils.ErrorLog.Error("Página no encontrada en SWAP", "pid", pid, "pagina", numPagina)
		return fmt.Errorf("no se encontró la página %d del proceso %d en SWAP", numPagina, pid)
	}

	utils.InfoLog.Info("Página encontrada en SWAP", "pid", pid, "pagina", numPagina, "offset", entrada.Offset, "tamanio", entrada.Tamanio)

	// Abrir archivo de SWAP
	swapFile, err := os.Open(config.SwapfilePath)
	if err != nil {
		utils.ErrorLog.Error("Error abriendo archivo SWAP para lectura", "archivo", config.SwapfilePath, "error", err)
		return fmt.Errorf("error al abrir archivo SWAP: %v", err)
	}
	defer swapFile.Close()

	// Leer datos
	datos := make([]byte, entrada.Tamanio)
	_, err = swapFile.ReadAt(datos, entrada.Offset)
	if err != nil {
		utils.ErrorLog.Error("Error leyendo desde SWAP", "archivo", config.SwapfilePath, "offset", entrada.Offset, "error", err)
		return fmt.Errorf("error al leer de SWAP: %v", err)
	}

	// Escribir datos en la memoria principal
	dirFisica := marco * config.PageSize
	copy(memoriaPrincipal[dirFisica:dirFisica+config.PageSize], datos)

	// Actualizar métricas
	actualizarMetricasSubidaMemoria(pid)

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Página %d recuperada de SWAP al marco %d", pid, numPagina, marco))

	utils.InfoLog.Info("Página recuperada exitosamente", "pid", pid, "pagina", numPagina, "marco", marco)

	return nil
}
