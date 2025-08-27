package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

func main() {
	// Inicializar loggers
	utils.InicializarLogger("INFO", "kernel")

	utils.InfoLog.Info("Kernel iniciando", "args", os.Args)

	// Verificar argumentos mínimos
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Uso: %s <archivo_configuracion> <archivo_pseudocódigo> <tamaño>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Ejemplo: %s configs/kernel-config-PlaniCortoFIFO scripts/PLANI_CORTO_PLAZO 0\n", os.Args[0])
		os.Exit(1)
	}

	// Obtener parámetros
	configPath := os.Args[1]                    // configs/kernel-config-PlaniCortoFIFO
	nombreArchivoInicial := os.Args[2]          // scripts/PLANI_CORTO_PLAZO
	tamanioInicial, err := strconv.Atoi(os.Args[3])  // 0
	if err != nil {
		utils.ErrorLog.Error("El tamaño del proceso inicial debe ser un número entero", "error", err, "valor", os.Args[3])
		os.Exit(1)
	}

	// Verificar que el archivo de configuración existe
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		utils.ErrorLog.Error("El archivo de configuración no existe", "archivo", configPath)
		os.Exit(1)
	}

	utils.InfoLog.Info("Parámetros procesados", 
		"config", configPath, 
		"script", nombreArchivoInicial, 
		"tamaño", tamanioInicial)

	// Inicializar kernel
	err = inicializarKernel(configPath)
	if err != nil {
		utils.ErrorLog.Error("Error durante la inicialización del Kernel", "error", err)
		os.Exit(1)
	}

	// Crear proceso inicial
	crearYAdmitirProcesoInicial(nombreArchivoInicial, tamanioInicial)

	utils.InfoLog.Info("Kernel listo y esperando conexiones")

	// Esperar Enter para iniciar planificadores
	fmt.Println("Presione ENTER para iniciar los planificadores...")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	utils.InfoLog.Info("Enter presionado, iniciando planificadores")
	fmt.Println("Planificadores iniciados. Sistema funcionando...")

	iniciarPlanificadores()

	// Configurar manejo de señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Esperar señal de terminación
	<-sigChan
	utils.InfoLog.Info("Ctrl+C recibido. Finalizando Kernel")
	fmt.Println("\nKernel finalizando...")
	os.Exit(0)
}
