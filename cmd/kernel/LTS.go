package main

import (
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

const (
	maxIntentosMemoria     = 5
	tiempoEsperaReintentos = 2 * time.Second
)

// PlanificarLargoPlazo optimizado
func PlanificarLargoPlazo() {
	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog.Error("PÁNICO EN PLANIFICADOR LTS", "error", r)
			panic(r)
		}
	}()

	utils.InfoLog.Info("Iniciando Planificador de Largo Plazo")

	for {
		var pcb *PCB

		// Esperar hasta que haya procesos disponibles (SUSP.READY tiene prioridad)
		for {
			// Revisar SUSP.READY primero (prioridad alta)
			suspReadyMutex.Lock()
			if len(colaSuspReady) > 0 {
				utils.InfoLog.Info("LTS encontró proceso en SUSP.READY", "cantidad", len(colaSuspReady))
				pcb = colaSuspReady[0]
				colaSuspReady = colaSuspReady[1:]
				suspReadyMutex.Unlock()

				semaforoMultiprogram.Wait()

				// Verificar si el proceso necesita desswap
				if pcb.EnSwap {
					// Proceso suspendido por timeout, necesita desswap
					go notificarDesswapAMemoria(pcb.PID)
					utils.InfoLog.Info("Proceso de SUSP.READY enviado a desswap", "pid", pcb.PID)
				} else {
					// Proceso completó IO, ya está en memoria
					pcb.CambiarEstado(EstadoReady)
					readyMutex.Lock()
					colaReady = append(colaReady, pcb)
					readyMutex.Unlock()
					condReady.Signal()
					utils.InfoLog.Info("Proceso movido de SUSP.READY a READY (ya en memoria)", "pid", pcb.PID)
				}
				break // Salir del loop interno para procesar siguiente
			}
			suspReadyMutex.Unlock()

			// Si no hay procesos en SUSP.READY, revisar NEW
			newMutex.Lock()
			if len(colaNew) > 0 {
				pcb = seleccionarProcesoLTS()
				if pcb != nil {
					newMutex.Unlock()
					break // Salir del loop interno para procesar
				}
			}

			// No hay procesos en ninguna cola, esperar señales
			utils.InfoLog.Info("LTS esperando procesos disponibles")
			condNew.Wait() // Espera señales de NEW o SUSP.READY
			newMutex.Unlock()
		}

		// Caso especial para proceso inicial (PID 0)
		if pcb.PID == 0 {
			utils.InfoLog.Info("Admitiendo proceso inicial", "pid", 0)
			removerDeCola(&colaNew, pcb)

			if inicializarEnMemoriaConReintentos(pcb) {
				pcb.CambiarEstado(EstadoReady)
				readyMutex.Lock()
				colaReady = append(colaReady, pcb)
				readyMutex.Unlock()
				condReady.Signal()
				utils.InfoLog.Info("Proceso inicial admitido a READY", "pid", pcb.PID)
			} else {
				utils.ErrorLog.Error("Error al inicializar proceso inicial", "pid", pcb.PID)
				FinalizarProceso(pcb, "ERROR_INICIALIZACION_MEMORIA_PROCESO_INICIAL")
			}
			continue
		}

		// Esperar semáforo antes de inicializar en memoria
		semaforoMultiprogram.Wait()

		if inicializarEnMemoriaConReintentos(pcb) {
			removerDeCola(&colaNew, pcb)
			pcb.CambiarEstado(EstadoReady)

			readyMutex.Lock()
			colaReady = append(colaReady, pcb)
			readyMutex.Unlock()
			condReady.Signal()
			utils.InfoLog.Info("Proceso admitido a READY", "pid", pcb.PID)
		} else {
			removerDeCola(&colaNew, pcb)
			FinalizarProceso(pcb, "ERROR_INICIALIZACION_MEMORIA")
			semaforoMultiprogram.Signal()
		}
	}
}

