VERDICT: CHANGES_REQUESTED

## Gesamturteil

Der geprüfte Stand ist ein reines Go-Backend ohne Endnutzer-UI. Damit entfallen Pflichttexte-, Cookie-/Consent- und Barrierefreiheitspflichten. Die Kernmechanik (In-Memory-Flag-Store, deterministische Evaluierung, Request-Body-Limit, PII-freies Access-Log, generische 500er-Fehler, sichere Key-Validierung, keine permissiven CORS-Header) ist sauber umgesetzt.

Offene Punkte bestehen vor allem bei Transportverschlüsselung, fehlender Absicherung der verwaltenden Endpunkte und unvollständiger CRA-Dokumentation. Es liegt kein fundamentaler, nicht behebbarer Verstoß vor, aber vor einer Marktfreigabe sind Änderungen erforderlich.

---

## 1. GDPR / Datenschutz

### G-01 — Unverschlüsselte Übertragung des Nutzerkennzeichens
- **Schweregrad:** hoch
- **Fundstelle:** `main.go`, Zeile `http.ListenAndServe(":8080", mux)`
- **Befund:** Der Endpunkt `GET /flags/{key}/evaluate?user=alice` verarbeitet einen Nutzerbezeichner. Der Server lauscht standardmäßig auf allen Interfaces ohne TLS. Der Query-Parameter `user` kann damit im Klartext über das Netz übertragen werden. Das betrifft Art. 5 Abs. 1 lit. f und Art. 32 DSGVO (Vertraulichkeit der Verarbeitung).
- **Konkrete Abhilfe:**
  - In `main.go` einen eigenen `http.Server` verwenden.
  - Standardbindung auf `127.0.0.1:8080` ändern, nicht `:8080`.
  - Optional TLS über Umgebungsvariablen aktivieren, z. B. `TLS_CERT_FILE` und `TLS_KEY_FILE`; wenn beide gesetzt sind, `srv.ListenAndServeTLS(...)` aufrufen.
  - Weiterhin `go run .` startfähig lassen, z. B. ohne TLS nur lokal auf Loopback binden.
  - In `README.md` dokumentieren: Produktivbetrieb ausschließlich hinter TLS oder TLS-terminierendem Proxy.

### G-02 — Fehlende Datenminimierung für `description` und unbegrenzter In-Memory-Store
- **Schweregrad:** mittel
- **Fundstelle:** `handlers_flags.go`, `validateFlagRequest`; `store.go`, `Create`
- **Befund:** Das Feld `description` wird ohne Längenbeschränkung akzeptiert (nur das globale 1-MiB-Body-Limit greift). Über die API können beliebig viele Flags und damit beliebig viele beschreibende Texte im Arbeitsspeicher gehalten werden, bis der Prozess beendet wird. Das widerspricht der Datenminimierung und Speicherbegrenzung nach Art. 5 Abs. 1 lit. c und Art. 25 DSGVO, sobald Beschreibungstexte personenbezogene Angaben enthalten.
- **Konkrete Abhilfe:**
  - In `validateFlagRequest` eine maximale UTF-8-Länge für `description` einführen, z. B. 256 Zeichen; bei Überschreitung `400` mit `{"error":"description too long"}`.
  - In `FlagStore` eine maximale Anzahl Flags einführen, z. B. `maxFlags = 10000`; bei Überschreitung in `Create` einen Fehler zurückgeben und im Handler als `507 Insufficient Storage` oder `400` beantworten.

### G-03 — Fehlende Cache-Unterdrückung für evaluierungsbezogene GET-Antworten
- **Schweregrad:** niedrig
- **Fundstelle:** `writeJSON` in `main.go`; betroffen: `handleEvaluate` in `evaluate.go`
- **Befund:** `user` steht als Query-Parameter in einer GET-URL. Der eigene Logger erfasst den Query-String nicht, aber zwischengeschaltete Caches, Reverse-Proxys oder Browser-Historien könnten die vollständige URL speichern. Die Antworten setzen keine `Cache-Control`-Header.
- **Konkrete Abhilfe:**
  - In `writeJSON` standardmäßig `Cache-Control: no-store` und `Pragma: no-cache` setzen.
  - Alternativ gezielt in `handleEvaluate` vor `writeJSON` setzen.

