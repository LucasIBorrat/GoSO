package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Estructura para leer config de memoria
type MemoriaConfig struct {
	PortMemory     int    `json:"PUERTO_MEMORIA"`
	IPMemory       string `json:"IP_MEMORIA"`
	MemorySize     int    `json:"TAM_MEMORIA"`
	PageSize       int    `json:"TAM_PAGINA"`
	EntriesPerPage int    `json:"ENTRADAS_POR_TABLA"`
	NumberOfLevels int    `json:"CANTIDAD_NIVELES"`
	MemoryDelay    int    `json:"RETARDO_MEMORIA"`
	SwapfilePath   string `json:"SWAPFILE_PATH"`
	SwapDelay      int    `json:"RETARDO_SWAP"`
	LogLevel       string `json:"LOG_LEVEL"`
	DumpPath       string `json:"DUMP_PATH"`
	ScriptsPath    string `json:"SCRIPTS_PATH"`
}

// Configuración de memoria obtenida dinámicamente
var (
	tamanoPagina     int
	entradasPorTabla int
	numeroDeNiveles  int
	configCargada    bool = false
)

// Cargar configuración directamente desde memoria-config.json
func cargarConfigMemoria() error {
	if configCargada {
		return nil
	}

	utils.InfoLog.Info("Cargando configuración de memoria desde archivo")

	rutaConfigMemoria := filepath.Join("configs", "memoria-config.json")

	if _, err := os.Stat(rutaConfigMemoria); os.IsNotExist(err) {
		utils.ErrorLog.Error("Archivo de configuración de memoria no encontrado", "ruta", rutaConfigMemoria)
		return fmt.Errorf("archivo de configuración de memoria no encontrado: %s", rutaConfigMemoria)
	}

	configMemoria := utils.CargarConfiguracion[MemoriaConfig](rutaConfigMemoria)

	tamanoPagina = configMemoria.PageSize
	entradasPorTabla = configMemoria.EntriesPerPage
	numeroDeNiveles = configMemoria.NumberOfLevels

	configCargada = true

	utils.InfoLog.Info("Configuración de memoria cargada",
		"page_size", tamanoPagina,
		"entries_per_page", entradasPorTabla,
		"number_of_levels", numeroDeNiveles)

	return nil
}

func calcularEntradasNiveles(direccionLogica int) ([]int, int) {
	numeroPagina := direccionLogica / tamanoPagina
	desplazamiento := direccionLogica % tamanoPagina

	entradas := make([]int, numeroDeNiveles)

	for nivel := 0; nivel < numeroDeNiveles; nivel++ {
		exponente := numeroDeNiveles - nivel - 1
		potencia := int(math.Pow(float64(entradasPorTabla), float64(exponente)))
		entradas[nivel] = (numeroPagina / potencia) % entradasPorTabla
	}

	return entradas, desplazamiento
}

// Traducir dirección lógica a física
func traducirDireccion(pid, direccionLogica int) int {
	if !configCargada {
		err := cargarConfigMemoria()
		if err != nil {
			utils.ErrorLog.Error("Error obteniendo configuración", "error", err)
			return -1
		}
	}

	numeroPagina := int(math.Floor(float64(direccionLogica) / float64(tamanoPagina)))
	desplazamiento := direccionLogica % tamanoPagina

	// Buscar en TLB si está habilitada
	if config.TLBEntries > 0 {
		marco := buscarEnTLB(pid, numeroPagina)
		if marco != -1 {
			utils.InfoLog.Info(fmt.Sprintf("PID: %d - TLB HIT - Página: %d", pid, numeroPagina))
			return marco*tamanoPagina + desplazamiento
		} else {
			utils.InfoLog.Info(fmt.Sprintf("PID: %d - TLB MISS - Página: %d", pid, numeroPagina))
		}
	}

	// Obtener marco desde memoria
	marco := obtenerMarcoDeMemoria(pid, numeroPagina)
	if marco == -1 {
		utils.ErrorLog.Error("Error obteniendo marco de memoria", "pid", pid, "pagina", numeroPagina)
		return -1
	}

	// Actualizar TLB si está habilitada
	if config.TLBEntries > 0 {
		actualizarTLB(pid, numeroPagina, marco)
	}

	return marco*tamanoPagina + desplazamiento
}

