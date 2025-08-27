package main

import (
	"fmt"
	"path/filepath"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

func handlerOperacion(msg *utils.Mensaje) (interface{}, error) {
	// Determinar el tipo de operación
	tipoOperacion := utils.ObtenerTipoOperacion(msg, "memoria")

	// Seleccionar el retardo adecuado
	retardo := config.MemoryDelay
	if tipoOperacion == "swap" {
		retardo = config.SwapDelay
	}

	return utils.HandlerGenerico(msg, retardo, procesarOperacion)
}

func handlerObtenerInstruccion(msg *utils.Mensaje) (interface{}, error) {
	// Extraer el PID y PC del mensaje
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado", "datos", datos)
		return map[string]interface{}{
			"error": "PID no proporcionado o formato incorrecto",
		}, nil
	}

	pc, ok := datos["pc"].(float64)
	if !ok {
		utils.ErrorLog.Error("PC no proporcionado", "datos", datos)
		return map[string]interface{}{
			"error": "PC no proporcionado o formato incorrecto",
		}, nil
	}

	pidInt := int(pid)
	pcInt := int(pc)

	utils.InfoLog.Info("Solicitud de instrucción", "pid", pidInt, "pc", pcInt)

	// Verificar si hay instrucciones para el PID
	instruccionesMutex.RLock()
	instrucciones, existe := instruccionesPorProceso[pidInt]
	instruccionesMutex.RUnlock()

	if !existe || len(instrucciones) == 0 {
		instruccionesMutex.Lock()
		// Doble verificación
		if instrucciones, existe = instruccionesPorProceso[pidInt]; !existe {
			if err := cargarInstrucciones(pidInt); err != nil {
				instruccionesMutex.Unlock()
				utils.ErrorLog.Error("Error cargando instrucciones", "pid", pidInt, "error", err)
				return map[string]interface{}{
					"error": fmt.Sprintf("No se pudieron cargar instrucciones para el PID %d: %v", pidInt, err),
				}, nil
			}
			instrucciones = instruccionesPorProceso[pidInt]
		}
		instruccionesMutex.Unlock()
	}

	// Verificar que el PC esté dentro del rango válido
	if pcInt < 0 || pcInt >= len(instrucciones) {
		utils.ErrorLog.Error("PC fuera de rango", "pid", pidInt, "pc", pcInt, "max", len(instrucciones)-1)
		return map[string]interface{}{
			"error": fmt.Sprintf("PC fuera de rango para PID %d: PC=%d, máximo=%d", pidInt, pcInt, len(instrucciones)-1),
		}, nil
	}

	// Obtener la instrucción
	instruccion := instrucciones[pcInt]

	// Log obligatorio del enunciado
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Obtener instrucción: %d - Instrucción: %s", pidInt, pcInt, instruccion))

	// Dumps intermedios automáticos
	if pcInt == 5 || pcInt == 10 || pcInt == 15 {
		if err := crearMemoryDump(pidInt); err != nil {
			utils.ErrorLog.Error("Error creando dump intermedio", "pid", pidInt, "pc", pcInt, "error", err)
		}
	}

	// Actualizar métricas
	actualizarMetricasInstruccion(pidInt)

	utils.InfoLog.Info("Instrucción entregada", "pid", pidInt, "pc", pcInt, "instruccion", instruccion)

	return map[string]interface{}{
		"status":      "OK",
		"instruccion": instruccion,
	}, nil
}

func handlerEspacioLibre(msg *utils.Mensaje) (interface{}, error) {
	espacioLibre := calcularEspacioLibre()

	utils.InfoLog.Info("Espacio libre consultado", "espacio_libre_bytes", espacioLibre)

	return map[string]interface{}{
		"status":        "OK",
		"espacio_libre": espacioLibre,
	}, nil
}

// calcularEspacioLibre calcula el espacio libre total en bytes
func calcularEspacioLibre() int {
	espacioLibre := 0
	for _, libre := range marcosLibres {
		if libre {
			espacioLibre += config.PageSize
		}
	}
	return espacioLibre
}

