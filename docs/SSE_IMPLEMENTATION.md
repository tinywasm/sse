# TinySSE Implementation Plan

> **Status:** Revised - Ready for Implementation  
> **Created:** December 2025  
> **Package:** `github.com/cdvelop/tinysse`

📊 **[Ver Diagramas de Arquitectura y Flujo](./ARCHITECTURE.md)**

## Overview

TinySSE es un paquete independiente que implementa SSE (Server-Sent Events) compatible con TinyGo/WASM. Sigue el principio de responsabilidad única, separado de `crudp`.

## Package Structure

```
tinysse/
├── tinysse.go       # Main struct, New(), Config
├── config.go        # Configuration struct
├── models.go        # Shared structs (SSEMessage, SSEError)
├── hub.go           # //go:build !wasm - SSE Hub & Client mgmt
├── server.go        # //go:build !wasm - HTTP handler
├── wasm.go          # //go:build wasm - EventSource wrapper
```

## Integration with CRUDP

```go
// In crudp/sse.go
import "github.com/cdvelop/tinysse"

func (cp *CrudP) initSSE() {
    cp.sse = tinysse.New(&tinysse.Config{
        BufferSize: 100,
        // ...
    })
}

// Optimized: Accepts already encoded []byte to avoid double encoding in CRUDP
func (cp *CrudP) routeToSSE(data []byte, broadcast []string, handlerID uint8) {
    cp.sse.Broadcast(data, broadcast, handlerID)
}
```

---

## Current State Analysis

### Existing Code in CRUDP

| File | Purpose | Status |
|------|---------|--------|
| `sse.go` | `routeToSSE()` stub | ⚠️ Only logs, will delegate to tinysse |
| `config.go` | `SSEEndpoint`, `UserProvider` | ✅ Configured |
| `interfaces.go` | `Response` interface with `broadcast []string` | ✅ Defined |

### Dependencies

- `tinytime` - Timer compatible with WASM ✅
- `tinystring` - Error handling ✅

---

## Decisions Taken

### Q1: Connection Management Strategy - Server-Only Hub

**Razón:** Optimización para TinyGo. El cliente WASM es una única conexión y no necesita lógica de gestión de múltiples clientes ni mutex.

```go
// hub.go (!wasm)
type SSEHub struct {
    mu      sync.RWMutex
    clients map[string]*SSEClient // Map for faster lookup
}

type SSEClient struct {
    ID        string   // Unique connection ID
    UserID    string   // From UserProvider
    Role      string   // For role-based broadcast
    Channels  []string // Subscribed channels
    Send      chan []byte
}
```

### Q2: Channel Subscription Model - Implicit via UserProvider

**Razón:** YAGNI - Empezar simple. Los canales se derivan automáticamente del contexto de usuario.

```go
// Auto-channels derivados de UserProvider
func (h *SSEHub) autoChannels(userID, role string) []string {
    return []string{
        "all",                    // Broadcast global
        "role:" + role,           // Por rol (admin, user, guest)
        "user:" + userID,         // Usuario específico
    }
}
```

### Q3: Reconnection Strategy - Hybrid (Native + Manual)

**Razón:** `EventSource` no permite cambiar el token en la URL automáticamente. Necesitamos reconexión manual para rotación de tokens y nativa para intermitencias de red simples.

**Solución WASM:**
1.  **Network Error:** Dejar que `EventSource` reintente (backoff nativo).
2.  **Auth Error / Token Expired:**
    *   Cerrar conexión actual.
    *   Pedir nuevo Token a `Config.TokenProvider`.
    *   Crear nueva conexión `EventSource` inyectando `Last-Event-ID` en URL query params.

**Servidor:**
*   Mantiene buffer de mensajes.
*   Al recibir conexión, chequea query param `lastEventId` (además del header estándar, por compatibilidad con reconexión manual).

### Q4: Message Format - SSE Standard wrapping JSON

**Razón:** SSE requiere formato `data: ...\n\n`. El payload será JSON.

