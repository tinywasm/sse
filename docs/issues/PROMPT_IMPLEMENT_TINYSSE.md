# Prompt: Implement TinySSE

## Context
We are implementing a new library `tinysse` (Server-Sent Events) that must be compatible with **TinyGo** (WASM) and standard Go.
We have already defined the Architecture, Implementation Plan, and critical Optimizations.

---

## Directives

### 1. Source of Truth
Follow the documentation strictly in this order:
1.  **[Implementation Plan](../SSE_IMPLEMENTATION.md)**: Steps and structure.
2.  **[Architecture](../ARCHITECTURE.md)**: Diagrams for Client/Server separation.
3.  **[Optimization: Pre-encoded Bytes](../../../crudp/docs/issues/FEAT_OPTIMIZE_SSE_BROADCAST.md)**: Important! Logic for passing `[]byte` instead of `any`.
4.  **[User Management Analysis](../USER_MANAGEMENT_ANALYSIS.md)**: Decoupling strategy (do NOT depend on UserProvider).

### 2. Testing Strategy (Critical)
You MUST follow the **Shared Test Logic** pattern used in `crudp` to ensure consistency.

*   **Structure:**
    *   `*_shared_test.go`: Core logic tests (agnostic).
    *   `*_stlib_test.go`: Tests specific to standard library (server HTTP handler).
    *   `*_wasm_test.go`: Tests specific to WASM (Client EventSource).
*   **Execution:**
    *   ALWAYS use the provided `test.sh` script to run tests.
    *   `./test.sh` runs both Stdlib and WASM tests (using `wasmbrowsertest`).
    *   **WASM Caveat:** If WASM tests fail due to environment limitations (e.g., `wasmbrowsertest` networking restrictions), **it is acceptable** as long as the logic is correct and shared tests pass. Prioritize logic correctness.

### 3. Implementation Focus
*   **Strict Separation:** Ensure `hub.go` and `server.go` have `//go:build !wasm`.
*   **Zero-Copy Broadcast:** The `Broadcast` method MUST accept `[]byte`.
*   **Hybrid Reconnection:** Implement the token rotation strategy defined in the Architecture (Manual reconnection on Auth error vs Native retry on Network error).

---

## Action Plan
1.  Read the documents listed above.
2.  Create the `tinysse` package structure.
3.  Implement Shared Types (`models.go`).
4.  Implement Server-Side Logic (`hub.go`, `server.go`).
5.  Implement Client-Side Logic (`wasm.go`).
6.  Create Tests (`tinysse_shared_test.go`, etc.).
7.  Run `./test.sh` and verify results.
