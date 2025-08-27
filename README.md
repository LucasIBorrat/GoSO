# GoSO - Simulador de Sistema Operativo

## Descripción

GoSO es un simulador de sistema operativo distribuido desarrollado en Go que implementa los componentes fundamentales de un sistema operativo moderno. El proyecto simula la arquitectura completa de un SO, incluyendo manejo de memoria virtual, planificación de procesos, gestión de entrada/salida y comunicación entre módulos.

## Arquitectura del Sistema

El simulador está compuesto por cuatro módulos principales que se comunican a través de HTTP:

### 🧠 **Kernel**
- **Ubicación**: `cmd/kernel/`
- **Función**: Núcleo del sistema operativo que coordina todos los demás módulos
- **Características**:
  - Planificador de corto plazo (STS)
  - Planificador de mediano/largo plazo (LTS)
  - Gestión de PCBs (Process Control Blocks)
  - Manejo de estados de procesos (NEW, READY, EXEC, BLOCKED, SUSPENDED, EXIT)
  - Algoritmos de planificación: FIFO, SJF, SRT, PMCP
  - Control de grado de multiprogramación

### 💾 **Memoria**
- **Ubicación**: `cmd/memoria/`
- **Función**: Gestión de memoria virtual y física
- **Características**:
  - Paginación y segmentación
  - Swap de procesos a disco
  - Gestión de marcos de página
  - Algoritmos de reemplazo de páginas
  - Tablas de páginas por proceso
  - Métricas de uso de memoria

### ⚡ **CPU**
- **Ubicación**: `cmd/cpu/`
- **Función**: Simulación de unidades de procesamiento
- **Características**:
  - Ejecución de instrucciones de pseudocódigo
  - TLB (Translation Lookaside Buffer) con algoritmos FIFO/LRU
  - Cache de datos con algoritmos CLOCK/CLOCK-M
  - Manejo de interrupciones
  - Ciclo fetch-decode-execute

### 🔌 **E/S (I/O)**
- **Ubicación**: `cmd/io/`
- **Función**: Simulación de dispositivos de entrada/salida
- **Características**:
  - Dispositivos configurables (DISCO, TECLADO, etc.)
  - Tiempos de operación simulados
  - Gestión de colas de E/S
  - Comunicación asíncrona con el kernel

## Características Principales

### Planificación de Procesos
- **Corto Plazo**: FIFO, SJF (Shortest Job First), SRT (Shortest Remaining Time)
- **Mediano/Largo Plazo**: FIFO, PMCP (Programación Multiprogramada Controlada por Prioridad)
- Control de grado de multiprogramación
- Suspensión y reanudación de procesos

### Gestión de Memoria
- **Paginación**: División de memoria en páginas de tamaño fijo
- **Swap**: Intercambio de procesos entre memoria y disco
- **TLB**: Cache de traducciones de direcciones virtuales
- **Cache**: Almacenamiento temporal de datos frecuentemente accedidos
- **Algoritmos**: FIFO, LRU, CLOCK, CLOCK-M

### Comunicación entre Módulos
- Protocolo HTTP/REST para comunicación entre módulos
- Mensajes estructurados para operaciones específicas
- Manejo de errores y reconexión automática
- Logging detallado de todas las operaciones

## Instrucciones de Compilación

### Prerrequisitos
- Go 1.23.1 o superior
- Sistema operativo: Windows, Linux o macOS

### Compilación
```bash
# Compilar todos los módulos
go build -o bin/kernel cmd/kernel/*.go
go build -o bin/memoria cmd/memoria/*.go
go build -o bin/cpu cmd/cpu/*.go
go build -o bin/io cmd/io/*.go
```

## Uso del Sistema

### Orden de Inicio de Módulos
1. **Memoria** (siempre primero)
2. **Dispositivos I/O**
3. **CPUs**
4. **Kernel** (último)

### Ejemplo de Ejecución Completa

