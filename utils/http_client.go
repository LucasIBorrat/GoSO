package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)


// Mensaje representa un mensaje genérico entre módulos
type Mensaje struct {
	Tipo      int         `json:"tipo"`
	Operacion string      `json:"operacion"`
	Origen    string      `json:"origen"`
	Datos     interface{} `json:"datos"`
}

// HTTPClient representa un cliente HTTP para comunicación entre módulos
type HTTPClient struct {
	BaseURL string
	Nombre  string
	client  *http.Client
}

// NewHTTPClient crea un nuevo cliente HTTP
func NewHTTPClient(ip string, puerto int, nombre string) *HTTPClient {
	return &HTTPClient{
		BaseURL: fmt.Sprintf("http://%s:%d", ip, puerto),
		Nombre:  nombre,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EnviarHTTPMensaje envía un mensaje a través de HTTP
func (c *HTTPClient) EnviarHTTPMensaje(tipo int, operacion string, datos interface{}) (interface{}, error) {
	mensaje := Mensaje{
		Tipo:      tipo,
		Operacion: operacion,
		Origen:    c.Nombre,
		Datos:     datos,
	}

	jsonData, err := json.Marshal(mensaje)
	if err != nil {
		return nil, fmt.Errorf("error al serializar mensaje: %v", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/mensaje", c.BaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("error al enviar mensaje HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("respuesta HTTP no exitosa: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var resultado interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resultado); err != nil {
		return nil, fmt.Errorf("error al decodificar respuesta: %v", err)
	}

	return resultado, nil
}


// VerificarConexion verifica si un módulo está disponible
func (c *HTTPClient) VerificarConexion() error {
	resp, err := c.client.Get(fmt.Sprintf("%s/health", c.BaseURL))
	if err != nil {
		return fmt.Errorf("error al verificar conexión con %s: %v", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("estado inesperado al verificar conexión: %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error al decodificar respuesta de verificación: %v", err)
	}

	slog.Info("Conexión verificada", "destino", c.BaseURL, "módulo", result["module"])
	return nil
}

// EnviarHTTPOperacion envía un mensaje de operación a través de HTTP
func (c *HTTPClient) EnviarHTTPOperacion(operacion string, datos map[string]interface{}) (interface{}, error) {
	return c.EnviarHTTPMensaje(MensajeOperacion, operacion, datos)
}