// Buscar página en TLB
func buscarEnTLB(pid, numeroPagina int) int {
	mutex.Lock()
	defer mutex.Unlock()

	for i, entrada := range tlbEntries {
		if entrada.PageNumber == numeroPagina && entrada.PID == pid {
			if config.TLBReplacement == "LRU" {
				tlbEntries[i].LastUsed = time.Now().UnixNano()
			}
			return entrada.FrameNumber
		}
	}

	return -1
}

// Actualizar TLB con nueva entrada
func actualizarTLB(pid, numeroPagina, marco int) {
	mutex.Lock()
	defer mutex.Unlock()

	// Buscar entrada libre
	indiceLibre := -1
	for i, entrada := range tlbEntries {
		if entrada.PageNumber == -1 {
			indiceLibre = i
			break
		}
	}

	if indiceLibre != -1 {
		tlbEntries[indiceLibre] = TLBEntry{
			PageNumber:  numeroPagina,
			FrameNumber: marco,
			PID:         pid,
			LastUsed:    time.Now().UnixNano(),
			LoadTime:    tlbCounter,
		}
		tlbCounter++
		return
	}

	// Aplicar algoritmo de reemplazo
	indiceVictima := 0

	switch config.TLBReplacement {
	case "FIFO":
		tiempoMasAntiguo := tlbEntries[0].LoadTime
		for i, entrada := range tlbEntries {
			if entrada.LoadTime < tiempoMasAntiguo {
				tiempoMasAntiguo = entrada.LoadTime
				indiceVictima = i
			}
		}
	case "LRU":
		menosUsada := tlbEntries[0].LastUsed
		for i, entrada := range tlbEntries {
			if entrada.LastUsed < menosUsada {
				menosUsada = entrada.LastUsed
				indiceVictima = i
			}
		}
	}

	tlbEntries[indiceVictima] = TLBEntry{
		PageNumber:  numeroPagina,
		FrameNumber: marco,
		PID:         pid,
		LastUsed:    time.Now().UnixNano(),
		LoadTime:    tlbCounter,
	}
	tlbCounter++
}

// Obtener marco de memoria para una página
func obtenerMarcoDeMemoria(pid, numeroPagina int) int {
	utils.InfoLog.Info("Buscando marco", "pid", pid, "pagina", numeroPagina)

	// Verificar en caché si está habilitada
	if config.CacheEntries > 0 {
		if marco := buscarEnCache(pid, numeroPagina); marco != -1 {
			return marco
		}
	}

	// Calcular entradas multinivel
	direccionLogica := numeroPagina * tamanoPagina
	entradas, _ := calcularEntradasNiveles(direccionLogica)

	// Preparar mensaje con info multinivel
	params := map[string]interface{}{
		"pid":              pid,
		"pagina":           numeroPagina,
		"entradas_niveles": entradas,
		"niveles":          numeroDeNiveles,
	}

	// Simular delay de cache si está configurado
	if config.CacheDelay > 0 {
		time.Sleep(time.Duration(config.CacheDelay) * time.Millisecond)
	}

	// Enviar solicitud a memoria
	respuesta, err := memoriaClient.EnviarHTTPMensaje(utils.MensajeObtenerMarco, "OBTENER_MARCO", params)
	if err != nil {
		utils.ErrorLog.Error("Error al solicitar marco a memoria", "error", err)
		return -1
	}

	// Extraer marco de la respuesta
	datos, ok := respuesta.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Error("Formato de datos incorrecto")
		return -1
	}

	var marcoInt int
	if marco, ok := datos["marco"].(float64); ok {
		marcoInt = int(marco)
	} else if marco, ok := datos["marco"].(int); ok {
		marcoInt = marco
	} else {
		utils.ErrorLog.Error("Formato de marco incorrecto", "marco", datos["marco"])
		return -1
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, numeroPagina, marcoInt))

	// Actualizar caché si está habilitada
	if config.CacheEntries > 0 {
		actualizarCache(pid, numeroPagina, marcoInt)
	}

	return marcoInt
}