func handlerInicializarProceso(msg *utils.Mensaje) (interface{}, error) {
	datos, ok := msg.Datos.(map[string]interface{})
	if !ok {
		utils.ErrorLog.Error("Formato de datos incorrecto", "datos", msg.Datos)
		return map[string]interface{}{"error": "Formato de datos incorrecto"}, nil
	}

	pid := int(datos["pid"].(float64))
	tamanio := int(datos["tamanio"].(float64))
	archivoOrigen := datos["archivo"].(string)

	utils.InfoLog.Info("Solicitud de inicialización de proceso", "pid", pid, "tamanio", tamanio, "archivo", archivoOrigen)

	// Verificar espacio libre
	if calcularEspacioLibre() < tamanio {
		utils.ErrorLog.Error("Espacio insuficiente", "pid", pid, "tamanio_requerido", tamanio, "espacio_libre", calcularEspacioLibre())
		return map[string]interface{}{
			"error": fmt.Sprintf("No hay suficiente espacio libre para inicializar el proceso %d", pid),
		}, nil
	}

	// Copiar el archivo de pseudocódigo
	destino := filepath.Join(config.ScriptsPath, fmt.Sprintf("%d.txt", pid))
	if err := copiarPseudocodigo(archivoOrigen, destino); err != nil {
		utils.ErrorLog.Error("Error copiando pseudocódigo", "archivo_origen", archivoOrigen, "destino", destino, "error", err)
		return map[string]interface{}{"error": err.Error()}, nil
	}

	// Crear tablas de páginas
	_, err := crearTablasPaginas(pid, tamanio)
	if err != nil {
		utils.ErrorLog.Error("Error creando tablas de páginas", "pid", pid, "error", err)
		return map[string]interface{}{"error": err.Error()}, nil
	}

	// Cargar instrucciones en memoria
	if err := cargarInstrucciones(pid); err != nil {
		utils.ErrorLog.Error("Error cargando instrucciones", "pid", pid, "error", err)
		liberarMemoriaProceso(pid)
		return map[string]interface{}{"error": err.Error()}, nil
	}

	utils.InfoLog.Info("Proceso inicializado correctamente", "pid", pid, "tamanio", tamanio, "archivo", archivoOrigen)

	return map[string]interface{}{
		"status": "OK",
	}, nil
}

func handlerFinalizarProceso(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado", "datos", datos)
		return map[string]interface{}{
			"error": "PID no proporcionado o formato incorrecto",
		}, nil
	}

	pidInt := int(pid)

	utils.InfoLog.Info("Solicitud de finalización de proceso", "pid", pidInt)

	// Crear dump final
	if err := crearMemoryDump(pidInt); err != nil {
		utils.ErrorLog.Error("Error creando dump final", "pid", pidInt, "error", err)
	}

	// Liberar memoria del proceso
	if err := liberarMemoriaProceso(pidInt); err != nil {
		utils.ErrorLog.Error("Error liberando memoria", "pid", pidInt, "error", err)
		return map[string]interface{}{
			"error": fmt.Sprintf("Error al liberar memoria del proceso %d: %v", pidInt, err),
		}, nil
	}

	// Log de métricas finales
	if metricas, existe := metricasPorProceso[pidInt]; existe {
		utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido - Métricas: ATP;%d;SWAP;%d;MemPrin;%d;LecMem;%d;EscMem;%d",
			pidInt,
			metricas.AccesosTablasPaginas,
			metricas.BajadasSwap,
			metricas.SubidasMemoria,
			metricas.LecturasMemoria,
			metricas.EscriturasMemoria))

		delete(metricasPorProceso, pidInt)
	}

	// Eliminar instrucciones del proceso
	instruccionesMutex.Lock()
	delete(instruccionesPorProceso, pidInt)
	instruccionesMutex.Unlock()

	utils.InfoLog.Info("Proceso finalizado correctamente", "pid", pidInt)

	return map[string]interface{}{
		"status": "OK",
	}, nil
}