// inicializarEnMemoriaConReintentos maneja reintentos automáticamente
func inicializarEnMemoriaConReintentos(pcb *PCB) bool {
	utils.InfoLog.Info("Inicializando proceso en memoria", "pid", pcb.PID, "max_intentos", maxIntentosMemoria)

	for intento := 1; intento <= maxIntentosMemoria; intento++ {
		if inicializarProcesoEnMemoria(pcb.PID, pcb.Tamanio, pcb.NombreArchivo) {
			utils.InfoLog.Info("Proceso inicializado en memoria", "pid", pcb.PID, "intento", intento)
			return true
		}

		if intento < maxIntentosMemoria {
			utils.InfoLog.Warn("Intento fallido, reintentando", "pid", pcb.PID, "intento", intento, "espera", tiempoEsperaReintentos)
			time.Sleep(tiempoEsperaReintentos)
		}
	}

	utils.ErrorLog.Error("Todos los intentos de inicialización fallaron", "pid", pcb.PID)
	return false
}

// seleccionarProcesoLTS selecciona el próximo proceso según algoritmo
func seleccionarProcesoLTS() *PCB {
	if len(colaNew) == 0 {
		return nil
	}

	algoritmo := kernelConfig.ReadyIngressAlgorithm
	utils.InfoLog.Info("Seleccionando proceso LTS", "algoritmo", algoritmo, "procesos_disponibles", len(colaNew))

	switch algoritmo {
	case "FIFO":
		return seleccionarFIFOLTS()
	case "PMCP":
		return seleccionarPMCP()
	default:
		utils.InfoLog.Warn("Algoritmo LTS no reconocido, usando FIFO", "algoritmo", algoritmo)
		return seleccionarFIFOLTS()
	}
}

// seleccionarFIFOLTS implementa selección FIFO
func seleccionarFIFOLTS() *PCB {
	return colaNew[0]
}

// seleccionarPMCP implementa Programación Multiprogramada Controlada por Prioridad
func seleccionarPMCP() *PCB {
	if len(colaNew) == 0 {
		return nil
	}

	// Crear copia para ordenar
	candidatos := make([]*PCB, len(colaNew))
	copy(candidatos, colaNew)

	// Ordenar por tamaño (menor tamaño = mayor prioridad)
	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].Tamanio == candidatos[j].Tamanio {
			return candidatos[i].HoraCreacion.Before(candidatos[j].HoraCreacion)
		}
		return candidatos[i].Tamanio < candidatos[j].Tamanio
	})

	seleccionado := candidatos[0]
	utils.InfoLog.Info("PMCP seleccionó proceso", "pid", seleccionado.PID, "tamaño", seleccionado.Tamanio)

	return seleccionado
}

// inicializarProcesoEnMemoria simplificado
func inicializarProcesoEnMemoria(pid int, tamanio int, nombreArchivo string) bool {
	cliente := GetMemoriaClient()
	if cliente == nil {
		utils.ErrorLog.Error("No se pudo obtener cliente de memoria", "pid", pid)
		return false
	}

	datos := map[string]interface{}{
		"pid":     pid,
		"tamanio": tamanio,
		"archivo": nombreArchivo,
	}

	respuesta, err := cliente.EnviarHTTPMensaje(utils.MensajeInicializarProceso, "default", datos)
	if err != nil {
		utils.ErrorLog.Error("Error de comunicación con Memoria", "pid", pid, "error", err.Error())
		return false
	}

	if respuestaMap, ok := respuesta.(map[string]interface{}); ok {
		status, _ := respuestaMap["status"].(string)
		if status == "OK" {
			utils.InfoLog.Info("Proceso inicializado en Memoria", "pid", pid)
			return true
		} else {
			message, _ := respuestaMap["message"].(string)
			utils.ErrorLog.Error("Memoria rechazó la inicialización", "pid", pid, "status", status, "message", message)
			return false
		}
	}

	utils.ErrorLog.Error("Respuesta de Memoria en formato inválido", "pid", pid)
	return false
}

// notificarDesswapAMemoria con log de notificación
func notificarDesswapAMemoria(pid int) bool {
	cliente := GetMemoriaClient()
	if cliente == nil {
		utils.ErrorLog.Error("No se pudo obtener cliente de memoria para desswap", "pid", pid)
		return false
	}

	// Log para visualizar la petición a Memoria para cargar desde SWAP
	utils.InfoLog.Info("Notificando a Memoria: Cargar desde SWAP", "pid", pid)

	datos := map[string]interface{}{
		"pid": pid,
	}

	respuesta, err := cliente.EnviarHTTPMensaje(utils.MensajeDessuspenderProceso, "default", datos)
	if err != nil {
		return false
	}

	if respuestaMap, ok := respuesta.(map[string]interface{}); ok {
		status, _ := respuestaMap["status"].(string)
		return status == "OK"
	}

	return false
}
