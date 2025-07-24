# --- Estágio de Build ---
# Usamos uma imagem oficial do Go como base para compilar nossa aplicação.
FROM golang:1.22-alpine AS builder

# Define o diretório de trabalho dentro do container.
WORKDIR /app

# Copia os arquivos de módulo e baixa as dependências primeiro.
# Isso aproveita o cache do Docker se as dependências não mudarem.
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código fonte.
COPY . .

# Compila a aplicação.
# -o /app/server: Define o nome e local do binário de saída.
# -ldflags="-w -s": Remove informações de debug, resultando em um binário menor.
# CGO_ENABLED=0: Garante que o binário seja estaticamente linkado.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server -ldflags="-w -s" ./cmd/server

# --- Estágio Final ---
# Usamos a imagem 'scratch', que é uma imagem vazia, para a segurança
# e tamanho mínimos. Alpine também é uma ótima opção se precisar de um shell.
FROM scratch

# Define o diretório de trabalho.
WORKDIR /app

# Copia apenas o binário compilado do estágio 'builder'.
COPY --from=builder /app/server .

# Expõe a porta que nossa aplicação irá escutar.
EXPOSE 8080

# Define o comando para executar quando o container iniciar.
ENTRYPOINT ["/app/server"]