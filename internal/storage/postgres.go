package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // Driver do PostgreSQL
	"github.com/luiszkm/go-todolist/internal/todo"
)

// Store define a interface para as operações de armazenamento de dados.
// Usar uma interface nos permite trocar a implementação (ex: para um mock nos testes).
type Store interface {
	CreateTodo(ctx context.Context, t todo.Todo) (*todo.Todo, error)
	GetTodo(ctx context.Context, id string) (*todo.Todo, error)
	ListTodos(ctx context.Context) ([]todo.Todo, error)
	UpdateTodo(ctx context.Context, id string, t todo.Todo) (*todo.Todo, error)
	DeleteTodo(ctx context.Context, id string) error
}

// PostgresStore é a implementação concreta da interface Store para o PostgreSQL.
type PostgresStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresStore cria uma nova conexão com o banco de dados e retorna uma instância de PostgresStore.
func NewPostgresStore(ctx context.Context, dsn string, logger *slog.Logger) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexão com o banco de dados: %w", err)
	}

	// Verifica se a conexão é válida.
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("falha ao verificar conexão com o banco de dados: %w", err)
	}

	logger.Info("conexão com o banco de dados PostgreSQL estabelecida com sucesso")

	return &PostgresStore{db: db, logger: logger}, nil
}

// Close fecha a conexão com o banco de dados.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// CreateTodo insere uma nova tarefa no banco de dados.
func (s *PostgresStore) CreateTodo(ctx context.Context, t todo.Todo) (*todo.Todo, error) {
	query := `
		INSERT INTO todos (title, description, completed)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRowContext(ctx, query, t.Title, t.Description, t.Completed).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: falha ao criar todo: %w", err)
	}
	return &t, nil
}

// GetTodo busca uma tarefa pelo seu ID.
func (s *PostgresStore) GetTodo(ctx context.Context, id string) (*todo.Todo, error) {
	var t todo.Todo
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM todos
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("storage: todo com id '%s' não encontrado: %w", id, err)
		}
		return nil, fmt.Errorf("storage: falha ao buscar todo: %w", err)
	}
	return &t, nil
}

// ListTodos retorna todas as tarefas do banco de dados.
func (s *PostgresStore) ListTodos(ctx context.Context) ([]todo.Todo, error) {
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM todos
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("storage: falha ao listar todos: %w", err)
	}
	defer rows.Close()

	var todos []todo.Todo
	for rows.Next() {
		var t todo.Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: falha ao escanear linha do todo: %w", err)
		}
		todos = append(todos, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: erro durante iteração das linhas de todos: %w", err)
	}

	return todos, nil
}

// UpdateTodo atualiza uma tarefa existente.
func (s *PostgresStore) UpdateTodo(ctx context.Context, id string, t todo.Todo) (*todo.Todo, error) {
	query := `
		UPDATE todos
		SET title = $1, description = $2, completed = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, title, description, completed, created_at, updated_at
	`
	var updatedTodo todo.Todo
	err := s.db.QueryRowContext(ctx, query, t.Title, t.Description, t.Completed, id).Scan(
		&updatedTodo.ID, &updatedTodo.Title, &updatedTodo.Description, &updatedTodo.Completed, &updatedTodo.CreatedAt, &updatedTodo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("storage: impossível atualizar, todo com id '%s' não encontrado: %w", id, err)
		}
		return nil, fmt.Errorf("storage: falha ao atualizar todo: %w", err)
	}
	return &updatedTodo, nil
}

// DeleteTodo remove uma tarefa do banco de dados.
func (s *PostgresStore) DeleteTodo(ctx context.Context, id string) error {
	query := `DELETE FROM todos WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("storage: falha ao deletar todo: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: falha ao verificar linhas afetadas ao deletar: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("storage: impossível deletar, todo com id '%s' não encontrado", id)
	}

	return nil
}
