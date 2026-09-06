VERDICT: BLOCKED

## Security-Report

### 1) Secrets
Keine hartkodierten Schlüssel, Passwörter, Tokens oder URLs im sichtbaren Code. Zugriffs-Logs geben bewusst nur Methode, Pfad (ohne Query-String) und Statuscode aus – der `user`-Parameter wird nicht protokolliert.  
**Befund:** Keine.

### 2) Injection & Eingaben
- **JSON-Bodies:** POST/PUT verwenden `http.MaxBytesReader` (1 MiB) und `json.Decoder` auf strukturierte Go-Typen. Keine unsichere Deserialisierung.  
- **Flag-Keys:** POST, GET, PUT und DELETE validieren Keys mittels `validKey` (max. 128 Zeichen, `[A-Za-z0-9._-]`).  
- **SQL/Command/Path-Injection:** Nicht anwendbar – In-Memory-Store, keine Shell-/DB-Aufrufe, kein Dateisystemzugriff.  
- **SSRF/XSS:** Keine externen Aufrufe; reine JSON-API ohne HTML, `Content-Type: application/json; charset=utf-8` wird gesetzt.

**Befunde hierzu:** keine kritischen Lücken. Einzelne Härtungen siehe unten.

### 3) AuthN/AuthZ
**Kritischer Befund:** Der Dienst besitzt keinerlei Authentifizierung oder Autorisierung. `main.go` registriert sämtliche Routen (`POST /flags`, `PUT /flags/{key}`, `DELETE /flags/{key}` usw.) ungeschützt. Jeder, der den Port erreichen kann, kann Flags anlegen, ändern, löschen und auslesen.  
- **Betroffene Stelle:** `main.go` (Routenregistrierung und `http.ListenAndServe`).  
- **Konkrete Lösung:** Eine Authentifizierungs-/Autorisierungs-Middleware einführen (z. B. statischer API-Key über Umgebungsvariable, `Authorization`-Header prüfen) und alle mutierenden bzw. auch lesenden Endpunkte dahinter schalten. Alternativ/ergänzend den Dienst nur auf `127.0.0.1` binden (siehe 5), wenn er ausschließlich lokal genutzt wird – dies allein schützt jedoch nicht gegen lokale Angreifer. Der Service muss mit den vorhandenen Tests weiterhin funktionieren, daher die Auth auf der `ServeMux`-Ebene ergänzen; Handler-Tests bleiben unberührt.

### 4) Dependencies
Keine externen Abhängigkeiten sichtbar; `go.mod` ist klein (vermutlich nur Modulname und Go-Version). Scanner-Output war nicht vorhanden (`go-backend`, keine semgrep/bandit etc.).  
**Befund:** Keine, aber der Bereich konnte mangels Scanner nicht maschinell geprüft werden.

### 5) Konfiguration & Transport
**Hoher Befund:** `main.go` startet mit `http.ListenAndServe(":"+port, handler)` –  
- lauscht damit auf **allen** Interfaces (0.0.0.0), nicht nur localhost;  
- nutzt **kein TLS** – Kommunikation inkl. Flag-Daten und Authentifizierungsinformationen (sofern künftig vorhanden) ist unverschlüsselt.  
- **Betroffene Stelle:** `main.go`, Zeile `http.ListenAndServe`.  
- **Konkrete Lösung:** `ListenAndServeTLS` mit Zertifikaten verwenden oder einen TLS-terminierenden Reverse Proxy davorschalten. Falls der Dienst nur intern gebraucht wird, auf `127.0.0.1` binden: `http.ListenAndServe("127.0.0.1:"+port, handler)`. Bei Deployment hinter einem Reverse Proxy muss der Proxy die Netzwerksegmentierung/ACL übernehmen.

---

### Weitere Findings (niedriger Schweregrad / Härtung)

#### M1 – `EvaluateFlag` validiert den Flag-Key nicht
- **Schweregrad:** Medium  
- **Datei/Stelle:** `internal/handlers/evaluate.go`, erster Block vor `s.Get(key)`.  
- **Problem:** `GET /flags/{key}/evaluate` prüft `key` nicht mit `validKey`. Ungültige Keys führen derzeit zu `404` (da sie nicht im Store existieren), verletzen aber AC-17 und könnten bei späteren Erweiterungen (z. B. Persistenz) zu Problemen führen.  
- **Fix:** `validKey(key)` prüfen und bei ungültigem Key `WriteError(w, http.StatusBadRequest, "invalid key")` zurückgeben.

#### M2 – `UpdateFlag` ignoriert den `bool`-Rückgabewert von `Store.Update`
- **Schweregrad:** Low  
- **Datei/Stelle:** `internal/handlers/flags.go`, Zeile `updated, _ := s.Update(key, existing)`.  
- **Problem:** Zwischen `Get` und `Update` könnte ein konkurrierender `Delete` stattfinden. `Update` liefert dann `(Flag{}, false)`; der Handler antwortet trotzdem mit `200` und leerem Flag. Kein direkter Exploit, aber inkonsistentes Verhalten.  
- **Fix:** `updated, ok := s.Update(...)` prüfen; bei `!ok` mit `404`/`409` antworten.

#### M3 – Kein Panic-Recovery
- **Schweregrad:** Low (Härtung)  
- **Datei/Stelle:** `main.go`, Middleware-Kette.  
- **Problem:** Eine unerwartete Panic in einem Handler beendet den gesamten Prozess.  
- **Fix:** Eine `recover`-Middleware ergänzen, die einen `500`-Fehler loggt und zurückgibt, ohne den Prozess zu crashen.

#### M4 – `ListFlags` ohne Pagination/Limit
- **Schweregrad:** Low  
- **Datei/Stelle:** `internal/handlers/flags.go`, `ListFlags`.  
- **Problem:** Bei sehr vielen Flags kann die Antwort unbegrenzt wachsen und Speicher/CPU belasten.  
- **Fix:** Pagination oder ein konfigurierbares Limit einführen (unter Beibehaltung der aktuellen Funktionalität für kleine Datenmengen).

---

### Zusammenfassung
Die Implementierung ist sauber in Bezug auf Eingabevalidierung, JSON-Behandlung, Größenbegrenzung und Logging-Datenschutz. Es existieren jedoch **kritische Lücken bei Authentifizierung und Transportverschlüsselung**, die ausnutzbar sind, sobald der Dienst nicht ausschließlich in einer abgeschotteten Umgebung läuft. Daher wird das Produkt in dieser Form nicht freigegeben.

**Verdict:** BLOCKED