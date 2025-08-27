package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var memoriaGeneralMutex sync.RWMutex

func cargarInstrucciones(pid int) error {
	utils.InfoLog.Info("Cargando instrucciones para proceso", "pid", pid)

	// Construir la ruta del archivo de pseudocódigo
	rutaArchivo := filepath.Clean(filepath.Join(config.ScriptsPath, fmt.Sprintf("%d.txt", pid)))
	utils.InfoLog.Info("Ruta del archivo", "pid", pid, "archivo", rutaArchivo)

	// Leer el archivo línea por línea
	contenido, err := os.ReadFile(rutaArchivo)
	if err != nil {
		utils.ErrorLog.Error("Error leyendo archivo de pseudocódigo", "pid", pid, "archivo", rutaArchivo, "error", err)
		return fmt.Errorf("error al leer el archivo de pseudocódigo para PID %d: %v", pid, err)
	}

	// Dividir el contenido en líneas (instrucciones)
	instrucciones := strings.Split(string(contenido), "\n")

	// Filtrar líneas vacías
	instruccionesFiltradas := []string{}
	for _, instruccion := range instrucciones {
		if strings.TrimSpace(instruccion) != "" {
			instruccionesFiltradas = append(instruccionesFiltradas, instruccion)
		}
	}

	utils.InfoLog.Info("Instrucciones procesadas", "pid", pid, "total_instrucciones", len(instruccionesFiltradas))

	instruccionesMutex.Lock()
	instruccionesPorProceso[pid] = instruccionesFiltradas
	instruccionesMutex.Unlock()

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Proceso Creado - Tamaño: %d",
		pid, len(instruccionesFiltradas)))

	utils.InfoLog.Info("Instrucciones cargadas exitosamente", "pid", pid, "instrucciones", len(instruccionesFiltradas))
	return nil
}

func copiarPseudocodigo(origen string, destino string) error {
	utils.InfoLog.Info("Copiando archivo de pseudocódigo", "origen", origen, "destino", destino)

	// Si el origen no incluye la ruta scripts/, agregarla
	rutaCompleta := origen
	if !strings.Contains(origen, string(filepath.Separator)) && !strings.HasPrefix(origen, "scripts") {
		rutaCompleta = filepath.Join("scripts", origen)
		utils.InfoLog.Info("Ruta ajustada", "ruta_original", origen, "ruta_completa", rutaCompleta)
	}

	input, err := os.ReadFile(rutaCompleta)
	if err != nil {
		utils.ErrorLog.Error("Error leyendo archivo origen", "archivo", rutaCompleta, "error", err)
		return err
	}

	err = os.WriteFile(destino, input, 0644)
	if err != nil {
		utils.ErrorLog.Error("Error escribiendo archivo destino", "archivo", destino, "error", err)
		return err
	}

	utils.InfoLog.Info("Archivo de pseudocódigo copiado", "origen", rutaCompleta, "destino", destino)
	return nil
}

// suspenderProceso guarda todas las páginas de un proceso en SWAP y libera sus marcos
func suspenderProceso(pid int) error {
	utils.InfoLog.Info("Iniciando suspensión de proceso", "pid", pid)

	// LOCK GLOBAL para evitar race conditions
	memoriaGeneralMutex.Lock()
	defer memoriaGeneralMutex.Unlock()

	// Obtener marcos asignados al proceso
	marcos, existe := marcosAsignadosPorProceso[pid]
	if !existe {
		utils.ErrorLog.Error("Proceso sin marcos asignados", "pid", pid)
		return fmt.Errorf("el proceso %d no tiene marcos asignados", pid)
	}

	// Crear una copia de los marcos para evitar modificaciones durante iteración
	marcosCopia := make([]int, len(marcos))
	copy(marcosCopia, marcos)

	// Obtener tabla de páginas del proceso
	tabla, existeTabla := tablasPaginas[pid]
	if !existeTabla {
		utils.ErrorLog.Error("Proceso sin tabla de páginas", "pid", pid)
		return fmt.Errorf("el proceso %d no tiene tabla de páginas", pid)
	}

	utils.InfoLog.Info("Proceso a suspender", "pid", pid, "marcos_asignados", len(marcosCopia))

	// Crear dump antes de SWAP
	if err := crearMemoryDump(pid); err != nil {
		utils.ErrorLog.Error("Error creando dump antes de SWAP", "pid", pid, "error", err)
	}

	// Para cada marco, mover su contenido a SWAP
	for _, marco := range marcosCopia {
		// Buscar la página asociada a este marco
		numPagina := encontrarPaginaPorMarco(pid, tabla, marco, 1)
		if numPagina == -1 {
			utils.InfoLog.Warn("No se encontró página asociada al marco", "pid", pid, "marco", marco)
			continue
		}

		utils.InfoLog.Info("Moviendo página a SWAP", "pid", pid, "pagina", numPagina, "marco", marco)

		// Mover a SWAP (esta función también necesita ser thread-safe)
		_, err := moverASwap(pid, numPagina, marco)
		if err != nil {
			utils.ErrorLog.Error("Error moviendo página a SWAP", "pid", pid, "pagina", numPagina, "marco", marco, "error", err)
			continue
		}

		// Marcar página como no presente
		marcarPaginaNoPresente(pid, tabla, numPagina, 1)

		// Marcar marco como libre de forma thread-safe
		if marcosLibres != nil {
			marcosLibres[marco] = true
		}
	}

	// Liberar la lista de marcos asignados al proceso (pero mantener la tabla)
	// Verificar que el map sigue existiendo antes de modificar
	if marcosAsignadosPorProceso != nil {
		marcosAsignadosPorProceso[pid] = []int{}
	}

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Proceso suspendido a SWAP", pid))

	utils.InfoLog.Info("Proceso suspendido correctamente", "pid", pid)
	return nil
}

