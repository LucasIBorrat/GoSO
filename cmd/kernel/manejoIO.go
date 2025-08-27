package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var (
	dispositivosIO      map[string]*utils.HTTPClient = make(map[string]*utils.HTTPClient)
	dispositivosIOMutex sync.RWMutex
	contadorBalanceador int
	balanceadorMutex    sync.Mutex
)

// RegistrarDispositivoIO optimizado
func RegistrarDispositivoIO(nombre string, ip string, puerto int) {
	dispositivosIOMutex.Lock()
	defer dispositivosIOMutex.Unlock()

	if _, existe := dispositivosIO[nombre]; !existe {
		dispositivosIO[nombre] = utils.NewHTTPClient(ip, puerto, "Kernel->"+nombre)
		utils.InfoLog.Info("Dispositivo IO registrado", "nombre", nombre, "ip", ip, "puerto", puerto)
	}
}

func ObtenerClienteIO(nombre string) (*utils.HTTPClient, bool) {
	dispositivosIOMutex.RLock()
	defer dispositivosIOMutex.RUnlock()
	cliente, existe := dispositivosIO[nombre]
	return cliente, existe
}

// EnviarSolicitudIO con la corrección definitiva
func EnviarSolicitudIO(pcb *PCB, dispositivo string, tiempo int) {
	cliente, existe := ObtenerClienteIO(dispositivo)
	if !existe {
		utils.ErrorLog.Error("Dispositivo IO no registrado, finalizando proceso", "dispositivo", dispositivo, "pid", pcb.PID)
		FinalizarProceso(pcb, "ERROR_IO_DEVICE_NOT_FOUND") // Finaliza si el dispositivo no existe
		return
	}

	utils.InfoLog.Info("Enviando petición a IO", "pid", pcb.PID, "dispositivo", dispositivo)

	datos := map[string]interface{}{
		"pid":       pcb.PID,
		"tiempo":    tiempo,
		"operacion": "IO_REQUEST",
	}

	_, err := cliente.EnviarHTTPOperacion("IO_REQUEST", datos)

	// --- CAMBIO CLAVE Y DEFINITIVO ---
	// Si hay un error de comunicación (ej: el IO está caído), finalizamos el proceso.
	if err != nil {
		utils.ErrorLog.Error("Error de comunicación con dispositivo IO. El proceso será finalizado.", "dispositivo", dispositivo, "pid", pcb.PID, "error", err.Error())
		FinalizarProceso(pcb, "ERROR_IO_CONNECTION")
		return // Importante: Salimos de la función aquí.
	}
}

// manejarCompletionIO maneja la finalización de IO considerando el estado actual del proceso
func manejarCompletionIO(pcb *PCB) {
	switch pcb.Estado {
	case EstadoBlocked:
		// BLOCKED -> READY (proceso en memoria)
		MoverProcesoAReady(pcb)
		go despacharProcesoSiCorresponde()
	case EstadoSuspBlocked:
		// SUSP.BLOCKED -> SUSP.READY (proceso en swap)
		MoverProcesoASuspReady(pcb)
	default:
		utils.InfoLog.Warn("IO completada para proceso en estado inesperado", "pid", pcb.PID, "estado", pcb.Estado)
		MoverProcesoAReady(pcb) // Fallback: intentar mover a READY de todas formas
	}
}

// ManejadorRegistroIO simplificado
func ManejadorRegistroIO(origen string, datos map[string]interface{}) (interface{}, bool) {
	tipoModulo, ok := datos["tipo"].(string)
	if !ok || !strings.HasPrefix(tipoModulo, "IO") {
		return nil, false
	}

	ip, okIP := datos["ip"].(string)
	puertoFloat, okPuerto := datos["puerto"].(float64)

	if !okIP || !okPuerto {
		return map[string]interface{}{
			"status":  "ERROR",
			"message": "Handshake incompleto",
		}, true
	}

	// Registrar con nombre completo y simplificado
	RegistrarDispositivoIO(tipoModulo, ip, int(puertoFloat))

	nombreSimplificado := strings.TrimPrefix(tipoModulo, "IO")
	if nombreSimplificado != tipoModulo {
		RegistrarDispositivoIO(nombreSimplificado, ip, int(puertoFloat))
		utils.InfoLog.Info("Dispositivo IO registrado", "completo", tipoModulo, "simple", nombreSimplificado)
	} else {
		utils.InfoLog.Info("Módulo IO registrado", "nombre", tipoModulo)
	}

	return map[string]interface{}{
		"status":  "OK",
		"message": fmt.Sprintf("IO '%s' registrado", tipoModulo),
	}, true
}

