# Feature-Flag-Service

Ein Feature-Flag-Service als REST-API in Go. Flags werden thread-sicher im
Arbeitsspeicher gehalten und über JSON-Endpunkte verwaltet; ein
Evaluate-Endpunkt liefert anhand eines stabilen Hashs deterministische
Ja/Nein-Entscheidungen pro Nutzer auf Basis von `rollout_percent`.

## Tech-Stack

- **Sprache**: Go (>= 1.22)
- **Framework**: `net/http` (Standardbibliothek, keine externen Web-Frameworks)
- **Tests**: Go-Tests mit `httptest`

## Installation

Voraussetzung: Go >= 1.22.

```bash
go build ./...
```

## Ausführen

```bash
go run .
```

Der Server lauscht standardmäßig auf Port `8080`. Über die Umgebungsvariable
`PORT` lässt sich ein anderer Port wählen:

```bash
# Windows (PowerShell)
$env:PORT="9090"; go run .
# Unix
PORT=9090 go run .
```

## Endpunkte

| Methode | Pfad | Beschreibung |
| --- | --- | --- |
| GET | `/healthz` | Health-Check, antwortet `200 {"status":"ok"}` |
| POST | `/flags` | Legt ein Flag an (`201`) |
| GET | `/flags` | Listet alle Flags (`200`) |
| GET | `/flags/{key}` | Liefert ein Flag (`200`/`404`) |
| PUT | `/flags/{key}` | Aktualisiert ein Flag (`200`/`404`) |
| DELETE | `/flags/{key}` | Entfernt ein Flag (`204`/`404`) |
| GET | `/flags/{key}/evaluate?user={id}` | Deterministische Auswertung (`200`) |

Fehlerantworten sind JSON mit der Form `{"error":"<kurzer Text>"}`. Alle
JSON-Antworten setzen `Content-Type: application/json; charset=utf-8`.

Flag-JSON:

```json
{"key":"string","enabled":true,"description":"string","rollout_percent":50}
```

## Features

- Thread-sichere In-Memory-Speicherung von Feature-Flags
- CRUD-Endpunkte für Flags
- Deterministischer Evaluate-Endpunkt pro Nutzer
- Zugriffs-Logging als Middleware (Methode, Pfad ohne Query-String, Statuscode)
- Saubere Statuscodes und JSON-Fehlerobjekte
