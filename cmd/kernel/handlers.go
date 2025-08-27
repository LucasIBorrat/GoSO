package main

import (
	"fmt"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// HandlerHandshake optimizado
func HandlerHandshake(msg *utils.Mensaje) (interface{}, error) {
	utils.InfoLog.Info("Handshake recibido", "origen", msg.Origen)

	datosMap, ok := msg.Datos.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Error("Datos inválidos en handshake", "datos", fmt.Sprintf("%v", msg.Datos))
		return map[string]interface{}{"status": "ERROR", "message": "Datos inválidos"}, nil
	}

	// Procesar IO
	if respuesta, manejado := ManejadorRegistroIO(msg.Origen, datosMap); manejado {
		utils.InfoLog.Info("Procesado como dispositivo IO", "origen", msg.Origen)
		return respuesta, nil
	}

	// Procesar CPU
	if esCPU(msg.Origen, datosMap) {
		utils.InfoLog.Info("Procesando como CPU", "origen", msg.Origen)
		respuesta, _ := manejarRegistroCPU(msg.Origen, datosMap)
		return respuesta, nil
	}

	utils.InfoLog.Info("Handshake genérico completado", "origen", msg.Origen)
	return map[string]interface{}{"status": "OK", "message": "Handshake recibido"}, nil
}

// esCPU simplificado
func esCPU(origen string, datos map[string]interface{}) bool {
	return origen == "CPU" ||
		datos["tipo"] == "CPU" ||
		datos["nombre"] == "CPU"
}

// manejarRegistroCPU optimizado
func manejarRegistroCPU(origen string, datos map[string]interface{}) (interface{}, error) {
	ip, ipOk := datos["ip"].(string)
	if !ipOk {
		return map[string]interface{}{"status": "ERROR", "message": "IP requerida"}, nil
	}

	puerto, puertoOk := extraerPuerto(datos["puerto"])
	if !puertoOk {
		return map[string]interface{}{"status": "ERROR", "message": "Puerto inválido"}, nil
	}

	// Usar identificador específico de la CPU
	identificadorCPU := origen
	if id, existe := datos["identificador"].(string); existe && id != "" {
		identificadorCPU = id
	}

	// Registro síncrono
	registrarCPU(identificadorCPU, ip, puerto)

	utils.InfoLog.Info("CPU registrada", "identificador", identificadorCPU, "ip", ip, "puerto", puerto)

	return map[string]interface{}{"status": "OK", "message": fmt.Sprintf("CPU %s registrada", identificadorCPU)}, nil
}

// extraerPuerto helper
func extraerPuerto(puerto interface{}) (int, bool) {
	switch p := puerto.(type) {
	case float64:
		return int(p), true
	case int:
		return p, true
	case string:
		if val, err := strconv.Atoi(p); err == nil {
			return val, true
		}
	}
	return 0, false
}

func HandlerOperacion(msg *utils.Mensaje) (interface{}, error) {
	return procesarOperacionEspecifica(msg)
}

// procesarOperacionEspecifica con pipeline optimizado
func procesarOperacionEspecifica(msg *utils.Mensaje) (interface{}, error) {
	datos, ok := msg.Datos.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"status": "ERROR", "mensaje": "Datos inválidos"}, nil
	}

	pid, pidOk := extraerPID(datos["pid"])
	if !pidOk {
		if _, esFinIO := datos["evento"].(string); !esFinIO {
			return map[string]interface{}{"status": "ERROR", "mensaje": "PID inválido o faltante"}, nil
		}
	}

	// Pipeline de procesamiento
	handlers := []func(int, map[string]interface{}) (interface{}, bool){
		ProcesarRetornoCPU,
		func(pid int, datos map[string]interface{}) (interface{}, bool) {
			return ProcesarSolicitudIO(datos)
		},
		func(pid int, datos map[string]interface{}) (interface{}, bool) {
			return ProcesarIOTerminada(datos)
		},
		procesarFinalizacionSiCorresponde,
	}

	for _, handler := range handlers {
		if respuesta, manejado := handler(pid, datos); manejado {
			return respuesta, nil
		}
	}

	return map[string]interface{}{"status": "ERROR", "mensaje": "Operación desconocida o no manejada"}, nil
}

// ProcesarRetornoCPU maneja retorno de procesos desde CPU
func ProcesarRetornoCPU(pid int, datos map[string]interface{}) (interface{}, bool) {
	motivo, ok := datos["motivo_retorno"].(string)
	if !ok {
		return nil, false
	}

	pcb := BuscarPCBPorPID(pid)
	if pcb == nil {
		utils.ErrorLog.Warn("Retorno de CPU para PID inexistente", "pid", pid)
		return map[string]interface{}{"status": "ERROR", "mensaje": "PID no encontrado"}, true
	}

	liberarCPU(pid)

	switch motivo {
	case "INTERRUPTED":
		utils.InfoLog.Info("Proceso interrumpido por Kernel", "pid", pid)
		MoverProcesoAReady(pcb)
		go despacharProcesoSiCorresponde()
		return map[string]interface{}{"status": "OK", "message": "Proceso movido a READY por interrupción"}, true

	case "SYSCALL_IO":
		return nil, false

	default:
		return nil, false
	}
}

// liberarCPU encuentra y libera la CPU que ejecutaba un proceso
func liberarCPU(pid int) string {
	execMutex.Lock()
	defer execMutex.Unlock()

	var cpuLiberada string
	for cpu, pcbEnExec := range colaExec {
		if pcbEnExec != nil && pcbEnExec.PID == pid {
			delete(colaExec, cpu)
			cpuLiberada = cpu
			break
		}
	}
	
	if cpuLiberada != "" {
		utils.InfoLog.Info("CPU liberada", "cpu", cpuLiberada, "pid", pid)
	}
	
	return cpuLiberada
}

// extraerPID helper
func extraerPID(pid interface{}) (int, bool) {
	if pidFloat, ok := pid.(float64); ok {
		return int(pidFloat), true
	}
	return 0, false
}

// procesarFinalizacionSiCorresponde con detección optimizada
func procesarFinalizacionSiCorresponde(pid int, datos map[string]interface{}) (interface{}, bool) {
	evento, _ := datos["evento"].(string)
	motivoRetorno, _ := datos["motivo_retorno"].(string)

	if evento == "PROCESO_TERMINADO" || motivoRetorno == "EXIT" || motivoRetorno == "ERROR" {
		liberarCPU(pid)
		respuesta, _ := procesarFinalizacion(pid, datos)
		return respuesta, true
	}
	return nil, false
}

// procesarFinalizacion optimizado
func procesarFinalizacion(pid int, datos map[string]interface{}) (interface{}, error) {
	pcb := BuscarPCBPorPID(pid)
	if pcb == nil {
		return map[string]interface{}{"status": "ERROR", "mensaje": "Proceso no encontrado"}, nil
	}

	motivo := determinarMotivo(datos)
	FinalizarProceso(pcb, motivo)

	// Operaciones post-finalización en paralelo
	go func() {
		intentarAdmitirProceso()
		despacharProcesoSiCorresponde()
	}()

	return map[string]interface{}{"status": "OK", "mensaje": "Proceso finalizado"}, nil
}

// determinarMotivo helper
func determinarMotivo(datos map[string]interface{}) string {
	if m, ok := datos["motivo"].(string); ok {
		return m
	}
	if motivoRetorno, ok := datos["motivo_retorno"].(string); ok && motivoRetorno == "ERROR" {
		return "ERROR_CPU"
	}
	return "EXIT_NORMAL"
}