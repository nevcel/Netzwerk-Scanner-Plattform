# In der ersten Phase wird die Scanner-Anwendung kompiliert.
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Das Go-Modul wird vor dem Quellcode kopiert,
# damit Docker vorhandene Build-Schichten wiederverwenden kann.
COPY go.mod ./

COPY . .

# Die Scanner-Anwendung wird als ausführbare Linux-Datei erstellt.
RUN CGO_ENABLED=0 GOOS=linux go build -o /scanner ./cmd/scanner


# Das endgültige Image enthält nur die benötigte Laufzeitumgebung.
FROM alpine:3.21

# Nmap wird direkt im Scanner-Container installiert.
# Eine separate Installation auf dem Benutzerrechner ist nicht notwendig.
RUN apk add --no-cache nmap nmap-scripts ca-certificates

# Der Scanner erhält einen eigenen Benutzer ohne Root-Rechte.
RUN addgroup -S scanner && adduser -S scanner -G scanner

WORKDIR /app


COPY --from=builder /scanner /app/scanner

USER scanner

# Der Scanner stellt seine HTTP-Schnittstelle auf Port 8081 bereit.
EXPOSE 8081

CMD ["/app/scanner"]