// ProcesarSolicitudIO optimizado
func ProcesarSolicitudIO(datos map[string]interface{}) (interface{}, bool) {
	evento, _ := datos["evento"].(string)
	motivo, _ := datos["motivo_retorno"].(string)

	if evento != "SOLICITUD_IO" && motivo != "IO_REQUEST" {
		return nil, false
	}

	pidFloat, pidOk := datos["pid"].(float64)
	if !pidOk {
		return map[string]interface{}{"status": "ERROR", "mensaje": "PID inválido"}, true
	}

	dispositivo, _ := datos["dispositivo"].(string)
	if dispositivo == "" {
		dispositivo, _ = datos["nombre_dispositivo"].(string)
	}

	tiempoFloat, _ := datos["tiempo"].(float64)
	if tiempoFloat == 0 {
		tiempoFloat, _ = datos["tiempo_bloqueo"].(float64)
	}

	pcb := BuscarPCBPorPID(int(pidFloat))
	if pcb == nil {
		return map[string]interface{}{"status": "ERROR", "mensaje": "Proceso no encontrado"}, true
	}

	dispositivoSeleccionado := SeleccionarDispositivoIO(dispositivo, pcb.PID)
	MoverProcesoABlocked(pcb, fmt.Sprintf("IO_%s", dispositivoSeleccionado))
	go EnviarSolicitudIO(pcb, dispositivoSeleccionado, int(tiempoFloat))
	go despacharProcesoSiCorresponde()

	return map[string]interface{}{"status": "OK", "mensaje": "IO procesando"}, true
}

// ProcesarIOTerminada optimizado
func ProcesarIOTerminada(datos map[string]interface{}) (interface{}, bool) {
	evento, _ := datos["evento"].(string)
	operacion, _ := datos["operacion"].(string)

	if evento != "IO_TERMINADA" && operacion != "IO_COMPLETADA" {
		return nil, false
	}

	pidFloat, pidOk := datos["pid"].(float64)
	if !pidOk {
		return map[string]interface{}{"status": "ERROR", "mensaje": "PID inválido"}, true
	}

	pcb := BuscarPCBPorPID(int(pidFloat))
	if pcb == nil {
		return map[string]interface{}{"status": "ERROR", "mensaje": "Proceso no encontrado"}, true
	}

	utils.InfoLog.Info(fmt.Sprintf("(%d) - Finalizó IO y pasa a READY", pcb.PID))
	utils.InfoLog.Info("IO finalizada, proceso pasa a READY", "pid", pcb.PID)
	pcb.PC++

	// Manejar transiciones según el estado actual
	switch pcb.Estado {
	case EstadoBlocked:
		// BLOCKED -> READY (proceso en memoria)
		utils.InfoLog.Info("IO finalizada, proceso pasa a READY", "pid", pcb.PID)
		MoverProcesoAReady(pcb)
		go despacharProcesoSiCorresponde()

	case EstadoSuspBlocked:
		// SUSP.BLOCKED -> SUSP.READY (proceso en swap)
		utils.InfoLog.Info("IO finalizada, proceso pasa a SUSP.READY", "pid", pcb.PID)
		MoverProcesoASuspReady(pcb)
	}

	return map[string]interface{}{"status": "OK", "mensaje": "IO completada"}, true
}

// SeleccionarDispositivoIO implementa balanceador de carga
func SeleccionarDispositivoIO(dispositivoSolicitado string, pid int) string {
	dispositivosIOMutex.RLock()
	defer dispositivosIOMutex.RUnlock()

	// Si el dispositivo existe directamente, usarlo
	if _, existe := dispositivosIO[dispositivoSolicitado]; existe {
		utils.InfoLog.Info("Usando dispositivo directo", "pid", pid, "dispositivo", dispositivoSolicitado)
		return dispositivoSolicitado
	}

	// Buscar dispositivos similares para distribución automática
	dispositivosSimilares := obtenerDispositivosSimilares(dispositivoSolicitado)
	if len(dispositivosSimilares) == 0 {
		utils.ErrorLog.Error("No hay dispositivos disponibles", "dispositivo_solicitado", dispositivoSolicitado, "pid", pid)
		return dispositivoSolicitado
	}

	// Round-robin para distribuir carga
	balanceadorMutex.Lock()
	dispositivoSeleccionado := dispositivosSimilares[contadorBalanceador%len(dispositivosSimilares)]
	contadorBalanceador++
	balanceadorMutex.Unlock()

	utils.InfoLog.Info("Balanceador IO", "pid", pid, "solicitado", dispositivoSolicitado, "seleccionado", dispositivoSeleccionado)
	return dispositivoSeleccionado
}

// obtenerDispositivosSimilares devuelve lista de dispositivos IO disponibles
func obtenerDispositivosSimilares(dispositivoBase string) []string {
	var dispositivos []string

	for nombre := range dispositivosIO {
		dispositivos = append(dispositivos, nombre)
	}

	sort.Strings(dispositivos)
	return dispositivos
}
