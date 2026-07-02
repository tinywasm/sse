# Plan — Refactor de `sse` al contrato de streaming de `tinywasm/router`

> `sse` sirve eventos server-sent: hoy implementa `http.Handler` y hace un
> *type-assert* a `http.Flusher` para empujar datos. El refactor lo pasa al contrato
> de **streaming** de `github.com/tinywasm/router`, para que el streaming sea
> isomórfico y no dependa de `net/http`. Autocontenido, en español.

---

## Reglas de Desarrollo

Las reglas del arnés viven en el **`AGENTS.md` de la raíz de esta librería** — léelo
antes de cualquier cambio. Este PLAN no las repite; describe solo el *cómo*.

Alcance (responsabilidad única): entregar un flujo de eventos incremental. El
mecanismo de *flush* concreto no debe filtrarse a su superficie.

---

## El contrato que consume (reexpresado para ser autocontenido)

El contrato `router` añade una capacidad de streaming: el handler **recibe** un
`Streamer` ya tipado (no se obtiene por aserción):

```go
// package router (github.com/tinywasm/router)
type Context interface { Method() string; Path() string; Body() []byte
	GetHeader(k string) string; SetHeader(k, v string); WriteStatus(code int); Write([]byte) (int, error) }

// Streamer es un Context que además empuja lo escrito de inmediato.
type Streamer interface {
	Context
	Flush() // envía al cliente lo escrito hasta ahora, sin cerrar la respuesta
}
type StreamFunc func(Streamer)
// Router: Stream(path string, h StreamFunc)  — registra una ruta de streaming
```

---

## Estado de partida

- `func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request)`.
- Dentro: `flusher, ok := w.(http.Flusher)`; si no, `http.Error(...)`; luego
  `w.WriteHeader(200)`, `flusher.Flush()`, y en bucle `w.Write(msg)`.
- `ResolveChannels(r *http.Request) ([]string, error)` — resuelve canales desde la
  petición.

O sea: `sse` es un **handler de streaming**, exactamente el caso que la capacidad
`Streamer` del contrato existe para cubrir.

---

## Cambios (antes → después)

| Antes (`net/http`) | Después (`router`) |
|---|---|
| `ServeHTTP(w http.ResponseWriter, r *http.Request)` | un `router.StreamFunc` — `func(s router.Streamer)` |
| `w.(http.Flusher)` + comprobación | innecesario: `Streamer` **ya** trae `Flush()` (la ruta se registró como streaming; el compilador garantiza la capacidad) |
| `w.WriteHeader(200)` / `w.Write(msg)` / `flusher.Flush()` | `s.WriteStatus(200)` / `s.Write(msg)` / `s.Flush()` |
| `ResolveChannels(r *http.Request)` | `ResolveChannels(ctx router.Context)` (usa `ctx.Path()`/`ctx.GetHeader`) |
| registro como `http.Handler` | `r.Stream(path, sseHandler)` |

Desaparece el `type-assert` a `http.Flusher`: el contrato hace imposible registrar un
handler de streaming sobre un transporte que no lo soporte (estado ilegal no
representable).

---

## Pasos de implementación

1. Añadir dependencia `github.com/tinywasm/router` en `go.mod`.
2. Convertir `ServeHTTP` en un `router.StreamFunc` que recibe `router.Streamer`.
3. Reemplazar el uso de `http.Flusher`/`ResponseWriter` por los métodos de
   `Streamer`.
4. Migrar `ResolveChannels` a `router.Context`.
5. Exponer el registro vía `Router.Stream(path, handler)`.

---

## Estrategia de pruebas y criterios de aceptación

- **Sin `net/http` en la superficie pública:** ninguna firma exportada nombra
  `http.ResponseWriter`/`http.Flusher`/`http.Request`.
- **Streaming real:** un `Streamer` de mentira que registre `Write` y cuente
  `Flush()` demuestra que los eventos se empujan uno a uno. `var _ router.StreamFunc
  = sseHandler` fija el contrato en compilación.
- **Canales:** `ResolveChannels` resuelve desde un `router.Context` de mentira sin
  tocar `net/http`.
