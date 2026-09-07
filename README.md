# Feature-Flag-Service

Ein REST-API-Dienst in Go, der Feature-Flags in einem thread-sicheren
In-Memory-Store verwaltet und deterministische Rollout-Entscheidungen pro
Nutzer liefert. Er bietet ein JSON-Fehlerobjekt für alle Fehlerantworten,
einen Health-Check und vollständige Go-Tests mit `httptest`.

## Tech-Stack

- **Sprache:** Go (Standardbibliothek `net/http`, keine externen Frameworks)
- **Testing:** `testing` + `net/http/httptest`
- **Build:** `go build ./...` / `go test ./...`

## Installation

Voraussetzung ist eine Go-Toolchain (Go 1.22 oder neuer). Es gibt keine
externen Dependencies — ein `go mod tidy` ist daher optional.

```sh
go mod download
```

## Starten (Entwicklung)

```sh
go run .
```

Der Dienst lauscht auf Port `8080`.

## Konfiguration

Der Dienst wird über die folgenden Umgebungsvariablen konfiguriert:

| Variable | Bedeutung | Default |
| --- | --- | --- |
| `FLAG_API_KEY` | API-Key, mit dem alle `/flags`-Endpunkte abgesichert werden (Bearer-Token-Prüfung). | — (gesetzt erforderlich) |
| `RATE_LIMIT_PER_SECOND` | Maximale Anzahl erlaubter Requests pro Sekunde. | `100` |
| `TLS_CERT_FILE` | Pfad zur TLS-Zertifikatsdatei (PEM) für den Produktivbetrieb. | — (ohne Angabe kein TLS) |
| `TLS_KEY_FILE` | Pfad zur TLS-Schlüsseldatei (PEM) für den Produktivbetrieb. | — (ohne Angabe kein TLS) |

## Tests

```sh
go test ./...
```

Für den Race-Detector:

```sh
go test -race ./...
```

## API

Alle Fehlerantworten verwenden das JSON-Fehlerobjekt `{"error":"<meldung>"}`;
bei Status 500 wird stets nur die generische Meldung `"internal server error"`
zurückgegeben.

### `GET /healthz`

Antwortet mit `200` und `{"status":"ok"}`.

### `POST /flags`

Legt ein Flag an. Body:

```json
{ "key": "my-flag", "enabled": true, "description": "optional", "rollout_percent": 50 }
```

Antwort `201` mit dem angelegten Flag, `400` bei Validierungsfehler, `409` bei
doppeltem Key, `413` bei Body größer 1 MiB.

### `GET /flags`

Listet alle Flags, sortiert nach Key. Antwort `200` mit `[Flag, ...]`.

### `GET /flags/{key}`

Liefert ein einzelnes Flag (`200`) oder `404` bei unbekanntem Key.

### `PUT /flags/{key}`

Aktualisiert `enabled`, `description` und `rollout_percent` (`200`), `400` bei
Validierungsfehler, `404` bei unbekanntem Key.

### `DELETE /flags/{key}`

Entfernt ein Flag (`204`, ohne Body) oder `404` bei unbekanntem Key.

### `GET /flags/{key}/evaluate?user={id}`

Liefert die deterministische Rollout-Entscheidung `{"enabled": bool}` (`200`),
`400` bei fehlendem/leerem `user`, `404` bei unbekanntem Key.

## Flag-Schema

```json
{ "key": "string", "enabled": true, "description": "string", "rollout_percent": 50 }
```

- `key` muss `^[A-Za-z0-9_-]{1,64}$` entsprechen.
- `rollout_percent` liegt zwischen 0 und 100 (Default 0).
- `description` erscheint nur, wenn sie gesetzt ist.

## Data processing (Datenverarbeitung)

Zweck der Verarbeitung ist die deterministische Rollout-Berechnung für
Feature-Flags pro Nutzer. Der Parameter `user` aus
`GET /flags/{key}/evaluate` wird ausschließlich als Eingabe eines
Hash-Verfahrens zur Bestimmung des Rollout-Buckets verwendet. Er wird weder
persistiert noch in Logs geschrieben (siehe Logging-Middleware). Der Dienst
speichert alle Daten ausschließlich im Arbeitsspeicher (In-Memory) und legt
keine Datenbank oder Dateiablage an.

Die Rechtsgrundlage der Verarbeitung ist vertrags- bzw. betriebsabhängig und
liegt typischerweise in Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung) oder
Art. 6 Abs. 1 lit. f DSGVO (berechtigtes Interesse) — festzulegen durch den
jeweiligen Betreiber im konkreten Einsatzkontext.

## Security & lifecycle (Sicherheit & Lebenszyklus)

- **Speicherung:** Der Dienst hält alle Daten im Arbeitsspeicher; es findet
  keine Persistenz statt. Ein Neustart setzt den Zustand zurück.
- **Transport:** Der Produktivbetrieb muss ausschließlich hinter TLS erfolgen —
  entweder über `TLS_CERT_FILE`/`TLS_KEY_FILE` oder über einen
  TLS-terminierenden Proxy.
- **Absicherung:** Der Zugriff auf die API wird über `FLAG_API_KEY`
  abgesichert.
- **Bindung:** Standardmäßig bindet der Dienst an `127.0.0.1` (nur lokal
  erreichbar). Für einen netzwerkweiten Einsatz ist ein TLS-terminierender
  Proxy vorgeschaltet.
- **Updates & Support:** Der Update-/Patch-Weg sowie der Supportzeitraum
  werden durch den Betreiber festgelegt.
- **Einbettung:** Wird der Dienst in ein Gesamtsystem eingebettet, benötigt der
  Betreiber eigene, auf das Gesamtsystem bezogene Datenschutzhinweise.