### G-04 — Rechtsgrundlage und Verarbeitungszweck nicht dokumentiert
- **Schweregrad:** niedrig
- **Fundstelle:** `README.md`, `evaluate.go`
- **Befund:** Der Code verarbeitet den `user`-Wert ausschließlich transient zur Berechnung eines FNV-32a-Hashs; er wird weder gespeichert noch geloggt. Die Rechtsgrundlage ist im sichtbaren Stand nicht benannt.
- **Konkrete Abhilfe:**
  - In `README.md` einen Abschnitt „Data processing“ ergänzen: Zweck ist deterministische Rollout-Berechnung; `user` wird nur als Hash-Eingabe verwendet, nicht persistiert, nicht geloggt; Rechtsgrundlage ist vertrags-/betriebsabhängig, typischerweise Art. 6 Abs. 1 lit. b oder lit. f DSGVO.

---

## 2. EU Cyber Resilience Act (CRA)

### C-01 — Verwaltungsendpunkte ohne Authentifizierung/Autorisierung
- **Schweregrad:** hoch
- **Fundstelle:** `main.go`, `newMux`; `handlers_flags.go`, `handleCreateFlag`, `handleUpdateFlag`, `handleDeleteFlag`
- **Befund:** `POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sind ohne Authentifizierung erreichbar. Jeder mit Netzzugriff kann Flags anlegen, ändern oder löschen und damit das Verhalten des Systems beeinflussen. Das verletzt den CRA-Grundsatz „security by design“ und den Schutz vor unbefugtem Zugriff.
- **Konkrete Abhilfe:**
  - Eine konfigurierbare Authentifizierungs-Middleware einführen, z. B. Bearer-Token oder API-Key aus `FLAG_API_KEY`.
  - Die Middleware vor `withLogging` für `POST/PUT/DELETE` setzen.
  - Standardmäßig nur an Loopback binden, wenn kein Token gesetzt ist, und eine Warnung loggen: `"FLAG_API_KEY not set; refusing non-loopback binds"`.
  - Tests entweder mit Test-Token ausführen oder die Middleware testbar konfigurierbar halten.

### C-02 — Nicht gehärteter HTTP-Server ohne Timeouts
- **Schweregrad:** mittel
- **Fundstelle:** `main.go`, `http.ListenAndServe(":8080", mux)`
- **Befund:** Es wird der Default-Handler ohne `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout` und ohne `MaxHeaderBytes` verwendet. Das begünstigt Ressourcenerschöpfung und Slowloris-ähnliche Angriffe.
- **Konkrete Abhilfe:**
  - In `main.go` einen `http.Server` mit mindestens folgenden Werten erzeugen:
    - `ReadHeaderTimeout: 5 * time.Second`
    - `ReadTimeout: 10 * time.Second`
    - `WriteTimeout: 10 * time.Second`
    - `IdleTimeout: 60 * time.Second`
    - `MaxHeaderBytes: 1 << 20`
  - Danach `srv.ListenAndServe()` bzw. `srv.ListenAndServeTLS()` verwenden.

### C-03 — Fehlende sichtbare Sicherheitsdokumentation, SBOM und Update-/Patch-Prozess
- **Schweregrad:** mittel
- **Fundstelle:** `README.md`, `go.mod`, Projektwurzel
- **Befund:** Im sichtbaren Stand sind keine SBOM, keine dokumentierten Sicherheitseigenschaften, kein Supportzeitraum und kein Patch-/Update-Prozess erkennbar. `go.mod` hat offenbar keine externen Dependencies, was das Supply-Chain-Risiko verringert; die CRA-Pflicht zur Bereitstellung von Sicherheitsaktualisierungen muss aber über den Betriebs-/Deploymentprozess abgebildet sein.
- **Konkrete Abhilfe:**
  - `README.md` um einen Abschnitt „Security & lifecycle“ ergänzen: Standardannahmen (In-Memory, keine Persistenz, TLS erforderlich), Update-/Patch-Weg, zuständige Stelle, Supportzeitraum.
  - Eine maschinenlesbare SBOM ergänzen, z. B. `sbom.json`/`sbom.spdx`, die `module`, `Go-Version` und Standardbibliothek ausweist.
  - Da der Product Type `go-backend` ist, den CI-Prozess so erweitern, dass bei jedem Build eine SBOM erzeugt wird.

### C-04 — Fehlendes Rate-Limiting
- **Schweregrad:** mittel
- **Fundstelle:** `middleware.go`, `handlers_flags.go`
- **Befund:** Neben dem 1-MiB-Body-Limit gibt es keine Begrenzung der Anfragefrequenz. Viele gültige Anfragen können den In-Memory-Store füllen oder CPU-Zeit der Hash-Berechnung beanspruchen.
- **Konkrete Abhilfe:**
  - Eine einfache Token-Bucket-Middleware für die API-Routen ergänzen, z. B. global oder pro Client-IP, ohne IP-Adressen zu loggen.
  - Konfigurierbar über `RATE_LIMIT_PER_SECOND`; Standardwert z. B. 100 Requests/s.
  - Bei Überschreitung `429` mit `{"error":"too many requests"}` antworten.

---

## 3. EU AI Act

- **Befund:** Keine KI-Funktion oder KI-Komponente im sichtbaren Code. Der Feature-Flag-Dienst verwendet einen deterministischen FNV-Hash; das ist keine KI im Sinne des AI Act.
- **Pflichten:** Keine AI-Act-Pflichten erkennbar.

---

## 4. Pflichttexte & UI

- **Befund:** Reines Backend ohne öffentliche Web-UI. Keine Impressums-, Datenschutzerklärungs-, Cookie- oder Widerrufsbelehrungspflichten auf Produktebene.
- **Hinweis:** Die datenschutzrechtliche Informationspflicht trifft den Betreiber, der das Backend in einen Dienst mit Endnutzern einbettet. In `README.md` sollte ein Hinweis aufgenommen werden, dass der Betreiber für die Einbindung in ein Gesamtsystem eigene Datenschutzhinweise benötigt.

---

## 5. Barrierefreiheit

- **Befund:** Keine öffentliche Web-UI vorhanden. WCAG/BITV/EAA sind für den sichtbaren Stand nicht anwendbar.
- **Pflichten:** Keine.

---

## Positiv bewertet

- Access-Logging protokolliert nur `Methode`, `Pfad`, `Statuscode`, `Dauer`; Query-Strings mit `user` erscheinen nicht (`middleware.go`, `middleware_test.go`).
- Request-Body-Limit von 1 MiB mit sauberer 413er-Fehlerantwort (`handlers_flags.go`, `decodeBody`).
- 500er-Antworten werden generisch ausgegeben; interne Fehlermeldungen, Pfade und Speicheradressen werden unterdrückt (`main.go`, `writeError`, `main_test.go`).
- Flag-Keys sind auf `[A-Za-z0-9_-]{1,64}` beschränkt, was Path-Traversal und unsichere Pfadsegmente verhindert.
- Keine permissiven CORS-Header; `Access-Control-Allow-Origin` wird nicht gesetzt.
- Thread-sicherer Store mit `sync.RWMutex`; keine persistenten PII-Datenquellen.

---

## Ergebnis

Es bestehen keine fundamentalen, nicht behebbaren Rechtsverstöße. Vor einer Marktfreigabe müssen jedoch insbesondere Transportverschlüsselung, Absicherung der Verwaltungsendpunkte und CRA-Dokumentations-/Härtungspflichten umgesetzt werden.