VERDICT: CHANGES_REQUESTED

## Sicherheitsreport

**Scanner-Abdeckung:** Für den Projekttyp `go-backend` wurde kein externer Security-Scanner ausgeführt. Eigene manuelle Codeanalyse durchgeführt.

### Positivbefund (keine kritischen Findings)
- Keine hartkodierten Secrets, Tokens oder Passwörter im Produktivcode. `auth_test.go` enthält ausschließlich Testwerte.
- Request-Bodies werden vor dem Einlesen auf 1 MiB begrenzt (`http.MaxBytesReader`).
- Flag-Keys werden bei POST/PUT mit `^[A-Za-z0-9_-]{1,64}$` validiert.
- Rollout-Prozent und Description-Längen werden validiert.
- Zugriffs-Logging nutzt `r.URL.Path` und protokolliert weder Query-String noch `user`-Parameter oder Body-Inhalte.
- Antworten mit Status >= 500 werden in `writeError` generisch auf `{"error":"internal server error"}` reduziert.
- Keine permissiven CORS-Header; Cache-Control/Pragma werden gesetzt.
- HTTP-Server besitzt `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` und `MaxHeaderBytes`.
- Der In-Memory-Store ist durch `sync.RWMutex` thread-sicher.

---

### Findings

#### 1. Fehlende Authentifizierung bei nicht gesetztem `FLAG_API_KEY`
**Schweregrad:** mittel  
**Betroffene Stelle:** `main.go` / `newMux` in `main.go`  
**Beschreibung:**  
Ist die Umgebungsvariable `FLAG_API_KEY` nicht gesetzt, werden alle `/flags`-Routen ohne Authentifizierung ausgeliefert (`withAuth` wird nur bei nicht-leerem API-Key aktiviert). Jeder Prozess/Benutzer mit Zugriff auf den Loopback-Port kann dann Flags anlegen, ändern und löschen. Das Risiko wird durch die feste Bindung an `127.0.0.1:8080` reduziert, ist aber für ein Produkt beim Kunden ein schwacher Standard.

**Konkreter Fix:**  
Den Start ohne API-Key nur mit explizitem Opt-in erlauben, z. B.:
```go
if apiKey == "" && os.Getenv("ALLOW_NO_AUTH") != "true" {
    log.Fatal("FLAG_API_KEY is required (or set ALLOW_NO_AUTH=true for local development)")
}
```
Alternativ in der Deployment-Dokumentation verbindlich festlegen, dass `FLAG_API_KEY` gesetzt sein muss, und den Dienst sonst nicht starten.

---

#### 2. Rate-Limit greift vor der Authentifizierung und ermöglicht lokalen DoS
**Schweregrad:** mittel  
**Betroffene Stelle:** `main.go` / `protect` in `newMux`  
**Beschreibung:**  
Der Token-Bucket wird **vor** der Auth-Prüfung angewendet:
```go
h = withTokenBucket(h, limiter)
if apiKey != "" {
    h = withAuth(h, apiKey)
}
```
Dadurch können unauthentifizierte Anfragen das globale Rate-Limit-Budget verbrauchen. Ein lokaler Angreifer ohne gültigen API-Key kann so legitime Anfragen dauerhaft mit `429 Too Many Requests` blockieren (Availability-Problem), obwohl er selbst keinen Zugriff auf die API erhält.

**Konkreter Fix:**  
Reihenfolge vertauschen, sodass die Authentifizierung vor dem Rate-Limit stattfindet:
```go
protect := func(h http.Handler) http.Handler {
    if apiKey != "" {
        h = withAuth(h, apiKey)
    }
    h = withTokenBucket(h, limiter)
    return withLogging(h)
}
```
`/healthz` bleibt hiervon unberührt, da er separat und ungeschützt registriert wird. Optional können separate Token-Buckets für anonyme und authentifizierte Anfragen eingeführt werden.

---

#### 3. JSON-Endpunkte erzwingen keinen `Content-Type: application/json`
**Schweregrad:** niedrig  
**Betroffene Stelle:** `handlers_flags.go` / `decodeBody`  
**Beschreibung:**  
Der JSON-Decoder akzeptiert beliebige oder fehlende Content-Type-Header. Das ist kein direkter Exploit, aber ein unnötig lockerer Umgang mit Eingabedaten und kann zu Fehlinterpretationen führen.

**Konkreter Fix:**  
Vor dem Dekodieren den Media-Type prüfen:
```go
ct := r.Header.Get("Content-Type")
if ct != "" {
    mt, _, err := mime.ParseMediaType(ct)
    if err != nil || mt != "application/json" {
        writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
        return false
    }
}
```
Dafür `mime` importieren. Die bestehenden internen Tests müssten bei Bedarf den Content-Type-Header setzen.

---

#### 4. Rate-Limit kann durch ungültige oder negative Konfiguration deaktiviert werden
**Schweregrad:** niedrig  
**Betroffene Stelle:** `main.go` / `newTokenBucket` und `RATE_LIMIT_PER_SECOND`-Parsing  
**Beschreibung:**  
`newTokenBucket(rps)` deaktiviert das Rate-Limit bei `rps <= 0`. Wenn ein Betreiber versehentlich `RATE_LIMIT_PER_SECOND=0` oder einen negativen Wert setzt, ist das Rate-Limit vollständig deaktiviert, ohne dass dies als Fehler sichtbar ist.

**Konkreter Fix:**  
Ungültige Werte als Konfigurationsfehler behandeln und auf den sicheren Standard zurückfallen:
```go
if v := os.Getenv("RATE_LIMIT_PER_SECOND"); v != "" {
    n, err := strconv.Atoi(v)
    if err != nil || n < 0 {
        log.Printf("invalid RATE_LIMIT_PER_SECOND %q; using default 100", v)
        rps = 100
    } else {
        rps = n
    }
}
```
Falls `0` bewusst zur Deaktivierung erlaubt bleiben soll, sollte dies als explizite Opt-in-Konfiguration dokumentiert oder separat abgesichert werden.

---

### Ergebnis
Es wurden keine ausnutzbaren Injection-, Path-Traversal-, Secret-Leak- oder Auth-Bypass-Schwachstellen mit hohem oder kritischem Risiko gefunden. Die Acceptance-Criteria-Sicherheitsanforderungen AC-11 bis AC-16 sind im Code erfüllt. Aufgrund der schwachen Standard-Authentifizierung und der Reihenfolge von Rate-Limit und Auth sind gezielte Härtungsmaßnahmen vor Auslieferung empfehlenswert.