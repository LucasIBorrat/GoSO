package main

import (
	"fmt"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var ( 
	modulo       *utils.Modulo
	kernelClient *utils.HTTPClient
)

func main() {
	// Verificar argumentos mínimos
	if len(os.Args) < 3 {
		fmt.Println("Uso: ./io <nombre_dispositivo> <ruta_configuracion>")
		fmt.Println("Ejemplo: ./io DISCO configs/io1-config.json")
		os.Exit(1)
	}

	nombreDispositivo := os.Args[1]
	rutaConfig := os.Args[2]

	// Verificar que el archivo de configuración existe
	if _, err := os.Stat(rutaConfig); os.IsNotExist(err) {
		fmt.Printf("Error: El archivo de configuración '%s' no existe\n", rutaConfig)
		os.Exit(1)
	}

	// Inicializar módulo
	inicializarModulo(rutaConfig, nombreDispositivo)

	// Mantener vivo el proceso
	select {}
}

func inicializarModulo(rutaConfig string, nombreDispositivo string) {
	// Crear módulo
	modulo = utils.NuevoModulo("IO", rutaConfig)

	// Inicializar logger
	loggerName := fmt.Sprintf("IO-%s", nombreDispositivo)
	utils.InicializarLogger("INFO", loggerName)

	// Cargar configuración
	config = utils.CargarConfiguracion[IOConfig](rutaConfig)

	// Actualizar nivel de log
	utils.InicializarLogger(config.LogLevel, loggerName)

	utils.InfoLog.Info("Módulo IO inicializado",
		"dispositivo", nombreDispositivo,
		"config_path", rutaConfig,
		"ip", config.IPIO,
		"puerto", config.PortIO,
		"nivel_log", config.LogLevel)

	// Registrar handlers
	registrarHandlers()

	// Iniciar servidor
	modulo.IniciarServidor(config.IPIO, config.PortIO)
	utils.InfoLog.Info("Servidor iniciado", "ip", config.IPIO, "puerto", config.PortIO)

	// Crear cliente HTTP directamente
    kernelClient = utils.NewHTTPClient(config.IPKernel, config.PortKernel, "IO->Kernel")
    utils.InfoLog.Info("Cliente HTTP creado")

	// Datos para handshake
	datosHandshake := map[string]interface{}{
		"nombre": nombreDispositivo,
		"tipo":   "IO" + nombreDispositivo,
		"ip":     config.IPIO,
		"puerto": config.PortIO,
	}

	// Conectar con Kernel
	go conectarConReintentos(kernelClient, "Kernel", datosHandshake)
	utils.InfoLog.Info("Conectando a Kernel", "ip", config.IPKernel, "puerto", config.PortKernel)
}


func registrarHandlers() {
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeHandshake), "handshake", handlerHandshake)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeOperacion), "EJECUTAR_PROCESO", handlerOperacion)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeOperacion), "IO_REQUEST", handlerOperacion)
	modulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeEjecutar), "default", handlerOperacion)

	utils.InfoLog.Info("Handlers registrados correctamente")
}