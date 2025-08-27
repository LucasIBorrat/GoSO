package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// crearMemoryDump crea un archivo con el contenido completo de la memoria de un proceso
func crearMemoryDump(pid int) error {
	utils.InfoLog.Info("Iniciando memory dump", "pid", pid)

	// Obtener timestamp
	timestamp := time.Now().Format("20060102-150405")

	// Construir nombre de archivo
	nombreArchivo := fmt.Sprintf("%d-%s.dmp", pid, timestamp)
	rutaCompleta := filepath.Join(config.DumpPath, nombreArchivo)

	utils.InfoLog.Info("Archivo de dump generado", "pid", pid, "archivo", nombreArchivo, "ruta", rutaCompleta)

	// Obtener marcos asignados al proceso
	marcos, existe := marcosAsignadosPorProceso[pid]
	if !existe {
		utils.ErrorLog.Error("Proceso sin marcos asignados", "pid", pid)
		return fmt.Errorf("el proceso %d no tiene marcos asignados", pid)
	}

	utils.InfoLog.Info("Marcos del proceso", "pid", pid, "cantidad_marcos", len(marcos))

	// Verificar que el directorio de dumps existe
	if err := os.MkdirAll(config.DumpPath, 0755); err != nil {
		utils.ErrorLog.Error("Error creando directorio dump", "error", err)
		return fmt.Errorf("error al crear directorio para dumps: %v", err)
	}

	// Crear archivo de dump
	dumpFile, err := os.Create(rutaCompleta)
	if err != nil {
		utils.ErrorLog.Error("Error creando archivo dump", "archivo", rutaCompleta, "error", err)
		return fmt.Errorf("error al crear archivo de dump: %v", err)
	}
	defer dumpFile.Close()

	// Calcular tamaño total del proceso
	tamanioTotal := len(marcos) * config.PageSize
	utils.InfoLog.Info("Tamaño total del dump", "pid", pid, "tamanio_bytes", tamanioTotal)

	// Crear buffer para almacenar todo el contenido del proceso
	contenidoProceso := make([]byte, tamanioTotal)

	// Copiar contenido de cada marco al buffer
	for i, marco := range marcos {
		dirFisica := marco * config.PageSize
		copy(contenidoProceso[i*config.PageSize:(i+1)*config.PageSize],
			memoriaPrincipal[dirFisica:dirFisica+config.PageSize])
	}

	// Escribir buffer en el archivo
	_, err = dumpFile.Write(contenidoProceso)
	if err != nil {
		utils.ErrorLog.Error("Error escribiendo dump", "archivo", rutaCompleta, "error", err)
		return fmt.Errorf("error al escribir en archivo de dump: %v", err)
	}

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d Memory Dump solicitado", pid))
	utils.InfoLog.Info("Memory dump completado", "pid", pid, "archivo", nombreArchivo)

	return nil
}

// handlerMemoryDump crea un volcado de memoria para un proceso
func handlerMemoryDump(msg *utils.Mensaje) (interface{}, error) {
	// Extraer el PID del mensaje
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado o formato incorrecto", "datos", datos)
		return map[string]interface{}{
			"error": "PID no proporcionado o formato incorrecto",
		}, nil
	}
	pidInt := int(pid)

	utils.InfoLog.Info("Solicitud de memory dump recibida", "pid", pidInt)

	// Crear memory dump
	err := crearMemoryDump(pidInt)
	if err != nil {
		utils.ErrorLog.Error("Error al crear memory dump", "pid", pidInt, "error", err)
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Aplicar el retardo de memoria
	utils.AplicarRetardo("memory", config.MemoryDelay)

	utils.InfoLog.Info("Memory dump completado exitosamente", "pid", pidInt)

	return map[string]interface{}{
		"status": "OK",
	}, nil
}
