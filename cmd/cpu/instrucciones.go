package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Fetch: Obtener instrucción desde memoria
func fetch(pid, pc int) string {
	utils.InfoLog.Info(fmt.Sprintf("PID: %d - FETCH - PC: %d", pid, pc))

	params := map[string]interface{}{
		"pid": pid,
		"pc":  pc,
	}

	respuesta, err := memoriaClient.EnviarHTTPMensaje(utils.MensajeFetch, "FETCH", params)
	if err != nil {
		utils.ErrorLog.Error("Error al solicitar instrucción a memoria", "error", err)
		return ""
	}

	respuestaMap, ok := respuesta.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Error("Formato de respuesta incorrecto", "respuesta", fmt.Sprintf("%v", respuesta))
		return ""
	}

	instruccion, ok := respuestaMap["instruccion"].(string)
	if !ok {
		utils.ErrorLog.Error("Formato de instrucción incorrecto", "respuesta", fmt.Sprintf("%v", respuestaMap))
		return ""
	}

	utils.InfoLog.Info("Instrucción obtenida", "pid", pid, "pc", pc, "instruccion", instruccion)
	return instruccion
}

// Decode y Execute: Interpretar y ejecutar instrucción
func decodeAndExecute(pid, pc int, instruccion string) (int, string, map[string]interface{}) {
	partes := strings.Fields(instruccion)
	if len(partes) == 0 {
		utils.ErrorLog.Error("Instrucción vacía", "pid", pid, "pc", pc)
		return pc, "ERROR", nil
	}

	operacion := partes[0]
	parametros := partes[1:]

	// Construir string de parámetros para el log
	var argsString string
	if len(parametros) > 0 {
		argsString = strings.Join(parametros, " ")
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Ejecutando: %s %s", pid, operacion, argsString))

	parametrosSyscall := make(map[string]interface{})
	motivoRetorno := ""
	siguientePC := pc

	switch operacion {
	case "NOOP":
		// No hacer nada, solo consumir tiempo

	case "WRITE":
		if len(parametros) >= 2 {
			direccion, err := strconv.Atoi(parametros[0])
			if err != nil {
				utils.ErrorLog.Error("Error en dirección WRITE", "error", err)
				motivoRetorno = "ERROR"
				break
			}
			datos := parametros[1]
			escribirEnMemoria(pid, direccion, datos)
		} else {
			utils.ErrorLog.Error("WRITE: parámetros insuficientes", "parametros", parametros)
			motivoRetorno = "ERROR"
		}

	case "READ":
		if len(parametros) >= 2 {
			direccion, err1 := strconv.Atoi(parametros[0])
			tamano, err2 := strconv.Atoi(parametros[1])
			if err1 != nil || err2 != nil {
				utils.ErrorLog.Error("Error en parámetros READ", "err1", err1, "err2", err2)
				motivoRetorno = "ERROR"
				break
			}
			leerDeMemoria(pid, direccion, tamano)
		} else {
			utils.ErrorLog.Error("READ: parámetros insuficientes", "parametros", parametros)
			motivoRetorno = "ERROR"
		}

	case "GOTO":
		if len(parametros) >= 1 {
			nuevoPC, err := strconv.Atoi(parametros[0])
			if err != nil {
				utils.ErrorLog.Error("Error en GOTO", "error", err)
				motivoRetorno = "ERROR"
				break
			}
			siguientePC = nuevoPC
			utils.InfoLog.Info("GOTO ejecutado", "pid", pid, "nuevo_pc", nuevoPC)
		} else {
			utils.ErrorLog.Error("GOTO: parámetros insuficientes", "parametros", parametros)
			motivoRetorno = "ERROR"
		}

	case "IO":
		if len(parametros) >= 2 {
			dispositivo := parametros[0]
			tiempo, err := strconv.Atoi(parametros[1])
			if err != nil {
				utils.ErrorLog.Error("Error en tiempo IO", "error", err)
				motivoRetorno = "ERROR"
				break
			}
			parametrosSyscall["dispositivo"] = dispositivo
			parametrosSyscall["tiempo"] = tiempo
			motivoRetorno = "SYSCALL_IO"
			utils.InfoLog.Info("IO solicitado", "pid", pid, "dispositivo", dispositivo, "tiempo", tiempo)
		} else {
			utils.ErrorLog.Error("IO: parámetros insuficientes", "parametros", parametros)
			motivoRetorno = "ERROR"
		}

	case "INIT_PROC":
		if len(parametros) >= 2 {
			archivo := parametros[0]
			tamano, err := strconv.Atoi(parametros[1])
			if err != nil {
				utils.ErrorLog.Error("Error en tamaño INIT_PROC", "error", err)
				motivoRetorno = "ERROR"
				break
			}
			parametrosSyscall["archivo"] = archivo
			parametrosSyscall["tamano"] = tamano
			motivoRetorno = "SYSCALL_INIT_PROC"
			utils.InfoLog.Info("INIT_PROC solicitado", "pid", pid, "archivo", archivo, "tamano", tamano)
		} else {
			utils.ErrorLog.Error("INIT_PROC: parámetros insuficientes", "parametros", parametros)
			motivoRetorno = "ERROR"
		}

	case "DUMP_MEMORY":
		motivoRetorno = "SYSCALL_DUMP_MEMORY"
		utils.InfoLog.Info("DUMP_MEMORY solicitado", "pid", pid)

	case "EXIT":
		motivoRetorno = "EXIT"
		utils.InfoLog.Info("EXIT ejecutado", "pid", pid)

	default:
		utils.ErrorLog.Error("Instrucción desconocida", "operacion", operacion)
		motivoRetorno = "ERROR"
	}

	return siguientePC, motivoRetorno, parametrosSyscall
}
