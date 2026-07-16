---
PLAN: "feat: sse.Publisher adapts SSEServer to events.Publisher"
TAG: v0.1.0
---

> Este plan se despacha vía el flujo CodeJob. Ver skill: **agents-workflow**.
> Orquestado por `tinywasm/app-releases/docs/REUSABLE_MODULES_MASTER_PLAN.md` — **Fase B5**.

# PLAN — `sse`: adaptar `SSEServer` a `events.Publisher`

Autocontenido, en español. Eres un agente **sin contexto previo** y **solo tienes este repo**
(`tinywasm/sse`). Todo el contrato y el código exacto van inline.

> `docs/ARCHITECTURE.md` de este repo está **desactualizado** (documenta un flujo de auth por token
> que ya no existe en el código — `client.go` dice explícitamente "No Authentication"). No lo uses
> como referencia; este plan se basa en el código real.

## 1. Qué cambia y por qué

`tinywasm/events` (nuevo, ya publicado `v0.0.2`) es el contrato tipado de pub/sub:

```go
// tinywasm/events (ya publicado)
type Event struct { Topic string; Payload model.Encodable }
type Publisher interface { Publish(e Event) }
type Subscriber interface { Subscribe(topic string, h Handler) }
```

Un módulo de dominio publica eventos (`item_catalog` publica `catalog.item.created`, etc.) contra
`events.Publisher`, inyectado — **sin conocer si el push llega a otro módulo en el mismo binario o al
navegador**. `sse` es la implementación que empuja al navegador. Este plan da a `sse` un adaptador
que satisface `events.Publisher` reusando lo que `SSEServer` ya hace.

**Solo `Publisher`, no `Subscriber`.** El lado "suscriptor" de una conexión SSE es el **navegador**,
vía la conexión HTTP persistente — no una función Go `func(events.Event)`. Ese "suscribirse" ya
existe y se llama distinto: `ChannelProvider.ResolveChannels(ctx router.Context) ([]string, error)`
decide a qué canales se suscribe una conexión ENTRANTE. Forzar `sse` a implementar
`events.Subscriber` sería una intersección falsa — no hay callback Go que registrar. El broker
**in-proc** (Fase C/D, otro repo) es quien implementa `events.Subscriber` para módulo↔módulo dentro
del mismo binario; `sse` se queda solo del lado `Publisher`.

## 2. Estado actual exacto (verificado, no supuesto)

- `*SSEServer` (`server.go:10-14`) ya expone:
  ```go
  func (s *SSEServer) Publish(data []byte, channel string)                          // server.go:86
  func (s *SSEServer) PublishEvent(event string, data []byte, channels ...string)    // server.go:97
  ```
  Ambos empujan al `hub` (`hub.go`, todo no exportado) vía un canal `broadcast`; el hub filtra por
  canal (`isSubscribed`) y formatea el frame SSE (`formatSSEMessage`).
- Ningún archivo de `sse` (no-test) importa un codec (`grep -rn "tinywasm/json" *.go` vacío,
  excluyendo tests) — el framing es manual, bytes ya listos entran, bytes salen.
- `go.mod` **ya** tiene `github.com/tinywasm/router v0.1.13` y `github.com/tinywasm/model v0.0.15`
  — **no hace falta ningún bump de versión** para este plan. `github.com/tinywasm/json v0.5.11`
  está como `// indirect`; pasa a directo al usarlo en el adaptador.
- `sse` **no implementa** `router.Context`/`router.Streamer` en ningún tipo propio — solo los
  *consume* (`StreamHandler() router.StreamFunc` recibe un `router.Streamer` ya construido por el
  llamante). El cambio de `Context.Decode`/`Encode` en `router@v0.1.13` **no requiere tocar nada
  aquí**: el único lugar donde aparece `Decode`/`Encode` en este repo es el test double
  `mockStreamer` (`server_test.go`), que embebe `router/mock.Context` y ya los hereda de ahí.
- `mcp.SSEPublisher` (`/home/cesar/Dev/Project/tinywasm/mcp/publish_sse.go`, otro repo, solo
  contexto): `type SSEPublisher interface { Publish(data []byte, channel string) }` —
  `*SSEServer.Publish` ya lo satisface estructuralmente (mismos 2 args). No relacionado con este
  plan directamente, pero confirma que `*SSEServer` ya juega bien como "el lado que envía bytes a un
  canal" — el adaptador de este plan hace exactamente ese mismo papel para `events.Event`.

## 3. El cambio exacto

Nuevo archivo `events_publisher.go` (package `sse`, sin build tag — es código de servidor, igual que
`server.go`):

