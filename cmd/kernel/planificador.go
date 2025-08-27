package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-LosCuervosXeneizes/utils"
)

var (
	proximoPID int = 0
	pidMutex   sync.Mutex

	// Colas de estados
	colaNew         []*PCB          = []*PCB{}
	colaReady       []*PCB          = []*PCB{}
	colaExec        map[string]*PCB = make(map[string]*PCB)
	colaBlocked     []*PCB          = []*PCB{}
	colaSuspReady   []*PCB          = []*PCB{}
	colaSuspBlocked []*PCB          = []*PCB{}
	colaExit        []*PCB          = []*PCB{}

	// Mutexes
	newMutex         sync.Mutex
	readyMutex       sync.Mutex
	execMutex        sync.Mutex
	blockedMutex     sync.Mutex
	suspReadyMutex   sync.Mutex
	suspBlockedMutex sync.Mutex
	exitMutex        sync.Mutex
	mapaMutex        sync.RWMutex

	// Conditions
	condNew   *sync.Cond
	condReady *sync.Cond

	mapaPCBs               map[int]*PCB = make(map[int]*PCB)
	gradoMultiprogramacion int
	semaforoMultiprogram   *utils.Semaforo
	timersSuspension       map[int]*time.Timer
	timersMutex            sync.Mutex
)

// InicializarPlanificador optimizado
func InicializarPlanificador(config *KernelConfig) {
	gradoMultiprogramacion = config.GradoMultiprogramacion
	if gradoMultiprogramacion <= 0 {
		gradoMultiprogramacion = 1
	}
	semaforoMultiprogram = utils.NewSemaforo(gradoMultiprogramacion)

	condNew = sync.NewCond(&newMutex)
	condReady = sync.NewCond(&readyMutex)
	timersSuspension = make(map[int]*time.Timer)

	utils.InfoLog.Info("Planificador inicializado",
		"algoritmo_sts", config.SchedulerAlgorithm,
		"algoritmo_lts", config.ReadyIngressAlgorithm,
		"multiprogramacion", gradoMultiprogramacion)
}

// GenerarNuevoPID devuelve un PID único
func GenerarNuevoPID() int {
	pidMutex.Lock()
	defer pidMutex.Unlock()
	pid := proximoPID
	proximoPID++
	return pid
}

// BuscarPCBPorPID busca un PCB en el mapa global
func BuscarPCBPorPID(pid int) *PCB {
	mapaMutex.RLock()
	defer mapaMutex.RUnlock()
	return mapaPCBs[pid]
}

// AgregarProcesoANew optimizado
func AgregarProcesoANew(pcb *PCB) {
	newMutex.Lock()
	colaNew = append(colaNew, pcb)
	newMutex.Unlock()
	condNew.Signal()
}

// MoverProcesoAReady optimizado
func MoverProcesoAReady(pcb *PCB) {
	// Si el proceso está en SUSP.BLOCKED, debe ir a SUSP.READY primero
	if pcb.Estado == EstadoSuspBlocked {
		utils.InfoLog.Info(" Proceso en SUSP.BLOCKED, moviendo a SUSP.READY", "pid", pcb.PID)
		MoverProcesoASuspReady(pcb)
		return
	}

	// Remover de la cola correspondiente según estado actual
	switch pcb.Estado {
	case EstadoBlocked:
		removerDeBlocked(pcb)
	case EstadoSuspBlocked:
		removerDeSuspBlocked(pcb)
	}

	// Cancelar timer de suspensión si existe (proceso terminó IO antes de ser suspendido)
	timersMutex.Lock()
	if timer, existe := timersSuspension[pcb.PID]; existe {
		timer.Stop()
		delete(timersSuspension, pcb.PID)
		utils.InfoLog.Info(" Timer de suspensión cancelado - proceso terminó IO", "pid", pcb.PID)
	}
	timersMutex.Unlock()

	pcb.CambiarEstado(EstadoReady)

	readyMutex.Lock()
	colaReady = append(colaReady, pcb)
	readyMutex.Unlock()
	condReady.Signal()
}

