VERDICT: BLOCKED

Hinweis: Für den Projekttyp `go-backend` wurden keine automatisierten Security-Scanner ausgeführt bzw. geliefert. Die folgende Bewertung basiert ausschließlich auf der manuellen Analyse des sichtbaren Quellcodes.

## Befund 1 — Hoch: Vollständig fehlende Authentifizierung und Autorisierung

- **Betroffene Stelle:** `main.go` (`newMux` registriert alle `/flags`-Routen ohne Auth), `handlers_flags.go`, `evaluate.go`
- **Beschreibung:** Alle Verwaltungs-Endpunkte (`POST /flags`, `PUT /flags/{key}`, `DELETE /flags/{key}`) sowie lesende Endpunkte (`GET /flags`, `GET /flags/{key}`, `GET /flags/{key}/evaluate`) sind ungeschützt. Jeder, der den Port `:8080` erreicht, kann Feature-Flags anlegen, verändern, löschen und auslesen. Da Feature-Flags die Funktionsfreischaltung einer Anwendung steuern, führt eine unbefugte Änderung direkt zu einem Integritäts- und Verfügbarkeitsproblem. Es gibt weder API-Key noch Token noch mTLS noch eine Netzwerkbeschränkung.
- **Fix:** Vor alle `/flags`-Routen eine Authentifizierungs-Middleware schalten (z. B. Prüfung eines Bearer-Tokens oder statischen API-Keys mit konstantem Zeitvergleich). `/healthz` kann öffentlich bleiben. Beispiel:
  - `Authorization: Bearer <token>` prüfen; bei fehlendem oder ungültigem Token `401 {"error":"unauthorized"}`.
  - Alternativ den Dienst ausschließlich an `127.0.0.1:8080` binden, wenn er nur lokal verwendet wird. Für einen produktiven, netzwerkbasierten Dienst ist eine echte Authentifizierung erforderlich.

## Befund 2 — Mittel: Transport unverschlüsselt (HTTP ohne TLS)

- **Betroffene Stelle:** `main.go` — `http.ListenAndServe(":8080", mux)`
- **Beschreibung:** Die gesamte API-Kommunikation läuft im Klartext. Flag-Konfigurationen sowie der Query-Parameter `user` aus `/flags/{key}/evaluate` sind auf dem Transportweg abhörbar und manipulierbar.
- **Fix:** TLS verwenden, z. B. `http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", mux)`, oder den Dienst hinter einem TLS-terminierenden Reverse-Proxy betreiben. Falls der Dienst nur lokal erreichbar sein muss, Bindung auf `127.0.0.1:8080` einschränken.

## Befund 3 — Niedrig bis Mittel: Fehlende Server-Timeouts (Slowloris / Ressourcenerschöpfung)

- **Betroffene Stelle:** `main.go` — Verwendung von `http.ListenAndServe` mit dem Default-Server ohne Timeouts
- **Beschreibung:** Der Standard-Server setzt keine `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` oder `IdleTimeout`. Langsame Clients können Verbindungen und Header über lange Zeit offen halten und damit Ressourcen binden (Slowloris-Angriffe).
- **Fix:** Einen eigenen `http.Server` mit Timeouts konfigurieren:
  ```go
  srv := &http.Server{
      Addr:              ":8080",
      Handler:           mux,
      ReadHeaderTimeout: 5 * time.Second,
      ReadTimeout:       10 * time.Second,
      WriteTimeout:      10 * time.Second,
      IdleTimeout:       60 * time.Second,
      MaxHeaderBytes:    1 << 20,
  }
  log.Fatal(srv.ListenAndServe())
  ```

## Befund 4 — Niedrig: Unbegrenzte Länge des `user`-Query-Parameters

- **Betroffene Stelle:** `evaluate.go` — `user := r.URL.Query().Get("user")`
- **Beschreibung:** Der Parameter `user` wird lediglich auf Nicht-Leerheit geprüft. Ein sehr langer Wert (bis zur HTTP-Server-Header-/Request-Line-Grenze) wird geparst und gehasht; in Kombination mit fehlender Authentifizierung und fehlendem Rate-Limiting ergibt sich ein kleiner DoS-Faktor.
- **Fix:** Länge vor dem Hashing begrenzen:
  ```go
  if len(user) > 256 {
      writeError(w, http.StatusBadRequest, "user too long")
      return
  }
  ```

## Befund 5 — Niedrig: Speicherwachstum durch unbegrenzte Flag-Anzahl und Beschreibungslänge

- **Betroffene Stelle:** `store.go` (FlagStore-Map), `handlers_flags.go` (`validateFlagRequest` begrenzt `description` nicht)
- **Beschreibung:** Ein Angreifer kann beliebig viele Flags mit bis zu 1 MiB großen Beschreibungen anlegen. `GET /flags` liefert anschließend eine unbegrenzt große Liste; der In-Memory-Speicher wächst unkontrolliert. Die primäre Ursache ist die fehlende Authentifizierung, aber auch mit Auth sollte eine Obergrenze bestehen.
- **Fix:** Zusätzlich zu Befund 1 eine maximale Länge für `description` (z. B. 4 KiB) und/oder eine maximale Flag-Anzahl im Store einführen; außerdem Rate-Limiting für Schreib-Endpunkte erwägen.

## Positiv geprüfte Punkte

- **Request-Body-Limit:** `http.MaxBytesReader` begrenzt POST- und PUT-Bodies auf 1 MiB; die 413-Antwort erfolgt als JSON-Fehlerobjekt. (AC-11 erfüllt)
- **Logging-Datenschutz:** `middleware.go` protokolliert nur Methode, `r.URL.Path` (ohne Query-String), Statuscode und Dauer. Query-Parameter wie `user` erscheinen nicht. (AC-12/AC-16 erfüllt)
- **Generische 500-Antworten:** `writeError` ersetzt ab Status 500 jede Meldung durch `"internal server error"`. (AC-13 erfüllt)
- **Key-Validierung:** `validKey` erzwingt `[A-Za-z0-9_-]{1,64}` für POST und PUT. (AC-14 erfüllt)
- **Keine permissiven CORS-Header:** Es werden keine `Access-Control-Allow-Origin`-Header gesetzt. (AC-15 erfüllt)
- **Thread-Sicherheit:** `FlagStore` verwendet `sync.RWMutex`; Lese- und Schreiboperationen sind korrekt geschützt.