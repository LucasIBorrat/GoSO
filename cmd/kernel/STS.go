package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Variables globales para gestión de CPUs
var (
	cpuClients               map[string]*utils.HTTPClient
	cpuClientsMutex          sync.Mutex
	ultimoLogCPUNoDisponible time.Time
)

// InicializarMapaCPUs inicializa el mapa de CPUs
func InicializarMapaCPUs() {
	cpuClientsMutex.Lock()
	defer cpuClientsMutex.Unlock()

	if cpuClients == nil {
		cpuClients = make(map[string]*utils.HTTPClient)
		utils.InfoLog.Info("Mapa cpuClients inicializado")
	}
}

// registrarCPU registra un nuevo cliente CPU
func registrarCPU(nombre string, ip string, puerto int) {
	cpuClientsMutex.Lock()
	defer cpuClientsMutex.Unlock()

	if cpuClients == nil {
		utils.ErrorLog.Error("Mapa cpuClients no inicializado")
		cpuClients = make(map[string]*utils.HTTPClient)
	}

	nombreCPU := nombre
	if nombreCPU == "" {
		nombreCPU = fmt.Sprintf("CPU_%s_%d", ip, puerto)
		utils.InfoLog.Info("Usando nombre generado para CPU", "nombre_generado", nombreCPU)
	}

	cpuClients[nombreCPU] = utils.NewHTTPClient(ip, puerto, "Kernel->"+nombreCPU)

	utils.InfoLog.Info("CPU registrada correctamente", "nombre", nombreCPU, "ip", ip, "puerto", puerto, "total_cpus", len(cpuClients))
}

// PlanificarCortoPlazo gestiona transición de procesos entre READY y EXEC
func PlanificarCortoPlazo() {
	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog.Error("PÁNICO EN PLANIFICADOR STS", "error", r)
			panic(r)
		}
	}()

	utils.InfoLog.Info("Iniciando Planificador de Corto Plazo")

	for {
		utils.InfoLog.Info("Esperando procesos en READY")
		readyMutex.Lock()
		for len(colaReady) == 0 {
			condReady.Wait()
		}
		utils.InfoLog.Info("Proceso detectado en READY", "procesos_en_ready", len(colaReady))

		pcb := seleccionarProcesoSTS()

		if pcb != nil {
			utils.InfoLog.Info("Proceso seleccionado", "pid", pcb.PID)
			removerDeCola(&colaReady, pcb)
		}

		if pcb == nil {
			readyMutex.Unlock()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		readyMutex.Unlock()

		// Buscar CPU disponible
		var nombreCPU string
		var cpuClient *utils.HTTPClient

		utils.InfoLog.Info("Buscando CPU disponible")
		for {
			nombreCPU, cpuClient = obtenerCPUDisponibleParaEjecucion()
			if cpuClient != nil {
				execMutex.Lock()
				colaExec[nombreCPU] = pcb
				execMutex.Unlock()
				utils.InfoLog.Info("CPU encontrada y reservada", "nombre", nombreCPU)
				break
			}
			utils.InfoLog.Warn("No hay CPU disponible, reintentando")
			time.Sleep(200 * time.Millisecond)
		}

		pcb.CambiarEstado(EstadoExec)
		utils.InfoLog.Info("Proceso despachado a CPU", "pid", pcb.PID, "cpu", nombreCPU)

		go despacharYProcesarCPU(nombreCPU, cpuClient, pcb)
	}
}

