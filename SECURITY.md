VERDICT: CHANGES_REQUESTED

## Sicherheitsbewertung

**Scanner-Abdeckung:** Für diesen Projekttyp wurde laut Ausgabe kein Scanner ausgeführt (`no applicable security scanners for this project type`). Die Bewertung beruht daher auf manueller Code-Analyse. Es wurden keine bekannten CVEs oder Drittanbieter-Abhängigkeiten gefunden, da das Projekt ausschließlich die Go-Standardbibliothek nutzt.

**Positiv geprüft:**
- Keine hartkodierten Secrets; der Admin-API-Key kommt aus `ADMIN_API_KEY`.
- Passwort-/Token-Vergleich erfolgt mit `crypto/subtle.ConstantTimeCompare`.
- Flag-Keys werden streng validiert: `[A-Za-z0-9._-]`, max. 128 Zeichen.
- POST-/PUT-Bodies werden über `http.MaxBytesReader` auf 1 MiB begrenzt.
- Fehlerantworten sind generisch und enthalten keine internen Fehlermeldungen/Stacktraces.
- JSON-Antworten setzen `Content-Type: application/json; charset=utf-8`.
- Access-Logs enthalten nur Methode, escaped Path ohne Query-String und Statuscode; der `user`-Parameter wird nicht geloggt.
- Evaluate verändert den Store nicht und persistiert weder Nutzer-IDs noch Ergebnisse.

## Festgestellte Befunde

### 1. Medium — Administrative Lese-Endpunkte sind unauthentifiziert erreichbar

**Datei/Stelle:** `main.go`, Route-Registrierung:
```go
mux.HandleFunc("GET /flags", handlers.ListFlags(s))
mux.HandleFunc("GET /flags/{key}", handlers.GetFlag(s))
```

**Beschreibung:** `POST`, `PUT` und `DELETE` werden durch `middleware.Auth` geschützt, `GET /flags` und `GET /flags/{key}` jedoch nicht. Diese Endpunkte liefern sämtliche Flag-Definitionen inklusive `description` und `rollout_percent`. Das ist ein inkonsistenter Authentifizierungs-/Autorisierungszustand und kann zu Information Disclosure führen, wenn der Dienst hinter einem Reverse-Proxy entfernt erreichbar gemacht wird. Die Bind-Adresse `127.0.0.1` reduziert das Risiko bei direkter Ausführung, hebt die Schwäche aber nicht auf.

**Konkreter Fix:**
```go
mux.Handle("GET /flags", middleware.Auth(handlers.ListFlags(s)))
mux.Handle("GET /flags/{key}", middleware.Auth(handlers.GetFlag(s)))
```
`GET /healthz` und `GET /flags/{key}/evaluate` bleiben öffentlich, damit die produktiv genutzte Evaluierungsfunktion ohne Admin-Key funktioniert.

---

### 2. Low — JSON-Dekodierer akzeptiert zusätzliche Daten nach dem JSON-Dokument

**Datei/Stelle:** `internal/handlers/flags.go`, `decodeJSONBody`.

**Beschreibung:** `json.Decoder.Decode` liest genau ein JSON-Dokument. Überschüssige Daten am Ende des Bodys werden ignoriert. Dadurch könnte beispielsweise ein Body der Form `{"key":"k","enabled":true}garbage` als gültig akzeptiert werden. Strenges Parsen verhindert Überraschungen und mögliche Smuggling-/Validierungsumgehungen.

**Konkreter Fix:** Statt des Decoders den begrenzten Body vollständig lesen und mit `json.Unmarshal` parsen, da `json.Unmarshal` nachfolgende Nicht-Whitespace-Zeichen ablehnt. Alternativ nach `Decode` prüfen, ob noch ein weiterer JSON-Token vorhanden ist, und dann mit `400 invalid JSON body` antworten.

---

### 3. Low — `user`-Query-Parameter ohne Längenbegrenzung

**Datei/Stelle:** `internal/handlers/evaluate.go`, Zeile ca. 21:
```go
user := r.URL.Query().Get("user")
...
h.Write([]byte(key + ":" + user))
```

**Beschreibung:** Der `user`-Parameter wird nur auf `""` geprüft, aber nicht auf eine sinnvolle Maximallänge. Da `EvaluateFlag` öffentlich ist, kann ein entfernter Client einen sehr langen Query-String senden und so unnötige Allokationen sowie Hash-Last erzeugen. `net/http` begrenzt Header in der Standardeinstellung zwar auf ca. 1 MiB, eine applikationsseitige Begrenzung ist aber robustere Härtung.

**Konkreter Fix:**
```go
const maxUserLen = 256

user := r.URL.Query().Get("user")
if user == "" {
    WriteError(w, http.StatusBadRequest, "user is required")
    return
}
if len(user) > maxUserLen {
    WriteError(w, http.StatusBadRequest, "user is too long")
    return
}
```
Die Grenze mit der tatsächlich erwarteten Nutzer-ID-Form (E-Mail, UUID etc.) abstimmen.

---

### 4. Low — HTTP-Server ohne Timeouts

**Datei/Stelle:** `main.go`, letzter Abschnitt:
```go
if err := http.ListenAndServe("127.0.0.1:"+port, handler); err != nil {
```

**Beschreibung:** `http.ListenAndServe` setzt keine `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` oder `IdleTimeout`. Damit ist der Dienst anfällig für Slow-Client-/Slowloris-artige Verbindungserschöpfung, insbesondere auf den öffentlich erreichbaren Endpunkten.

**Konkreter Fix:**
```go
srv := &http.Server{
    Addr:              "127.0.0.1:" + port,
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
}
if err := srv.ListenAndServe(); err != nil {
    log.Fatalf("server error: %v", err)
}
```

---

## Zusammenfassung

Die Implementierung erfüllt die wesentlichen Sicherheitsanforderungen aus der Spezifikation: Eingabevalidierung, Body-Limit, eine generische Fehlerstruktur, Logging ohne PII und ein konstantzeitlicher API-Key-Vergleich sind vorhanden. Blockierende Schwachstellen wie Injection/RCE, hartkodierte Secrets oder ein Auth-Bypass wurden nicht gefunden.

Die ungeschützten administrativen Lese-Endpunkte sind jedoch ein mittleres Informationsoffenlegungsrisiko und sollten vor Auslieferung behoben werden.