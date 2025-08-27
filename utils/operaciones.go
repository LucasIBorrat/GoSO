package utils

import (
	"log/slog"
	"time"
)

// AplicarRetardo aplica un retardo simulado y lo registra
func AplicarRetardo(operacion string, duracionMs int) {
	slog.Info("Aplicando retardo", "operación", operacion, "duración_ms", duracionMs)
	time.Sleep(time.Duration(duracionMs) * time.Millisecond)
	slog.Info("Retardo completado", "operación", operacion)
}

// ExtraerRetardo extrae el retardo de una operación del mensaje
func ExtraerRetardo(msg *Mensaje, valorPorDefecto int) int {
	if datosMap, ok := msg.Datos.(map[string]interface{}); ok {
		if retardo, ok := datosMap["retardo"].(float64); ok {
			return int(retardo)
		}
	}
	return valorPorDefecto
}

// ObtenerTipoOperacion obtiene el tipo de operación del mensaje
func ObtenerTipoOperacion(msg *Mensaje, valorPorDefecto string) string {
	if datosMap, ok := msg.Datos.(map[string]interface{}); ok {
		if tipo, ok := datosMap["tipo"].(string); ok {
			return tipo
		}
	}
	return valorPorDefecto
}

// HandlerGenerico es un handler genérico para tratar operaciones con retardo
func HandlerGenerico(msg *Mensaje, retardoPorDefecto int, procesador func(msg *Mensaje) (interface{}, error)) (interface{}, error) {
	slog.Info("Operación recibida", "origen", msg.Origen, "tipo", msg.Tipo)

	retardo := ExtraerRetardo(msg, retardoPorDefecto)
	AplicarRetardo("procesamiento", retardo)

	return procesador(msg)
}
