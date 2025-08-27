package main

import (
	"fmt"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

type Configuracion struct {
	PageSize int
}

type TablaPagina struct {
	// Definición de la estructura de la tabla de páginas
}

// Asigna un marco libre para un proceso
func asignarMarco(pid int) (int, error) {
	utils.InfoLog.Info("Buscando marco libre", "pid", pid)

	// Buscar un marco libre
	for i, libre := range marcosLibres {
		if libre {
			// Marcar el marco como ocupado
			marcosLibres[i] = false

			// Registrar que este marco está asignado al proceso
			marcosAsignadosPorProceso[pid] = append(marcosAsignadosPorProceso[pid], i)

			utils.InfoLog.Info("Marco asignado", "pid", pid, "marco", i)
			return i, nil
		}
	}

	utils.ErrorLog.Error("No hay marcos libres disponibles", "pid", pid)
	return 0, fmt.Errorf("no hay marcos libres disponibles")
}

// Cuenta el número de marcos libres disponibles
func contarMarcosLibres() int {
	count := 0
	for _, libre := range marcosLibres {
		if libre {
			count++
		}
	}

	utils.InfoLog.Info("Marcos libres contados", "marcos_libres", count, "total_marcos", len(marcosLibres))
	return count
}

// liberarMemoriaProceso libera todos los marcos asignados a un proceso
func liberarMemoriaProceso(pid int) error {
	utils.InfoLog.Info("Liberando memoria del proceso", "pid", pid)

	// Verificar si existe el proceso
	marcos, existe := marcosAsignadosPorProceso[pid]
	if !existe {
		utils.ErrorLog.Error("No existe asignación de memoria", "pid", pid)
		return fmt.Errorf("no existe asignación de memoria para el proceso %d", pid)
	}

	utils.InfoLog.Info("Marcos a liberar", "pid", pid, "cantidad_marcos", len(marcos), "marcos", marcos)

	// Marcar como libres todos los marcos asignados al proceso
	for _, marco := range marcos {
		marcosLibres[marco] = true

		// Limpiar la memoria (poner en ceros)
		inicio := marco * config.PageSize
		fin := inicio + config.PageSize
		for i := inicio; i < fin && i < len(memoriaPrincipal); i++ {
			memoriaPrincipal[i] = 0
		}

		utils.InfoLog.Info("Marco limpiado", "pid", pid, "marco", marco)
	}

	// Eliminar la entrada del proceso del mapa de asignaciones
	delete(marcosAsignadosPorProceso, pid)

	// Eliminar la tabla de páginas del proceso
	delete(tablasPaginas, pid)

	utils.InfoLog.Info("Memoria liberada completamente", "pid", pid, "marcos_liberados", len(marcos))

	return nil
}