func handlerLeerMemoria(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado", "datos", datos)
		return map[string]interface{}{"error": "PID no proporcionado o formato incorrecto"}, nil
	}
	pidInt := int(pid)

	// Dirección puede ser física o lógica
	dirFisica, ok := datos["direccion_fisica"].(float64)
	if !ok {
		dirLogica, ok := datos["direccion_logica"].(float64)
		if !ok {
			utils.ErrorLog.Error("Dirección no proporcionada", "datos", datos)
			return map[string]interface{}{"error": "Dirección no proporcionada o formato incorrecto"}, nil
		}

		dirFisicaInt, err := traducirDireccion(pidInt, int(dirLogica))
		if err != nil {
			utils.ErrorLog.Error("Error traduciendo dirección", "pid", pidInt, "dir_logica", int(dirLogica), "error", err)
			return map[string]interface{}{"error": fmt.Sprintf("Error traduciendo dirección: %v", err)}, nil
		}
		dirFisica = float64(dirFisicaInt)
	}

	tamanio, ok := datos["tamanio"].(float64)
	if !ok {
		tamanio = 1
	}

	// Verificar límites
	if int(dirFisica) < 0 || int(dirFisica)+int(tamanio) > len(memoriaPrincipal) {
		utils.ErrorLog.Error("Dirección fuera de rango", "pid", pidInt, "dir_fisica", int(dirFisica), "tamanio", int(tamanio))
		return map[string]interface{}{"error": "Dirección fuera de rango"}, nil
	}

	// Leer de memoria
	valor := memoriaPrincipal[int(dirFisica):int(dirFisica+tamanio)]

	// Actualizar métricas
	actualizarMetricasLectura(pidInt)

	// Log obligatorio
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Lectura - Dir Física: %d - Tamaño: %d",
		pidInt, int(dirFisica), int(tamanio)))

	utils.InfoLog.Info("Lectura de memoria realizada", "pid", pidInt, "dir_fisica", int(dirFisica), "tamanio", int(tamanio))

	return map[string]interface{}{
		"status": "OK",
		"valor":  string(valor),
	}, nil
}

func handlerEscribirMemoria(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado", "datos", datos)
		return map[string]interface{}{"error": "PID no proporcionado o formato incorrecto"}, nil
	}
	pidInt := int(pid)

	// Dirección puede ser física o lógica
	dirFisica, ok := datos["direccion_fisica"].(float64)
	if !ok {
		dirLogica, ok := datos["direccion_logica"].(float64)
		if !ok {
			utils.ErrorLog.Error("Dirección no proporcionada", "datos", datos)
			return map[string]interface{}{"error": "Dirección no proporcionada o formato incorrecto"}, nil
		}

		dirFisicaInt, err := traducirDireccion(pidInt, int(dirLogica))
		if err != nil {
			utils.ErrorLog.Error("Error traduciendo dirección", "pid", pidInt, "dir_logica", int(dirLogica), "error", err)
			return map[string]interface{}{"error": fmt.Sprintf("Error traduciendo dirección: %v", err)}, nil
		}
		dirFisica = float64(dirFisicaInt)
	}

	valor, ok := datos["valor"].(string)
	if !ok {
		utils.ErrorLog.Error("Valor no proporcionado", "datos", datos)
		return map[string]interface{}{"error": "Valor no proporcionado o formato incorrecto"}, nil
	}

	// Verificar límites
	if int(dirFisica) < 0 || int(dirFisica)+len(valor) > len(memoriaPrincipal) {
		utils.ErrorLog.Error("Dirección fuera de rango para escritura", "pid", pidInt, "dir_fisica", int(dirFisica), "tamanio_valor", len(valor))
		return map[string]interface{}{"error": "Dirección fuera de rango"}, nil
	}

	// Escribir en memoria
	copy(memoriaPrincipal[int(dirFisica):int(dirFisica)+len(valor)], []byte(valor))

	// Actualizar métricas
	actualizarMetricasEscritura(pidInt)

	// Log obligatorio
	utils.InfoLog.Info(fmt.Sprintf("## PID: %d - Escritura - Dir Física: %d - Tamaño: %d",
		pidInt, int(dirFisica), len(valor)))

	utils.InfoLog.Info("Escritura en memoria realizada", "pid", pidInt, "dir_fisica", int(dirFisica), "tamanio", len(valor))

	return map[string]interface{}{
		"status": "OK",
	}, nil
}