```bash
# Terminal 1: Iniciar Memoria
./bin/memoria configs/memoria-config-PlaniCorto.json

# Terminal 2: Iniciar Dispositivo I/O
./bin/io DISCO1 configs/io1-config-PlaniCorto.json

# Terminal 3: Iniciar CPU
./bin/cpu CPU1 configs/cpu1-config-PlaniCorto.json

# Terminal 4: Iniciar Kernel
./bin/kernel configs/kernel-config-PlaniCortoFIFO.json scripts/PLANI_CORTO_PLAZO 0
```

### Parámetros del Kernel
```bash
./kernel <archivo_configuracion> <script_inicial> <tamaño_proceso>
```

- **archivo_configuracion**: Archivo JSON con la configuración del kernel
- **script_inicial**: Script de pseudocódigo a ejecutar
- **tamaño_proceso**: Tamaño en bytes del proceso inicial

## Configuración

### Archivos de Configuración
Los archivos de configuración se encuentran en la carpeta `configs/` y permiten personalizar:

- **Direcciones IP y puertos** de comunicación
- **Algoritmos de planificación**
- **Tamaños de memoria y cache**
- **Algoritmos de reemplazo**
- **Tiempos de operación**
- **Grado de multiprogramación**

### Scripts de Pseudocódigo
Los scripts se ubican en `scripts/` e incluyen instrucciones como:
- `NOOP`: No operación
- `INIT_PROC`: Crear nuevo proceso
- `IO`: Operación de entrada/salida
- `EXIT`: Finalizar proceso
- `GOTO`: Salto condicional/incondicional

## Pruebas Disponibles

El proyecto incluye múltiples escenarios de prueba:

### Planificación
- **Corto Plazo**: Pruebas con FIFO, SJF, SRT
- **Mediano/Largo Plazo**: Pruebas con FIFO, PMCP
- **Multiprogramación**: Control de grado de multiprogramación

### Memoria
- **Cache**: Algoritmos CLOCK y CLOCK-M
- **TLB**: Algoritmos FIFO y LRU
- **Swap**: Intercambio de procesos a disco
- **Paginación**: Gestión de páginas y marcos

### Estabilidad
- **Estabilidad General**: Pruebas integrales con múltiples módulos
- **Stress Testing**: Pruebas con alta carga de trabajo

## Estructura del Proyecto

```
GoSO/
├── cmd/                    # Módulos principales
│   ├── kernel/            # Kernel del SO
│   ├── memoria/           # Gestión de memoria
│   ├── cpu/               # Unidades de procesamiento
│   └── io/                # Dispositivos de E/S
├── configs/               # Archivos de configuración
├── scripts/               # Scripts de pseudocódigo
├── utils/                 # Utilidades compartidas
├── swap/                  # Archivos de intercambio
└── dump/                  # Volcados de memoria
```

## Utilidades Compartidas

### `utils/`
- **logger.go**: Sistema de logging unificado
- **http_client.go**: Cliente HTTP para comunicación
- **http_server.go**: Servidor HTTP base
- **semaforo.go**: Implementación de semáforos
- **operaciones.go**: Operaciones comunes
- **modulo.go**: Base para todos los módulos

## Características Técnicas

### Concurrencia
- Uso extensivo de goroutines para operaciones asíncronas
- Sincronización con mutexes y condition variables
- Semáforos para control de recursos
- Canales para comunicación entre goroutines

### Logging
- Sistema de logging estructurado
- Niveles de log configurables (DEBUG, INFO, WARN, ERROR)
- Logs específicos por módulo
- Formato consistente con timestamps

### Configuración
- Archivos JSON para configuración flexible
- Validación de configuraciones al inicio
- Parámetros específicos por módulo y algoritmo

## Autores

- **Equipo**: Los Cuervos Xeneizes
- **Proyecto**: Trabajo Práctico - Sistemas Operativos
- **Institución**: Universidad Tecnológica Nacional

## Licencia

Este proyecto es parte de un trabajo práctico académico para la materia Sistemas Operativos.

---

**Nota**: Para ejecutar las pruebas completas, consulte el archivo `SCRIPTS TESTS.txt` que contiene todos los comandos necesarios para diferentes escenarios de prueba.
