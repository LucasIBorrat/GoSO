package main

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// KernelConfig define la configuración del módulo Kernel
type KernelConfig struct {
	IPKernel               string  `json:"IP_KERNEL"`
	PortKernel             int     `json:"PUERTO_KERNEL"`
	IPMemory               string  `json:"IP_MEMORIA"`
	PortMemory             int     `json:"PUERTO_MEMORIA"`
	LogLevel               string  `json:"LOG_LEVEL"`
	SchedulerAlgorithm     string  `json:"ALGORITMO_CORTO_PLAZO"`
	ReadyIngressAlgorithm  string  `json:"ALGORITMO_INGRESO_A_READY"`
	Alpha                  float64 `json:"ALFA"`
	InitialEstimate        int     `json:"ESTIMACION_INICIAL"`
	SuspensionTime         int     `json:"TIEMPO_SUSPENSION"`
	GradoMultiprogramacion int     `json:"GRADO_MULTIPROGRAMACION"`
	ScriptsPath            string  `json:"SCRIPTS_PATH,omitempty"`
}

var (
	kernelModulo  *utils.Modulo
	kernelConfig  *KernelConfig
	memoriaClient *utils.HTTPClient
)

// inicializarKernel optimizado
func inicializarKernel(configPath string) error {
	kernelModulo = utils.NuevoModulo("Kernel", configPath)
	kernelConfig = utils.CargarConfiguracion[KernelConfig](configPath)

	utils.InicializarLogger(kernelConfig.LogLevel, "Kernel")
	utils.InfoLog.Info("Inicializando Kernel", "config_path", configPath)

	// Inicializar el mapa de CPUs ANTES de cualquier otra operación
	inicializarMapaCPUs()

	InicializarPlanificador(kernelConfig)

	// Inicializar y conectar con Memoria
	memoriaClient = utils.NewHTTPClient(kernelConfig.IPMemory, kernelConfig.PortMemory, "Kernel->Memoria")
	if err := conectarAMemoria(10); err != nil {
		utils.ErrorLog.Error("No se pudo conectar con Memoria", "error", err)
		return err
	}

	registrarHandlers()
	kernelModulo.IniciarServidor(kernelConfig.IPKernel, kernelConfig.PortKernel)

	utils.InfoLog.Info("Kernel inicializado correctamente")
	return nil
}

// registrarHandlers registra todos los manejadores HTTP
func registrarHandlers() {
	kernelModulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeHandshake), "handshake", HandlerHandshake)
	kernelModulo.RegistrarHandler(fmt.Sprintf("%d", utils.MensajeOperacion), "default", HandlerOperacion)
	
	utils.InfoLog.Info("Handlers registrados correctamente")
}

// conectarAMemoria intenta conectar con el módulo de Memoria con reintentos
func conectarAMemoria(intentosMax int) error {
	utils.InfoLog.Info("Conectando con Memoria", "intentos_max", intentosMax)
	
	for i := 0; i < intentosMax; i++ {
		err := memoriaClient.VerificarConexion()
		if err == nil {
			utils.InfoLog.Info("Conexión establecida con Memoria")
			return nil
		}
		
		utils.InfoLog.Warn("Fallo al conectar con Memoria, reintentando", "intento", i+1, "error", err)
		time.Sleep(3 * time.Second)
	}
	
	return fmt.Errorf("no se pudo establecer conexión después de %d intentos", intentosMax)
}

// crearYAdmitirProcesoInicial crea el PCB inicial y lo coloca en NEW
func crearYAdmitirProcesoInicial(nombreArchivo string, tamanio int) {
	utils.InfoLog.Info("Creando proceso inicial", "archivo", nombreArchivo, "tamaño", tamanio)
	
	pcb := NuevoPCB(-1, tamanio) // Usar -1 para generar PID 0
	pcb.NombreArchivo = nombreArchivo

	utils.InfoLog.Info("Proceso inicial creado", "pid", pcb.PID, "estado", "NEW")
	AgregarProcesoANew(pcb)
}

// iniciarPlanificadores se llama después de presionar Enter
func iniciarPlanificadores() {
	utils.InfoLog.Info("Iniciando planificadores")
	go PlanificarLargoPlazo()
	go PlanificarCortoPlazo()
	utils.InfoLog.Info("Planificadores iniciados")
}

// inicializarMapaCPUs inicializa el mapa de CPUs durante el arranque del kernel
func inicializarMapaCPUs() {
	InicializarMapaCPUs()
	utils.InfoLog.Info("Mapa de CPUs inicializado correctamente")
}

// GetMemoriaClient proporciona acceso seguro al cliente de memoria
func GetMemoriaClient() *utils.HTTPClient {
	if memoriaClient == nil {
		utils.ErrorLog.Error("Cliente de memoria no inicializado")
		return nil
	}
	return memoriaClient
}
