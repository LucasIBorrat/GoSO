package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var (
	modulo        *utils.Modulo
	identificador string
	kernelClient  *utils.HTTPClient
	memoriaClient *utils.HTTPClient
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Uso: ./cpu [identificador] [archivo_config_opcional]")
		os.Exit(1)
	}

	identificador = os.Args[1]

	// Inicializar módulo
	inicializarModulo()

	utils.InfoLog.Info("CPU iniciada correctamente", "identificador", identificador)

	// Inicializar componentes de la CPU
	inicializarCPU()

	// Mantener vivo el proceso
	select {}
}

func inicializarModulo() {
	// Determinar archivo de configuración
	var rutaConfig string
	if len(os.Args) >= 3 {
		rutaConfig = os.Args[2]
	} else {
		rutaConfig = filepath.Join("configs", "cpu-config.json")
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(rutaConfig); os.IsNotExist(err) {
		fmt.Printf("Error: El archivo de configuración '%s' no existe\n", rutaConfig)
		os.Exit(1)
	}

	// Crear módulo
	modulo = utils.NuevoModulo("CPU", rutaConfig)

	var loggerName string
    if strings.HasPrefix(identificador, "CPU") {
        loggerName = identificador  // Ya tiene "CPU", usar tal como está
    } else {
        loggerName = fmt.Sprintf("CPU-%s", identificador)  // Agregar prefijo
    }
	utils.InicializarLogger("INFO", loggerName)

	// Cargar configuración
	config = utils.CargarConfiguracion[CPUConfig](rutaConfig)

	// Actualizar nivel de log
	utils.InicializarLogger(config.LogLevel, loggerName)
	utils.InfoLog.Info("Configuración cargada", "nivel_log", config.LogLevel, "config_path", rutaConfig)

	// Datos para el handshake
	datosHandshake := map[string]interface{}{
		"nombre":        "CPU",
		"tipo":          "CPU",
		"ip":            config.IPCPU,
		"puerto":        config.PortCPU,
		"identificador": identificador,
	}

	// Registrar handlers
	RegistrarHandlers()

	// Iniciar servidor
	modulo.IniciarServidor(config.IPCPU, config.PortCPU)
	utils.InfoLog.Info("Servidor iniciado", "ip", config.IPCPU, "puerto", config.PortCPU)

	// Crear clientes HTTP directamente
	kernelClient = utils.NewHTTPClient(config.IPKernel, config.PortKernel, "CPU->Kernel")
	memoriaClient = utils.NewHTTPClient(config.IPMemory, config.PortMemory, "CPU->Memoria")

	utils.InfoLog.Info("Clientes HTTP creados")

	// Conectar con reintentos
	go conectarConReintentos(kernelClient, "Kernel", datosHandshake)
	go conectarConReintentos(memoriaClient, "Memoria", datosHandshake)
}