// dessuspenderProceso carga todas las páginas de un proceso desde SWAP a la memoria
func dessuspenderProceso(pid int) error {
	utils.InfoLog.Info("Iniciando dessuspensión de proceso", "pid", pid)

	// Verificar si existe la tabla de páginas
	tabla, existeTabla := tablasPaginas[pid]
	if !existeTabla {
		utils.ErrorLog.Error("Proceso sin tabla de páginas", "pid", pid)
		return fmt.Errorf("el proceso %d no tiene tabla de páginas", pid)
	}

	// Buscar todas las entradas de SWAP para este proceso
	swapMutex.Lock()
	paginasEnSwap := []int{}
	for _, entrada := range mapaSwap {
		if entrada.PID == pid && entrada.EnUso {
			paginasEnSwap = append(paginasEnSwap, entrada.Pagina)
		}
	}
	swapMutex.Unlock()

	// Si no hay páginas en SWAP, no hay nada que hacer
	if len(paginasEnSwap) == 0 {
		utils.InfoLog.Info("No hay páginas en SWAP para dessuspender", "pid", pid)
		return nil
	}

	utils.InfoLog.Info("Páginas en SWAP detectadas", "pid", pid, "paginas_en_swap", len(paginasEnSwap))

	// Verificar si hay suficientes marcos libres
	marcosNecesarios := len(paginasEnSwap)
	marcosDisponibles := contarMarcosLibres()
	if marcosDisponibles < marcosNecesarios {
		utils.ErrorLog.Error("Marcos insuficientes para dessuspensión", "pid", pid, "necesarios", marcosNecesarios, "disponibles", marcosDisponibles)
		return fmt.Errorf("no hay suficientes marcos libres para dessuspender el proceso %d: "+
			"necesita %d, disponibles %d", pid, marcosNecesarios, marcosDisponibles)
	}

	// Asignar marcos para cada página
	marcosAsignados := []int{}
	for _, numPagina := range paginasEnSwap {
		utils.InfoLog.Info("Recuperando página desde SWAP", "pid", pid, "pagina", numPagina)

		// Asignar un nuevo marco
		marco, err := asignarMarco(pid)
		if err != nil {
			// Limpiar los marcos ya asignados y retornar error
			for _, m := range marcosAsignados {
				marcosLibres[m] = true
			}
			utils.ErrorLog.Error("Error asignando marco", "pid", pid, "error", err)
			return fmt.Errorf("error asignando marco: %v", err)
		}
		marcosAsignados = append(marcosAsignados, marco)

		// Recuperar desde SWAP
		err = recuperarDeSwap(pid, numPagina, marco)
		if err != nil {
			// Limpiar los marcos ya asignados y retornar error
			for _, m := range marcosAsignados {
				marcosLibres[m] = true
			}
			utils.ErrorLog.Error("Error recuperando de SWAP", "pid", pid, "error", err)
			return fmt.Errorf("error recuperando de SWAP: %v", err)
		}

		// Actualizar tabla de páginas
		actualizarTablaPaginas(pid, tabla, numPagina, marco, 1)
	}

	// Guardar la lista de marcos asignados al proceso
	marcosAsignadosPorProceso[pid] = marcosAsignados

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Proceso dessuspendido desde SWAP", pid))

	utils.InfoLog.Info("Proceso dessuspendido correctamente", "pid", pid, "marcos_asignados", len(marcosAsignados))
	return nil
}
