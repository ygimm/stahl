FROM golang:alpine AS builder

# Установка зависимостей для сборки
RUN apk update && \
    apk add --no-cache ca-certificates git && \
    update-ca-certificates

# Создание рабочей директории
WORKDIR /build

# Копирование и скачивание зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY api/ ./api/
# В зависимости от структуры проекта, возможно, нужно скопировать другие директории

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o main -ldflags="-s -w" -a -installsuffix cgo ./cmd/stahl/main.go

# Создание минимального финального образа
FROM alpine:latest

# Установка необходимых зависимостей для запуска
RUN apk --no-cache add ca-certificates && \
    update-ca-certificates && \
    # Создание непривилегированного пользователя
    adduser -D -H -h /app stahl

# Рабочая директория
WORKDIR /app

# Копирование только исполняемого файла из этапа сборки
COPY --from=builder /build/main .

# Копирование конфигурационных файлов
COPY .stahl/config/ ./config/

# Переименование конфигурационного файла и настройка прав доступа
RUN cp ./config/config.example.yml ./config/config.yml && \
    chmod 644 ./config/config.yml && \
    chown -R stahl:stahl ./config/

# Добавим команду для проверки доступности хостов
RUN echo "hosts: files dns" > /etc/nsswitch.conf

# Указание порта для работы приложения
EXPOSE 8000

# Переход на пользователя с меньшими привилегиями
USER stahl

# Запуск приложения
ENTRYPOINT ["./main"]