package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
)

// Modulo representa un módulo genérico del sistema
type Modulo struct {
	Nombre      string
	Server      *HTTPServer
	Clientes    map[string]*HTTPClient
	ConfigPath  string
	HandlerFunc map[string]map[string]HTTPHandlerFunc
}

// NuevoModulo crea una nueva instancia de un módulo
func NuevoModulo(nombre string, configPath string) *Modulo {
	return &Modulo{
		Nombre:      nombre,
		Clientes:    make(map[string]*HTTPClient),
		ConfigPath:  configPath,
		HandlerFunc: make(map[string]map[string]HTTPHandlerFunc),
	}
}

// RegistrarHandler registra un handler para un tipo de mensaje y operación específicos
func (m *Modulo) RegistrarHandler(tipo string, operacion string, handler HTTPHandlerFunc) {
	if _, existe := m.HandlerFunc[tipo]; !existe {
		m.HandlerFunc[tipo] = make(map[string]HTTPHandlerFunc)
	}
	m.HandlerFunc[tipo][operacion] = handler
}

// IniciarServidor crea e inicializa el servidor HTTP del módulo
func (m *Modulo) IniciarServidor(ip string, puerto int) {
	m.Server = NewHTTPServer(ip, puerto, m.Nombre)

	// Registrar handlers para el servidor HTTP
	for tipoStr, handlersPorOperacion := range m.HandlerFunc {
		tipo, err := strconv.Atoi(tipoStr)
		if err != nil {
			slog.Error("Error al convertir tipo de mensaje a entero", "tipo", tipoStr, "error", err)
			continue
		}

		m.Server.RegisterHTTPHandler(tipo, func(msg *Mensaje) (interface{}, error) {
			operacion := msg.Operacion
			if operacion == "" {
				operacion = "default"
			}

			handler, existe := handlersPorOperacion[operacion]
			if !existe {
				handler, existe = handlersPorOperacion["default"]
				if !existe {
					slog.Error("No hay handler para operación", "tipo", tipo, "operacion", operacion)
					return nil, fmt.Errorf("no hay handler para operación %s", operacion)
				}
			}

			return handler(msg)
		})
	}

	go func() {
		err := m.Server.Start()
		if err != nil {
			slog.Error("Error al iniciar servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Servidor HTTP iniciado", "módulo", m.Nombre, "dirección", fmt.Sprintf("%s:%d", ip, puerto))
}


func CargarConfiguracion[T any](ruta string) *T {
	slog.Info("Cargando configuración", "ruta", ruta)

	// Crear directorio si no existe
	dir := filepath.Dir(ruta)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("Error al crear directorio de configuración", "error", err)
		os.Exit(1)
	}

	// Obtener ruta absoluta
	absPath, err := filepath.Abs(ruta)
	if err != nil {
		slog.Error("Error obteniendo ruta absoluta", "error", err, "ruta", ruta)
		os.Exit(1)
	}

	// Abrir archivo
	file, err := os.Open(absPath)
	if err != nil {
		slog.Error("Error abriendo archivo de configuración", "error", err, "archivo", absPath)
		os.Exit(1)
	}
	defer file.Close()

	// Decodificar JSON directamente al tipo genérico
	var config T
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		slog.Error("Error decodificando configuración", "error", err, "archivo", absPath)
		os.Exit(1)
	}

	slog.Info("Configuración cargada correctamente")
	return &config
}

// ============================================================================
// Constantes para tipos de mensajes entre módulos
// ============================================================================
const (
    // === COMUNICACIÓN BÁSICA (1-9) ===
    MensajeHandshake = 1  // Conexión inicial
    MensajeOperacion = 2  // Operaciones genéricas
    
    // === OPERACIONES DE MEMORIA (10-19) ===
    MensajeLeer         = 10  // Leer datos
    MensajeEscribir     = 11  // Escribir datos
    MensajeObtenerMarco = 12  // Obtener marco
    MensajeFetch        = 13  // Fetch instrucción
    MensajeEspacioLibre = 14  // Consultar espacio
    MensajeMemoryDump   = 15  // Volcado memoria
    
    // === GESTIÓN DE PROCESOS (20-29) ===
    MensajeInicializarProceso  = 20  // Crear proceso
    MensajeFinalizarProceso    = 21  // Terminar proceso
    MensajeSuspenderProceso    = 22  // Suspender proceso
    MensajeDessuspenderProceso = 23  // Reactivar proceso
    
    // === EJECUCIÓN DE CPU (30-39) ===
    MensajeEjecutar           = 30  // Ejecutar en CPU
    MensajeObtenerInstruccion = 31  // Obtener instrucción
    MensajeInterrupcion       = 32  // Interrumpir CPU
)