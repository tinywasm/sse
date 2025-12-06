# Design Principles & Decisions

This document outlines the core architectural decisions behind `tinysse`.

## Design Principles

1.  **Strict Separation of Concerns**: `tinysse` handles SSE transport only. It does not handle authentication, user management, or business logic.
2.  **TinyGo/WASM Optimization**:
    *   **No JSON Dependency**: The library passes raw `[]byte` to the user. JSON parsing is left to the consumer to reduce binary size.
    *   **Environment Build Tags**: Logic is split using `//go:build wasm` and `//go:build !wasm`.
3.  **Standard SSE Format**: Complies with the Server-Sent Events specification (`id:`, `event:`, `data:`).
4.  **No User Management**: The library identifies connections, not users. Mapping users to connections is done via the `ChannelProvider`.

## Key Decisions

| Decision | Description | Reason |
| :--- | :--- | :--- |
| **Server-Only Hub** | The `Hub` logic resides only on the server (`!wasm`). | Reduces WASM binary size; the client only needs a single connection. |
| **ChannelProvider** | An interface (`ChannelProvider`) resolves channels from `http.Request`. | Decouples the library from any specific authentication or session system (like `crudp`). |
| **Raw Data Delivery** | `SSEMessage.Data` is `[]byte`. | Avoids forced `encoding/json` import in the library. |
| **Hybrid Reconnection** | Uses browser native reconnection for network drops, but supports manual configuration for retry strategies. | Balances reliability and control. |
| **Implicit Broadcasting** | Broadcasting is done to "channels" (strings). | Simple and flexible. A "user" is just a channel named `user:ID`. |
| **Error Handling** | Uses `tinystring` for error formatting. | Consistent with the ecosystem and lightweight. |

## Architecture Overview

See [ARCHITECTURE.md](./ARCHITECTURE.md) for diagrams and flow details.