// despacharYProcesarCPU maneja el ciclo de vida de un proceso en la CPU
func despacharYProcesarCPU(nombreCPU string, cpuClient *utils.HTTPClient, pcb *PCB) {
	utils.InfoLog.Info("Iniciando ejecución en CPU", "pid", pcb.PID, "cpu", nombreCPU)

	defer func() {
		utils.InfoLog.Info("Liberando CPU", "pid", pcb.PID, "cpu", nombreCPU)
		execMutex.Lock()
		delete(colaExec, nombreCPU)
		execMutex.Unlock()
	}()

	// Ciclo de ejecución en CPU
	for {
		// VERIFICACIÓN CRÍTICA: Comprobar si el proceso sigue existiendo
		mapaMutex.Lock()
		_, procesoExiste := mapaPCBs[pcb.PID]
		estadoActual := pcb.Estado
		mapaMutex.Unlock()

		// Si el proceso ya no existe o está en EXIT, terminar inmediatamente
		if !procesoExiste || estadoActual == EstadoExit {
			utils.InfoLog.Info("Proceso finalizado o no existe, terminando ejecución", "pid", pcb.PID, "existe", procesoExiste, "estado", estadoActual)
			break
		}

		// Si el proceso ya no está en EXEC, salir del bucle
		if estadoActual != EstadoExec {
			utils.InfoLog.Info("Proceso cambió de estado", "pid", pcb.PID, "nuevo_estado", estadoActual)
			break
		}

		utils.InfoLog.Info("Enviando proceso a CPU", "pid", pcb.PID, "cpu", nombreCPU, "pc", pcb.PC)
		fueExitoso := EnviarProcesoCPU(pcb, nombreCPU)

		if !fueExitoso {
			utils.ErrorLog.Error("Ciclo de ejecución en CPU falló", "pid", pcb.PID, "cpu", nombreCPU)
			break
		}

		// VERIFICACIÓN POST-EJECUCIÓN: Verificar nuevamente el estado
		mapaMutex.Lock()
		_, procesoSigueExistiendo := mapaPCBs[pcb.PID]
		estadoPostEjecucion := pcb.Estado
		mapaMutex.Unlock()

		// Si el proceso fue finalizado durante EnviarProcesoCPU, terminar
		if !procesoSigueExistiendo || estadoPostEjecucion == EstadoExit {
			utils.InfoLog.Info("Proceso finalizado durante ejecución", "pid", pcb.PID, "existe", procesoSigueExistiendo, "estado", estadoPostEjecucion)
			break
		}

		// Si cambió a otro estado (IO, BLOCKED, etc.), salir del bucle
		if estadoPostEjecucion != EstadoExec {
			utils.InfoLog.Info("Proceso cambió de estado después de ejecución", "pid", pcb.PID, "nuevo_estado", estadoPostEjecucion)
			break
		}
	}
}

// obtenerCPUDisponibleParaEjecucion busca CPU que no esté ejecutando
func obtenerCPUDisponibleParaEjecucion() (string, *utils.HTTPClient) {
	// Obtener CPUs registradas
	cpuClientsMutex.Lock()
	cpusDisponibles := make(map[string]*utils.HTTPClient)
	for nombre, cliente := range cpuClients {
		cpusDisponibles[nombre] = cliente
	}
	cpuClientsMutex.Unlock()

	if len(cpusDisponibles) == 0 {
		if time.Since(ultimoLogCPUNoDisponible) > 5*time.Second {
			utils.InfoLog.Warn("No hay CPUs registradas")
			ultimoLogCPUNoDisponible = time.Now()
		}
		return "", nil
	}

	// Verificar cuáles no están en ejecución
	execMutex.Lock()
	defer execMutex.Unlock()

	for nombre, cliente := range cpusDisponibles {
		if _, ocupada := colaExec[nombre]; !ocupada {
			utils.InfoLog.Info("CPU libre encontrada", "nombre", nombre)
			return nombre, cliente
		}
	}

	utils.InfoLog.Info("Todas las CPUs están ocupadas")
	return "", nil
}

// seleccionarProcesoSTS selecciona proceso según algoritmo configurado
func seleccionarProcesoSTS() *PCB {
	if len(colaReady) == 0 {
		return nil
	}

	algoritmo := kernelConfig.SchedulerAlgorithm
	utils.InfoLog.Info("Seleccionando proceso STS", "algoritmo", algoritmo, "procesos_disponibles", len(colaReady))

	switch algoritmo {
	case "FIFO":
		return seleccionarFIFO()
	case "SJF":
		return seleccionarSJF()
	case "SRT":
		return seleccionarSRT()
	default:
		utils.InfoLog.Warn("Algoritmo STS no reconocido, usando FIFO", "algoritmo", algoritmo)
		return seleccionarFIFO()
	}
}

// seleccionarFIFO implementa selección FIFO
func seleccionarFIFO() *PCB {
	if len(colaReady) == 0 {
		return nil
	}
	return colaReady[0]
}

