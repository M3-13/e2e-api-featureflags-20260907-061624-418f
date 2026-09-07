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