```go
// Wire Format Example
// id: 12345
// data: {"json_payload":...}
// <empty line>
```

**Seguridad:**
La estructura interna `SSEMessage` contiene datos de enrutamiento (`Targets`) que **NO** deben enviarse al cliente. Se usará un DTO o serialización selectiva.

```go
// models.go (Shared)
type SSEMessage struct {
    ID        string   `json:"id"`
    HandlerID uint8    `json:"handler_id"`
    Data      []byte   `json:"data"`
    Targets   []string `json:"-"` // Ignored in JSON, internal use ONLY
}
```

### Q5: Authentication Flow - Query Token with Rotation

**Razón:** Compatible con SPA/PWA y TinyGo/WASM.

**Flujo corregido:**
```
1. Login → JWT principal
2. POST /auth/sse-token → Primer SSE token
3. new EventSource("/events?token=t1")
4. Token expira -> Server cierra o retorna 401
5. WASM detecta error -> Pide nuevo token -> new EventSource("/events?token=t2&lastEventId=xxx")
```

### Q6: Error Handling - tinystring.MessageType Extended

**Razón:** Reutilizar `MessageType` existente de `tinystring` para consistencia y zero allocations.

```go
// tinysse/error.go
import . "github.com/cdvelop/tinystring"

type SSEError struct {
    Type    MessageType
    Err     error
    Context any
}

// Nuevos tipos en tinystring.Msg:
// Msg.Connect, Msg.Auth, Msg.Parse, Msg.Timeout, Msg.Broadcast
```

---

## TinySSE Config Structure

```go
// config.go
type Config struct {
    BufferSize    int
    Endpoint      string
    RetryInterval int
    MaxRetryDelay int
    
    // Callbacks
    OnConnect    func(clientID string) // Server
    OnDisconnect func(clientID string) // Server
    OnMessage    func(msg *SSEMessage) // Client
    OnError      func(err error)
    
    // Auth
    TokenValidator func(token string) (userID, role string, err error) // Server
    TokenProvider  func() (token string, err error)                    // Client
}
```

---

## Implementation Steps

1. [Implement FEAT_MESSAGETYPE_SSE in tinystring](#step-1-implement-feat_messagetype_sse-in-tinystring)
2. [Create tinysse/config.go and models.go](#step-2-create-tinysseconfiggo-and-modelsgo)
3. [Create tinysse/hub.go (Server Only)](#step-3-create-tinyssehubgo-server-only)
4. [Create tinysse/server.go (HTTP Handler)](#step-4-create-tinysseservergo-http-handler)
5. [Create tinysse/wasm.go (EventSource Client)](#step-5-create-tinyssewasmgo-eventsource-client)
6. [Integration tests](#step-6-integration-tests)
7. [Update crudp to use tinysse](#step-7-update-crudp-to-use-tinysse)

---

## Step 1: Implement FEAT_MESSAGETYPE_SSE in tinystring
*Add constants to `tinystring`: `Connect`, `Auth`, `Parse`, `Timeout`, `Broadcast`.*

## Step 2: Create tinysse/config.go and models.go
*Define `Config`, `SSEMessage` (with `json:"-"` on targets), and `SSEError`.*

## Step 3: Create tinysse/hub.go (Server Only)
*Implement `SSEHub` with map of clients in `//go:build !wasm`. Logic for broadcasting filtered by role/user.*

## Step 4: Create tinysse/server.go (HTTP Handler)
*Implement `ServeHTTP`. Parse token, register client, listen for disconnect. Format output as `id:..\ndata:..\n\n`.*

## Step 5: Create tinysse/wasm.go (EventSource Client)
*Implement manual reconnection loop for token rotation. Handle `onmessage`, `onerror`. Parse JSON data payload.*

## Step 6: Integration tests
*Unit tests for Hub routing. End-to-end test with mock server.*

## Step 7: Update crudp to use tinysse
*Replace stub with real implementation.*

---

## Ready for Implementation?

Plan actualizado con correcciones de arquitectura y seguridad.