func handlerObtenerMarco(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado", "datos", datos)
		return map[string]interface{}{"error": "PID no proporcionado o formato incorrecto"}, nil
	}
	pidInt := int(pid)

	numPagina, ok := datos["pagina"].(float64)
	if !ok {
		utils.ErrorLog.Error("Número de página no proporcionado", "datos", datos)
		return map[string]interface{}{"error": "Número de página no proporcionado o formato incorrecto"}, nil
	}

	utils.InfoLog.Info("Solicitud de marco", "pid", pidInt, "pagina", int(numPagina))

	// Obtener la tabla de páginas del proceso
	tabla, existe := tablasPaginas[pidInt]
	if !existe {
		utils.ErrorLog.Error("No existe tabla de páginas", "pid", pidInt)
		return map[string]interface{}{"error": "No existe tabla de páginas para el PID proporcionado"}, nil
	}

	// Obtener el marco para la página solicitada
	marco, err := obtenerMarcoDesdeTabla(pidInt, tabla, int(numPagina), 1)
	if err != nil {
		utils.ErrorLog.Error("Error obteniendo marco", "pid", pidInt, "pagina", int(numPagina), "error", err)
		return map[string]interface{}{"error": fmt.Sprintf("Error obteniendo marco: %v", err)}, nil
	}

	// Log obligatorio
	utils.InfoLog.Info(fmt.Sprintf("PID: %d OBTENER MARCO Página: %d Marco: %d",
		pidInt, int(numPagina), marco))

	utils.InfoLog.Info("Marco obtenido", "pid", pidInt, "pagina", int(numPagina), "marco", marco)

	return map[string]interface{}{
		"status": "OK",
		"marco":  marco,
	}, nil
}

func handlerSuspenderProceso(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado para suspensión", "datos", datos)
		return map[string]interface{}{
			"error": "PID no proporcionado o formato incorrecto",
		}, nil
	}
	pidInt := int(pid)

	utils.InfoLog.Info("Solicitud de suspensión", "pid", pidInt)

	err := suspenderProceso(pidInt)
	if err != nil {
		utils.ErrorLog.Error("Error suspendiendo proceso", "pid", pidInt, "error", err)
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	utils.InfoLog.Info("Proceso suspendido correctamente", "pid", pidInt)

	return map[string]interface{}{
		"status": "OK",
	}, nil
}

func handlerDessuspenderProceso(msg *utils.Mensaje) (interface{}, error) {
	datos := msg.Datos.(map[string]interface{})
	pid, ok := datos["pid"].(float64)
	if !ok {
		utils.ErrorLog.Error("PID no proporcionado para dessuspensión", "datos", datos)
		return map[string]interface{}{
			"error": "PID no proporcionado o formato incorrecto",
		}, nil
	}
	pidInt := int(pid)

	utils.InfoLog.Info("Solicitud de dessuspensión", "pid", pidInt)

	err := dessuspenderProceso(pidInt)
	if err != nil {
		utils.ErrorLog.Error("Error dessuspendiendo proceso", "pid", pidInt, "error", err)
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	utils.InfoLog.Info("Proceso dessuspendido correctamente", "pid", pidInt)

	return map[string]interface{}{
		"status": "OK",
	}, nil
}