// Buscar en caché
func buscarEnCache(pid, numeroPagina int) int {
	mutex.Lock()
	defer mutex.Unlock()

	for i, entrada := range cacheEntries {
		if entrada.PageNumber == numeroPagina && entrada.PID == pid {
			utils.InfoLog.Info(fmt.Sprintf("PID: %d - Cache Hit - Página: %d", pid, numeroPagina))

			// Actualizar bit de referencia para CLOCK
			cacheEntries[i].Referenced = true

			return entrada.FrameNumber
		}
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Cache Miss - Página: %d", pid, numeroPagina))
	return -1
}

// Actualizar caché
func actualizarCache(pid, numeroPagina int, marco int) {
	mutex.Lock()
	defer mutex.Unlock()

	// Buscar entrada libre
	indiceLibre := -1
	for i, entrada := range cacheEntries {
		if entrada.PageNumber == -1 {
			indiceLibre = i
			break
		}
	}

	if indiceLibre != -1 {
		cacheEntries[indiceLibre] = CacheEntry{
			PageNumber:  numeroPagina,
			FrameNumber: marco,
			Content:     "",
			PID:         pid,
			Modified:    false,
			Referenced:  true,
		}
		utils.InfoLog.Info(fmt.Sprintf("PID: %d - Cache Add - Página: %d", pid, numeroPagina))
		return
	}

	// Aplicar algoritmo de reemplazo
	if config.CacheReplacement == "CLOCK" {
		aplicarCLOCK(pid, numeroPagina, marco)
	} else if config.CacheReplacement == "CLOCK-M" {
		aplicarCLOCKM(pid, numeroPagina, marco)
	}
}

// Algoritmo CLOCK para caché
func aplicarCLOCK(pid, numeroPagina int, marco int) {
	for {
		if !cacheEntries[clockPointer].Referenced {
			if cacheEntries[clockPointer].Modified {
				actualizarMemoria(cacheEntries[clockPointer].PID, cacheEntries[clockPointer].PageNumber)
			}

			cacheEntries[clockPointer] = CacheEntry{
				PageNumber:  numeroPagina,
				FrameNumber: marco,
				Content:     "",
				PID:         pid,
				Modified:    false,
				Referenced:  true,
			}
			utils.InfoLog.Info(fmt.Sprintf("PID: %d - Cache Add - Página: %d", pid, numeroPagina))

			clockPointer = (clockPointer + 1) % len(cacheEntries)
			return
		}

		cacheEntries[clockPointer].Referenced = false
		clockPointer = (clockPointer + 1) % len(cacheEntries)
	}
}

// Algoritmo CLOCK-M para caché
func aplicarCLOCKM(pid, numeroPagina int, marco int) {
	// Primera vuelta: buscar (0,0) - no referenciada, no modificada
	punteroInicial := clockPointer
	for {
		if !cacheEntries[clockPointer].Referenced && !cacheEntries[clockPointer].Modified {
			break
		}
		clockPointer = (clockPointer + 1) % len(cacheEntries)
		if clockPointer == punteroInicial {
			break
		}
	}

	// Segunda vuelta: buscar (0,1) - no referenciada, modificada
	if clockPointer == punteroInicial {
		for {
			if !cacheEntries[clockPointer].Referenced && cacheEntries[clockPointer].Modified {
				break
			}
			clockPointer = (clockPointer + 1) % len(cacheEntries)
			if clockPointer == punteroInicial {
				break
			}
		}
	}

	// Tercera vuelta: quitar referencias y buscar
	if clockPointer == punteroInicial {
		for i := range cacheEntries {
			cacheEntries[i].Referenced = false
		}

		for {
			if !cacheEntries[clockPointer].Referenced {
				break
			}
			clockPointer = (clockPointer + 1) % len(cacheEntries)
		}
	}

	if cacheEntries[clockPointer].Modified {
		actualizarMemoria(cacheEntries[clockPointer].PID, cacheEntries[clockPointer].PageNumber)
	}

	cacheEntries[clockPointer] = CacheEntry{
		PageNumber:  numeroPagina,
		FrameNumber: marco,
		Content:     "",
		PID:         pid,
		Modified:    false,
		Referenced:  true,
	}
	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Cache Add - Página: %d", pid, numeroPagina))

	clockPointer = (clockPointer + 1) % len(cacheEntries)
}

// Actualizar memoria desde caché
func actualizarMemoria(pid, numeroPagina int) {
	var contenido string
	var marco int = -1

	for _, e := range cacheEntries {
		if e.PageNumber == numeroPagina && e.PID == pid {
			contenido = e.Content
			marco = e.FrameNumber
			break
		}
	}

	if marco == -1 {
		utils.ErrorLog.Error("No se encontró la página en caché para actualizar memoria", "pid", pid, "pagina", numeroPagina)
		return
	}

	params := map[string]interface{}{
		"pid":              pid,
		"direccion_fisica": marco * tamanoPagina,
		"valor":            contenido,
	}

	_, err := memoriaClient.EnviarHTTPMensaje(utils.MensajeEscribir, "ESCRIBIR", params)
	if err != nil {
		utils.ErrorLog.Error("Error al actualizar memoria", "error", err)
		return
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Memory Update - Página: %d - Frame: %d", pid, numeroPagina, marco))
}

// Escribir en memoria
func escribirEnMemoria(pid, direccionLogica int, valor string) {
	direccionFisica := traducirDireccion(pid, direccionLogica)

	// Verificar si está en caché
	if config.CacheEntries > 0 {
		numeroPagina := int(math.Floor(float64(direccionLogica) / float64(tamanoPagina)))

		mutex.Lock()
		for i, entrada := range cacheEntries {
			if entrada.PageNumber == numeroPagina && entrada.PID == pid {
				cacheEntries[i].Content = valor
				cacheEntries[i].Modified = true
				cacheEntries[i].Referenced = true

				utils.InfoLog.Info(fmt.Sprintf("PID: %d - Acción: ESCRIBIR - Dir Física: %d - Valor: %s", pid, direccionFisica, valor))
				mutex.Unlock()
				return
			}
		}
		mutex.Unlock()
	}

	// Escribir en memoria
	params := map[string]interface{}{
		"pid":       pid,
		"direccion": direccionFisica,
		"valor":     valor,
	}

	_, err := memoriaClient.EnviarHTTPMensaje(utils.MensajeEscribir, "ESCRIBIR", params)
	if err != nil {
		utils.ErrorLog.Error("Error al escribir en memoria", "error", err)
		return
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Acción: ESCRIBIR - Dir Física: %d - Valor: %s", pid, direccionFisica, valor))
}

// Leer de memoria
func leerDeMemoria(pid, direccionLogica, tamano int) string {
	direccionFisica := traducirDireccion(pid, direccionLogica)

	// Verificar si está en caché
	if config.CacheEntries > 0 {
		numeroPagina := int(math.Floor(float64(direccionLogica) / float64(tamanoPagina)))

		mutex.Lock()
		for i, entrada := range cacheEntries {
			if entrada.PageNumber == numeroPagina && entrada.PID == pid {
				valor := entrada.Content
				cacheEntries[i].Referenced = true

				utils.InfoLog.Info(fmt.Sprintf("PID: %d - Acción: LEER - Dir Física: %d - Valor: %s", pid, direccionFisica, valor))
				mutex.Unlock()
				return valor
			}
		}
		mutex.Unlock()
	}

	// Leer de memoria
	params := map[string]interface{}{
		"pid":       pid,
		"direccion": direccionFisica,
		"tamano":    tamano,
	}

	respuesta, err := memoriaClient.EnviarHTTPMensaje(utils.MensajeLeer, "LEER", params)
	if err != nil {
		utils.ErrorLog.Error("Error al leer de memoria", "error", err)
		return ""
	}

	datos, ok := respuesta.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Error("Formato de datos incorrecto")
		return ""
	}

	valor, ok := datos["valor"].(string)
	if !ok {
		utils.ErrorLog.Error("Formato de valor incorrecto")
		return ""
	}

	utils.InfoLog.Info(fmt.Sprintf("PID: %d - Acción: LEER - Dir Física: %d - Valor: %s", pid, direccionFisica, valor))
	return valor
}
