VERDICT: APPROVED

Das Produkt ist ein reines Go-Backend ohne Endnutzer-UI. Daher sind Anforderungen zu Legal Notice, Cookies, Barrierefreiheit und AI Act nicht einschlägig. Die sichtbaren datenschutz- und sicherheitsrelevanten Anforderungen sind überwiegend sauber umgesetzt.

---

## 1. DSGVO / Datenschutz

### Befund: Transiente Verarbeitung der Nutzerkennung ist datenschutzrechtlich vertretbar
Der Query-Parameter `user` wird ausschließlich in `evaluate.go` verarbeitet, dort mit `fnv.New32a()` verhasht und nicht gespeichert, nicht geloggt und nicht zurückgegeben. Es entsteht kein personenbezogenes Profil. Die Längenbegrenzung auf 256 Zeichen und das Fehlen einer Persistenz sind datenminimierend.

**Schweregrad:** niedrig (kein Blocker)  
**Empfehlung:** In `COMPLIANCE.md` oder `README.md` ausdrücklich dokumentieren, dass `user` nur transient für die Rollout-Entscheidung verarbeitet und nicht protokolliert wird. Das schafft Klarheit für Betreiber und Auditoren.

### Befund: Zugriffsprotokoll erfüllt die Datenschutzvorgaben
`middleware.go` protokolliert ausschließlich Methode, `r.URL.Path`, Statuscode und Dauer. Query-Parameter werden nicht geloggt. Das erfüllt AC-12 und AC-16. Die Tests bestätigen den Ausschluss von `user=alice` und `role=admin`.

**Schweregrad:** kein Befund

### Befund: Fehlerausgaben geben keine personenbezogenen Daten preis
Interne 5xx-Antworten werden durch `writeError` auf `{"error":"internal server error"}` vereinheitlicht. Clientfehler (4xx) enthalten kontrollierte, generische Texte. Es werden keine Stacktraces, Dateipfade oder Speicheradressen ausgegeben.

**Schweregrad:** kein Befund

### Befund: `writeJSON` protokolliert bei Encode-Fehlern potenziell interne Fehlerdetails
In `main.go`:
```go
log.Printf("writeJSON: failed to encode response: %v", err)
```
Diese Zeile kann bei einem unerwarteten Fehler Details der Go-Laufzeitumgebung in das Betriebslog schreiben. Das ist kein plausibles Leck personenbezogener Daten, aber es widerspricht dem Prinzip der sauberen Fehlerbehandlung.

**Schweregrad:** niedrig  
**Konkrete Abhilfe:** In `main.go` die Zeile ändern zu:
```go
log.Printf("writeJSON: response encoding failed")
```
oder nur den Fehlertyp ohne Details ausgeben.

---

## 2. EU Cyber Resilience Act (CRA)

### Befund: Security-by-Design ist im Wesentlichen umgesetzt
Sichtbar sind folgende Maßnahmen:
- Request-Body-Limit: `http.MaxBytesReader` mit 1 MiB (`handlers_flags.go`)
- Header-Limit: `MaxHeaderBytes: 1 << 20` (`main.go`)
- Zeitlimits: Read-, Write-, Idle- und ReadHeader-Timeout gesetzt (`main.go`)
- Kein CORS, insbesondere kein `Access-Control-Allow-Origin: *` (`main.go`, Tests in `main_test.go`)
- Optionale Bearer-Token-Authentifizierung mit `crypto/subtle` (`auth.go`)
- Globale Token-Bucket-Rate-Limitierung (`ratelimit.go`)
- Generische 500-Antworten, keine internen Details (`handlers_flags.go`, `main.go`)

**Schweregrad:** kein Befund

### Befund: Sichere Standardbindung und TLS-Option vorhanden
Der Dienst bindet standardmäßig an `127.0.0.1:8080`, also nur an die Loopback-Schnittstelle. TLS kann über `TLS_CERT_FILE` und `TLS_KEY_FILE` aktiviert werden. Das ist eine sichere Standardkonfiguration.

**Schweregrad:** kein Befund  
**Hinweis:** Für einen Produktionsbetrieb außerhalb von `localhost` muss zwingend TLS aktiviert und ein API-Key gesetzt werden. In `README.md` oder `SECURITY.md` sollte diese Betriebsvoraussetzung klar benannt sein.

### Befund: Abhängigkeiten und SBOM vorhanden
Das Projekt verwendet ausschließlich die Go-Standardbibliothek. `go.mod` und `sbom.json` sind vorhanden. Es gibt keine externen Frameworks oder Bibliotheken, daher besteht aktuell kein erkennbares Risiko durch ungepatchte Drittanbieter.

**Schweregrad:** kein Befund

---

## 3. EU AI Act

Nicht einschlägig. Das Produkt enthält keine KI-Funktion im Sinne des AI Act. Es werden lediglich deterministische Hash-Berechnungen für Feature-Flag-Rollouts ausgeführt.

---

## 4. Pflichttexte und UI

Nicht einschlägig. Das Produkt ist ein reines Backend ohne Endnutzer-UI, ohne Cookies und ohne Warenkorb/Kaufabwicklung. Es sind keine Legal Notice, Datenschutzerklärung im UI oder Cookie-Banner im Code erforderlich. Für den späteren Betrieb sollte der Betreiber dennoch eine externe Datenschutzerklärung über die Verarbeitung des `user`-Parameters bereitstellen.

---

## 5. Barrierefreiheit

Nicht einschlägig. Es gibt keine öffentliche Web-Oberfläche, daher bestehen keine WCAG/BITV/EAA-Pflichten für dieses Artefakt.

---

## Gesamtfazit

Keine kritischen, hohen oder mittleren Rechtsrisiken sichtbar. Datenschutz, Zugriffsprotokollierung, Eingabevalidierung, Fehlerbehandlung und sichere Standardkonfiguration sind sauber umgesetzt. Lediglich die kleine Protokollzeile in `main.go` sollte im Sinne der Datenminimierung und sauberen Fehlerbehandlung bereinigt werden. Das Produkt kann aus Compliance-Sicht für den vorgesehenen Verwendungszweck freigegeben werden.