// MoverProcesoASuspReady mueve un proceso de SUSP.BLOCKED a SUSP.READY
func MoverProcesoASuspReady(pcb *PCB) {
	// Remover de SUSP.BLOCKED
	if !removerDeSuspBlocked(pcb) {
		utils.InfoLog.Warn("Proceso no encontrado en SUSP.BLOCKED", "pid", pcb.PID)
		return
	}

	// Cancelar timer de suspensión si existe
	timersMutex.Lock()
	if timer, existe := timersSuspension[pcb.PID]; existe {
		timer.Stop()
		delete(timersSuspension, pcb.PID)
	}
	timersMutex.Unlock()

	// Cambiar estado y agregar a SUSP.READY
	pcb.CambiarEstado(EstadoSuspReady)
	pcb.EnSwap = false // El proceso está ahora en memoria debido a la finalización de IO

	suspReadyMutex.Lock()
	colaSuspReady = append(colaSuspReady, pcb)
	suspReadyMutex.Unlock()

	// Señalizar al LTS que hay procesos en SUSP.READY disponibles para admisión
	condNew.Signal()

	utils.InfoLog.Info("Timer de suspensión cancelado - proceso terminó IO", "pid", pcb.PID)
	utils.InfoLog.Info("Proceso movido de SUSP.BLOCKED -> SUSP.READY", "pid", pcb.PID)
}

// MoverProcesoABlocked optimizado
func MoverProcesoABlocked(pcb *PCB, motivo string) {
	execMutex.Lock()
	for cpu, pcbEnExec := range colaExec {
		if pcbEnExec != nil && pcbEnExec.PID == pcb.PID {
			delete(colaExec, cpu)
			break
		}
	}
	execMutex.Unlock()

	pcb.MotivoBloqueo = motivo
	pcb.CambiarEstado(EstadoBlocked)

	// Log específico para bloqueo por IO
	if motivo != "" && (motivo[:3] == "IO_" || motivo == "DUMP_MEMORY") {
		dispositivoNombre := motivo
		if motivo[:3] == "IO_" {
			dispositivoNombre = motivo[3:] // Remover "IO_" del prefijo
		}
		utils.InfoLog.Info(fmt.Sprintf("(%d) - Bloqueado por IO: %s", pcb.PID, dispositivoNombre))
	}

	utils.InfoLog.Info("Proceso bloqueado", "pid", pcb.PID, "motivo", motivo)

	blockedMutex.Lock()
	colaBlocked = append(colaBlocked, pcb)
	blockedMutex.Unlock()

	go iniciarTimerSuspension(pcb)
}

// iniciarTimerSuspension con log de inicio
func iniciarTimerSuspension(pcb *PCB) {
	tiempoSuspension := time.Duration(kernelConfig.SuspensionTime) * time.Millisecond
	if tiempoSuspension <= 0 {
		tiempoSuspension = 4500 * time.Millisecond
	}

	// Log para visualizar cuándo se arma el timer
	utils.InfoLog.Info("Iniciado timer de suspensión", "pid", pcb.PID, "duracion_ms", tiempoSuspension.Milliseconds())

	timersMutex.Lock()
	if timer, existe := timersSuspension[pcb.PID]; existe {
		timer.Stop()
	}

	timer := time.AfterFunc(tiempoSuspension, func() {
		suspenderProceso(pcb.PID)
	})
	timersSuspension[pcb.PID] = timer
	timersMutex.Unlock()
}

// suspenderProceso con log de notificación a Memoria
func suspenderProceso(pid int) {
	pcb := BuscarPCBPorPID(pid)
	if pcb == nil {
		utils.InfoLog.Warn("Proceso no encontrado para suspensión", "pid", pid)
		return
	}

	if pcb.Estado != EstadoBlocked {
		utils.InfoLog.Warn("Proceso no válido para suspensión", "pid", pid, "estado_actual", pcb.Estado)
		return
	}

	if !removerDeBlocked(pcb) {
		utils.InfoLog.Warn("No se pudo remover proceso de BLOCKED", "pid", pid)
		return
	}

	// Log para saber que el timer se disparó
	utils.InfoLog.Info("Timer de suspensión finalizado. Suspendiendo proceso.", "pid", pcb.PID)

	pcb.CambiarEstado(EstadoSuspBlocked)
	pcb.EnSwap = true // Marcar que el proceso estará en SWAP

	suspBlockedMutex.Lock()
	colaSuspBlocked = append(colaSuspBlocked, pcb)
	suspBlockedMutex.Unlock()

	go notificarSwapAMemoria(pcb.PID) // La función notificarSwapAMemoria ya la tienes bien.
	semaforoMultiprogram.Signal()
}

