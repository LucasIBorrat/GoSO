package main

import (
	"fmt"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Traduce una dirección lógica a una dirección física
func traducirDireccion(pid int, dirLogica int) (int, error) {
	// Obtener tabla de páginas de nivel 1 para el proceso
	tabla, existe := tablasPaginas[pid]
	if !existe {
		utils.ErrorLog.Error("No existe tabla de páginas", "pid", pid)
		return 0, fmt.Errorf("no existe tabla de páginas para PID %d", pid)
	}

	// Calcular componentes de la dirección lógica
	numPagina := dirLogica / config.PageSize
	desplazamiento := dirLogica % config.PageSize

	utils.InfoLog.Info("Traduciendo dirección", 
		"pid", pid, 
		"dir_logica", dirLogica, 
		"pagina", numPagina, 
		"desplazamiento", desplazamiento)

	// Obtener marco mediante función recursiva que navegue los niveles
	marco, err := obtenerMarcoDesdeTabla(pid, tabla, numPagina, 1)
	if err != nil {
		utils.ErrorLog.Error("Error obteniendo marco", "pid", pid, "pagina", numPagina, "error", err)
		return 0, err
	}

	// Calcular dirección física
	dirFisica := marco*config.PageSize + desplazamiento

	utils.InfoLog.Info("Dirección traducida", 
		"pid", pid, 
		"dir_logica", dirLogica, 
		"dir_fisica", dirFisica, 
		"marco", marco)

	return dirFisica, nil
}

// Calcula el número de páginas necesarias para un tamaño dado
func calcularNumeroPaginas(tamanio int) int {
	numPaginas := (tamanio + config.PageSize - 1) / config.PageSize
	utils.InfoLog.Info("Páginas calculadas", "tamanio", tamanio, "paginas_necesarias", numPaginas)
	return numPaginas
}
