package main

import (
	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

// Funciones para actualizar métricas

// Actualizar métricas de acceso a tablas de páginas
func actualizarMetricasAccesoTabla(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].AccesosTablasPaginas++
	
	utils.InfoLog.Info("Acceso a tabla de páginas", "pid", pid, "total_accesos", metricasPorProceso[pid].AccesosTablasPaginas)
}

// Actualizar métricas de instrucciones solicitadas
func actualizarMetricasInstruccion(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].InstruccionesSolicitadas++
	
	utils.InfoLog.Info("Instrucción solicitada", "pid", pid, "total_instrucciones", metricasPorProceso[pid].InstruccionesSolicitadas)
}

// Actualizar métricas de bajadas a SWAP
func actualizarMetricasBajadaSwap(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].BajadasSwap++
	
	utils.InfoLog.Info("Bajada a SWAP", "pid", pid, "total_bajadas", metricasPorProceso[pid].BajadasSwap)
}

// Actualizar métricas de subidas a memoria
func actualizarMetricasSubidaMemoria(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].SubidasMemoria++
	
	utils.InfoLog.Info("Subida a memoria", "pid", pid, "total_subidas", metricasPorProceso[pid].SubidasMemoria)
}

// Actualizar métricas de lecturas de memoria
func actualizarMetricasLectura(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].LecturasMemoria++
	
	utils.InfoLog.Info("Lectura de memoria", "pid", pid, "total_lecturas", metricasPorProceso[pid].LecturasMemoria)
}

// Actualizar métricas de escrituras en memoria
func actualizarMetricasEscritura(pid int) {
	if _, existe := metricasPorProceso[pid]; !existe {
		metricasPorProceso[pid] = &MetricasProceso{}
	}
	metricasPorProceso[pid].EscriturasMemoria++
	
	utils.InfoLog.Info("Escritura en memoria", "pid", pid, "total_escrituras", metricasPorProceso[pid].EscriturasMemoria)
}
