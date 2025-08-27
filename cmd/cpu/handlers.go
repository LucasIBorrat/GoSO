package main

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Registrar todos los handlers de mensajes
func RegistrarHandlers() {
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeHandshake), "handshake", manejarHandshake)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeOperacion), "EJECUTAR_PROCESO", manejarEjecutar)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeEjecutar), "default", manejarEjecutar)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeInterrupcion), "INTERRUPCION", manejarInterrupcion)
	
	utils.InfoLog.Info("Handlers registrados correctamente")
}

func manejarHandshake(msg *utils.Mensaje) (interface{}, error) {
    utils.InfoLog.Info("Handshake recibido", "origen", msg.Origen)
    return map[string]interface{}{"status": "OK"}, nil
}

// Handler para ejecutar instrucción
func manejarEjecutar(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, okPid := datos["pid"].(float64)
	pc, okPc := datos["pc"].(float64)

	if !okPid || !okPc {
		utils.ErrorLog.Error("Formato de mensaje incorrecto", "datos", fmt.Sprintf("%v", datos))
		return map[string]interface{}{
			"error": "Formato de mensaje incorrecto",
		}, nil
	}

	pidInt := int(pid)
	pcInt := int(pc)

	utils.InfoLog.Info("Proceso recibido para ejecutar", "pid", pidInt, "pc", pcInt)

	// Ejecutar ciclo de instrucción
	siguientePC, motivo, parametrosSyscall := ejecutarCiclo(pidInt, pcInt)

	// Preparar respuesta
	respuesta := map[string]interface{}{
		"pid": pidInt,
		"pc":  siguientePC,
	}

	// Agregar motivo de retorno si existe
	if motivo != "" {
		respuesta["motivo_retorno"] = motivo
		if parametrosSyscall != nil {
			respuesta["parametros"] = parametrosSyscall
		}
	}

	utils.InfoLog.Info("Proceso devuelto al Kernel", "pid", pidInt, "pc", siguientePC, "motivo", motivo)

	return respuesta, nil
}

// Handler para interrupciones
func manejarInterrupcion(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)

	if !ok {
		utils.ErrorLog.Error("Formato de interrupción incorrecto", "datos", fmt.Sprintf("%v", datos))
		return map[string]interface{}{
			"error": "Formato de interrupción incorrecto",
		}, nil
	}

	pidInt := int(pid)

	mutex.Lock()
	interrupcionPendiente = true
	pidInterrumpido = pidInt
	mutex.Unlock()

	utils.InfoLog.Info("Interrupción configurada", "pid", pidInt)

	return map[string]interface{}{"ok": true}, nil
}

func conectarConReintentos(c *utils.HTTPClient, nombreModulo string, datosHandshake map[string]interface{}) {
	utils.InfoLog.Info("Iniciando conexión", "destino", nombreModulo)

	for i := 1; ; i++ {
		_, err := c.EnviarHTTPMensaje(utils.MensajeHandshake, "handshake", datosHandshake)
		if err == nil {
			utils.InfoLog.Info("Conexión establecida", "destino", nombreModulo)
			return
		}

		utils.InfoLog.Warn("Reintentando conexión",
			"destino", nombreModulo,
			"intento", i,
			"próximo_en", "2s")
		time.Sleep(2 * time.Second)
	}
}