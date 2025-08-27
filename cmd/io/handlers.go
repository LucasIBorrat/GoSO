package main

import (
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Handler para handshake
func handlerHandshake(msg *utils.Mensaje) (interface{}, error) {
	utils.InfoLog.Info("Handshake recibido", "origen", msg.Origen)
	return map[string]interface{}{"status": "OK"}, nil
}

// Handler para operaciones IO
func handlerOperacion(msg *utils.Mensaje) (interface{}, error) {
	return utils.HandlerGenerico(msg, config.RetardoBase, procesarOperacion)
}

func conectarConReintentos(cliente *utils.HTTPClient, nombreModulo string, datosHandshake map[string]interface{}) {
    utils.InfoLog.Info("Iniciando conexión", "destino", nombreModulo)

    for i := 1; ; i++ {
        _, err := cliente.EnviarHTTPMensaje(utils.MensajeHandshake, "handshake", datosHandshake)
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