package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv" // Importa a nova biblioteca
	"github.com/luiszkm/go-todolist/internal/api"
	"github.com/luiszkm/go-todolist/internal/storage"
)

func main() {
	// Inicializa o logger estruturado.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Carrega as variáveis de ambiente do arquivo .env.
	// É idiomático ignorar o erro se o arquivo não existir,
	// pois em produção as variáveis serão injetadas diretamente.
	if err := godotenv.Load(); err != nil {
		logger.Info("nenhum arquivo .env encontrado, usando variáveis de ambiente do sistema")
	}

	// Captura o sinal de interrupção (Ctrl+C) para um encerramento gracioso.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Obtém a string de conexão do banco de dados a partir das variáveis de ambiente.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("a variável de ambiente DATABASE_URL não está definida")
		os.Exit(1)
	}

	// O restante do arquivo permanece o mesmo...
	store, err := storage.NewPostgresStore(ctx, dbURL, logger)
	if err != nil {
		logger.Error("falha ao conectar com o banco de dados", "erro", err)
		os.Exit(1)
	}
	defer store.Close()

	serverAddr := ":8080"
	apiServer := api.NewAPIServer(serverAddr, logger, store)
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	go func() {
		logger.Info("servidor iniciando", "endereço", serverAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("falha ao iniciar o servidor HTTP", "erro", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	stop()
	logger.Info("servidor recebendo sinal para desligar")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("falha no graceful shutdown do servidor", "erro", err)
		os.Exit(1)
	}

	logger.Info("servidor desligado com sucesso")
}
