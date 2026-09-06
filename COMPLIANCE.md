VERDICT: CHANGES_REQUESTED

## 1. GDPR / Datenschutz

**Befund 1.1 — Log-Injection über nicht sanitisierte URL-Pfade**  
Schweregrad: hoch  

Die Middleware `internal/middleware/logging.go` protokolliert `r.URL.Path`. Dieses Feld enthält den bereits URL-dekodierten Pfad. Ein Angreifer kann über Pfadsegmente wie `%0A` Steuerzeichen (z. B. Zeilenumbrüche) einschleusen und so die Logdatei manipulieren oder weitere Logzeilen unterschieben. Das konterkariert die Anforderung „Logs enthalten ausschließlich Methode, Pfad und Statuscode“ (AC-18) und ist ein Sicherheitsrisiko für die Integrität der Protokolle.  
Maßnahme: In `internal/middleware/logging.go` statt `r.URL.Path` den codierten Pfad `r.URL.EscapedPath()` verwenden oder den Wert mit `%q` formatieren, z. B. `log.Printf("%s %q %d", r.Method, r.URL.Path, rw.status)`. Dadurch bleiben legitime Anfragen unverändert, Steuerzeichen werden aber neutralisiert.

**Befund 1.2 — Fehlende dokumentierte Rechtsgrundlage für die Verarbeitung des `user`-Parameters**  
Schweregrad: mittel  

Der Evaluate-Endpunkt verarbeitet die Nutzer-ID (`user`) transient in Form eines Hashs. Eine Speicherung erfolgt nicht, und die ID wird nicht geloggt. Das ist datenschutzfreundlich. Es fehlt jedoch eine dokumentierte Rechtsgrundlage für diese Verarbeitung, etwa Art. 6 Abs. 1 lit. f DSGVO (berechtigtes Interesse an der deterministischen Feature-Auslieferung) oder Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung), sofern der Dienst gegenüber dem Nutzer eine Leistung erbringt.  
Maßnahme: In `README.md` einen Datenschutz-Abschnitt ergänzen: „Der Query-Parameter `user` wird ausschließlich transient zur Berechnung des Feature-Rollouts verarbeitet, nicht gespeichert, nicht geloggt und nach der Antwort verworfen. Rechtsgrundlage: berechtigtes Interesse gemäß Art. 6 Abs. 1 lit. f DSGVO.“  

**Positiv**  
- `user` wird nicht in Logs geschrieben; Logs enthalten nur Methode, Pfad und Statuscode.  
- Der Evaluate-Endpunkt verändert den Store nicht und speichert weder Nutzer-IDs noch Evaluationsergebnisse.  
- Keine PII im Klartext in Logs oder Antworten.

---

## 2. EU Cyber Resilience Act (CRA)

**Befund 2.1 — Schreibende Endpunkte ohne Authentifizierung/Autorisierung**  
Schweregrad: hoch  

`POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sind ohne jeden Zugriffsschutz implementiert (`main.go`). Jeder mit Netzwerkzugriff kann Flags anlegen, verändern oder löschen. Das verletzt die CRA-Anforderung an Security by Design/Default, da das Produkt in einer unsicheren Standardkonfiguration ausgeliefert wird, und kann zu Integritäts- und Verfügbarkeitsverlusten führen.  
Maßnahme: Eine Authentifizierungs-/Autorisierungs-Middleware für die Verwaltungsrouten ergänzen, z. B. API-Key- oder Basic-Auth-Prüfung, oder den Server alternativ nur an ein lokales Interface binden (`http.ListenAndServe("127.0.0.1:"+port, handler)`), falls das Produkt rein intern betrieben wird. Die gewählte Variante ist in `README.md` zu dokumentieren.

**Befund 2.2 — Fehlende dokumentierte Sicherheitseigenschaften, SBOM und Update-/Patch-Prozess**  
Schweregrad: mittel  

Der Code selbst ist schlank und nutzt keine externen Abhängigkeiten. Es fehlen jedoch die unter der CRA geforderten dokumentierten Sicherheitseigenschaften, eine Software-Stückliste (SBOM) und Angaben zum Schwachstellenmanagement und zu Updates.  
Maßnahme: In `README.md` einen Abschnitt „Security & Compliance“ ergänzen:  
- Verwendete Komponenten: Go-Standardbibliothek, keine externen Module — SBOM via `go list -m all` oder manuell (`go.mod`).  
- Unterstützter Betriebszeitraum und Art der Update-Bereitstellung (z. B. neues Deployment bei Codeänderungen).  
- Kontakt/Prozess für Sicherheitsmeldungen (z. B. `security@example.com`).  
- Dokumentierte Sicherheitseigenschaften: Body-Limit (1 MiB), Input-Validierung, Fehlerobjekte ohne interne Details, Logging ohne Query-Parameter, thread-sicherer In-Memory-Store.

**Positiv**  
- Keine externen Abhängigkeiten; reduziertes Supply-Chain-Risiko.  
- Body-Limit und Eingabevalidierung vorhanden.  
- Fehlerantworten geben keine internen Implementierungsdetails preis.

---

## 3. EU AI Act

Nicht anwendbar. Der Dienst enthält kein KI-System im Sinne des AI Act. Keine Befunde.

---

## 4. Pflichttexte & UI (Impressum, Datenschutzerklärung, Cookie-Banner)

Nicht anwendbar. Es handelt sich um ein reines Backend ohne öffentliche Web-UI. Ein Impressum, eine Datenschutzerklärung für Webseitenbesucher oder ein Cookie-Banner sind für diesen Projekttyp nicht erforderlich. Betreiberpflichten für die Bereitstellung der API (z. B. Datenschutzhinweise gegenüber API-Kunden) liegen beim Betreiber, nicht im Code.

---

## 5. Barrierefreiheit (WCAG / BITV / EAA)

Nicht anwendbar. Keine öffentliche Web-UI vorhanden. Keine Befunde.

---

## 6. Sonstige Beobachtung

**Befund 6.1 — Inkonsistente Key-Validierung im Evaluate-Handler**  
Schweregrad: niedrig  

`internal/handlers/evaluate.go` prüft den Pfad-Key nicht mit `validKey`, anders als `GetFlag`, `UpdateFlag` und `DeleteFlag`. Da der Store nur über validierende Schreibpfade befüllt wird, entsteht derzeit keine direkte Gefahr. Für Konsistenz und zur Vermeidung von Sonderzeichen im Pfad sollte `validKey` auch hier angewendet werden.  
Maßnahme: Zu Beginn von `EvaluateFlag` nach dem Auslesen des Keys `if !validKey(key) { WriteError(w, http.StatusBadRequest, "invalid key"); return }` einfügen.