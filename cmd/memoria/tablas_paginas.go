package main

import (
	"fmt"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Variables globales
var tablasMemoria = make(map[int]*TablaPaginas)
var ultimoID = 0
var mutexTablas = &sync.Mutex{}

// Función para crear tabla de páginas para un nuevo proceso
func crearTablasPaginas(pid int, tamanio int) (*TablaPaginas, error) {
	// LOCK GLOBAL para evitar race conditions
	memoriaGeneralMutex.Lock()
	defer memoriaGeneralMutex.Unlock()

	utils.InfoLog.Info("Creando tabla de páginas", "pid", pid, "tamanio", tamanio)

	// Calcular número de páginas necesarias
	numPaginas := calcularNumeroPaginas(tamanio)
	utils.InfoLog.Info("Páginas requeridas", "pid", pid, "paginas", numPaginas)

	// Verificar si hay suficientes marcos libres
	marcosFree := contarMarcosLibres()
	if marcosFree < numPaginas {
		utils.ErrorLog.Error("Marcos insuficientes", "pid", pid, "marcos_libres", marcosFree, "paginas_requeridas", numPaginas)
		return nil, fmt.Errorf("no hay suficientes marcos libres (%d) para el proceso %d que requiere %d páginas",
			marcosFree, pid, numPaginas)
	}

	// Verificar que no exista ya una tabla para este PID (prevención adicional)
	if _, existe := tablasPaginas[pid]; existe {
		utils.InfoLog.Warn("Tabla de páginas ya existe para el proceso", "pid", pid)
		return nil, fmt.Errorf("tabla de páginas ya existe para el proceso %d", pid)
	}

	// Crear tabla de nivel 1
	tablaNivel1 := &TablaPaginas{
		Entradas: make([]EntradaTabla, config.EntriesPerPage),
		Nivel:    1,
	}

	// Inicializar las entradas como inválidas
	for i := range tablaNivel1.Entradas {
		tablaNivel1.Entradas[i].Valido = false
	}

	utils.InfoLog.Info("Tabla de nivel 1 creada", "pid", pid, "entradas", len(tablaNivel1.Entradas))

	// Asignar tabla al proceso
	tablasPaginas[pid] = tablaNivel1

	// Registrar que este proceso necesita estas páginas
	marcosAsignadosPorProceso[pid] = []int{}

	utils.InfoLog.Info("Tabla de páginas creada exitosamente", "pid", pid, "paginas", numPaginas)

	return tablaNivel1, nil
}
// Navega recursivamente los niveles de tablas para obtener el marco final
func obtenerMarcoDesdeTabla(pid int, tabla *TablaPaginas, numPagina int, nivelActual int) (int, error) {
	utils.InfoLog.Info("Navegando tabla de páginas", "pid", pid, "pagina", numPagina, "nivel", nivelActual)

	// Actualizar métricas de acceso a tablas de páginas
	actualizarMetricasAccesoTabla(pid)

	// Calcular índice en el nivel actual
	indice := calcularIndiceEnNivel(numPagina, nivelActual)
	utils.InfoLog.Info("Índice calculado", "pid", pid, "nivel", nivelActual, "indice", indice)

	// Verificar si la entrada es válida
	if indice >= len(tabla.Entradas) || !tabla.Entradas[indice].Valido {
		utils.InfoLog.Info("Entrada no válida, creando estructura", "pid", pid, "nivel", nivelActual, "indice", indice)

		if nivelActual < config.NumberOfLevels {
			// Crear tabla para el siguiente nivel
			nuevaTabla := crearTablaSiguienteNivel(pid, tabla, indice, nivelActual+1)
			return obtenerMarcoDesdeTabla(pid, nuevaTabla, numPagina, nivelActual+1)
		} else {
			// En el último nivel, asignar un marco
			utils.InfoLog.Info("Último nivel, asignando marco", "pid", pid, "pagina", numPagina)
			marco, err := asignarMarco(pid)
			if err != nil {
				utils.ErrorLog.Error("Error asignando marco", "pid", pid, "error", err)
				return 0, err
			}

			// Actualizar la entrada
			tabla.Entradas[indice].Marco = marco
			tabla.Entradas[indice].Presente = true
			tabla.Entradas[indice].Valido = true

			utils.InfoLog.Info("Marco asignado en último nivel", "pid", pid, "pagina", numPagina, "marco", marco)
			return marco, nil
		}
	}

	// Si estamos en el último nivel, devolver el marco
	if nivelActual == config.NumberOfLevels {
		if !tabla.Entradas[indice].Presente {
			utils.InfoLog.Info("Página no presente, trayendo de SWAP", "pid", pid, "pagina", numPagina)
			// Traer página de SWAP si es necesario
			err := traerPaginaDeSwap(pid, numPagina, tabla.Entradas[indice].Marco)
			if err != nil {
				utils.ErrorLog.Error("Error trayendo página de SWAP", "pid", pid, "pagina", numPagina, "error", err)
				return 0, err
			}
			tabla.Entradas[indice].Presente = true
		}

		marco := tabla.Entradas[indice].Marco
		utils.InfoLog.Info("Marco obtenido del último nivel", "pid", pid, "pagina", numPagina, "marco", marco)
		return marco, nil
	}

	// Si no estamos en el último nivel, obtener la siguiente tabla
	siguienteTabla := obtenerTablaSiguienteNivel(tabla.Entradas[indice].Direccion)
	utils.InfoLog.Info("Descendiendo al siguiente nivel", "pid", pid, "nivel_actual", nivelActual, "siguiente_nivel", nivelActual+1)

	// Llamada recursiva al siguiente nivel
	return obtenerMarcoDesdeTabla(pid, siguienteTabla, numPagina, nivelActual+1)
}

// Calcula el índice en un nivel específico de la tabla de páginas
func calcularIndiceEnNivel(numPagina int, nivel int) int {
	potencia := 1
	for i := 0; i < config.NumberOfLevels-nivel; i++ {
		potencia *= config.EntriesPerPage
	}
	indice := (numPagina / potencia) % config.EntriesPerPage

	utils.InfoLog.Info("Índice calculado", "pagina", numPagina, "nivel", nivel, "indice", indice)
	return indice
}

// Función auxiliar para crear una nueva tabla del siguiente nivel
func crearTablaSiguienteNivel(pid int, tablaActual *TablaPaginas, indice int, nuevoNivel int) *TablaPaginas {
	utils.InfoLog.Info("Creando tabla del siguiente nivel", "pid", pid, "nuevo_nivel", nuevoNivel)

	nuevaTabla := &TablaPaginas{
		Entradas: make([]EntradaTabla, config.EntriesPerPage),
		Nivel:    nuevoNivel,
	}

	// Inicializar entradas como inválidas
	for i := range nuevaTabla.Entradas {
		nuevaTabla.Entradas[i].Valido = false
	}

	// Asignar dirección en la tabla del nivel anterior
	direccionTabla := almacenarTablaEnMemoria(nuevaTabla)
	tablaActual.Entradas[indice].Direccion = direccionTabla
	tablaActual.Entradas[indice].Valido = true
	tablaActual.Entradas[indice].Presente = true

	utils.InfoLog.Info("Tabla del siguiente nivel creada", "pid", pid, "nivel", nuevoNivel, "direccion", direccionTabla)

	return nuevaTabla
}

// Almacena una tabla de páginas y devuelve un ID único
func almacenarTablaEnMemoria(tabla *TablaPaginas) int {
	mutexTablas.Lock()
	defer mutexTablas.Unlock()

	ultimoID++
	tablasMemoria[ultimoID] = tabla

	utils.InfoLog.Info("Tabla almacenada en memoria", "id", ultimoID, "nivel", tabla.Nivel)
	return ultimoID
}

// Obtiene una tabla usando su ID
func obtenerTablaSiguienteNivel(id int) *TablaPaginas {
	mutexTablas.Lock()
	defer mutexTablas.Unlock()

	tabla := tablasMemoria[id]
	if tabla != nil {
		utils.InfoLog.Info("Tabla obtenida", "id", id, "nivel", tabla.Nivel)
	} else {
		utils.ErrorLog.Error("Tabla no encontrada", "id", id)
	}

	return tabla
}

// marcarPaginaNoPresente marca una página como no presente en la tabla de páginas
func marcarPaginaNoPresente(pid int, tabla *TablaPaginas, numPagina int, nivelActual int) {
	utils.InfoLog.Info("Marcando página como no presente", "pid", pid, "pagina", numPagina, "nivel", nivelActual)

	if nivelActual == config.NumberOfLevels {
		indice := calcularIndiceEnNivel(numPagina, nivelActual)
		if indice < len(tabla.Entradas) && tabla.Entradas[indice].Valido {
			tabla.Entradas[indice].Presente = false
			utils.InfoLog.Info("Página marcada como no presente", "pid", pid, "pagina", numPagina, "indice", indice)
		}
	} else {
		indice := calcularIndiceEnNivel(numPagina, nivelActual)
		if indice < len(tabla.Entradas) && tabla.Entradas[indice].Valido {
			siguienteTabla := obtenerTablaSiguienteNivel(tabla.Entradas[indice].Direccion)
			marcarPaginaNoPresente(pid, siguienteTabla, numPagina, nivelActual+1)
		}
	}
}

// actualizarTablaPaginas actualiza la entrada de la tabla para una página
func actualizarTablaPaginas(pid int, tabla *TablaPaginas, numPagina int, marco int, nivelActual int) {
	utils.InfoLog.Info("Actualizando tabla de páginas", "pid", pid, "pagina", numPagina, "marco", marco, "nivel", nivelActual)

	if nivelActual == config.NumberOfLevels {
		indice := calcularIndiceEnNivel(numPagina, nivelActual)
		if indice < len(tabla.Entradas) {
			tabla.Entradas[indice].Marco = marco
			tabla.Entradas[indice].Presente = true
			tabla.Entradas[indice].Valido = true
			utils.InfoLog.Info("Entrada actualizada en último nivel", "pid", pid, "pagina", numPagina, "marco", marco, "indice", indice)
		}
	} else {
		indice := calcularIndiceEnNivel(numPagina, nivelActual)
		if indice < len(tabla.Entradas) {
			if !tabla.Entradas[indice].Valido {
				// Crear nueva tabla si no existe
				nuevaTabla := crearTablaSiguienteNivel(pid, tabla, indice, nivelActual+1)
				actualizarTablaPaginas(pid, nuevaTabla, numPagina, marco, nivelActual+1)
			} else {
				siguienteTabla := obtenerTablaSiguienteNivel(tabla.Entradas[indice].Direccion)
				actualizarTablaPaginas(pid, siguienteTabla, numPagina, marco, nivelActual+1)
			}
		}
	}
}

// encontrarPaginaPorMarco busca recursivamente qué página corresponde a un marco
func encontrarPaginaPorMarco(pid int, tabla *TablaPaginas, marco int, nivelActual int) int {
	utils.InfoLog.Info("Buscando página por marco", "pid", pid, "marco", marco, "nivel", nivelActual)

	if nivelActual == config.NumberOfLevels {
		// En el último nivel, buscamos el marco en las entradas
		for i, entrada := range tabla.Entradas {
			if entrada.Valido && entrada.Marco == marco {
				// Calculamos el número de página a partir del índice en el último nivel
				potencia := 1
				for j := 0; j < config.NumberOfLevels-nivelActual; j++ {
					potencia *= config.EntriesPerPage
				}
				numPagina := i * potencia
				utils.InfoLog.Info("Página encontrada por marco", "pid", pid, "marco", marco, "pagina", numPagina)
				return numPagina
			}
		}
		return -1
	} else {
		// En niveles intermedios, buscamos en las tablas siguientes
		for _, entrada := range tabla.Entradas {
			if entrada.Valido {
				siguienteTabla := obtenerTablaSiguienteNivel(entrada.Direccion)
				numPagina := encontrarPaginaPorMarco(pid, siguienteTabla, marco, nivelActual+1)
				if numPagina != -1 {
					utils.InfoLog.Info("Página encontrada en nivel intermedio", "pid", pid, "marco", marco, "pagina", numPagina)
					return numPagina
				}
			}
		}
		return -1
	}
}
