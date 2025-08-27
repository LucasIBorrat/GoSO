package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var (
	modulo     *utils.Modulo
	httpServer interface{}
)

func main() {
	// Verificar argumentos
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Uso: %s <archivo_configuracion>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Ejemplo: %s configs/memoria-config-PlaniCorto.json\n", os.Args[0])
		os.Exit(1)
	}

	// Inicializar logger ANTES de usarlo
	utils.InicializarLogger("INFO", "Memoria")

	utils.InfoLog.Info("Iniciando módulo Memoria")

	// Inicializar módulo
	inicializarModulo()

	utils.InfoLog.Info("Memoria inicializada correctamente")

	// Mantener el programa corriendo
	select {}
}

func inicializarModulo() {
	// Usar el archivo de configuración pasado como argumento
	rutaConfig := os.Args[1]

	// Verificar que el archivo existe
	if _, err := os.Stat(rutaConfig); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: El archivo de configuración no existe: %s\n", rutaConfig)
		os.Exit(1)
	}

	// Crear módulo
	modulo = utils.NuevoModulo("Memoria", rutaConfig)

	// Cargar configuración
	config = utils.CargarConfiguracion[MemoryConfig](rutaConfig)

	// Actualizar logger con configuración del archivo
	utils.InicializarLogger(config.LogLevel, "Memoria")
	utils.InfoLog.Info("Configuración cargada", "nivel_log", config.LogLevel, "config_path", rutaConfig)

	// Verificar directorio de dumps
	if err := os.MkdirAll(config.DumpPath, 0755); err != nil {
		utils.InfoLog.Warn("No se pudo crear directorio para dumps", "error", err)
	} else {
		utils.InfoLog.Info("Directorio para dumps verificado", "ruta", config.DumpPath)
	}

	// Inicializar componentes
	inicializarMemoria()
	inicializarMetricas()

	// Inicializar mapa de instrucciones
	instruccionesPorProceso = make(map[int][]string)
	utils.InfoLog.Info("Mapa de instrucciones inicializado")

	// Registrar handlers
	registrarHandlers()

	// Iniciar servidor
	modulo.IniciarServidor(config.IPMemory, config.PortMemory)
	utils.InfoLog.Info("Servidor iniciado", "ip", config.IPMemory, "puerto", config.PortMemory)

	httpServer = modulo.Server
}

func registrarHandlers() {
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeHandshake), "handshake", handlerHandshake)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeOperacion), "default", handlerOperacion)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeObtenerInstruccion), "default", handlerObtenerInstruccion)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeFetch), "default", handlerObtenerInstruccion)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeEspacioLibre), "default", handlerEspacioLibre)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeInicializarProceso), "default", handlerInicializarProceso)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeFinalizarProceso), "default", handlerFinalizarProceso)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeLeer), "default", handlerLeerMemoria)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeEscribir), "default", handlerEscribirMemoria)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeObtenerMarco), "default", handlerObtenerMarco)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeSuspenderProceso), "default", handlerSuspenderProceso)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeDessuspenderProceso), "default", handlerDessuspenderProceso)
	modulo.RegistrarHandler(strconv.Itoa(utils.MensajeMemoryDump), "default", handlerMemoryDump)

	utils.InfoLog.Info("Handlers registrados correctamente")
}

// Handler para handshake
func handlerHandshake(msg *utils.Mensaje) (interface{}, error) {
	utils.InfoLog.Info("Handshake recibido", "origen", msg.Origen)

	// Aplicar retardo de memoria
	utils.AplicarRetardo("handshake", config.MemoryDelay)

	return map[string]interface{}{
		"status":           "OK",
		"tam_pagina":       config.PageSize,
		"entradas_por_pag": config.EntriesPerPage,
		"niveles":          config.NumberOfLevels,
	}, nil
}

func procesarOperacion(msg *utils.Mensaje) (interface{}, error) {
	tipoOperacion := utils.ObtenerTipoOperacion(msg, "memoria")
	utils.InfoLog.Info("Operación procesada", "tipo", tipoOperacion)

	return map[string]interface{}{
		"status":  "OK",
		"mensaje": "Operación de memoria completada exitosamente",
	}, nil
}