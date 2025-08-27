package main

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Notificar al Kernel que la operación IO ha terminado
func notificarIOTerminadaAKernel(pid int) {
    datos := map[string]interface{}{
        "evento":    "IO_TERMINADA",
        "operacion": "IO_COMPLETADA",
        "pid":       pid,
        "timestamp": time.Now().UnixNano() / int64(time.Millisecond),
    }

    if kernelClient == nil {
        utils.ErrorLog.Error("Cliente de Kernel no inicializado")
        return
    }

    _, err := kernelClient.EnviarHTTPOperacion("IO_COMPLETADA", datos)
    if err != nil {
        utils.ErrorLog.Error("Error notificando IO terminada a Kernel", "error", err.Error(), "pid", pid)
    } else {
        utils.InfoLog.Info("IO terminada notificada a Kernel", "pid", pid)
    }
}

// Procesar operación IO
func procesarOperacion(msg *utils.Mensaje) (interface{}, error) {
	datos, ok := msg.Datos.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Warn("Formato de datos inválido en operación IO")
		return map[string]interface{}{
			"status":  "ERROR",
			"mensaje": "Formato de datos inválido",
		}, nil
	}

	// Extraer PID
	pidFloat, pidOk := datos["pid"].(float64)
	if !pidOk {
		utils.ErrorLog.Warn("Operación IO sin PID válido", "datos", datos)
		return map[string]interface{}{
			"status":  "ERROR",
			"mensaje": "PID inválido en solicitud IO",
		}, nil
	}
	pid := int(pidFloat)

	// Extraer tiempo
	tiempoFloat, tiempoOk := datos["tiempo"].(float64)
	if !tiempoOk {
		utils.ErrorLog.Warn("Operación IO sin tiempo válido", "datos", datos)
		return map[string]interface{}{
			"status":  "ERROR",
			"mensaje": "Tiempo inválido en solicitud IO",
		}, nil
	}
	tiempo := int(tiempoFloat)

	// Log de inicio de IO
	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Inicio de IO - Tiempo: %d", pid, tiempo))

	// Simular la operación IO con el retardo configurado
	utils.AplicarRetardo("io_operacion", tiempo)

	// Log de fin de IO
	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Fin de IO", pid))

	// Notificar al Kernel que la operación IO ha terminado
	go notificarIOTerminadaAKernel(pid)

	return map[string]interface{}{
		"status":  "OK",
		"mensaje": "Operación I/O completada exitosamente",
	}, nil
}