// seleccionarSJF implementa Shortest Job First
func seleccionarSJF() *PCB {
	if len(colaReady) == 0 {
		return nil
	}

	candidatos := make([]*PCB, len(colaReady))
	copy(candidatos, colaReady)

	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].EstimacionSiguienteRafaga == candidatos[j].EstimacionSiguienteRafaga {
			return candidatos[i].HoraListo.Before(candidatos[j].HoraListo)
		}
		return candidatos[i].EstimacionSiguienteRafaga < candidatos[j].EstimacionSiguienteRafaga
	})

	seleccionado := candidatos[0]
	utils.InfoLog.Info("SJF seleccionó proceso", "pid", seleccionado.PID, "estimacion", seleccionado.EstimacionSiguienteRafaga)

	return seleccionado
}

// seleccionarSRT implementa Shortest Remaining Time
func seleccionarSRT() *PCB {
	if len(colaReady) == 0 {
		return nil
	}

	mejorCandidatoReady := encontrarMejorCandidatoReady()

	execMutex.Lock()
	procesoADesalojar := encontrarProcesoADesalojar(mejorCandidatoReady)
	execMutex.Unlock()

	if procesoADesalojar != nil {
		utils.InfoLog.Info(fmt.Sprintf("(%d) - Desalojado por algoritmo SJF/SRT", procesoADesalojar.PID))
		utils.InfoLog.Info("Desalojando proceso por SRT", "desalojado", procesoADesalojar.PID, "nuevo", mejorCandidatoReady.PID)
		go desalojarProcesoActual(procesoADesalojar)
		return nil
	}

	return mejorCandidatoReady
}

func encontrarMejorCandidatoReady() *PCB {
	if len(colaReady) == 0 {
		return nil
	}

	mejorProceso := colaReady[0]
	for _, pcb := range colaReady[1:] {
		if pcb.EstimacionSiguienteRafaga < mejorProceso.EstimacionSiguienteRafaga {
			mejorProceso = pcb
		} else if pcb.EstimacionSiguienteRafaga == mejorProceso.EstimacionSiguienteRafaga {
			if pcb.HoraListo.Before(mejorProceso.HoraListo) {
				mejorProceso = pcb
			}
		}
	}
	return mejorProceso
}

func encontrarProcesoADesalojar(candidato *PCB) *PCB {
	var procesoMasLargo *PCB = nil
	if candidato == nil {
		return nil
	}

	for _, pcbEnExec := range colaExec {
		if candidato.EstimacionSiguienteRafaga < pcbEnExec.EstimacionSiguienteRafaga {
			if procesoMasLargo == nil || pcbEnExec.EstimacionSiguienteRafaga > procesoMasLargo.EstimacionSiguienteRafaga {
				procesoMasLargo = pcbEnExec
			}
		}
	}
	return procesoMasLargo
}

// desalojarProcesoActual maneja el desalojo de un proceso por SRT
func desalojarProcesoActual(pcb *PCB) {
	var cpuADesalojar string
	execMutex.Lock()
	for cpu, pcbEnExec := range colaExec {
		if pcbEnExec != nil && pcbEnExec.PID == pcb.PID {
			cpuADesalojar = cpu
			break
		}
	}
	execMutex.Unlock()

	if cpuADesalojar == "" {
		utils.InfoLog.Warn("Intento de desalojar proceso que ya no está en ejecución", "pid", pcb.PID)
		return
	}

	cpuClientsMutex.Lock()
	cpuClient, existe := cpuClients[cpuADesalojar]
	cpuClientsMutex.Unlock()

	if !existe {
		utils.ErrorLog.Error("No se encontró cliente para CPU a desalojar", "cpu", cpuADesalojar)
		return
	}

	utils.InfoLog.Info("Enviando interrupción a CPU", "cpu", cpuADesalojar, "pid", pcb.PID)
	_, err := cpuClient.EnviarHTTPOperacion("INTERRUPT", nil)
	if err != nil {
		utils.ErrorLog.Error("Fallo al enviar interrupción a CPU", "cpu", cpuADesalojar, "error", err)
	}
}