// FinalizarProceso optimizado
func FinalizarProceso(pcb *PCB, motivo string) {
	mapaMutex.Lock()
	if _, existe := mapaPCBs[pcb.PID]; !existe || pcb.Estado == EstadoExit {
		mapaMutex.Unlock()
		return
	}
	mapaMutex.Unlock()

	estadoPrevio := pcb.Estado

	// Limpiar timer
	timersMutex.Lock()
	if timer, existe := timersSuspension[pcb.PID]; existe {
		timer.Stop()
		delete(timersSuspension, pcb.PID)
	}
	timersMutex.Unlock()

	// Remover de cola actual
	fueRemovido := false
	switch estadoPrevio {
	case EstadoExec:
		execMutex.Lock()
		for cpu, pcbEnExec := range colaExec {
			if pcbEnExec != nil && pcbEnExec.PID == pcb.PID {
				delete(colaExec, cpu)
				fueRemovido = true
				break
			}
		}
		execMutex.Unlock()
	case EstadoReady:
		fueRemovido = removerDeReady(pcb)
	case EstadoBlocked:
		fueRemovido = removerDeBlocked(pcb)
	case EstadoSuspReady:
		fueRemovido = removerDeSuspReady(pcb)
	case EstadoSuspBlocked:
		fueRemovido = removerDeSuspBlocked(pcb)
	case EstadoNew:
		fueRemovido = removerDeNew(pcb)
	case EstadoExit:
		return
	}

	pcb.CambiarEstado(EstadoExit)

	exitMutex.Lock()
	colaExit = append(colaExit, pcb)
	exitMutex.Unlock()

	// Liberar multiprogramación
	if estadoPrevio == EstadoReady || estadoPrevio == EstadoExec || estadoPrevio == EstadoBlocked {
		semaforoMultiprogram.Signal()
	}

	go notificarFinalizacionAMemoria(pcb.PID)

	if estadoPrevio != EstadoExit {
		utils.InfoLog.Info(fmt.Sprintf("(%d) - Finaliza el proceso", pcb.PID))
		utils.InfoLog.Info("Proceso finalizado", "pid", pcb.PID, "motivo", motivo)
		pcb.CalcularMetricas()
	}

	mapaMutex.Lock()
	delete(mapaPCBs, pcb.PID)
	mapaMutex.Unlock()

	// Usar variable para evitar warning del compilador
	_ = fueRemovido
}

// notificarFinalizacionAMemoria simplificado
func notificarFinalizacionAMemoria(pid int) {
	cliente := GetMemoriaClient()
	if cliente == nil {
		utils.ErrorLog.Error("No se pudo obtener cliente de memoria para finalización", "pid", pid)
		return
	}

	datos := map[string]interface{}{
		"pid": pid,
	}

	_, err := cliente.EnviarHTTPMensaje(utils.MensajeFinalizarProceso, "default", datos)
	if err != nil {
		utils.ErrorLog.Error("Error notificando finalización a Memoria", "pid", pid, "error", err.Error())
	}
}

// Funciones auxiliares optimizadas
func removerDeReady(pcb *PCB) bool {
	readyMutex.Lock()
	defer readyMutex.Unlock()
	return removerDeCola(&colaReady, pcb)
}

func removerDeBlocked(pcb *PCB) bool {
	blockedMutex.Lock()
	defer blockedMutex.Unlock()
	return removerDeCola(&colaBlocked, pcb)
}

func removerDeNew(pcb *PCB) bool {
	newMutex.Lock()
	defer newMutex.Unlock()
	return removerDeCola(&colaNew, pcb)
}

func removerDeSuspReady(pcb *PCB) bool {
	suspReadyMutex.Lock()
	defer suspReadyMutex.Unlock()
	return removerDeCola(&colaSuspReady, pcb)
}

func removerDeSuspBlocked(pcb *PCB) bool {
	suspBlockedMutex.Lock()
	defer suspBlockedMutex.Unlock()
	return removerDeCola(&colaSuspBlocked, pcb)
}

// Template function para remover de cualquier cola
func removerDeCola(cola *[]*PCB, pcb *PCB) bool {
	for i, p := range *cola {
		if p.PID == pcb.PID {
			*cola = append((*cola)[:i], (*cola)[i+1:]...)
			return true
		}
	}
	return false
}

// Funciones de planificación optimizadas
func intentarAdmitirProceso() {
	newMutex.Lock()
	defer newMutex.Unlock()

	if len(colaNew) > 0 {
		utils.InfoLog.Info("Señal enviada al LTS", "procesos_en_new", len(colaNew))
		condNew.Signal()
	}
}

func despacharProcesoSiCorresponde() {
	condReady.Signal()
}

func notificarSwapAMemoria(pid int) {
	cliente := GetMemoriaClient()
	if cliente == nil {
		utils.ErrorLog.Error("No se pudo obtener cliente de memoria para swap", "pid", pid)
		return
	}

	datos := map[string]interface{}{
		"pid": pid,
	}
	cliente.EnviarHTTPMensaje(utils.MensajeSuspenderProceso, "default", datos)
}
