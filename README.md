# Netzwerk Scanner

Lokale, containerisierte Netzwerk-Scanning-Plattform für autorisierte IPv4-Netzwerke. Entwickelt von Nevzat Celik im Rahmen der Diplomarbeit HF Informatik an der TEKO.

Die Anwendung ermöglicht es, Scanaufträge über eine Weboberfläche zu erstellen, Netzwerkdienste mit Nmap zu untersuchen und Ergebnisse dauerhaft in PostgreSQL zu speichern.

## Funktionen

- Browserbasierte Bedienung mit HTML, CSS und JavaScript.
- Go-Backend mit REST-API.
- Separate Scanner-Komponente mit kontrollierter Nmap-Ausführung.
- Asynchrone Verarbeitung mit zwei parallelen Scans und bis zu 100 wartenden Aufträgen.
- Anzeige von IP-Adressen, Portzuständen und verfügbaren Dienst-, Produkt- und Versionsinformationen.
- Persistente Speicherung und erneuter Aufruf über die Scan-Historie.
- Eingabevalidierung, Statusverwaltung und Fehlerprotokollierung.

## Voraussetzungen

Für den Betrieb aus dem Quellcode:

- Docker Desktop mit aktivierten Linux-Containern und Docker Compose.
- Netzwerkzugriff vom Docker-System auf die autorisierten Zielgeräte.
- Internetzugang beim ersten Build zum Laden der Basis-Images und Abhängigkeiten.
- Freie lokale Ports 3000, 8080, 8081 und 5432.

Go, PostgreSQL und Nmap müssen für den Docker-Betrieb nicht separat auf dem Computer installiert werden.

Für die lokale Ausführung der Go-Tests wird Go 1.25 oder eine kompatible neuere Version benötigt. Die PowerShell-Testskripte setzen PowerShell voraus.

## Installation und Start

Die folgenden Befehle werden in PowerShell im Projektordner ausgeführt.

### 1. Konfiguration erstellen

```powershell
Copy-Item .env.example .env
```

Die Datei `.env` öffnen und `POSTGRES_PASSWORD` auf ein eigenes Passwort setzen. Den Beispielwert `change_me` ersetzen.

| Variable | Verwendung |
|---|---|
| `POSTGRES_DB` | Name der PostgreSQL-Datenbank |
| `POSTGRES_USER` | Datenbankbenutzer |
| `POSTGRES_PASSWORD` | Passwort des Datenbankbenutzers |

Die lokale `.env` enthält Zugangsdaten und gehört nicht in ein öffentliches Repository oder Abgabearchiv.

### 2. Anwendung bauen und starten

```powershell
docker compose up -d --build
```

### 3. Containerstatus prüfen

```powershell
docker compose ps
```

### 4. Weboberfläche öffnen