// EnviarProcesoCPU envía un PCB a la CPU para su ejecución
func EnviarProcesoCPU(pcb *PCB, nombreCPU string) bool {
	cpuClientsMutex.Lock()
	cpuClient, existe := cpuClients[nombreCPU]
	cpuClientsMutex.Unlock()

	if !existe {
		utils.ErrorLog.Error("CPU no encontrada", "cpu_id", nombreCPU)
		return false
	}

	datos := map[string]interface{}{
		"pid": pcb.PID,
		"pc":  pcb.PC,
	}

	utils.InfoLog.Info("Enviando proceso a CPU", "pid", pcb.PID, "pc", pcb.PC, "cpu", nombreCPU)

	respuesta, err := cpuClient.EnviarHTTPOperacion("EJECUTAR_PROCESO", datos)

	if err != nil {
		utils.ErrorLog.Error("Error enviando proceso a CPU", "pid", pcb.PID, "error", err.Error())
		MoverProcesoAReady(pcb)
		return false
	}

	// Procesar respuesta
	if respuestaMap, ok := respuesta.(map[string]interface{}); ok {
		if errorMsg, tieneError := respuestaMap["error"].(string); tieneError {
			utils.ErrorLog.Error("Error reportado por CPU", "pid", pcb.PID, "mensaje", errorMsg)
			return false
		}

		// Actualizar PC
		pcActualizadoPorCPU := false
		if nuevoPC, hayPC := respuestaMap["pc"].(float64); hayPC {
			pcb.PC = int(nuevoPC)
			pcActualizadoPorCPU = true
		}

		// Verificar motivo de retorno
		if motivoRetorno, hayMotivo := respuestaMap["motivo_retorno"].(string); hayMotivo {
			utils.InfoLog.Info("Motivo de retorno recibido", "pid", pcb.PID, "motivo", motivoRetorno)

			switch motivoRetorno {
			case "SYSCALL_INIT_PROC":
				utils.InfoLog.Info(fmt.Sprintf("(%d) - Solicitó syscall: INIT_PROC", pcb.PID))
				if parametros, ok := respuestaMap["parametros"].(map[string]interface{}); ok {
					archivo, _ := parametros["archivo"].(string)
					tamano, _ := parametros["tamano"].(float64)

					utils.InfoLog.Info("Procesando INIT_PROC", "pid", pcb.PID, "archivo", archivo, "tamaño", int(tamano))

					nuevoPCB := NuevoPCB(-1, int(tamano))
					nuevoPCB.NombreArchivo = archivo
					utils.InfoLog.Info("Nuevo proceso creado", "nuevo_pid", nuevoPCB.PID, "estado", "NEW")
					AgregarProcesoANew(nuevoPCB)
				}

				pcb.PC++
				utils.InfoLog.Info("PC incrementado después de INIT_PROC", "pid", pcb.PID, "nuevo_pc", pcb.PC)
				return true

			case "SYSCALL_IO":
				utils.InfoLog.Info(fmt.Sprintf("(%d) - Solicitó syscall: IO", pcb.PID))
				utils.InfoLog.Info("Procesando IO", "pid", pcb.PID)
				pcb.CambiarEstado(EstadoBlocked)

				if parametros, ok := respuestaMap["parametros"].(map[string]interface{}); ok {
					dispositivo, _ := parametros["dispositivo"].(string)
					tiempo, _ := parametros["tiempo"].(float64)

					dispositivoReal := SeleccionarDispositivoIO(dispositivo, pcb.PID)
					MoverProcesoABlocked(pcb, fmt.Sprintf("IO_%s", dispositivoReal))
					go EnviarSolicitudIO(pcb, dispositivoReal, int(tiempo))
				}
				return true

			case "SYSCALL_DUMP_MEMORY":
				utils.InfoLog.Info(fmt.Sprintf("(%d) - Solicitó syscall: DUMP_MEMORY", pcb.PID))
				utils.InfoLog.Info("Procesando DUMP_MEMORY", "pid", pcb.PID)
				pcb.CambiarEstado(EstadoBlocked)
				MoverProcesoABlocked(pcb, "DUMP_MEMORY")
				return true

			case "EXIT":
				utils.InfoLog.Info(fmt.Sprintf("(%d) - Solicitó syscall: EXIT", pcb.PID))
				utils.InfoLog.Info("Proceso solicita EXIT", "pid", pcb.PID)
				FinalizarProceso(pcb, "EXIT")
				return true

			case "ERROR":
				utils.ErrorLog.Error("Error en ejecución de proceso", "pid", pcb.PID)
				FinalizarProceso(pcb, "ERROR")
				return true
			}
		}

		// Continuar ejecución
		if !pcActualizadoPorCPU {
			pcb.PC++
		}
		utils.InfoLog.Info("Continuando ejecución", "pid", pcb.PID, "nuevo_pc", pcb.PC)

		return true
	}

	utils.ErrorLog.Warn("Formato de respuesta inválido de CPU", "respuesta", fmt.Sprintf("%v", respuesta))
	return false
}
