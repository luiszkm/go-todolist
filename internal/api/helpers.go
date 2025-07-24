package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// respondWithError envia uma resposta de erro JSON.
func respondWithError(w http.ResponseWriter, logger *slog.Logger, code int, message string) {
	// Logamos o erro internamente antes de responder ao cliente.
	logger.Error("resposta de erro da API", "status", code, "mensagem", message)
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON envia uma resposta JSON.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		// Se houver um erro ao serializar a resposta, é um problema do servidor.
		// Logamos o erro e enviamos um 500 genérico.
		// O logger não é passado aqui, mas em um app real, teríamos acesso a ele.
		// Para simplicidade, usamos o logger padrão do http.
		http.Error(w, "Erro ao gerar resposta JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
