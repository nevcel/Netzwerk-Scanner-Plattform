# ============================================================
# Build-Phase: Go-Backend kompilieren
# ============================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Moduldateien zuerst kopieren, damit Docker bereits
# heruntergeladene Abhängigkeiten wiederverwenden kann.
COPY go.mod go.sum ./

RUN go mod download

# Anschliessend wird der eigentliche Quellcode übernommen.
COPY . .

# Backend als ausführbare Linux-Datei erstellen.
RUN CGO_ENABLED=0 GOOS=linux go build -o /backend ./cmd/api


# ============================================================
# Laufzeit-Phase: Minimales Backend-Image
# ============================================================
FROM alpine:3.21

# Zertifikate werden für spätere sichere HTTP-Verbindungen
# und wget für die Health-Prüfung bereitgestellt.
RUN apk add --no-cache ca-certificates

# Die Anwendung wird nicht mit Root-Rechten ausgeführt.
RUN addgroup -S backend && adduser -S backend -G backend

WORKDIR /app

# Nur die kompilierte Anwendung wird übernommen.
COPY --from=builder /backend /app/backend

USER backend

# REST-Schnittstelle des Backends.
EXPOSE 8080

CMD ["/app/backend"]