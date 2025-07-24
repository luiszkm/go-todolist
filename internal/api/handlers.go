package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/luiszkm/go-todolist/internal/storage"
	"github.com/luiszkm/go-todolist/internal/todo"
)

// APIServer encapsula as dependências do servidor da API, como o logger e o storage.
type APIServer struct {
	addr   string
	logger *slog.Logger
	store  storage.Store
}

// NewAPIServer cria uma nova instância do nosso servidor da API.
func NewAPIServer(addr string, logger *slog.Logger, store storage.Store) *APIServer {
	return &APIServer{
		addr:   addr,
		logger: logger,
		store:  store,
	}
}

// RegisterRoutes registra todos os handlers da nossa API no mux.
func (s *APIServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/todos", s.handleTodos)
	mux.HandleFunc("/todos/", s.handleTodoByID) // Note a barra no final para capturar /todos/qualquer-coisa
}

// handleTodos é um dispatcher que decide entre Listar e Criar baseado no método HTTP.
func (s *APIServer) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTodos(w, r)
	case http.MethodPost:
		s.handleCreateTodo(w, r)
	default:
		respondWithError(w, s.logger, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

// handleTodoByID é um dispatcher para rotas que incluem um ID.
func (s *APIServer) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	// Extrai o ID da URL. Ex: /todos/uuid-vai-aqui
	id := strings.TrimPrefix(r.URL.Path, "/todos/")
	if id == "" {
		respondWithError(w, s.logger, http.StatusBadRequest, "ID do To-Do não pode ser vazio")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTodo(w, r, id)
	case http.MethodPut:
		s.handleUpdateTodo(w, r, id)
	case http.MethodDelete:
		s.handleDeleteTodo(w, r, id)
	default:
		respondWithError(w, s.logger, http.StatusMethodNotAllowed, "Método não permitido")
	}
}

func (s *APIServer) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	var newTodo todo.Todo
	if err := json.NewDecoder(r.Body).Decode(&newTodo); err != nil {
		respondWithError(w, s.logger, http.StatusBadRequest, "Payload da requisição inválido")
		return
	}

	if newTodo.Title == "" {
		respondWithError(w, s.logger, http.StatusBadRequest, "O título é obrigatório")
		return
	}

	createdTodo, err := s.store.CreateTodo(r.Context(), newTodo)
	if err != nil {
		respondWithError(w, s.logger, http.StatusInternalServerError, "Falha ao criar to-do")
		return
	}

	respondWithJSON(w, http.StatusCreated, createdTodo)
}

func (s *APIServer) handleListTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := s.store.ListTodos(r.Context())
	if err != nil {
		respondWithError(w, s.logger, http.StatusInternalServerError, "Falha ao listar to-dos")
		return
	}

	// Retorna uma lista vazia em vez de nula se não houver tarefas.
	if todos == nil {
		todos = []todo.Todo{}
	}

	respondWithJSON(w, http.StatusOK, todos)
}

func (s *APIServer) handleGetTodo(w http.ResponseWriter, r *http.Request, id string) {
	foundTodo, err := s.store.GetTodo(r.Context(), id)
	if err != nil {
		// Podemos ser mais específicos aqui, verificando se o erro é 'não encontrado'.
		if strings.Contains(err.Error(), "não encontrado") {
			respondWithError(w, s.logger, http.StatusNotFound, "To-do não encontrado")
		} else {
			respondWithError(w, s.logger, http.StatusInternalServerError, "Falha ao buscar to-do")
		}
		return
	}
	respondWithJSON(w, http.StatusOK, foundTodo)
}

func (s *APIServer) handleUpdateTodo(w http.ResponseWriter, r *http.Request, id string) {
	var updatedTodo todo.Todo
	if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
		respondWithError(w, s.logger, http.StatusBadRequest, "Payload da requisição inválido")
		return
	}

	if updatedTodo.Title == "" {
		respondWithError(w, s.logger, http.StatusBadRequest, "O título é obrigatório")
		return
	}

	result, err := s.store.UpdateTodo(r.Context(), id, updatedTodo)
	if err != nil {
		if strings.Contains(err.Error(), "não encontrado") {
			respondWithError(w, s.logger, http.StatusNotFound, "To-do não encontrado para atualizar")
		} else {
			respondWithError(w, s.logger, http.StatusInternalServerError, "Falha ao atualizar to-do")
		}
		return
	}
	respondWithJSON(w, http.StatusOK, result)
}

func (s *APIServer) handleDeleteTodo(w http.ResponseWriter, r *http.Request, id string) {
	err := s.store.DeleteTodo(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "não encontrado") {
			respondWithError(w, s.logger, http.StatusNotFound, "To-do não encontrado para deletar")
		} else {
			respondWithError(w, s.logger, http.StatusInternalServerError, "Falha ao deletar to-do")
		}
		return
	}
	respondWithJSON(w, http.StatusNoContent, nil)
}