```go
package sse

import (
	"github.com/tinywasm/events"
	"github.com/tinywasm/json"
)

// Publisher adapts an *SSEServer to events.Publisher. It is a separate wrapper type — NOT a
// method named Publish directly on *SSEServer — because SSEServer already has
// Publish(data []byte, channel string): a byte-level API existing callers use today. Colliding
// the name would either break them or force events.Publisher's single-arg shape onto that call
// site. Composition, not renaming, is the fix.
type Publisher struct {
	Server *SSEServer
}

// Publish encodes e.Payload with the ecosystem's typed codec (never `any`, never a hand-rolled
// format) and pushes it as an SSE message: Topic becomes BOTH the SSE "event" name (so a browser
// listener can dispatch by type) and the channel (so only connections ChannelProvider resolved
// into that channel receive it). A nil or IsNil Payload sends an empty body — Topic alone is
// still meaningful (e.g. a "heartbeat" event with no data).
func (p Publisher) Publish(e events.Event) {
	var data []byte
	if e.Payload != nil && !e.Payload.IsNil() {
		if err := json.Encode(e.Payload, &data); err != nil {
			return // fire-and-forget: events.Publisher makes no delivery-error promise (see its doc)
		}
	}
	p.Server.PublishEvent(e.Topic, data, e.Topic)
}

var _ events.Publisher = Publisher{}
```

`go.mod`: añade `github.com/tinywasm/events` (última versión publicada); `github.com/tinywasm/json`
pasa de `// indirect` a dependencia directa (ya resuelta, solo cambia de sección al usarla en código
no-test).

## 4. Test con forma de consumidor (obligatorio, arnés de construcción)

Sigue el patrón exacto de `TestStreamHandlerPublishEvent` (`server_test.go:100-170`), ya existente:
levanta un `SSEServer`, corre `StreamHandler()` contra un `mockStreamer` en una goroutine, y en vez
de llamar `server.PublishEvent(...)` a mano, pasa por el adaptador nuevo:

```go
// events_publisher_test.go
package sse

import (
	"testing"
	"time"

	"github.com/tinywasm/events"
	"github.com/tinywasm/model"
)

type fakePayload struct{ Value string }

func (p *fakePayload) IsNil() bool                      { return p == nil }
func (p *fakePayload) EncodeFields(w model.FieldWriter) { w.String("value", p.Value) }

func TestPublisher_PushesTypedEventOverSSE(t *testing.T) {
	tSSE := New(&Config{Log: testLog(t)})
	srv := tSSE.Server(&ServerConfig{ChannelProvider: &mockChannelProvider{channels: []string{"catalog"}}})
	pub := Publisher{Server: srv}

	st := newMockStreamer()
	go srv.StreamHandler()(st)
	time.Sleep(50 * time.Millisecond) // let the connection register — same wait TestStreamHandlerPublishEvent uses

	pub.Publish(events.Event{Topic: "catalog", Payload: &fakePayload{Value: "hello"}})
	time.Sleep(100 * time.Millisecond)

	out := st.Output()
	verifyMessage(t, /* ajusta a la firma real de verifyMessage en shared_test.go */ nil, "catalog", nil)
	if !contains(out, "event: catalog") {
		t.Errorf("expected SSE event name %q in output, got: %s", "catalog", out)
	}
	if !contains(out, `"value":"hello"`) {
		t.Errorf("expected encoded payload in output, got: %s", out)
	}
}
```

> `mockChannelProvider`/`newMockStreamer`/`testLog`/`verifyMessage`/`contains` ya existen en
> `server_test.go`/`shared_test.go` — **reutilízalos**, no los reescribas. Ajusta la llamada a
> `verifyMessage` a su firma real (leída en `shared_test.go`) en vez de la marcada `/* ajusta */`
> arriba — este plan no reproduce esa firma exacta porque no es necesaria para el diseño, solo para
> la mecánica del test.

## 5. Fuera de alcance

- **No** implementes `events.Subscriber` en `sse` (§1 justifica por qué no aplica).
- **No** renombres ni toques `SSEServer.Publish`/`PublishEvent` — el adaptador los envuelve, no los
  reemplaza.
- **No** arregles `docs/ARCHITECTURE.md` (desactualizado, pero fuera del alcance de este plan — es
  un documento aparte, no bloquea esta migración).
- **No** toques `client.go`/`SSEClient` (lado navegador) — este plan es servidor→navegador únicamente
  por el lado que ya existe; el navegador sigue recibiendo SSE crudo, sin cambios.

## 6. Criterios de aceptación

- `go build ./...` verde, **sin** bump de `router`/`model` (ya están al día).
- `sse.Publisher` satisface `events.Publisher` (`var _ events.Publisher = Publisher{}` compila).
- El test de §4 verde: el payload tipado llega al navegador simulado, codificado, con el nombre de
  evento y canal correctos.
- `grep -n "func (s \*SSEServer) Publish\b" server.go` sigue mostrando la firma original de 2
  argumentos, intacta.
- `gotest ./...` (o `go test ./...` + `GOOS=js GOARCH=wasm go test ./...`) verde en ambos targets.

## 7. Etapas

| # | Etapa | Archivo(s) | Criterio |
|---|---|---|---|
| 1 | Dependencia | `go.mod` | añade `tinywasm/events`; `tinywasm/json` pasa a directa |
| 2 | Adaptador | `events_publisher.go` (nuevo) | `Publisher{Server}` satisface `events.Publisher` |
| 3 | Test consumidor | `events_publisher_test.go` (nuevo) | §4, verde |
| 4 | Verificación | — | `gotest ./...` verde en ambos targets |
