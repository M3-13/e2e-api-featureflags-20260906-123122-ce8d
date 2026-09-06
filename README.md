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

## Datenschutz

Der Query-Parameter `user` des Evaluate-Endpunkts wird ausschließlich
**transient** verarbeitet, um die Feature-Rollout-Entscheidung zu berechnen.
Er wird weder gespeichert noch in Logs geschrieben und unmittelbar nach der
Antwort verworfen. Der Endpunkt verändert den Flag-Store nicht und hält keine
Nutzer-IDs oder Evaluationsergebnisse vor. Rechtsgrundlage für diese
Verarbeitung ist das berechtigte Interesse gemäß Art. 6 Abs. 1 lit. f DSGVO.

## Security & Compliance

- **Komponenten**: Es kommen ausschließlich Pakete der Go-Standardbibliothek
  zum Einsatz; es werden keine externen Module verwendet. Ein SBOM lässt sich
  direkt aus `go.mod` bzw. via `go list -m all` ableiten.
- **Betrieb & Updates**: Sicherheits- und Funktionsupdates werden über ein
  neues Deployment bereitgestellt, sobald Codeänderungen eingespielt werden.
- **Sicherheitskontakt**: Meldungen zu Sicherheitslücken bitte an
  security@example.com.
- **Sicherheitseigenschaften**:
  - 1-MiB-Body-Limit für POST- und PUT-Anfragen (413 bei Überschreitung)
  - Eingabevalidierung von Keys und Request-Bodies
  - Fehlerobjekte ohne interne Details, Stacktraces oder Implementierungsdetails
  - Logging ohne Query-Parameter (nur Methode, Pfad und Statuscode)
  - Thread-sicherer In-Memory-Store
  - API-Key-Schutz der schreibenden Endpunkte (`ADMIN_API_KEY`)
  - Bindung des Servers an `127.0.0.1`