[http://127.0.0.1:3000](http://127.0.0.1:3000)

## Bedienung

1. Eine eigene oder ausdrücklich autorisierte private IPv4-Adresse beziehungsweise einen zulässigen Netzwerkbereich eingeben.
2. Auf **Scan starten** klicken.
3. Den Bearbeitungszustand verfolgen.
4. Nach Abschluss die erkannten Hosts, Ports und Dienstinformationen prüfen.
5. Frühere Ergebnisse über die **Scan-Historie** erneut öffnen.

Beispiele für das Eingabeformat:

```text
192.168.178.1
192.168.178.0/24
```

Die Beispiele sind an das tatsächlich autorisierte Zielnetz anzupassen.

## Architektur

| Komponente | Aufgabe | Interner Port |
|---|---|---:|
| Frontend | Nginx, Weboberfläche und Weiterleitung der API-Anfragen | 8080 |
| Backend | REST-API, Auftragskoordination und Datenbankzugriff | 8080 |
| Scanner | Warteschlange, Worker und Nmap-Ausführung | 8081 |
| PostgreSQL | Speicherung der Scanläufe und Ergebnisse | 5432 |

Der Browser greift über den lokalen Port 3000 auf das Frontend zu. Nginx leitet API-Anfragen intern an das Backend weiter. Das Backend kommuniziert mit dem Scanner und PostgreSQL.

Die Entwicklungs-Konfiguration veröffentlicht Backend, Scanner und Datenbank zusätzlich auf der Loopback-Adresse des Computers.

## Scanprofil

Die Anwendung verwendet einen TCP-Connect-Scan mit fest vorgegebenen Parametern:

| Parameter | Bedeutung |
|---|---|
| `-sT` | TCP-Connect-Scan |
| `-sV` | Dienst- und Versionserkennung |
| `--version-light` | Reduzierte Intensität der Diensterkennung |
| `--top-ports 100` | Prüfung der 100 häufigsten TCP-Ports |
| `-T4` | Beschleunigtes Timing-Profil |
| `--max-retries 2` | Höchstens zwei Wiederholungen bei unbeantworteten Portprüfungen |
| `-oX -` | XML-Ausgabe zur Verarbeitung durch das Backend |

Bei Netzwerkbereichen erfolgt die Host-Erkennung über:

```text
-PS22,80,443,445,3389
```

Einzelne IP-Adressen werden mit `-Pn` ohne vorgelagerte Host-Erkennung geprüft. Bei solchen Scans ist die Statusangabe `up` allein kein unabhängiger Erreichbarkeitsnachweis.

Die maximale Nmap-Laufzeit beträgt **30 Minuten pro gestarteter Ausführung**. Die Wartezeit in der Auftragswarteschlange ist darin nicht enthalten.

## Daten und Neustartverhalten

PostgreSQL speichert die Daten in einem benannten Docker-Volume. Ein normales Stoppen oder Neuerstellen der Container entfernt dieses Volume nicht.

Die Scanner-Komponente hält ihre Aufträge im Arbeitsspeicher. Bei einem Scanner-Neustart können wartende oder laufende Aufträge nicht fortgesetzt werden. Sobald der Scanner wieder erreichbar ist und einen offenen Auftrag nicht mehr kennt, setzt das Backend dessen gespeicherten Status auf `failed`.

Bereits abgeschlossene und gespeicherte Scanresultate bleiben erhalten.

Das Datenbankschema wird bei der ersten Initialisierung eines leeren Datenbank-Volumes angelegt. Änderungen an `.env` ersetzen keine Zugangsdaten einer bereits initialisierten Datenbank.

## Betrieb und Diagnose

Container stoppen und entfernen; gespeicherte Daten behalten:

```powershell
docker compose down
```

Anwendung erneut starten:

```powershell
docker compose up -d
```

Logs anzeigen:

```powershell
docker compose logs --tail 100 backend scanner
```

Logs laufend verfolgen:

```powershell
docker compose logs -f backend scanner
```

Scanner nach einer Quellcodeänderung neu bauen:

```powershell
docker compose up -d --build scanner
```

Lokale Scanner-Image-ID zur Dokumentation eines Teststands anzeigen:

```powershell
docker inspect --format '{{.Image}}' network-scanner-scanner
```

Das Entfernen des Datenbank-Volumes löscht die gespeicherten Scanläufe und Ergebnisse. Für einen normalen Neustart darf `docker compose down` deshalb nicht um `-v` ergänzt werden.

## REST-Schnittstellen

| Methode | Endpunkt | Funktion |
|---|---|---|
| `GET` | `/health` | Erreichbarkeit des Backends prüfen |
| `GET` | `/api/scanner/health` | Verbindung zur Scanner-Komponente prüfen |
| `POST` | `/api/scans` | Scanauftrag erstellen |
| `GET` | `/api/scans` | Die letzten 50 gespeicherten Scanläufe abrufen |
| `GET` | `/api/scans/{id}` | Status eines Scanauftrags abrufen |
| `GET` | `/api/scans/{id}/results` | Strukturierte Scanergebnisse abrufen |

Beispiel für das Erstellen eines Auftrags:

```powershell
Invoke-RestMethod `
    -Uri "http://127.0.0.1:8080/api/scans" `
    -Method Post `
    -ContentType "application/json" `
    -Body '{"target":"192.168.178.1"}'
```

Ein angenommener Auftrag wird mit HTTP 202 beantwortet. Die Antwort enthält die Scan-ID.

Mögliche Scanstatus:

| Status | Bedeutung |
|---|---|
| `pending` | Auftrag wartet auf Verarbeitung |
| `running` | Nmap-Ausführung läuft |
| `completed` | Nmap-Ausführung erfolgreich abgeschlossen |
| `failed` | Auftrag fehlgeschlagen oder abgebrochen |

## Tests

### Automatisierte Go-Tests

Im Projektordner ausführen:

```powershell
Push-Location backend
go test -count=1 ./...
Pop-Location

Push-Location scanner
go test -count=1 ./...
Pop-Location
```

Die Tests überprüfen unter anderem Eingabevalidierung, Controller-Verhalten, XML-Verarbeitung und Auftragsverwaltung. Sie bilden keine vollständige Testabdeckung sämtlicher Komponenten ab.

### Negative API-Tests

Bei laufender Anwendung:

```powershell
& ".\tests\Sicherheitsprüfung_REST-API.ps1"
```

Das Skript prüft sieben ungültige beziehungsweise unzulässige Anfragen.

### Container-Konfiguration prüfen

```powershell
& ".\tests\Container_Sicherheitstest.ps1"
```

Das Skript zeigt Benutzer, Berechtigungen und Sicherheitseinstellungen der Container an. Die Ausgabe dient der manuellen Prüfung und ist kein umfassender automatisierter Sicherheitsnachweis.

Der Praxistest eines realen lokalen Netzwerkbereichs ist in der Diplomarbeitsdokumentation als **PT04** beschrieben.

## Projektstruktur

| Pfad | Inhalt |
|---|---|
| `backend/` | Go-REST-API, Geschäftslogik, XML-Parser und Datenbankzugriff |
| `scanner/` | Go-Scanner-API, Auftragsverwaltung und Nmap-Ausführung |
| `frontend/` | HTML, CSS, JavaScript und Nginx-Konfiguration |
| `database/` | Datenbankschema, Initialisierung und Dockerfile |
| `tests/` | PowerShell-Skripte für API- und Containerprüfungen |
| `docker-compose.yml` | Lokaler Build und Betrieb aus dem Quellcode |
| `docker-compose.release.yml` | Separate Konfiguration mit veröffentlichten Images |
| `.env.example` | Vorlage für die lokale Konfiguration |
| `go.work` | Go-Workspace für Backend und Scanner |

Diese Anleitung beschreibt den Build aus dem mitgelieferten Quellcode über `docker-compose.yml`.

## Bekannte Einschränkungen

- Unterstützt werden private und Loopback-IPv4-Adressen.
- Netzwerkbereiche sind auf höchstens 256 Adressen begrenzt; zulässig sind Netzpräfixe `/24` oder länger.
- IPv6 und UDP-Scans werden nicht unterstützt.
- Untersucht werden die 100 häufigsten TCP-Ports, nicht sämtliche Ports.
- Firewalls, Segmentierung und die Host-Erkennung können dazu führen, dass Geräte oder Dienste nicht erfasst werden.
- Produkt- und Versionsinformationen sind nur verfügbar, wenn Nmap sie bestimmen kann.
- Broadcast-Adressen können in der Ergebnisliste erscheinen und sind nicht als eigenständige Geräte zu interpretieren.
- Es gibt keine Benutzeranmeldung oder Rechteverwaltung.
- Unterbrochene Scanner-Aufträge werden nicht automatisch erneut ausgeführt.

## Verantwortliche Nutzung

Die Anwendung darf ausschliesslich zur Untersuchung eigener oder ausdrücklich autorisierter Netzwerke verwendet werden.

Sie ist für den lokalen Betrieb in einer kontrollierten Umgebung konzipiert. Eine Bereitstellung im Internet ist nicht vorgesehen. Die Ergebnisse dienen als technische Bestandsaufnahme und ersetzen keine vollständige Sicherheitsprüfung.

## Autor und Kontext

**Autor:** Nevzat Celik  
**Ausbildung:** HF Informatik  
**Schule:** TEKO  
**Projekt:** Containerisierte Netzwerk-Scanning-Plattform