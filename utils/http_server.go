package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// HTTPHandlerFunc es el tipo para los manejadores de mensajes HTTP
type HTTPHandlerFunc func(*Mensaje) (interface{}, error)

// HTTPServer representa un servidor HTTP para cualquier módulo
type HTTPServer struct {
	IP       string
	Puerto   int
	Nombre   string
	server   *http.Server
	handlers map[int]HTTPHandlerFunc
	Listener net.Listener
}

// NewHTTPServer crea un nuevo servidor HTTP
func NewHTTPServer(ip string, puerto int, nombre string) *HTTPServer {
	return &HTTPServer{
		IP:       ip,
		Puerto:   puerto,
		Nombre:   nombre,
		handlers: make(map[int]HTTPHandlerFunc),
	}
}

// RegisterHTTPHandler registra un manejador para un tipo específico de mensaje
func (s *HTTPServer) RegisterHTTPHandler(tipoMensaje int, handler HTTPHandlerFunc) {
	s.handlers[tipoMensaje] = handler
}

// Start inicia el servidor HTTP
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// Endpoint para recibir mensajes
	mux.HandleFunc("/mensaje", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var mensaje Mensaje
		err := json.NewDecoder(r.Body).Decode(&mensaje)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error decodificando mensaje: %v", err), http.StatusBadRequest)
			return
		}

		handler, exists := s.handlers[mensaje.Tipo]
		if !exists {
			http.Error(w, fmt.Sprintf("No hay manejador para el tipo de mensaje %d", mensaje.Tipo), http.StatusBadRequest)
			return
		}

		respuesta, err := handler(&mensaje)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error en el manejador: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respuesta)
	})

	// Endpoint de healthcheck
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "module": s.Nombre})
	})

	// Si ya tiene Listener asignado (caso IO)
	if s.Listener != nil {
		slog.Info("Servidor HTTP escuchando", "módulo", s.Nombre, "dirección", s.Listener.Addr().String())
		return http.Serve(s.Listener, mux)
	}

	// Caso normal para otros módulos
	address := fmt.Sprintf("%s:%d", s.IP, s.Puerto)
	s.server = &http.Server{
		Addr:    address,
		Handler: mux,
	}

	slog.Info("Servidor HTTP escuchando", "módulo", s.Nombre, "dirección", address)
	return s.server.ListenAndServe()
}

