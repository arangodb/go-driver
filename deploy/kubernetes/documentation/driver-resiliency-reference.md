# **ArangoDB Driver Resiliency Reference**

> **Internal use only.** This document is for ArangoDB **driver maintainers** (Go, Java, JavaScript, Python, and other official drivers) who implement shared resiliency / Toxiproxy tests against the common Kubernetes test infrastructure.  
> It is **not** end-user or public API documentation, and it is **not** intended for external product docs, customer guides, or third-party consumers of the drivers.

**Authoritative multi-driver reference** for reproducing the same resiliency and network-fault tests in Java, JavaScript, Python, Go, and other ArangoDB drivers. Other driver teams can use this document as a **complete reference** to implement Part A (#0–#8) and Part B (#1–#14) scenario steps, error categories A–F, and pass/fail rules in their own test suites. Pair with [Harness env contract](#harness-env-contract-all-drivers) below (required) and optionally the overview slides in `deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html`.

**This document answers three questions for each scenario:**

1. **What fault is injected? (ingress restart, coordinator kill, network proxy, …)**
2. **What should the driver do? (fail cleanly, recover, no hang/panic)**
3. **What errors are acceptable? (with real example messages you can match in your language)**

> **Go driver v2** implements these scenarios under `v2/tests/` **(build tags** `resiliency` **and** `toxiproxy`**).**  
> **Shared infrastructure (all drivers):** `deploy/kubernetes/run-driver-tests.sh` (kind + kube-arangodb + Ingress) and `test/toxiproxy.sh` (start/stop Toxiproxy container + create the `arangodb` proxy).  
> **Per-language only:** test helpers that call the Toxiproxy **admin API** and assert driver errors (Go: `toxiproxy_helper_test.go`, `network_fault_error_util_test.go`).

---

## **How to use this document**


| **If you want to…**                | **Read…**                                                                                                                     |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **See all scenarios at a glance**  | **[Scenario catalog](#scenario-catalog)**                                                                                     |
| **Know which errors to accept**    | **[Error categories](#error-categories-language-agnostic)**                                                                   |
| **See which scenario gets A–E**    | **[Which scenarios produce which categories?](#which-scenarios-produce-which-categories)**                                    |
| **Implement one scenario**         | [Part A — Kubernetes scenarios](#part-a--kubernetes-scenarios) or [Part B — Toxiproxy network faults](#part-b--toxiproxy-network-faults) |
| **Wire harness env vars**          | **[Harness env contract (all drivers)](#harness-env-contract-all-drivers)** — endpoints, auth, Host header, Toxiproxy ports |
| **Reuse Toxiproxy across drivers** | **[Shared setup (Part B)](#shared-setup-part-b)** — `test/toxiproxy.sh`, listen vs admin ports, HTTP vs TLS CI                |
| **Run the Go reference suite**     | **[Running tests (Go reference)](#running-tests-go-reference)**                                                               |
| **Match errors in your language**  | **[Example error messages](#example-error-messages-by-category)**                                                             |
| **HTTP/1 vs HTTP/2 differences**   | **[Why HTTP/2 can show both JSON and HTML](#why-http2-can-show-both-json-410503-and-html-in-one-test)**                       |


**Important: Do not require one exact error string. Accept any error in the same category (connection loss, timeout, HTTP 503 JSON gateway error, HTTP 502/503/504 with non-JSON `Content-Type` or non-JSON body, etc.). HTTP/1 and HTTP/2 often report different text for the same fault — classify by category, not by one exact message.**

---

## **Scenario catalog**

**All Kubernetes resiliency scenarios run against 3 coordinators through ingress (**`arangodb.local`**). Most chaos scenarios run twice: HTTP/1 and HTTP/2. Load-balancer and failover scenarios use inline HTTP/1 and HTTP/2 subtests.**


| **#** | **Scenario**                               | **Fault**                                          | **Traffic during fault?**                      | **Error validation**                                                            |
| ----- | ------------------------------------------ | -------------------------------------------------- | ---------------------------------------------- | ------------------------------------------------------------------------------- |
| **0** | **LoadBalancerCoordinatorDistribution**    | **None (observational)**                           | **Yes —** `GET /_admin/status` **probes**      | **No — logs routing only**                                                      |
| **1** | **IngressCoordinatorFailover**             | **Delete 1 random coordinator pod**                | **Yes —** `GET /_admin/status` **probes**      | **No — only check recovery**                                                    |
| **2** | **IngressRestartWhileIdle**                | **Restart ingress-nginx**                          | **No**                                         | **No — only check recovery**                                                    |
| **3** | **IngressRestartDuringActiveWorkload**     | **Restart ingress-nginx**                          | **Yes —** `GET /_api/version` **every 100 ms** | **Yes — transient errors only**                                                 |
| **4** | **CoordinatorRestartWhileIdle**            | **Delete all 3 coordinator pods**                  | **No**                                         | **No — only check recovery**                                                    |
| **5** | **CoordinatorRestartDuringActiveWorkload** | **Delete all 3 coordinator pods**                  | **Yes —** `GET /_api/version` **every 100 ms** | **Recovery required;** `failuresDuring` **may be 0; if > 0, validate A–D only** |
| **6** | **CoordinatorKillDuringRead**              | **Delete all 3 coordinators during cursor**        | **Yes — streaming cursor**                     | **Yes — cursor interrupt + dead cursor**                                        |
| **7** | **CoordinatorKillDuringInsert**            | **Delete 1 coordinator during inserts**            | **Yes — insert loop**                          | **Yes — if any during-fault error, A–D only;** `failuresDuring` **may be 0**    |
| **8** | **CoordinatorKillDuringCursorIteration**   | **Delete all 3 coordinators after 30 cursor docs** | **Yes — streaming cursor**                     | **Yes — cursor interrupt + dead cursor**                                        |


**Toxiproxy tests (Part B) inject network faults through a local proxy — no Kubernetes chaos required.**

**Typical full suite duration: ~18 minutes on kind/WSL2 (HTTP/1 + HTTP/2 for all scenarios).**

---

## **Error categories (language-agnostic)**

**Classify every failure into one of these categories. Your test helpers should accept the category, not a single exact string.**


| **Category**                                | **Meaning**                                                                                                                  | **Accept in tests?**                        |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| **A — Connection lost**                     | **TCP connection broken before a valid HTTP response**                                                                       | **Yes (during faults)**                     |
| **B — Timeout**                             | **Client or transport gave up waiting**                                                                                      | **Yes (during faults)**                     |
| **C — Gateway HTTP error**                  | **HTTP 502/503/504 with JSON body** — proxy or coordinator unavailable                                                       | **Yes (during faults)**                     |
| **D — HTTP status / non-JSON Content-Type** | **HTTP 502/503/504 (typical) with non-JSON body and/or** `Content-Type` **other than** `application/json` (e.g. `text/html`) | **Yes (during faults)**                     |
| **E — Cursor gone (HTTP)**                  | **HTTP 404/409/410 with JSON body** — cursor no longer valid on cursor APIs                                                  | **Yes (cursor interrupt / resume / close)** |
| **F — Application error**                   | **Auth (401), validation, not-found on normal API**                                                                          | **No during fault window**                  |


### **Which scenarios produce which categories?**

**Use this table when your colleague asks “in which scenario do we see error type X?”**


| **Category** | **Kubernetes scenarios**                                                                                                                                                                                                      | **Phase / when**                                                         | **Covered by automated test?** |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------ |
| **A**        | **IngressCoordinatorFailover**, **IngressRestartDuringActiveWorkload**, **CoordinatorRestartDuringActiveWorkload**, **CoordinatorKillDuringRead**, **CoordinatorKillDuringInsert**, **CoordinatorKillDuringCursorIteration** | **During fault / probe retry / cursor interrupt**                        | **Yes**                        |
| **B**        | **IngressCoordinatorFailover**, **IngressRestartDuringActiveWorkload**, **CoordinatorRestartDuringActiveWorkload**, **CoordinatorKillDuringRead**, **CoordinatorKillDuringInsert**, **CoordinatorKillDuringCursorIteration** | **During fault / cursor interrupt** (`context deadline exceeded`, I/O timeout) | **Yes**                        |
| **C**        | **IngressCoordinatorFailover**, **IngressRestartDuringActiveWorkload**, **CoordinatorRestartDuringActiveWorkload**, **CoordinatorKillDuringRead**, **CoordinatorKillDuringInsert**, **CoordinatorKillDuringCursorIteration** | **During fault; cursor interrupt/resume when ArangoDB returns JSON**     | **Yes**                        |
| **D**        | **IngressCoordinatorFailover**, **IngressRestartDuringActiveWorkload**, **CoordinatorRestartDuringActiveWorkload**, **CoordinatorKillDuringRead**, **CoordinatorKillDuringInsert**, **CoordinatorKillDuringCursorIteration** | **During fault; dead-cursor resume when ingress has no healthy backend** | **Yes**                        |
| **E**        | **CoordinatorKillDuringRead**, **CoordinatorKillDuringCursorIteration**                                                                                                                                                       | **Cursor interrupted, dead cursor resume, close cursor (404)**           | **Yes**                        |
| **F**        | **None during fault windows**                                                                                                                                                                                                 | **Wrong auth, bad request, normal API not-found**                        | **N/A — must not appear**      |


**Toxiproxy (Part B):**


| **Category** | **Toxiproxy tests**                                                                   | **When**                                                               |
| ------------ | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| **A**        | **1–3, 9, 11–14** (connection cut, reset, packet loss, query/cursor/write disconnect) | **During proxy fault**                                                 |
| **B**        | **5, 7–8, 9, 10** (extreme latency, timeouts, full packet loss; partial loss may time out) | **Client or transport timeout** ( #9 accepts A **or** B on failures) |
| **C, D, E**  | **Not expected**                                                                      | **Proxy faults are transport-level, not ArangoDB cursor/ingress HTML** |


**Coverage summary:** All categories **A–E** used in Kubernetes resiliency scenarios are covered by the Go reference suite (scenarios **0–8** + Toxiproxy **1–14**). Category **F** is never accepted during a fault window. Idle scenarios (**IngressRestartWhileIdle**, **CoordinatorRestartWhileIdle**) send no traffic during the outage, so no fault-time errors are expected there. **CoordinatorRestartDuringActiveWorkload** may complete with `failuresDuring = 0`; categories A–D apply only when a version request fails during the coordinator outage ().

### **Quick decision guide**

```text
Did the request fail because the network/ingress/cluster was down?
  → Categories A, B, C, or D  → ACCEPT during fault

Did the cursor fail because the coordinator holding it was killed?
  → Category E (404, 409, 410 on cursor APIs), or C (503 JSON), or A/C/D  → ACCEPT

Did the client authenticate wrong or send a bad request?
  → Category F  → REJECT (test bug or wrong setup)
```

### **HTTP status quick map**


| **HTTP status**                                         | **Category** | **Layer**                                |
| ------------------------------------------------------- | ------------ | ---------------------------------------- |
| **502, 503, 504** + JSON                                | **C**        | Gateway / service unavailable            |
| **502, 503** + non-JSON body or non-JSON `Content-Type` | **D**        | Ingress / gateway (e.g. nginx HTML page) |
| **404, 409, 410** + JSON on cursor APIs                 | **E**        | Cursor gone / invalid                    |
| **(no HTTP response)** timeout                          | **B**        | Client or transport deadline             |
| **(no HTTP response)** connection broken                | **A**        | TCP / HTTP transport                     |


---

## **Example error messages by category**

**Use these as patterns to match in your driver. Wording varies by language and HTTP library.**

### **A — Connection lost**


| **Pattern**            | **Example messages (observed)**                                                         |
| ---------------------- | --------------------------------------------------------------------------------------- |
| **Connection refused** | `connect: connection refused`**,** `Connection refused`**,** `ECONNREFUSED`             |
| **Connection reset**   | `connection reset by peer`**,** `Connection reset`**,** `ECONNRESET`                    |
| **Unexpected EOF**     | `unexpected EOF`**,** `EOF`**,** `end of stream`**,** `Premature end of Content-Length` |
| **Broken pipe**        | `broken pipe`**,** `EPIPE`**,** `write: broken pipe`                                    |
| **Closed connection**  | `use of closed network connection`**,** `Socket closed`                                 |


**Full message example (HTTP/1):**

```text
Get "https://arangodb.local/_api/version": dial tcp 127.0.0.1:443: connect: connection refused
```

**Full message example (HTTP/2):**

```text
Get "https://arangodb.local/_api/version": unexpected EOF
```

---

### **B — Timeout**


| **Pattern**             | **Example messages**                                                                                  |
| ----------------------- | ----------------------------------------------------------------------------------------------------- |
| **Client deadline**     | `context deadline exceeded`**,** `TimeoutError`**,** `Request timed out`**,** `TaskCanceledException` |
| **Socket I/O timeout**  | `i/o timeout`**,** `read timed out`**,** `SocketTimeoutException`                                     |
| **HTTP header timeout** | `timeout awaiting response headers`**,** `Read timed out`                                             |


---

### **C — Gateway HTTP error (JSON response)**

**The server returns HTTP 502, 503, or 504 and the response body is valid JSON** (ArangoDB error object or proxy JSON error). These are **infrastructure / gateway** statuses — the request reached HTTP, but a proxy or coordinator could not complete it normally.


| **HTTP status** | **Meaning**             | **Example (Go driver)** | **What to check in any driver**       |
| --------------- | ----------------------- | ----------------------- | ------------------------------------- |
| **502**         | **Bad gateway**         | `ArangoError: Code 502` | `error.code == 502` **or status 502** |
| **503**         | **Service unavailable** | `ArangoError: Code 503` | `error.code == 503` **or status 503** |
| **504**         | **Gateway timeout**     | `ArangoError: Code 504` | `error.code == 504` **or status 504** |


**Also seen during coordinator shutdown (treat as category C / transient even if HTTP status is unavailable):**

```text
shutdown in progress
soft shutdown
Coordinator soft shutdown ongoing.
```

**How to classify as C:** HTTP status is **502, 503, or 504** (or mapped application error code) **and** the body parses as JSON — **or** the error message clearly indicates coordinator soft-shutdown / “shutdown in progress” (Go reference accepts those strings without requiring a parsed 503).

**Not category C:** HTTP **404**, **409**, and **410** are **not** gateway errors. They are **cursor API** responses (category **E**). See below.

**C vs B (gateway timeout):** HTTP **504** (category **C**) means the **proxy or upstream** timed out and returned a JSON error. **Category B** (`context deadline exceeded`, `i/o timeout`) means **your client or transport** gave up waiting — often with **no** JSON body. Both can happen during chaos; classify by what your driver observed.

If your driver exposes `Content-Type`, `**application/json`** supports **C** — but **JSON parse success is sufficient**; header assertion is optional.

---

### **D — HTTP status / non-JSON Content-Type**

**Category D** is defined at the **HTTP layer**: the client receives **HTTP 502, 503, or 504** (typical) with a **non-JSON** response — usually `Content-Type: text/html` and an nginx HTML error page when **no healthy coordinator backend** exists. The underlying transient fault is a gateway/ingress outage; classification should use **HTTP status and/or non-JSON `Content-Type`** when your driver exposes them.

**How to classify as D (preferred — when your HTTP layer exposes headers):**


| **Signal**                                                      | **Example**                                       |
| --------------------------------------------------------------- | ------------------------------------------------- |
| **HTTP status 502, 503, or 504** and body is **not valid JSON** | Status **503**, body starts with `<html`          |
| `**Content-Type` is not** `application/json`                    | `Content-Type: text/html` with status **502/503** |


**Go driver v2 (observable symptom — not the category definition):** go-driver v2 does **not** attach HTTP status or `Content-Type` to the error when JSON decode fails on the public API. It therefore often surfaces category **D** only as a **JSON parse failure**:


| **Go v2 observable (symptom)**    | **Example**                                            |
| --------------------------------- | ------------------------------------------------------ |
| JSON parse error on non-JSON body | `invalid character '<' looking for beginning of value` |


This is **driver misbehavior** (see team note): for `Content-Type: text/html`, the driver should not attempt JSON deserialization and should instead return an HTTP/status-aware error. A go-driver v2 fix is planned; until then, resiliency tests accept the parse-error text as the Go reference observable for **D**.

**Other drivers:** classify **D** directly from **HTTP status and/or non-JSON `Content-Type`** when available. JSON parse errors on HTML bodies are an optional fallback when headers are not exposed.


| **Pattern (other languages / fallback)** | **Example messages**                                                                                                       |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| JSON parse on non-JSON body              | `Unexpected token < in JSON`, `JSONDecodeError: Expecting value`, `SyntaxError: Unexpected token <`, `JsonReaderException` |


**Treat like a transient gateway outage — same acceptance rules as HTTP 503 from ArangoDB.**

**Seen in:** **IngressRestartDuringActiveWorkload**, **CoordinatorRestartDuringActiveWorkload**, **CoordinatorKillDuringRead**, **CoordinatorKillDuringInsert**, **CoordinatorKillDuringCursorIteration** during coordinator/ingress outage; especially **dead cursor resume** when ingress cannot reach any coordinator.

### **How to tell C vs D**


| **Observation**                                                                       | **Category**                |
| ------------------------------------------------------------------------------------- | --------------------------- |
| HTTP **502/503/504** + body parses as JSON (`ArangoError`, `shutdown in progress`, …) | **C**                       |
| HTTP **502/503/504** + non-JSON body or non-JSON `Content-Type` (e.g. `text/html`)    | **D**                       |
| Go v2 only: JSON parse fails on HTML body (`invalid character '<'…`)                  | **D** (observable symptom)  |
| No HTTP response (connection lost, timeout)                                           | **A** or **B** — not C or D |


**Do not require one exact signal.** Match the **category** using whatever your driver exposes: application error code, HTTP status, parse error, and optionally `Content-Type`.

---

### **E — Cursor gone (HTTP JSON response)**

**After a coordinator kill, cursor operations should fail (not hang, not return more documents).** These are **ArangoDB cursor API** errors — JSON responses with HTTP **404**, **409**, or **410**. They are **not** gateway errors (502/504) and **not** gateway timeout.


| **HTTP status**   | **Meaning**                           | **When in resiliency tests**              | **Example**             |
| ----------------- | ------------------------------------- | ----------------------------------------- | ----------------------- |
| **410 Gone**      | **Cursor not on this coordinator**    | Cursor interrupted after coordinator kill | `ArangoError: Code 410` |
| **404 Not Found** | **Cursor unknown or already removed** | Close or resume dead cursor               | `ArangoError: Code 404` |
| **409 Conflict**  | **Cursor state conflict**             | Dead cursor resume on wrong coordinator   | `ArangoError: Code 409` |


**503 on cursor APIs:** A cursor read may also return HTTP **503** with JSON (`shutdown in progress`) when the coordinator is shutting down. Classify that as **C** (gateway/service unavailable), not **E**. Cursor-specific codes for **E** are **404**, **409**, and **410** only.

**Go reference note:** `isArangoGatewayOrCursorError()` accepts **502/503/504** and **404/409/410** in one helper for cursor-kill tests — implementation convenience only. In documentation and other drivers, keep **C** (gateway) and **E** (cursor gone) separate as above.

---

## **Master table — scenario → expected errors**

**This is the main reference for other drivers. For each scenario phase, accept any error from the listed categories.**


| **Scenario**                                                            | **Phase**                                            | **Accept categories**          | **Common HTTP statuses** | **Example messages**                                                                                                                                                                        |
| ----------------------------------------------------------------------- | ---------------------------------------------------- | ------------------------------ | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **IngressCoordinatorFailover**                                          | **During kill window**                               | **A, B, C, D**                 | **502, 503**             | `connection refused`, `unexpected EOF`, `invalid character '<'…`                                                                                                                            |
| **IngressRestartDuringActiveWorkload**                                  | **During fault**                                     | **A, B, C, D**                 | **502, 503**             | `connection reset by peer`, `connection refused`, `EOF`, `unexpected EOF`, `context deadline exceeded`; **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`) |
| **CoordinatorRestartDuringActiveWorkload**                              | **During fault (only if a request overlaps outage)** | **A, B, C, D**                 | **502, 503**             | **Optional —** `failuresDuring` **often 0; see** [CoordinatorRestartDuringActiveWorkload](#coordinatorrestartduringactiveworkload)                                                          |
| **CoordinatorKillDuringInsert**                                         | **During fault**                                     | **A, B, C, D** *(or no error)* | **503** *(if any)*       | `shutdown in progress`, `invalid character '<'…`; **failuresDuring = 0 is OK**                                                                                                              |
| **CoordinatorKillDuringRead**, **CoordinatorKillDuringCursorIteration** | **Cursor interrupted**                               | **A, B, C, D, E**              | **410, 502, 503**        | `Code 410`, `Code 503`, timeout; **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`)                                                                       |
| **CoordinatorKillDuringRead**, **CoordinatorKillDuringCursorIteration** | **Resume dead cursor**                               | **A, B, C, D, E**              | **409, 502, 503**        | `Code 503`, `Code 409`, timeout; **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`)                                                                       |
| **CoordinatorKillDuringRead**, **CoordinatorKillDuringCursorIteration** | **Close dead cursor**                                | **E** primary; also **A, B, C, D** | **404** (typical)   | `Code 404` typical; also **410**/**503**/transport/HTML OK (Go: `isDeadCursorError`)                                                                                                       |
| **Toxiproxy — connection cut**                                          | **During fault**                                     | **A**                          | **—**                    | `connection reset by peer`, `connection refused`, `unexpected EOF`                                                                                                                          |
| **Toxiproxy — query startup disconnect**                                | **During db.Query()** (`POST /_api/cursor`)          | **A**                          | **—**                    | HTTP/1: `EOF`; HTTP/2: `unexpected EOF`                                                                                                                                                     |
| **Toxiproxy — streaming disconnect**                                    | **Next cursor read** (`POST /_api/cursor/…`)         | **A**                          | **—**                    | `connection refused`, `EOF`, `broken pipe`                                                                                                                                                  |
| **Toxiproxy — insert / commit disconnect**                              | **During CreateDocument / Commit**                   | **A**                          | **—**                    | `EOF`, `connection refused`, `unexpected EOF`; **write/commit outcome unknown**                                                                                                             |


### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**                       | **HTTP/1 (typical)**                                                                                                                        | **HTTP/2 (typical)**                                                                                                       |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| **TCP reset**                   | `connection reset by peer`                                                                                                                  | `unexpected EOF`                                                                                                           |
| **Coordinator kill (JSON)**     | `Code 410`                                                                                                                                  | `Code 503` **or** `Code 410`                                                                                               |
| **Gateway outage (category D)** | HTTP **502/503** + non-JSON `Content-Type` (e.g. `text/html`); Go v2 observable: `invalid character '<'…` — often during **recovery retry** | Same **D** definition; often on **dead-cursor resume** when no healthy backend; Go v2 observable: `invalid character '<'…` |
| **Ingress outage (TCP)**        | `connection refused`                                                                                                                        | `connection refused`                                                                                                       |


### **Why HTTP/2 can show both JSON (410/503) and HTML in one test**

**This is normal for** **CoordinatorKillDuringCursorIteration** **and** **CoordinatorKillDuringRead**. Each scenario has three phases; each phase can hit a different failure layer:**

```text
Phase 1 — cursor interrupted (next read after kill)
  → Request may still reach ArangoDB on a surviving/restarting coordinator
  → JSON error: Code 410 or 503 (category C or E); HTTP 503 + JSON body

Phase 2 — dead cursor resume (another read on same cursor)
  → All coordinators are gone; ingress has no healthy backend
  → HTTP 502/503 + `Content-Type: text/html` (category D)
  → Go v2 may surface only: invalid character '<'… (same category D)
  → OR JSON 409/503 if a coordinator answers but cursor is invalid (category E/C)

Phase 3 — close cursor
  → Code 404 (category E) — cursor already removed
```

**HTTP/1 vs HTTP/2 differences are about transport and timing, not different rules:**


| **Question**                                             | **Answer**                                                                                                                                                                                                                                                                                                                                                                                                |
| -------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Why does HTTP/1 often show HTML (D) first?**           | After killing all coordinators, the next cursor read often hits ingress with zero ready backends → nginx returns HTTP **502/503** with an HTML body.                                                                                                                                                                                                                                                      |
| **Why does HTTP/2 sometimes show 410 (E) on interrupt?** | HTTP/2 may still deliver the request to a coordinator process that is shutting down → ArangoDB returns JSON **410 Gone** (category **E**).                                                                                                                                                                                                                                                                |
| **Why can HTTP/2 show both 410 and HTML in one run?**    | **Different phases** (interrupt vs resume) and **race with pod recovery** — not contradictory. Accept any listed category per phase.                                                                                                                                                                                                                                                                      |
| **Must we assert `Content-Type`?**                       | **Preferred:** classify **D** from **HTTP status + non-JSON `Content-Type`** when your driver exposes them. **Go v2 reference:** accepts `invalid character '<'…` as the observable symptom because the driver does not surface status/`Content-Type` on JSON decode failure (known misbehavior). Optional: log `Content-Type: text/html` vs `application/json` to explain **D** vs **C** in test output. |


---

# **Part A — Kubernetes scenarios**

## **Shared setup**

```text
Driver  →  https://arangodb.local  →  ingress-nginx  →  coordinator (1 of 3)  →  ArangoDB
```


| **Requirement** | **Value**                                                   |
| --------------- | ----------------------------------------------------------- |
| **Cluster**     | **kube-arangodb enterprise, 3 coordinators**                |
| **Endpoint**    | `https://arangodb.local` **(or** `http://` **without TLS)** |
| **Auth**        | Harness string `basic:root:rootpw` (user `root` / password `rootpw`) |
| **Fault tool**  | `kubectl` **(restart ingress, delete coordinator pods)**    |
| **Host header** | Always send `Host: arangodb.local` (or `$TEST_INGRESS_HOST`) when the URL host is an IP |


**Why cursor tests kill all 3 coordinators: Ingress load-balances requests. You cannot reliably know which coordinator owns an open cursor. Kill all coordinator pods for cursor tests. Insert and failover tests kill only one coordinator.**

### **Harness env contract (all drivers)**

`deploy/kubernetes/run-driver-tests.sh` exports these names by default (override the *name* with `K8S_TEST_*_ENV` if your driver expects different variable names):


| **Env var (default name)** | **Part A (resiliency / ingress)** | **Part B (Toxiproxy)** | **Driver must…** |
| -------------------------- | --------------------------------- | ---------------------- | ---------------- |
| `TEST_ENDPOINTS_OVERRIDE`  | `http(s)://arangodb.local`        | `http(s)://127.0.0.1:17001` | Use as the client connection URL (map to your driver’s URL env if needed) |
| `TEST_AUTHENTICATION_OVERRIDE` (also `TEST_AUTHENTICATION`) | `basic:root:rootpw` | same | Parse `basic:<user>:<password>` (default password `rootpw`) |
| `TEST_INGRESS_HOST`        | usually unset when URL host is `arangodb.local` | `arangodb.local` | When set, send this value as the HTTP `Host` header on every request (required for Toxiproxy IP URLs) |
| `TEST_NET_OVERRIDE` / `K8S_TEST_DOCKER_EXTRA_ARGS` | Docker `--add-host` / mounts | same | Forward into your test container as the runner prints them |
| `TOXIPROXY_LISTEN_PORT` / `TOXIPROXY_ADMIN_PORT` | — | `17001` / `8474` | Pass through to `test/toxiproxy.sh`; driver traffic uses listen; tests use admin |
| `TOXIPROXY_UPSTREAM` / `TOXIPROXY_PROXY_NAME` | — | Ingress upstream / `arangodb` | Pass through to `test/toxiproxy.sh` |
| `TEST_TOXIPROXY_ADMIN` / `TEST_TOXIPROXY_PROXY` | — | `http://127.0.0.1:8474` / `arangodb` | Admin helpers in your language |

**Go remapping:** the Go Makefile copies `TEST_ENDPOINTS_OVERRIDE` → `TEST_ENDPOINTS` and `TEST_AUTHENTICATION_OVERRIDE` → `TEST_AUTHENTICATION`. Other drivers should either read the `*_OVERRIDE` names directly or perform the same remap in their wrapper.

---

## **LoadBalancerCoordinatorDistribution**

**Scenario #0 — Load balancer coordinator distribution**

**Goal: Observe how nginx ingress routes out-of-cluster driver traffic across coordinators (no fault injected).**


|                        |                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------- |
| **Fault**              | **None**                                                                                                |
| **API**                | `GET /_admin/status` **— read** `serverInfo.serverId` **from response**                                 |
| **Subtests**           | **Shared HTTP/2; shared HTTP/1; fresh HTTP/1 per request; fresh HTTP/2 per request**                    |
| **Pass**               | **All probes succeed; log which coordinator answered each request**                                     |
| **Assert stickiness?** | **No — ingress may spread requests across 1–3 coordinators**                                            |
| **Why it matters**     | **Baseline for understanding ingress LB before chaos tests; explains why cursor tests kill all 3 pods** |


**Request flow:**

```text
go-driver (Docker) → https://arangodb.local → nginx ingress → coordinator pod IP → GET /_admin/status
```

**ClientIP session affinity on the coordinator Service does not apply on this path.**

---

## **IngressCoordinatorFailover**

**Scenario #1 — Ingress coordinator failover**

**Goal: After killing one coordinator, fresh probes succeed again through ingress and kube-arangodb restores the deployment.**


| **Step** | **Action**                                                                    |
| -------- | ----------------------------------------------------------------------------- |
| **1**    | **Record baseline coordinator via** `GET /_admin/status` **(fresh client)**   |
| **2**    | **Delete 1 random coordinator pod**                                           |
| **3**    | **Retry** `GET /_admin/status` **with fresh clients until a response (90 s)** |
| **4**    | **Log whether coordinator ID changed (either outcome is valid)**              |
| **5**    | **Wait for cluster recovery (3 pods, ArangoDeployment ready, ingress OK)**    |
| **6**    | **Assert coordinator count restored to original**                             |
| **7**    | **Record final coordinator ID after recovery**                                |



|                           |                                                                                          |
| ------------------------- | ---------------------------------------------------------------------------------------- |
| **Fault**                 | **Delete 1 coordinator pod (**`kubectl delete pod … --grace-period=0 --force`**)**       |
| **API**                   | `GET /_admin/status` **with fresh HTTP client per probe (avoids stale TCP)**             |
| **Subtests**              | **Shared HTTP/1 client + fresh HTTP/1 probes; shared HTTP/2 + fresh HTTP/2 probes**      |
| **Pass**                  | **Probe succeeds after kill; coordinator count restored; probe succeeds after recovery** |
| **During kill — accept**  | **Categories A, B, C, D on probe retries (not asserted individually)**                   |
| **Coordinator ID change** | **Logged only — unchanged ID is valid if ingress keeps routing to a survivor**           |


---

## **IngressRestartWhileIdle**

**Scenario #2 — Ingress restart while idle**

**Goal:** A connected client with **no traffic during the outage** can call `GET /_api/version` again on the **same client** after ingress-nginx recovers.

### **How this differs from IngressRestartDuringActiveWorkload**


|                          | **IngressRestartWhileIdle**                  | **IngressRestartDuringActiveWorkload**      |
| ------------------------ | -------------------------------------------- | ------------------------------------------- |
| **Traffic during fault** | **None** — no requests while ingress is down | **Yes** — version loop every 100 ms         |
| **During-fault errors**  | **None expected**                            | **Required to tolerate** categories **A–D** |
| **Pass focus**           | Version OK before and after on same client   | ≥ 1 success before and after; no hang       |


### **Execution order**


| **Step** | **Action**                                                                                            |
| -------- | ----------------------------------------------------------------------------------------------------- |
| **1**    | Connect to cluster through ingress (`https://arangodb.local` or `http://…`)                           |
| **2**    | `GET /_api/version` — must succeed (baseline)                                                         |
| **3**    | **No requests** during the fault window                                                               |
| **4**    | Restart ingress-nginx: `kubectl rollout restart deployment/ingress-nginx-controller -n ingress-nginx` |
| **5**    | Wait until ingress deployment rollout is ready (budget ~6 min)                                        |
| **6**    | `GET /_api/version` on the **same client** — must succeed                                             |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**        | **What happens**                   | **Accept categories** | **Notes**              |
| ----- | --------------------- | ---------------------------------- | --------------------- | ---------------------- |
| **1** | Before restart        | `GET /_api/version`                | —                     | Must succeed           |
| **2** | During ingress outage | No API calls                       | —                     | **No errors expected** |
| **3** | After recovery        | `GET /_api/version` on same client | —                     | Must succeed           |


**Reject:** Failure before restart; failure after recovery; hang; category **F**.

### **CircleCI reference run (HTTP ingress)**

No fault-time errors on either protocol — only ingress restart and wait logs.

```text
Restarting ingress deployment ingress-nginx/ingress-nginx-controller
Waiting for ingress deployment ingress-nginx/ingress-nginx-controller to become ready
--- PASS: IngressRestartWhileIdle/HTTP/1 (22.12s)
--- PASS: IngressRestartWhileIdle/HTTP/2 (49.03s)
```


| **Subtest** | **Errors during fault** | **Result** |
| ----------- | ----------------------- | ---------- |
| **HTTP/1**  | *(none — idle)*         | Pass       |
| **HTTP/2**  | *(none — idle)*         | Pass       |


---

## **IngressRestartDuringActiveWorkload**

**Scenario #3 — Ingress restart during active workload**

**Goal:** One client survives a continuous `GET /_api/version` loop while ingress-nginx is restarted; transient failures during the outage are OK; the client must work again after recovery.

### **Execution order**


| **Step** | **Action**                                                                                        |
| -------- | ------------------------------------------------------------------------------------------------- |
| **1**    | Connect through ingress; `GET /_api/version` — must succeed                                       |
| **2**    | Start background version loop: `GET /_api/version` every **100 ms**, **10 s** per-request timeout |
| **3**    | Wait until ≥ **1** successful request **before** restart                                          |
| **4**    | Mark fault window open                                                                            |
| **5**    | Restart ingress-nginx (`kubectl rollout restart …`)                                               |
| **6**    | Wait until ingress deployment rollout is ready (~6 min max)                                       |
| **7**    | Mark recovery complete                                                                            |
| **8**    | Wait until ≥ **1** successful request **after** recovery                                          |
| **9**    | Stop version loop; assert no hang                                                                 |
| **10**   | Assert recovery — each during-fault error is category **A–D** only                                |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**              | **What happens**                             | **Accept categories** | **Common messages**                                                                                                                                                                                     |
| ----- | --------------------------- | -------------------------------------------- | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1** | Before restart              | Version loop running                         | —                     | `successesBefore ≥ 1` and `successesBefore ≥ failuresBefore` (typically `failuresBefore = 0`)                                                                                                           |
| **2** | During fault                | Version requests hit dead/restarting ingress | **A, B, C, D**        | `connection reset by peer`, `connection refused`, `unexpected EOF`, `EOF`, `context deadline exceeded`, `Code 503`; **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`) |
| **3** | After recovery              | Version loop resumes                         | —                     | `successesAfter ≥ 1` and `successesAfter ≥ failuresAfter` (typically `failuresAfter = 0`)                                                                                                               |
| **4** | During-fault classification | Each error in fault window                   | **A–D** only          | Reject category **F**; empty during-fault list is rare but OK if recovery checks pass                                                                                                                   |


**Do not require a specific failure count.** `failuresDuring` depends on restart duration and poll timing (~50–80 on slower runs is typical; CircleCI below shows ~56–59). Prefer seeing during-fault errors, but do **not** fail solely because `failuresDuring = 0` if before/after recovery assertions pass.

### **CircleCI reference run (`http://arangodb.local`)**


| **Subtest** | **successesBefore** | **successesAfter** | **failuresDuring** | **Primary during-fault error**   | **Category** |
| ----------- | ------------------- | ------------------ | ------------------ | -------------------------------- | ------------ |
| **HTTP/1**  | 3                   | 158                | 59                 | `read: connection reset by peer` | **A**        |
| **HTTP/2**  | 3                   | 160                | 56                 | `read: connection reset by peer` | **A**        |


```text
# HTTP/1
workload summary: successesBefore=3 successesAfter=158 failures=59 (before=0 during=59 after=0) totalAttempts=220
during-fault error: Get "http://arangodb.local/_api/version": read tcp …: read: connection reset by peer (transient=true)
during-fault error: Get "http://arangodb.local/_api/version": EOF (transient=true)

# HTTP/2
workload summary: successesBefore=3 successesAfter=160 failures=56 (before=0 during=56 after=0) totalAttempts=219
during-fault error: Get "http://arangodb.local/_api/version": read tcp …: read: connection reset by peer (transient=true)
```

### **Why `connection reset by peer` (not always `connection refused`)**

During ingress rollout, existing TCP connections to the old controller pods are **reset** when pods terminate. The version loop (every 100 ms) often hits those dying connections → category **A** (`connection reset by peer`, `EOF`, `unexpected EOF` on HTTP/2).

Other valid during-fault outcomes on different runs:


| **Category** | **Example**                                                                      | **When**                                                                        |
| ------------ | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **A**        | `connection refused`                                                             | New connection before new ingress pod listens                                   |
| **A**        | `connection reset by peer`, `EOF`, `http2: client conn could not be established` | Connection to pod being terminated / dial race while ingress is down            |
| **B**        | `context deadline exceeded`                                                      | Request waits through 10 s timeout                                              |
| **C**        | `Code 503`                                                                       | JSON gateway error from proxy/coordinator                                       |
| **D**        | HTTP **502/503/504** + non-JSON `Content-Type` (e.g. `text/html`)                | Ingress returns non-JSON error page; Go v2 observable: `invalid character '<'…` |


**Reject:** `successesAfter = 0`; hang after cancel; category **F** (401, etc.); requiring zero `failuresDuring` as a hard rule (failures during ingress restart are typical). Do **not** require `failuresBefore = 0` / `failuresAfter = 0` as hard asserts — require successes to dominate failures outside the fault window (Go: `successes* ≥ failures*`).

---

## **CoordinatorRestartWhileIdle**

**Scenario #4 — Coordinator restart while idle**

**Goal: Same client works after all 3 coordinator pods are recreated.**


|                  |                                                            |
| ---------------- | ---------------------------------------------------------- |
| **Fault**        | **Delete all 3 coordinator pods; operator recreates them** |
| **API**          | `GET /_api/version` **before and after**                   |
| **Pass**         | **Version succeeds after all 3 pods ready (~10 min max)**  |
| **During fault** | **No requests (idle)**                                     |


---

## **CoordinatorRestartDuringActiveWorkload**

**Scenario #5 — Coordinator restart during active workload**

**Goal: One client survives continuous version requests while all 3 coordinator pods are deleted and recreated. The driver must recover; it does not need to fail during the outage.**

### What to implement (same in every driver)


| Item             | Specification                                                                                  |
| ---------------- | ---------------------------------------------------------------------------------------------- |
| **Precondition** | 3 coordinator pods Ready; client reaches cluster through ingress                               |
| **Protocols**    | Run twice: HTTP/1 and HTTP/2 (skip HTTP/2 if server < 3.7.1)                                   |
| **Client**       | Single client instance for the whole subtest                                                   |
| **Workload**     | Background loop: `GET /_api/version` every **100 ms**, **10 s** per-request timeout            |
| **Fault**        | Force-delete **all 3** coordinator pods; wait until **3/3** Ready again                        |
| **Phases**       | Track `successesBefore`, `successesAfter`, `failuresBefore`, `failuresDuring`, `failuresAfter` |


### Procedure


| Step  | Action                                               |
| ----- | ---------------------------------------------------- |
| **1** | Start version loop in background                     |
| **2** | Wait until `successesBefore ≥ 1`                     |
| **3** | Mark fault window open (`restartStarted`)            |
| **4** | Delete all 3 coordinator pods; wait for recovery     |
| **5** | Mark recovery complete (`ingressReady` / equivalent) |
| **6** | Wait until `successesAfter ≥ 1`                      |
| **7** | Cancel workload; assert no hang                      |
| **8** | Assert recovery (table below)                        |


### Pass criteria (required)


| Check                  | Rule                                                                 |
| ---------------------- | -------------------------------------------------------------------- |
| **Baseline traffic**   | `successesBefore ≥ 1`                                                |
| **Recovery traffic**   | `successesAfter ≥ 1`                                                 |
| **Baseline quality**   | `successesBefore ≥ failuresBefore` (typically `failuresBefore = 0`)  |
| **Recovery quality**   | `successesAfter ≥ failuresAfter` (typically `failuresAfter = 0`)     |
| **No hang**            | Workload goroutine exits after cancel                                |
| **During-fault count** | `failuresDuring` **may be 0** — not a failure                        |


**Successes that occur while pods are still restarting** may be counted toward `successesAfter` once recovery is marked (Go reference: `successesPending` → `successesAfter` on `markIngressReady()`).

### During-fault errors (conditional — only when `failuresDuring > 0`)

**Do not require failures during the outage.** Unlike **IngressRestartDuringActiveWorkload**, coordinator pod recreation is often fast enough that no `Version()` call overlaps downtime — especially on CI.


| Environment             | Typical `failuresDuring` | Notes                                                          |
| ----------------------- | ------------------------ | -------------------------------------------------------------- |
| **Fast CI / kind**      | **0**                    | Common passing outcome (e.g. `totalAttempts=6`, all successes) |
| **Slower local / WSL2** | **1–5**                  | More overlap between 100 ms poll and pod downtime              |


**When `failuresDuring > 0`, each error must be category A, B, C, or D** (never F). Assert category, not one exact string.


| Category | Common HTTP status                     | Example messages (Go v2)                                                                                                  | Layer                               |
| -------- | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| **A**    | — (no HTTP body)                       | `connection refused`, `unexpected EOF`                                                                                    | TCP / connection                    |
| **B**    | — (timeout)                            | `context deadline exceeded`, `i/o timeout`                                                                                | Client or transport timeout         |
| **C**    | **503** (JSON)                         | `shutdown in progress`, `ArangoError: Code 503`                                                                           | ArangoDB coordinator shutting down  |
| **D**    | **502, 503** + non-JSON `Content-Type` | **Define:** HTTP **502/503** + non-JSON `Content-Type` (e.g. `text/html`). **Go v2 observable:** `invalid character '<'…` | Gateway / ingress non-JSON response |


**HTTP/1 vs HTTP/2:** Same pass rules. Message text differs (e.g. HTTP/2 may use `unexpected EOF` where HTTP/1 uses `connection reset by peer`). Category match is sufficient.

### Example passing workload summaries

```text
# Fast restart — zero during-fault failures (valid pass)
workload summary: successesBefore=3 successesAfter=3 failures=0 (before=0 during=0 after=0) totalAttempts=6

# Slower restart — one transient failure during fault (valid pass)
workload summary: successesBefore=3 successesAfter=14 failures=1 (before=0 during=1 after=0) totalAttempts=18
during-fault error: shutdown in progress (transient=true)
```

### Reject


| Outcome                        | Why                                        |
| ------------------------------ | ------------------------------------------ |
| `successesAfter = 0`           | Client did not recover                     |
| Hang after cancel              | Driver leak or blocked I/O                 |
| Category **F** during fault    | Wrong error class (e.g. 401)               |
| Requiring `failuresDuring > 0` | Not part of this test — optional by design |


---

## **CoordinatorKillDuringRead**

**Scenario #6 — Coordinator kill during cursor read**

**Goal:** Open cursor is interrupted cleanly when all coordinators die; dead cursor cannot resume; client recovers.

### **AQL query — no documents need to be inserted**

The query does **not** read from the collection:

```aql
FOR i IN 1..200 RETURN { i: i, burn: SLEEP(0.25) }
```

`FOR i IN 1..200` generates **200 synthetic rows in memory** (AQL range). `SLEEP(0.25)` keeps each batch slow so the cursor stays open. The collection is created for test setup only; the query does not read from it.

### **Execution order**


| **Step** | **Action**                                                                                                                                               |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1**    | Connect to cluster through ingress; create database and collection                                                                                       |
| **2**    | Start streaming AQL cursor: `FOR i IN 1..200 RETURN { i: i, burn: SLEEP(0.25) }` with `batchSize: 1`, `stream: true`                                     |
| **3**    | Read **1** document successfully                                                                                                                         |
| **4**    | Delete **all 3** coordinator pods                                                                                                                        |
| **5**    | Next cursor read on the in-flight iteration **must fail** (≤ 90 s) — **checkpoint: cursor interrupted**                                                  |
| **6**    | Another read on the **same** cursor **must fail** — **checkpoint: dead cursor resume** (run **before** cluster recovery; coordinators may still be down) |
| **7**    | Wait for cluster recovery (3 coordinators ready; `GET /_api/version` succeeds through ingress)                                                           |
| **8**    | Close cursor — **404 is OK** — **checkpoint: close dead cursor**                                                                                         |


### **Checkpoints — acceptable errors**

**Accept categories A, B, C, D, E** at checkpoints 2 and 3 (not one fixed HTTP code). Close primarily expects **E** (404); also accept A/B/C/D (Go: `isDeadCursorError`).


| **#** | **Checkpoint**           | **What happens**                                   | **Accept categories**   | **Common HTTP codes**                       |
| ----- | ------------------------ | -------------------------------------------------- | ----------------------- | ------------------------------------------- |
| —     | Cursor fails before kill | Query or reads complete all 200 rows before step 4 | —                       | **FAIL** test                               |
| **2** | Cursor interrupted       | Next read after coordinator kill                   | **A, B, C, D, E**       | **503**, **410**, **502**+non-JSON, timeout |
| **3** | Dead cursor resume       | Another read on same cursor before recovery        | **A, B, C, D, E**       | **503**, **409**, **410**, **502**+non-JSON |
| **4** | Cluster recovery         | Wait for pods and ingress                          | Retries may show **D**  | Benign during wait                          |
| **5** | Close dead cursor        | `close cursor` after recovery                      | **E** primary; also **A, B, C, D** | **404**, also **410**/**503**/transport OK |


### **CircleCI reference run (HTTP ingress)**


| **Checkpoint**     | **HTTP/1**              | **HTTP/2**              | **Category**  |
| ------------------ | ----------------------- | ----------------------- | ------------- |
| Cursor interrupted | `ArangoError: Code 503` | `ArangoError: Code 503` | **C**         |
| Dead cursor resume | `ArangoError: Code 409` | `ArangoError: Code 503` | **E** / **C** |
| Close dead cursor  | `ArangoError: Code 404` | `ArangoError: Code 404` | **E**         |


```text
# HTTP/1
cursor interrupted after coordinator kill: ArangoError: Code 503, ErrorNum 0
dead cursor resume error: ArangoError: Code 409, ErrorNum 0
cursor close on dead cursor: ArangoError: Code 404, ErrorNum 0

# HTTP/2
cursor interrupted after coordinator kill: ArangoError: Code 503, ErrorNum 0
dead cursor resume error: ArangoError: Code 503, ErrorNum 0
cursor close on dead cursor: ArangoError: Code 404, ErrorNum 0
```

### **Why HTTP/1 gets 409 and HTTP/2 gets 503 on dead cursor resume**

Both are **valid** for checkpoint **3**. The resume call is `POST /_api/cursor/{cursorId}` (fetch next batch). After killing all coordinators:


| **Code**                 | **Typical meaning**                 | **When**                                                                                                                                         |
| ------------------------ | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **503**                  | Service unavailable / shutting down | Request reaches a coordinator that is down or restarting                                                                                         |
| **409**                  | Conflict                            | Ingress routed the cursor request to a **different** coordinator than the one that owned the cursor — cursor state does not match on that server |
| **410**                  | Gone                                | Cursor no longer exists on the answering coordinator                                                                                             |
| **D** (non-JSON gateway) | Ingress has no healthy backend      | `HTTP 502/503` + `Content-Type: text/html` (or non-JSON body)                                                                                    |


**HTTP/1 vs HTTP/2 difference** is **routing and timing**, not different test rules:

- **HTTP/1** (CircleCI): next cursor request may land on a **surviving/restarting** peer → **409 Conflict** (wrong coordinator for that cursor ID).
- **HTTP/2** (CircleCI): same request may hit a coordinator still returning **503** (shutting down / not ready).

Connection reuse, ingress LB, and pod restart order vary per run — **accept any code in categories A, B, C, D, E** for checkpoints 2 and 3.

### **Other observed runs (kind/WSL2 — also valid)**


| **Checkpoint**     | **HTTP/1**                                                                          | **HTTP/2**     |
| ------------------ | ----------------------------------------------------------------------------------- | -------------- |
| Cursor interrupted | **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`) | `Code 410` (E) |
| Dead cursor resume | **D:** HTTP **502/503** + non-JSON `Content-Type` (Go v2: `invalid character '<'…`) | `Code 503` (C) |
| Close              | `Code 404`                                                                          | `Code 404`     |


**Reject:** Cursor reads all 200 documents after kill; hang past 90 s; panic; category **F** (401, etc.).

---

## **CoordinatorKillDuringInsert**

**Scenario #7 — Coordinator kill during insert**

**Goal:** Continuous document inserts survive **one** coordinator dying (the other two keep serving); inserts succeed again after recovery. Unlike cursor-kill scenarios, this does **not** delete all 3 coordinators.

### **How this differs from cursor-kill scenarios**


|                         | **CoordinatorKillDuringInsert**                                          | **CoordinatorKillDuringRead / CursorIteration**                         |
| ----------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| **Fault**               | Delete **1** coordinator pod                                             | Delete **all 3** coordinator pods                                       |
| **Workload**            | Insert loop (`CreateDocument` every 50 ms)                               | Streaming AQL cursor                                                    |
| **During-fault errors** | Optional — `failuresDuring` may be **0**                                 | Required — cursor must fail                                             |
| **Why 1 pod?**          | Inserts are not pinned to one cursor owner; survivors can absorb traffic | Cursor ownership is sticky; must kill all so the cursor cannot continue |


### **Execution order**


| **Step** | **Action**                                                                                               |
| -------- | -------------------------------------------------------------------------------------------------------- |
| **1**    | Connect to cluster through ingress; create database and collection                                       |
| **2**    | Start background insert loop: create document every **50 ms** (per-request timeout ~10 s)                |
| **3**    | Wait until ≥ **5** successful inserts (**before** fault)                                                 |
| **4**    | Delete **1** coordinator pod (the one currently answering this client, when known)                       |
| **5**    | Wait for cluster recovery (3 coordinators ready; ArangoDeployment ready; ingress `GET /_api/version` OK) |
| **6**    | Wait until ≥ **5** successful inserts (**after** recovery)                                               |
| **7**    | Stop insert loop; assert recovery (no hang; during-fault errors only category **A–D** if any)            |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**              | **What happens**                                       | **Accept categories**       | **Notes**                                                                      |
| ----- | --------------------------- | ------------------------------------------------------ | --------------------------- | ------------------------------------------------------------------------------ |
| **1** | Inserts before kill         | ≥ 5 successes                                          | —                           | Must succeed                                                                   |
| **2** | During fault                | Insert may fail while the killed pod decays            | **A, B, C, D** if any error | `**failuresDuring = 0` is OK** — remaining coordinators may absorb all traffic |
| **3** | After recovery              | ≥ 5 successes; errors after recovery must not dominate | Same client continues       | No hang after cancel                                                           |
| **4** | During-fault classification | If `failuresDuring > 0`, each error must be transient  | **A, B, C, D**              | Reject category **F**                                                          |


**Common during-fault messages when errors do occur:** `shutdown in progress` (**C**), `Code 503` (**C**), `invalid character '<'…` (**D**), `connection refused` / `unexpected EOF` (**A**), `context deadline exceeded` (**B**).

### **CircleCI reference run (HTTP ingress)**

Both HTTP/1 and HTTP/2 completed with **zero failures during the fault window** — expected when only one coordinator is killed and ingress routes inserts to survivors.


| **Subtest** | **successesBefore** | **successesAfter** | **failuresDuring** | **during-fault errors** |
| ----------- | ------------------- | ------------------ | ------------------ | ----------------------- |
| **HTTP/1**  | 5                   | 24                 | **0**              | *(none)*                |
| **HTTP/2**  | 5                   | 25                 | **0**              | *(none)*                |


```text
# HTTP/1
Deleting coordinator pod default/arangodb-driver-tests-crdn-…
… ingress recovered after 1 Version() attempt(s)
insert workload summary: successesBefore=5 successesAfter=24 failures=0 (before=0 during=0 after=0) totalAttempts=29

# HTTP/2
Deleting coordinator pod default/arangodb-driver-tests-crdn-…
… ingress recovered after 1 Version() attempt(s)
insert workload summary: successesBefore=5 successesAfter=25 failures=0 (before=0 during=0 after=0) totalAttempts=30
```

### **Why `failuresDuring = 0` is valid**

Only **one** of three coordinators is deleted. Ingress can keep routing successful `CreateDocument` calls to the remaining pods, so the insert loop may never observe an error. The test still passes if:

1. There were successes **before** the kill
2. There are successes **after** recovery
3. Any errors that *did* occur during the fault are category **A–D**
4. The workload stops cleanly (no hang)

Do **not** require `failuresDuring > 0` for this scenario.

### **Other observed runs (kind/WSL2 — also valid when during-fault errors appear)**


| **Subtest** | **failuresDuring** | **Example errors**                               | **Category** |
| ----------- | ------------------ | ------------------------------------------------ | ------------ |
| HTTP/1      | 2                  | `shutdown in progress`, `invalid character '<'…` | **C**, **D** |
| HTTP/2      | 1                  | `invalid character '<'…`                         | **D**        |


**Reject:** `successesAfter = 0`; hang after stop; category **F** during fault; requiring forced failures during the kill window.

---

## **CoordinatorKillDuringCursorIteration**

**Scenario #8 — Coordinator kill during cursor iteration**

**Goal:** Same as **CoordinatorKillDuringRead**, but the kill happens **after 30 documents** (mid-iteration), not after the first read. Same AQL query, options (`batchSize: 1`, `stream: true`), checkpoints, and accepted error categories.

**Why keep this scenario:** After ~30 successful batch fetches, the cursor is deeper into the stream and more connection/LB state may exist. Killing then still must interrupt cleanly (no hang, no finishing the remaining ~170 rows). That is a different timing window from “kill after 1 doc,” even though both share the same test helper pattern.

### **Execution order**


| **Step** | **Action**                                                                                                           |
| -------- | -------------------------------------------------------------------------------------------------------------------- |
| **1**    | Connect to cluster through ingress; create database and collection                                                   |
| **2**    | Start streaming AQL cursor: `FOR i IN 1..200 RETURN { i: i, burn: SLEEP(0.25) }` with `batchSize: 1`, `stream: true` |
| **3**    | Read **30** documents successfully                                                                                   |
| **4**    | Delete **all 3** coordinator pods                                                                                    |
| **5**    | Next cursor read **must fail** (≤ 90 s) — **checkpoint: cursor interrupted**                                         |
| **6**    | Another read on the **same** cursor **must fail** — **checkpoint: dead cursor resume** (**before** cluster recovery) |
| **7**    | Wait for cluster recovery (3 coordinators ready; `GET /_api/version` through ingress)                                |
| **8**    | Close cursor — **404 is OK** — **checkpoint: close dead cursor**                                                     |


### **Checkpoints — acceptable errors**

Same acceptance rules as **CoordinatorKillDuringRead**: **A, B, C, D, E** for interrupt/resume; **E** (**404** primary) for close (also A/B/C/D OK).


| **#** | **Checkpoint**              | **What happens**                     | **Accept categories**   | **Common HTTP codes**                       |
| ----- | --------------------------- | ------------------------------------ | ----------------------- | ------------------------------------------- |
| —     | Cursor finishes before kill | All 200 rows read before step 4      | —                       | **FAIL** test                               |
| **2** | Cursor interrupted          | Next read after kill (after 30 docs) | **A, B, C, D, E**       | **410**, **503**, **502**+non-JSON, timeout |
| **3** | Dead cursor resume          | Another read before recovery         | **A, B, C, D, E**       | **503**, **409**, **410**, **502**+non-JSON |
| **4** | Cluster recovery            | Wait for pods and ingress            | Retries may show **D**  | Benign during wait                          |
| **5** | Close dead cursor           | Close after recovery                 | **E** primary; also **A, B, C, D** | **404**, also **410**/**503**/transport OK |


### **CircleCI reference run (HTTP ingress)**


| **Checkpoint**     | **HTTP/1**              | **HTTP/2**                               | **Category**  |
| ------------------ | ----------------------- | ---------------------------------------- | ------------- |
| Cursor interrupted | `ArangoError: Code 410` | `invalid character '<'…` (non-JSON body) | **E** / **D** |
| Dead cursor resume | `ArangoError: Code 503` | `invalid character '<'…` (non-JSON body) | **C** / **D** |
| Close dead cursor  | `ArangoError: Code 404` | `ArangoError: Code 404`                  | **E**         |


```text
# HTTP/1
cursor interrupted after coordinator kill: ArangoError: Code 410, ErrorNum 0
dead cursor resume error: ArangoError: Code 503, ErrorNum 0
… ingress not ready yet (attempt 1): invalid character '<'…   ← recovery only (benign)
cursor close on dead cursor: ArangoError: Code 404, ErrorNum 0

# HTTP/2
cursor interrupted after coordinator kill: invalid character '<' looking for beginning of value
dead cursor resume error: invalid character '<' looking for beginning of value
cursor close on dead cursor: ArangoError: Code 404, ErrorNum 0
```

### **Why category D appears (non-JSON gateway response)**

Category **D** is defined at the HTTP layer: the client got **HTTP 502/503/504** with a **non-JSON** response — typically `Content-Type: text/html` from nginx when **no healthy coordinator backend** is ready.

**Go driver v2** often reports this only as `invalid character '<' looking for beginning of value` because it attempts JSON deserialization on the HTML body instead of returning an HTTP/status-aware error (known driver misbehavior; fix planned).


| **Where in the logs**                       | **HTTP/1**                                                                               | **HTTP/2**                                                   |
| ------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| **Cursor interrupted / resume**             | JSON **410** / **503** — request still reached ArangoDB (or a shutting-down coordinator) | **D** — ingress had no usable backend, returned HTML 502/503 |
| **Recovery wait** (`ingress not ready yet`) | **D** once — benign retry until `Version()` succeeds                                     | Not required in this run                                     |


**Important:** HTML is **not** always present on both HTTP/1 and HTTP/2. In this CircleCI run, **interrupt/resume HTML is HTTP/2 only**. HTTP/1’s `invalid character '<'` appeared during **ingress recovery**, not as the cursor-interrupt assertion error.

Compared to **CoordinatorKillDuringRead** on the same CI run (interrupt **503**, resume **409**/**503**), mid-iteration kill spent longer on the streaming cursor; by the time HTTP/2 issued the post-kill read, ingress backends were often fully gone → **D**, while HTTP/1 still got a JSON **410** from an answering coordinator.

**Reject:** Cursor finishes remaining documents after kill; hang; panic; category **F**.

---

## **Cluster recovery checklist (after coordinator kill)**

**Before asserting post-recovery behavior, wait until:**

1. **3 coordinator pods are Running and Ready (**`1/1`**)**
2. **ArangoDeployment status is Ready**
3. **Service endpoints exist for** `arangodb-driver-tests-ea`
4. `**GET /_api/version`**** succeeds through ingress (retry up to 5 minutes)**

---

## **Benign log lines (not failures)**


| **Log / message**                               | **Why it is OK**                           |
| ----------------------------------------------- | ------------------------------------------ |
| `coordinator pods ready: 2/3`                   | **Pods still recreating between subtests** |
| `ingress not ready yet: invalid character '<'…` | **Recovery retry hitting nginx HTML page** |
| `cursor close on dead cursor: Code 404`         | **Expected after coordinator kill**        |
| `Removing DB … Code 503`                        | **Test teardown raced with recovery**      |
| `Unable to get SUT health`                      | **Harness probe during restart**           |


---

# **Part B — Toxiproxy network faults**

**Proxy:** `127.0.0.1:17001` **→ ArangoDB ingress**  
**Image:** `ghcr.io/shopify/toxiproxy:2.9.0`


| **#**  | **Test**                              | **Fault**                                       | **API**                   | **Pass**                          | **Error category**    |
| ------ | ------------------------------------- | ----------------------------------------------- | ------------------------- | --------------------------------- | --------------------- |
| **1**  | **AbruptTCPConnectionClose**          | `reset_peer` **toxic**                          | **Version**               | **Fail then recover**             | **A — reset / EOF**   |
| **2**  | **NetworkDisconnect**                 | **Disable proxy**                               | **Version**               | **Fail then recover**             | **A — refused / EOF** |
| **3**  | **ConnectionResetByPeer**             | `reset_peer` **downstream**                     | **Version**               | **Fail then recover**             | **A**                 |
| **4**  | **HighLatency**                       | **2000 ms latency**                             | **Version**               | **Succeed, slower**               | **None**              |
| **5**  | **ExtremeLatency**                    | **30 s latency, 10 s client timeout**           | **Version**               | **Fail**                          | **B**                 |
| **6**  | **LatencyRemoved**                    | **Remove latency toxic**                        | **Version**               | **Faster after remove**           | **None**              |
| **7**  | **ContextTimeout**                    | **20 s latency, 2 s client timeout**            | **Version**               | **Fail < 5 s**                    | **B**                 |
| **8**  | **ServerTimeout**                     | **20 s downstream latency, 2 s header timeout** | **Version (HTTP/1 only)** | **Fail**                          | **B**                 |
| **9**  | **PartialPacketLoss**                 | **30% reset_peer**                              | **Version × 20**          | **Mix fail/success**              | **A or B**            |
| **10** | **FullPacketLoss**                    | **timeout toxic**                               | **Version**               | **Timeout**                       | **B**                 |
| **11** | **DisconnectDuringCursorIteration**   | **Disable proxy after 5 docs**                  | **Cursor read**           | **Next read fails**               | **A**                 |
| **12** | **DisconnectDuringQueryExecution**    | **Disable proxy during db.Query()**             | **AQL query start**       | **Query fails**                   | **A**                 |
| **13** | **DisconnectDuringInsert**            | **Disable proxy during CreateDocument**         | **Document create**       | **Write fails; outcome unknown**  | **A**                 |
| **14** | **DisconnectDuringTransactionCommit** | **Disable proxy during Commit()**               | **Transaction commit**    | **Commit fails; outcome unknown** | **A**                 |


**Detailed step-by-step scenarios:** **#1–#14** below (all Toxiproxy tests). Kubernetes scenarios **#0–#8** are in Part A.

### **Toxiproxy error examples**


| **Test**                      | **HTTP/1**                                                        | **HTTP/2**                                   |
| ----------------------------- | ----------------------------------------------------------------- | -------------------------------------------- |
| **reset_peer**                | `connection reset by peer`                                        | `unexpected EOF`                             |
| **proxy disabled**            | `connection refused`                                              | `connection refused` **or** `unexpected EOF` |
| **client timeout**            | `context deadline exceeded`                                       | **same**                                     |
| **cursor disconnect**         | `connection refused`**,** `EOF`**,** `broken pipe`                | **same**                                     |
| **query startup disconnect**  | `EOF` on `POST /_api/cursor`                                      | `unexpected EOF` on `POST /_api/cursor`      |
| **write / commit disconnect** | `EOF` on `POST /_api/document/…` **or** `PUT /_api/transaction/…` | `unexpected EOF` on the same APIs            |


## **Shared setup (Part B)**

### **Traffic path**

```text
Driver  ──HTTP(S)──►  127.0.0.1:17001  (Toxiproxy listen / data port)
                              │
                              ▼
                         Toxiproxy process
                              │
                              ▼
                    Ingress :80 or :443  (Host: arangodb.local required)
                              │
                              ▼
                         Coordinator
```

**Required:** every driver request through `127.0.0.1:17001` must send HTTP `Host: arangodb.local` (from `TEST_INGRESS_HOST`). Without it, Ingress will not route to ArangoDB.

Go reference (Docker Desktop / WSL): listen binds `0.0.0.0:17001` inside the container; published as `127.0.0.1:17001` on the host; upstream is `host.docker.internal:80` so Toxiproxy can reach kind Ingress on the Docker host.

**Run (HTTP Ingress — CI default):**

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy
```

**Run (HTTPS Ingress — local / optional):**

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy-tls
```

**Cluster:** `K8S_COORDINATORS_COUNT=1` for toxiproxy targets; Cluster mode still deploys 3 DB servers + 1 agent.

Every Toxiproxy test runs **HTTP/1** and **HTTP/2** subtests (except **ServerTimeout**, HTTP/1 only).

### **Listen port vs admin port (both required)**


| **Port**            | **Default** | **Who uses it** | **Purpose**                                                                                                                                       |
| ------------------- | ----------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Listen (data)**   | `17001`     | **Driver**      | Application traffic: `Driver → Toxiproxy → Ingress`. Faults (`reset_peer`, latency, `Disable`) apply on this TCP path.                            |
| **Admin (control)** | `8474`      | **Tests only**  | REST API to create the proxy and call `AddToxic` / `RemoveToxic` / `Enable` / `Disable`. The driver never talks to this port for normal requests. |


Without the **listen** port, the driver has no proxied endpoint. Without the **admin** port, tests cannot inject faults.

### **What every driver reuses vs what each language owns**


| **Component**                                             | **Reuse across drivers?** | **Role**                                                                                                                                                    |
| --------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `deploy/kubernetes/run-driver-tests.sh` (`run-toxiproxy`) | **Yes**                   | Deploy kube-arangodb + Ingress; export endpoint/auth/`TOXIPROXY_`* env; run `<driver-test-command>`.                                                        |
| `test/toxiproxy.sh`                                       | **Yes (copy or share)**   | Language-agnostic: `docker run` Shopify Toxiproxy image, wait for admin ready, `POST /proxies` named `arangodb`, cleanup.                                   |
| Toxiproxy scenario steps (#1–#14) + error categories A/B  | **Yes (this doc)**        | Same faults, APIs, pass/fail rules.                                                                                                                         |
| Admin-API helpers (`AddToxic`, `Disable`, …)              | **No — per language**     | Go: `v2/tests/toxiproxy_helper_test.go` (Shopify Go client → admin `:8474`). Java/JS/Python implement the same HTTP admin calls with their own HTTP client. |
| Error classifiers                                         | **No — per language**     | Go: `network_fault_error_util_test.go`. Map to categories A/B in your language.                                                                             |


**Shopify Toxiproxy** is the open-source proxy project (historically from Shopify). The Go package `github.com/Shopify/toxiproxy/v2/client` is only a thin HTTP client for admin `:8474` — other languages do not need that Go package; they call the same REST admin API.

**Go container name:** `test/toxiproxy.sh` names the container `${TESTCONTAINER}-toxiproxy`. Go Makefile sets `TESTCONTAINER=go-driver-test`, so you see `go-driver-test-toxiproxy` (`docker rm -f go-driver-test-toxiproxy`).

### **HTTP vs HTTPS / TLS modes (CI recommendation)**


| **Target**                          | **Flags**                              | **Meaning**                                                                                                                   | **CI**                                                                                                                                                                    |
| ----------------------------------- | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `make run-k8s-v2-toxiproxy`         | `K8S_INGRESS_TLS=false`, `K8S_TLS=false` | Plain HTTP to Ingress                                                                                                       | **Yes — default.** Covers all Part B fault categories on the TCP path.                                                                                                    |
| `make run-k8s-v2-toxiproxy-tls`     | `K8S_INGRESS_TLS=true`, `K8S_TLS=false`  | HTTPS at Ingress (self-signed); ArangoDB pods remain HTTP                                                                   | **Optional / local.** Same toxics still apply (Toxiproxy is L4).                                                                                                          |
| `make run-k8s-v2-toxiproxy-e2e-tls` | `K8S_INGRESS_TLS=true`, `K8S_TLS=true`   | HTTPS Driver↔Ingress **and** HTTPS Ingress↔ArangoDB (`backend-protocol: HTTPS`)                                             | **Optional / local.** Not in CircleCI until explicitly requested. Same Part B scenarios; validates TLS on both hops.                                                      |

Resiliency mirrors the same three modes: `run-k8s-v2-resiliency`, `run-k8s-v2-resiliency-tls`, `run-k8s-v2-resiliency-e2e-tls`.

Part B is primarily about **transport faults through a proxy**, not production PKI. HTTP CI is sufficient for scenarios **#1–#14**; Ingress TLS and end-to-end TLS are extra modes to run locally (or add to CI after team confirmation).

### **Proxy vs toxic**


| **Concept**                    | **Meaning**                                                                                                            |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------- |
| **Proxy**                      | Permanent TCP tunnel (listen → ingress). Stays up for the whole suite.                                                 |
| **Toxic**                      | Temporary fault rule on that tunnel (`reset_peer`, `latency`, `timeout`, …). Added for the fault window, then removed. |
| `proxy.Disable()` / `Enable()` | Hard cut of the listen port (no tunnel) vs restoring it.                                                               |


**Recovery:** remove toxics and/or `proxy.Enable()`, then wait for a successful `Version()` through the proxy (1 min local / 3 min k8s).

### **Go execution flow (reference)**

```text
make run-k8s-v2-toxiproxy
  → run-driver-tests.sh run-toxiproxy make run-v2-tests-toxiproxy-k8s
      → deploy cluster + Ingress; export TEST_ENDPOINTS_OVERRIDE=http://127.0.0.1:17001,
        TEST_INGRESS_HOST=arangodb.local, TOXIPROXY_UPSTREAM=…, TOXIPROXY_* ports
      → make run-v2-tests-toxiproxy-k8s
          → (Go) remap TEST_ENDPOINTS_OVERRIDE → TEST_ENDPOINTS
          → test/toxiproxy.sh start   # container go-driver-test-toxiproxy, ports 8474 + 17001
          → go test -tags toxiproxy -run '^TestToxiproxy_'
          → test/toxiproxy.sh cleanup
```

Prefer `TOXIPROXY_TEST_RUN` (or leave unset for `^TestToxiproxy_`) over a leftover `TESTOPTIONS=-test.run …` — a shell `TESTOPTIONS` from other suites can accidentally run non-Toxiproxy tests through the proxy without injecting faults.

**Helpers** (`network_fault_error_util_test.go`):


| **Helper**                                                 | **Used by**                        | **Accepts**                         |
| ---------------------------------------------------------- | ---------------------------------- | ----------------------------------- |
| `isConnectionError()`                                      | Connection-loss tests **#1–#2**    | Category **A** transport failures   |
| `isResetOrEOFError()`                                      | **ConnectionResetByPeer** (**#3**) | `connection reset` / `EOF`          |
| `isStreamingInterruptError()`                              | Streaming tests **#11–#12**        | Category **A** + cancel mid-stream  |
| `isWriteInterruptError()`                                  | Write tests **#13–#14**            | Category **A** mid-flight write cut |
| `isDriverTimeoutError()` / `isContextDeadlineExceeded()`   | Latency/timeout tests              | Category **B**                      |
| `isIntermittentNetworkError()`                             | Partial packet loss **#9**         | Category **A** or **B**             |


**Reject (all Part B):** panic; hang past the test timeout; category **F**; no recovery after the fault is cleared.

---

## **AbruptTCPConnectionClose**

**Toxiproxy test #1 — Upstream `reset_peer` on a live pooled connection**

**Goal:** Validate that `Version()` fails with a **clean transport error** (category **A**) when Toxiproxy injects a TCP RST on the **upstream** (client → server) stream, and that the **same client** recovers after the toxic is removed.

**Why keep this scenario:** Models an abrupt TCP close on an already-open keep-alive connection (LB idle kill, middlebox RST, peer abort). Distinct from **#2** (`proxy.Disable()` — nothing listening) and **#3** (downstream RST on the response path).

### **Execution order**


| **Step** | **Action**                                                                                                                                       |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **1**    | Connect through Toxiproxy; wait for successful `Version()` (establishes HTTP keep-alive in pool)                                                 |
| **2**    | Add toxic: `reset_peer` / **upstream** / toxicity `1.0` / `timeout: 0` (RST immediately)                                                         |
| **3**    | Call `Version()` with 15 s context — **must fail fast** (milliseconds; deadline is only a safety bound) — **checkpoint: connection interrupted** |
| **4**    | `RemoveToxic("reset_peer")` — proxy stays up; only the fault rule is deleted                                                                     |
| **5**    | Wait for successful `Version()` on the same client — **checkpoint: recovery**                                                                    |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**         | **What happens**                     | **Accept categories** | **Common Go v2 messages**                    |
| ----- | ---------------------- | ------------------------------------ | --------------------- | -------------------------------------------- |
| —     | Baseline               | Step 1 — `Version()` succeeds        | —                     | **Required**                                 |
| **1** | Connection interrupted | `Version()` after `reset_peer` added | **A**                 | `connection reset by peer`, `unexpected EOF` |
| **2** | Recovery               | `Version()` after toxic removed      | —                     | **Required**                                 |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Result** | **Duration** | **Notes**                 |
| ----------- | ---------- | ------------ | ------------------------- |
| HTTP/1      | **PASS**   | ~0.04 s      | Fault fails fast; no hang |
| HTTP/2      | **PASS**   | ~0.02 s      | Fault fails fast; no hang |


**Expected failing API:** `GET …/_api/version` (or driver `Version()` equivalent).

**Expected error messages** (from Go v2 reference comments / error catalog; HTTP/1 vs HTTP/2 differ):


| **Subtest** | **Example error**                                                      | **Category** |
| ----------- | ---------------------------------------------------------------------- | ------------ |
| HTTP/1      | `read tcp …: read: connection reset by peer`                           | **A**        |
| HTTP/2      | `unexpected EOF` (HTTP/2 framing sees abrupt close as incomplete data) | **A**        |


```text
# HTTP/1 (typical)
Get "http://127.0.0.1:17001/_api/version": read tcp …: read: connection reset by peer

# HTTP/2 (typical)
Get "http://127.0.0.1:17001/_api/version": unexpected EOF
```

### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**                                  | **HTTP/1 (typical)**       | **HTTP/2 (typical)** |
| ------------------------------------------ | -------------------------- | -------------------- |
| Upstream `reset_peer` on pooled connection | `connection reset by peer` | `unexpected EOF`     |


**Helper:** `isConnectionError()` — transport-level only (TCP reset, broken pipe, EOF, dial errors). **Not** ArangoDB HTTP errors (401, 503) or application responses.

**Reject:** Silent success while toxic is active; panic; hang until the 15 s deadline; failure that is not category **A**; no recovery after `RemoveToxic`.

---

## **NetworkDisconnect**

**Toxiproxy test #2 — Full outage via `proxy.Disable()` / `Enable()`**

**Goal:** Validate that `Version()` fails with a **clean transport error** (category **A**) while the Toxiproxy listen port is disabled (nothing accepts connections), and that the **same client** recovers after `proxy.Enable()`.

**Why keep this scenario:** Models a complete network path outage (cable unplug, firewall drop, proxy/LB process down) — the dial fails because nothing is listening. Distinct from **#1** / **#3**, which keep the proxy up and inject RST on an existing tunnel.

**Source:** `toxiproxy_connection_loss_test.go` (`TestToxiproxy_NetworkDisconnect`).

### **Execution order**


| **Step** | **Action**                                                                                  |
| -------- | ------------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                  |
| **2**    | `proxy.Disable()` — listen port closed; active connections dropped; no traffic passes       |
| **3**    | Call `Version()` with 15 s context — **must fail** — **checkpoint: connection interrupted** |
| **4**    | `proxy.Enable()` — network path restored                                                    |
| **5**    | Wait for successful `Version()` on the same client — **checkpoint: recovery**               |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**         | **What happens**                 | **Accept categories** | **Common Go v2 messages**                     |
| ----- | ---------------------- | -------------------------------- | --------------------- | --------------------------------------------- |
| —     | Baseline               | Step 1 — `Version()` succeeds    | —                     | **Required**                                  |
| **1** | Connection interrupted | `Version()` while proxy disabled | **A**                 | `connection refused`, `unexpected EOF`, `EOF` |
| **2** | Recovery               | `Version()` after `Enable()`     | —                     | **Required**                                  |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**      | **Example error**                                       | **Category** |
| ----------- | -------------------- | ------------------------------------------------------- | ------------ |
| HTTP/1      | `GET …/_api/version` | `dial tcp 127.0.0.1:17001: connect: connection refused` | **A**        |
| HTTP/2      | `GET …/_api/version` | `dial tcp 127.0.0.1:17001: connect: connection refused` | **A**        |


```text
# HTTP/1
Get "http://127.0.0.1:17001/_api/version": dial tcp 127.0.0.1:17001: connect: connection refused

# HTTP/2
Get "http://127.0.0.1:17001/_api/version": dial tcp 127.0.0.1:17001: connect: connection refused
```

**Suite timing (this run):** total ~0.06 s (HTTP/1 ~0.02 s, HTTP/2 ~0.02 s) — fails fast on dial; no hang.

### **HTTP/1 vs HTTP/2 — same fault, same message (this run)**


| **Fault**                   | **HTTP/1 (this run)** | **HTTP/2 (this run)** | **Also acceptable**                                                   |
| --------------------------- | --------------------- | --------------------- | --------------------------------------------------------------------- |
| `proxy.Disable()` then dial | `connection refused`  | `connection refused`  | `unexpected EOF` / `EOF` if a pooled connection is reused before dial |


**Helper:** `isConnectionError()` — transport-level only. **Not** ArangoDB HTTP errors (401, 503).

**Reject:** Silent success while proxy is disabled; panic; hang until the 15 s deadline; failure that is not category **A**; no recovery after `Enable()`.

---

## **ConnectionResetByPeer**

**Toxiproxy test #3 — Downstream `reset_peer` (remote peer aborts the response path)**

**Goal:** Validate that `Version()` fails with a **clean transport error** (category **A**) when Toxiproxy injects a TCP RST on the **downstream** (server → client) stream, and that the **same client** recovers after the toxic is removed.

**Why keep this scenario:** Models the remote peer aborting the connection during the response path (server/LB RST while reading the reply). Distinct from **#1** (upstream RST on the request path) and **#2** (`proxy.Disable()` — dial fails with nothing listening).

**Source:** `toxiproxy_connection_loss_test.go` (`TestToxiproxy_ConnectionResetByPeer`).

### **Execution order**


| **Step** | **Action**                                                                                         |
| -------- | -------------------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                         |
| **2**    | Add toxic: `reset_peer` / **downstream** / toxicity `1.0` / `timeout: 0` (named `reset_peer_down`) |
| **3**    | Call `Version()` with 15 s context — **must fail** — **checkpoint: connection interrupted**        |
| **4**    | `RemoveToxic("reset_peer_down")` — RST injection stops; proxy stays up                             |
| **5**    | Wait for successful `Version()` on the same client — **checkpoint: recovery**                      |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**         | **What happens**                          | **Accept categories** | **Common Go v2 messages**                    |
| ----- | ---------------------- | ----------------------------------------- | --------------------- | -------------------------------------------- |
| —     | Baseline               | Step 1 — `Version()` succeeds             | —                     | **Required**                                 |
| **1** | Connection interrupted | `Version()` after downstream `reset_peer` | **A**                 | `connection reset by peer`, `unexpected EOF` |
| **2** | Recovery               | `Version()` after toxic removed           | —                     | **Required**                                 |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**      | **Example error**                                                           | **Category** |
| ----------- | -------------------- | --------------------------------------------------------------------------- | ------------ |
| HTTP/1      | `GET …/_api/version` | `read tcp 127.0.0.1:45486->127.0.0.1:17001: read: connection reset by peer` | **A**        |
| HTTP/2      | `GET …/_api/version` | `unexpected EOF`                                                            | **A**        |


```text
# HTTP/1
Get "http://127.0.0.1:17001/_api/version": read tcp 127.0.0.1:45486->127.0.0.1:17001: read: connection reset by peer

# HTTP/2
Get "http://127.0.0.1:17001/_api/version": unexpected EOF
```

**Suite timing (this run):** total ~0.07 s (HTTP/1 ~0.04 s, HTTP/2 ~0.02 s) — fails fast; no hang.

### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**               | **HTTP/1 (this run)**      | **HTTP/2 (this run)** |
| ----------------------- | -------------------------- | --------------------- |
| Downstream `reset_peer` | `connection reset by peer` | `unexpected EOF`      |


**Helper:** `isResetOrEOFError()` — accepts `connection reset` and `EOF` / `unexpected EOF`. Narrower than `isConnectionError()` (used by **#1–#2**).

**Reject:** Silent success while toxic is active; panic; hang until the 15 s deadline; failure that is not reset/EOF; no recovery after `RemoveToxic`.

---

## **HighLatency**

**Toxiproxy test #4 — Moderate upstream latency; request still succeeds**

**Goal:** Validate that `Version()` **succeeds** when Toxiproxy adds **~2 s** upstream latency, as long as the caller context timeout (**10 s**) is larger than the delay. The call must take **materially longer** than a baseline call without the toxic (at least baseline + 1.5 s).

**Why keep this scenario:** Models a slow but still-working network path. The driver must not treat high latency as a hard failure when the deadline allows it. Distinct from **#5** (latency exceeds deadline → category **B**) and **#6** (latency removed → duration recovers).

**Source:** `toxiproxy_latency_test.go` (`TestToxiproxy_HighLatency`).

### **How the timing works (confirmation)**


| **Piece**                    | **Value**                              | **Role**                                                                        |
| ---------------------------- | -------------------------------------- | ------------------------------------------------------------------------------- |
| Upstream latency toxic       | `toxiproxyHighLatencyMs` = **2000 ms** | Toxiproxy delays client→server bytes by ~2 s                                    |
| `measureVersionCall` context | **10 s**                               | Must be **greater than** the injected latency so `Version()` can still complete |
| Assertion                    | `slow > baseline + 1.5s`               | Proves the toxic actually slowed the call; not a silent no-op                   |


So yes: with **2 s** latency and a **10 s** deadline, execution completes successfully inside the timeout. The test compares **baseline** (no toxic) vs **slow** (with toxic). There is **no error** expected — category **None**.

### **Execution order**


| **Step** | **Action**                                                              |
| -------- | ----------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy                                               |
| **2**    | Measure baseline `Version()` with 10 s context → `baseline`             |
| **3**    | Add toxic: upstream `latency` **2000 ms** (`latency_up`)                |
| **4**    | Measure `Version()` again with 10 s context → `slow` — **must succeed** |
| **5**    | Assert `slow > baseline + 1.5s` — **checkpoint: latency observed**      |


### **Checkpoints**


| **#** | **Checkpoint**   | **What happens**                                    | **Accept categories** | **Notes**                     |
| ----- | ---------------- | --------------------------------------------------- | --------------------- | ----------------------------- |
| —     | Baseline         | `Version()` without toxic succeeds                  | —                     | **Required**                  |
| **1** | Latency observed | `Version()` with 2 s latency succeeds and is slower | **None**              | No error; duration check only |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Result** | **Duration** | **Notes**                     |
| ----------- | ---------- | ------------ | ----------------------------- |
| HTTP/1      | **PASS**   | ~2.02 s      | Matches ~2 s injected latency |
| HTTP/2      | **PASS**   | ~2.03 s      | Same                          |


**Reject:** `Version()` fails under 2 s latency with a 10 s deadline; duration does not increase by ~1.5 s+; panic.

---

## **ExtremeLatency**

**Toxiproxy test #5 — Latency exceeds client deadline (category B)**

**Goal:** Validate that `Version()` **fails** with **context deadline exceeded** (category **B**) when upstream latency (**30 s**) is longer than the caller timeout (**10 s**). The driver must not panic or hang past the deadline.

**Why keep this scenario:** Models a path so slow the client correctly gives up. Complements **#4** (succeed when deadline > latency).

**Source:** `toxiproxy_latency_test.go` (`TestToxiproxy_ExtremeLatency`).

### **Execution order**


| **Step** | **Action**                                                                                |
| -------- | ----------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                |
| **2**    | Add toxic: upstream `latency` **30000 ms** (`latency_up`)                                 |
| **3**    | Call `Version()` with **10 s** context — **must fail** at ~10 s — **checkpoint: timeout** |
| **4**    | Cleanup removes toxic                                                                     |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint** | **What happens**                        | **Accept categories** | **Common Go v2 messages**   |
| ----- | -------------- | --------------------------------------- | --------------------- | --------------------------- |
| **1** | Timeout        | `Version()` with 30 s latency, 10 s ctx | **B**                 | `context deadline exceeded` |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**      | **Example error**           | **Category** | **Duration** |
| ----------- | -------------------- | --------------------------- | ------------ | ------------ |
| HTTP/1      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~10.02 s     |
| HTTP/2      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~10.04 s     |


```text
# HTTP/1 and HTTP/2 (same message)
Get "http://127.0.0.1:17001/_api/version": context deadline exceeded
```

**Suite timing (this run):** total ~20.07 s (one ~10 s timeout per protocol).

**Helper:** `isContextDeadlineExceeded()`.

**Reject:** Success despite 30 s latency / 10 s deadline; hang past ~10 s; panic; category **A** connection errors instead of deadline (unless the path is also broken).

---

## **LatencyRemoved**

**Toxiproxy test #6 — Duration recovers after latency toxic is removed**

**Goal:** Validate that after a **2 s** upstream latency toxic is **removed**, a subsequent `Version()` is **much faster** than the throttled call (`afterRemoval < withLatency / 2`).

**Why keep this scenario:** Proves the driver and connection pool recover when the network path returns to normal — not only that latency can be injected.

**Source:** `toxiproxy_latency_test.go` (`TestToxiproxy_LatencyRemoved`).

### **Execution order**


| **Step** | **Action**                                                                                         |
| -------- | -------------------------------------------------------------------------------------------------- |
| **1**    | Measure baseline `Version()` (15 s context)                                                        |
| **2**    | Add upstream `latency` **2000 ms**; measure `withLatency` — must be `> baseline + 1.5s`            |
| **3**    | `RemoveToxic("latency_up")`                                                                        |
| **4**    | Measure `afterRemoval` — assert `afterRemoval < withLatency / 2` — **checkpoint: recovered speed** |


### **Checkpoints**


| **#** | **Checkpoint**  | **What happens**                 | **Accept categories** | **Notes**                |
| ----- | --------------- | -------------------------------- | --------------------- | ------------------------ |
| **1** | Latency applied | `withLatency > baseline + 1.5s`  | **None**              | Succeeds, slower         |
| **2** | Recovered speed | `afterRemoval < withLatency / 2` | **None**              | No error; duration check |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **withLatency/2** | **afterRemoval** | **Result** |
| ----------- | ----------------- | ---------------- | ---------- |
| HTTP/1      | ~1.014 s          | ~2.98 ms         | **PASS**   |
| HTTP/2      | ~1.002 s          | ~3.16 ms         | **PASS**   |


```text
# HTTP/1
afterRemoval: 2.982327ms
withLatency/2: 1.013668215s

# HTTP/2
afterRemoval: 3.155844ms
withLatency/2: 1.002460038s
```

**Suite timing (this run):** total ~4.09 s.

**Reject:** After toxic removal, call still ~as slow as with latency; panic; failure to remove toxic.

---

## **ContextTimeout**

**Toxiproxy test #7 — Caller context deadline shorter than injected latency (category B)**

**Goal:** Validate that `Version()` fails with **context deadline exceeded** (category **B**) when upstream latency (**20 s**) is longer than the caller timeout (**2 s**), and that the failure happens **near the deadline** (elapsed **< 5 s**), not after waiting for the full 20 s toxic delay.

**Why keep this scenario:** Models application-level timeouts (`context.WithTimeout`) on a slow path. Distinct from **#5** (same idea, longer 10 s / 30 s window) and **#8** (transport `ResponseHeaderTimeout`, not caller context).

**Source:** `toxiproxy_timeout_test.go` (`TestToxiproxy_ContextTimeout`).

### **How the timing works**


| **Piece**              | **Value**                                         | **Role**                                                        |
| ---------------------- | ------------------------------------------------- | --------------------------------------------------------------- |
| Upstream latency toxic | `toxiproxyContextTimeoutLatencyMs` = **20000 ms** | Round-trip would take ~20 s if allowed to finish                |
| Caller context         | `toxiproxyContextTimeoutDeadline` = **2 s**       | Cancels `Version()` before the toxic delay completes            |
| Elapsed assertion      | **< 5 s**                                         | Proves the client stopped near the 2 s deadline, not after 20 s |


### **Execution order**


| **Step** | **Action**                                                                                             |
| -------- | ------------------------------------------------------------------------------------------------------ |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                             |
| **2**    | Add toxic: upstream `latency` **20000 ms** (`latency_up`)                                              |
| **3**    | Call `Version()` with **2 s** context — **must fail** with deadline exceeded — **checkpoint: timeout** |
| **4**    | Assert elapsed **< 5 s** — **checkpoint: failed fast**                                                 |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint** | **What happens**                       | **Accept categories** | **Common Go v2 messages**   |
| ----- | -------------- | -------------------------------------- | --------------------- | --------------------------- |
| **1** | Timeout        | `Version()` with 20 s latency, 2 s ctx | **B**                 | `context deadline exceeded` |
| **2** | Failed fast    | Elapsed < 5 s                          | —                     | **Required**                |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**      | **Example error**           | **Category** | **Duration** |
| ----------- | -------------------- | --------------------------- | ------------ | ------------ |
| HTTP/1      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~2.02 s      |
| HTTP/2      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~2.02 s      |


```text
# HTTP/1 and HTTP/2 (same message)
Get "http://127.0.0.1:17001/_api/version": context deadline exceeded
```

**Suite timing (this run):** total ~4.06 s.

**Helper:** `isContextDeadlineExceeded()`.

**Reject:** Success despite 20 s latency / 2 s deadline; hang until ~20 s; panic; elapsed ≥ 5 s.

---

## **ServerTimeout**

**Toxiproxy test #8 — HTTP/1 transport response-header timeout on delayed response (category B)**

**Goal:** Validate that `Version()` fails with a **driver/transport timeout** (category **B**) when the **server→client** response is delayed (**20 s** downstream latency) beyond the HTTP/1 `ResponseHeaderTimeout` (**2 s**), without hanging for the full delay (elapsed **< 5 s**).

**Why keep this scenario:** Models transport-level timeouts when response headers never arrive in time — different from **#7** (caller `context` deadline). HTTP/2 is **skipped** because `http2.Transport` has no `ResponseHeaderTimeout`.

**Source:** `toxiproxy_timeout_test.go` (`TestToxiproxy_ServerTimeout`); connection factory `connectionToxiproxyHttpServerTimeout` in `toxiproxy_util_test.go`.

### **How the timing works**


| **Piece**                | **Value**                                                | **Role**                                       |
| ------------------------ | -------------------------------------------------------- | ---------------------------------------------- |
| Downstream latency toxic | `toxiproxyServerTimeoutResponseLatencyMs` = **20000 ms** | Delays server→client bytes (response path)     |
| `ResponseHeaderTimeout`  | `toxiproxyServerTimeoutDeadline` = **2 s**               | HTTP/1 transport gives up waiting for headers  |
| Max wait assertion       | `toxiproxyServerTimeoutMaxWait` = **5 s**                | Must not hang for the full 20 s toxic          |
| HTTP/2                   | **SKIP**                                                 | No `ResponseHeaderTimeout` on HTTP/2 transport |


### **Execution order**


| **Step** | **Action**                                                                                                |
| -------- | --------------------------------------------------------------------------------------------------------- |
| **1**    | Connect with HTTP/1 factory that sets `ResponseHeaderTimeout = 2s`                                        |
| **2**    | Wait for successful `Version()`                                                                           |
| **3**    | Add toxic: **downstream** `latency` **20000 ms** (`latency_down`)                                         |
| **4**    | Call `Version()` with background context — **must fail** with transport timeout — **checkpoint: timeout** |
| **5**    | Assert elapsed **< 5 s** — **checkpoint: failed fast**                                                    |
| —        | HTTP/2 subtest **skipped** with message about missing `ResponseHeaderTimeout`                             |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint** | **What happens**                                           | **Accept categories** | **Common Go v2 messages**                     |
| ----- | -------------- | ---------------------------------------------------------- | --------------------- | --------------------------------------------- |
| **1** | Timeout        | `Version()` with 20 s downstream delay, 2 s header timeout | **B**                 | `net/http: timeout awaiting response headers` |
| **2** | Failed fast    | Elapsed < 5 s                                              | —                     | **Required**                                  |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Result** | **Example error**                                                                        | **Category** | **Duration** |
| ----------- | ---------- | ---------------------------------------------------------------------------------------- | ------------ | ------------ |
| HTTP/1      | **PASS**   | `net/http: timeout awaiting response headers`                                            | **B**        | ~2.05 s      |
| HTTP/2      | **SKIP**   | `HTTP/2 transport has no ResponseHeaderTimeout; server response delay covered on HTTP/1` | —            | 0 s          |


```text
# HTTP/1
Get "http://127.0.0.1:17001/_api/version": net/http: timeout awaiting response headers

# HTTP/2
--- SKIP: TestToxiproxy_ServerTimeout/HTTP/2
    HTTP/2 transport has no ResponseHeaderTimeout; server response delay covered on HTTP/1
```

**Suite timing (this run):** total ~2.07 s.

**Helper:** `isDriverTimeoutError()` (accepts response-header timeout and context deadlines).

### **ContextTimeout (#7) vs ServerTimeout (#8)**


|                     | **#7 ContextTimeout**       | **#8 ServerTimeout**                |
| ------------------- | --------------------------- | ----------------------------------- |
| **What fires**      | Caller `context` deadline   | HTTP/1 `ResponseHeaderTimeout`      |
| **Toxic direction** | Upstream latency            | Downstream latency                  |
| **Typical error**   | `context deadline exceeded` | `timeout awaiting response headers` |
| **HTTP/2**          | Runs                        | **Skipped**                         |


**Reject:** Success despite delayed headers; hang until ~20 s; panic; HTTP/2 required to pass (it must skip).

---

## **PartialPacketLoss**

**Toxiproxy test #9 — Intermittent ~30% upstream connection resets (category A or B)**

**Goal:** Validate that under ~**30%** upstream `reset_peer` toxicity, a mix of `Version()` calls **fail** with intermittent transport errors (category **A**, or **B** if a call times out) and **succeed**, with **no panic**, and that the driver recovers after the toxic is removed.

**Why keep this scenario:** Models flaky networks / partial packet loss. Toxiproxy 2.9 has no `packet_loss` toxic — this uses `reset_peer` with toxicity **0.3** on **new TCP links**. HTTP/1 uses **DisableKeepAlives** so each attempt opens a fresh connection; HTTP/2 is **skipped** (one multiplexed TCP connection cannot show per-request loss the same way).

**Source:** `toxiproxy_packet_loss_test.go` (`TestToxiproxy_PartialPacketLoss`); factory `connectionToxiproxyHttpNoKeepAlive`.

### **How it works**


| **Piece**  | **Value**                                              | **Role**                           |
| ---------- | ------------------------------------------------------ | ---------------------------------- |
| Toxic      | upstream `reset_peer`, toxicity **0.3**, timeout **0** | ~30% of new links get RST          |
| Attempts   | **20** `Version()` calls                               | Expect both failures and successes |
| Connection | HTTP/1, keep-alives **disabled**                       | Fresh TCP link per attempt         |
| HTTP/2     | **SKIP**                                               | Multiplexes one TCP connection     |


### **Execution order**


| **Step** | **Action**                                                                              |
| -------- | --------------------------------------------------------------------------------------- |
| **1**    | Connect with no-keep-alive HTTP/1 factory; wait for successful `Version()`              |
| **2**    | Add toxic: upstream `reset_peer` toxicity **0.3** (`reset_peer_partial`)                |
| **3**    | Loop **20** times: `Version()` on a **fresh** client/connection — count fail/success    |
| **4**    | Each failure must be `isIntermittentNetworkError` — **checkpoint: intermittent faults** |
| **5**    | Assert `failures > 0` and `successes > 0`                                               |
| **6**    | `RemoveToxic`; wait for successful `Version()` — **checkpoint: recovery**               |
| —        | HTTP/2 subtest **skipped**                                                              |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**      | **What happens**                    | **Accept categories** | **Common Go v2 messages**       |
| ----- | ------------------- | ----------------------------------- | --------------------- | ------------------------------- |
| **1** | Intermittent faults | Some of 20 calls fail, some succeed | **A or B** (on failures) | `connection reset by peer`, EOF, timeout |
| **2** | Recovery            | `Version()` after toxic removed     | —                        | **Required**                             |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Result** | **Example failure**                                                                                  | **Category** | **Notes**                      |
| ----------- | ---------- | ---------------------------------------------------------------------------------------------------- | ------------ | ------------------------------ |
| HTTP/1      | **PASS**   | `read tcp …: read: connection reset by peer` (multiple attempts; `isIntermittentNetworkError: true`) | **A**        | Mix of fail/success in ~0.14 s |
| HTTP/2      | **SKIP**   | `HTTP/2 multiplexes one TCP connection; per-request packet loss covered on HTTP/1`                   | —            | Expected                       |


```text
# HTTP/1 (example failures during the 20-attempt loop)
Get "http://127.0.0.1:17001/_api/version": read tcp 127.0.0.1:56862->127.0.0.1:17001: read: connection reset by peer
isIntermittentNetworkError: true

# HTTP/2
--- SKIP: TestToxiproxy_PartialPacketLoss/HTTP/2
    HTTP/2 multiplexes one TCP connection; per-request packet loss covered on HTTP/1
```

**Helper:** `isIntermittentNetworkError()` (connection / reset / EOF / timeout).

**Reject:** All 20 succeed or all 20 fail; panic; non-transport error on failure; no recovery after toxic removal; HTTP/2 required to pass.

---

## **FullPacketLoss**

**Toxiproxy test #10 — 100% upstream data block via `timeout` toxic (category B)**

**Goal:** Validate that when Toxiproxy blocks **all** upstream data (`timeout` toxic, toxicity **1.0**), `Version()` fails with a **timeout or transport error** (typically category **B**), returns within the call deadline (**10 s** + small slack), and recovers after the toxic is removed.

**Why keep this scenario:** Models a black-hole / full packet-loss path where bytes never arrive. Distinct from **#9** (partial intermittent RST) and **#2** (`proxy.Disable()` → immediate `connection refused`).

**Source:** `toxiproxy_packet_loss_test.go` (`TestToxiproxy_FullPacketLoss`).

### **How the timing works**


| **Piece**     | **Value**                                           | **Role**                   |
| ------------- | --------------------------------------------------- | -------------------------- |
| Toxic         | upstream `timeout`, toxicity **1.0**, timeout **0** | Stops all data on the link |
| Call context  | `toxiproxyFullPacketLossCallTimeout` = **10 s**     | Client gives up waiting    |
| Elapsed bound | deadline **+ 3 s**                                  | Must not hang indefinitely |


### **Execution order**


| **Step** | **Action**                                                                            |
| -------- | ------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                            |
| **2**    | Add toxic: upstream `timeout` toxicity **1.0** (`timeout_full`)                       |
| **3**    | Call `Version()` with **10 s** context — **must fail** — **checkpoint: blocked path** |
| **4**    | Assert elapsed within deadline + 3 s                                                  |
| **5**    | `RemoveToxic`; wait for successful `Version()` — **checkpoint: recovery**             |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint** | **What happens**                       | **Accept categories** | **Common Go v2 messages**   |
| ----- | -------------- | -------------------------------------- | --------------------- | --------------------------- |
| **1** | Blocked path   | `Version()` while timeout toxic active | **B** (or **A**)      | `context deadline exceeded` |
| **2** | Recovery       | `Version()` after toxic removed        | —                     | **Required**                |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**      | **Example error**           | **Category** | **Duration** |
| ----------- | -------------------- | --------------------------- | ------------ | ------------ |
| HTTP/1      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~10.03 s     |
| HTTP/2      | `GET …/_api/version` | `context deadline exceeded` | **B**        | ~10.55 s     |


```text
# HTTP/1 and HTTP/2 (same message)
Get "http://127.0.0.1:17001/_api/version": context deadline exceeded
```

**Suite timing (this run):** total ~20.59 s.

**Helper:** `isDriverTimeoutError()` **or** `isIntermittentNetworkError()`.

**Reject:** Success while toxic is active; hang past deadline + 3 s; panic; no recovery after `RemoveToxic`.

---

## **DisconnectDuringCursorIteration**

**Toxiproxy test #11 — Network cut during cursor streaming**

**Goal:** Validate that `cursor.ReadDocument()` returns a **clean transport error** (category **A**) when the proxy is disabled **after** the cursor is open and several documents have already been read. The driver must not panic; it must work again after the proxy is re-enabled.

**Why keep this scenario:** Simulates losing the connection while **streaming a large result set** — a common production failure (load balancer idle timeout, pod restart, network blip mid-batch). This is distinct from test **#12**, which fails during **query startup** before a cursor handle is returned.

### **Execution order**


| **Step** | **Action**                                                                                 |
| -------- | ------------------------------------------------------------------------------------------ |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                 |
| **2**    | Create database and collection                                                             |
| **3**    | Seed **100** documents (`4000`-byte payload each)                                          |
| **4**    | Open cursor: `FOR d IN <collection> RETURN d` with `batchSize: 1`                          |
| **5**    | Read **5** documents successfully via `ReadDocument()`                                     |
| **6**    | `proxy.Disable()` — hard network cut                                                       |
| **7**    | Next `ReadDocument()` **must fail** (≤ 30 s) — **checkpoint: cursor interrupted**          |
| **8**    | `proxy.Enable()`; wait for successful `Version()` through proxy — **checkpoint: recovery** |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**     | **What happens**                        | **Accept categories** | **Common Go v2 messages**                  |
| ----- | ------------------ | --------------------------------------- | --------------------- | ------------------------------------------ |
| —     | Reads before cut   | Steps 5 — all 5 reads succeed           | —                     | **Required**                               |
| **1** | Cursor interrupted | Next `ReadDocument()` after `Disable()` | **A**                 | `connection refused`, `EOF`, `broken pipe` |
| **2** | Recovery           | `Version()` succeeds after `Enable()`   | —                     | **Required**                               |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**                                      | **Example error**                                       | **Category** |
| ----------- | ---------------------------------------------------- | ------------------------------------------------------- | ------------ |
| HTTP/1      | `POST …/_api/cursor/<id>/<batch>` (next batch fetch) | `dial tcp 127.0.0.1:17001: connect: connection refused` | **A**        |
| HTTP/2      | `POST …/_api/cursor/<id>/<batch>` (next batch fetch) | `dial tcp 127.0.0.1:17001: connect: connection refused` | **A**        |


```text
# HTTP/1 and HTTP/2 (same message in this run)
Post "http://127.0.0.1:17001/_db/<db>/_api/cursor/<cursorId>/<batch>": dial tcp 127.0.0.1:17001: connect: connection refused
```

**Reject:** Panic; hang past 30 s context; cursor silently completes remaining documents; category **F** (e.g. AQL syntax error); no recovery after `Enable()`.

---

## **DisconnectDuringQueryExecution**

**Toxiproxy test #12 — Network cut during query startup**

**Goal:** Validate that `db.Query()` returns a **clean transport error** (category **A**) when the proxy is disabled **while the initial** `POST /_api/cursor` **is still in flight** — before the driver receives a cursor handle. The driver must not panic; it must work again after the proxy is re-enabled.

**Why keep this scenario:** Simulates network/LB/pod failure **during query startup** — before any cursor exists on the client. This is a different failure point from test **#11** (mid-stream `ReadDocument()`), even though both use category **A** transport errors.

### **Execution order**


| **Step** | **Action**                                                                                        |
| -------- | ------------------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; wait for successful `Version()`                                        |
| **2**    | Create database (no collection seeding required)                                                  |
| **3**    | Start `db.Query()` in a goroutine with slow AQL: `FOR i IN 1..1 RETURN { i: i, burn: SLEEP(30) }` |
| **4**    | Wait **500 ms** (query POST is in flight; server blocked in `SLEEP`)                              |
| **5**    | `proxy.Disable()` — hard network cut                                                              |
| **6**    | `db.Query()` **must return error** (≤ 30 s) — **checkpoint: query interrupted**                   |
| **7**    | `proxy.Enable()`; wait for successful `Version()` through proxy — **checkpoint: recovery**        |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**    | **What happens**                       | **Accept categories** | **Common Go v2 messages**                                       |
| ----- | ----------------- | -------------------------------------- | --------------------- | --------------------------------------------------------------- |
| —     | Query before cut  | Goroutine issued `POST /_api/cursor`   | —                     | Server still executing `SLEEP(30)`                              |
| **1** | Query interrupted | `db.Query()` returns after `Disable()` | **A**                 | `EOF`, `unexpected EOF`, `connection refused` on `/_api/cursor` |
| **2** | Recovery          | `Version()` succeeds after `Enable()`  | —                     | **Required**                                                    |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**                    | **Example error**                                                    | **Category** |
| ----------- | ---------------------------------- | -------------------------------------------------------------------- | ------------ |
| HTTP/1      | `POST …/_api/cursor` (query start) | `Post "http://127.0.0.1:17001/_db/<db>/_api/cursor": EOF`            | **A**        |
| HTTP/2      | `POST …/_api/cursor` (query start) | `Post "http://127.0.0.1:17001/_db/<db>/_api/cursor": unexpected EOF` | **A**        |


```text
# HTTP/1
Post "http://127.0.0.1:17001/_db/<db>/_api/cursor": EOF

# HTTP/2
Post "http://127.0.0.1:17001/_db/<db>/_api/cursor": unexpected EOF
```

### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**                  | **HTTP/1 (typical)** | **HTTP/2 (typical)** |
| -------------------------- | -------------------- | -------------------- |
| Proxy disabled mid-request | `EOF`                | `unexpected EOF`     |
| Proxy disabled (dial)      | `connection refused` | `connection refused` |


**Reject:** Panic; hang past 30 s; `db.Query()` succeeds (disconnect too late); AQL syntax error (e.g. invalid `LET` binding); category **F**; no recovery after `Enable()`.

**Note:** Do **not** use `LET _ = SLEEP(…)` — `_` is not a valid AQL identifier. Use `RETURN { i: i, burn: SLEEP(30) }` as in the reference query above.

---

## **DisconnectDuringInsert**

**Toxiproxy test #13 — Network cut during document create (outcome unknown)**

**Goal:** Validate that `CreateDocument` returns a **clean transport error** (category **A**, no panic/hang) when the proxy is disabled **while the write is in flight**. After recovery, the client **must not assume** whether the server committed the document.

**Why keep this scenario:** Mid-flight write disconnect is one of the hardest distributed-systems cases — the request may have reached ArangoDB and been persisted before the response was lost. The driver must surface a clean network error and remain usable; applications must treat the write as **at-most-once / unknown** and reconcile if needed.

**Source:** `toxiproxy_writes_test.go` (`TestToxiproxy_DisconnectDuringInsert`).

### **How the fault is timed**


| **Piece**          | **Value**         | **Role**                                                      |
| ------------------ | ----------------- | ------------------------------------------------------------- |
| Downstream latency | **10000 ms**      | Delays the response so the request can reach the server first |
| Disconnect delay   | **500 ms**        | Cut after the write has likely left the client                |
| Fault              | `proxy.Disable()` | Hard network cut mid-flight                                   |


### **Execution order**


| **Step** | **Action**                                                                          |
| -------- | ----------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; create database and collection                           |
| **2**    | Add **downstream** `latency` **10000 ms**                                           |
| **3**    | Start `CreateDocument` in a goroutine                                               |
| **4**    | Wait **500 ms**; `proxy.Disable()`                                                  |
| **5**    | `CreateDocument` **must return error** (≤ 30 s) — **checkpoint: write interrupted** |
| **6**    | `proxy.Enable()`; remove toxic; wait for `Version()` — **checkpoint: recovery**     |
| **7**    | Optionally `ReadDocument` and **log** present/absent — **do not fail** on either    |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**    | **What happens**                   | **Accept categories** | **Notes**                                     |
| ----- | ----------------- | ---------------------------------- | --------------------- | --------------------------------------------- |
| **1** | Write interrupted | `CreateDocument` after `Disable()` | **A**                 | `EOF`, `connection refused`, `broken pipe`, … |
| **2** | Recovery          | `Version()` succeeds               | —                     | **Required**                                  |
| —     | Document state    | Present or absent after recovery   | —                     | **Informational only**                        |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**                     | **Example error** | **Category** | **After recovery (informational)**                 |
| ----------- | ----------------------------------- | ----------------- | ------------ | -------------------------------------------------- |
| HTTP/1      | `POST …/_api/document/<collection>` | `EOF`             | **A**        | Document **present** (server committed before cut) |
| HTTP/2      | `POST …/_api/document/<collection>` | `unexpected EOF`  | **A**        | Document **present** (server committed before cut) |


```text
# HTTP/1
Post "http://127.0.0.1:17001/_db/<db>/_api/document/<collection>": EOF
write outcome cannot be determined after transport interruption (server may or may not have committed)
after recovery: document insert_cut_… is present (the document exists after recovery)

# HTTP/2
Post "http://127.0.0.1:17001/_db/<db>/_api/document/<collection>": unexpected EOF
write outcome cannot be determined after transport interruption (server may or may not have committed)
after recovery: document insert_cut_… is present (the document exists after recovery)
```

**Suite timing (this run):** total ~18.40 s (HTTP/1 ~1.51 s, HTTP/2 ~16.85 s).

### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**                                                          | **HTTP/1 (this run)** | **HTTP/2 (this run)** |
| ------------------------------------------------------------------ | --------------------- | --------------------- |
| Mid-flight insert cut (`proxy.Disable()` after downstream latency) | `EOF`                 | `unexpected EOF`      |


**Why “document present” does not fail the test:** Downstream latency lets the write reach the server; cutting the proxy drops the **response**. Seeing the document after recovery is a valid illustration of **unknown outcome** — the client got an error even though the server committed. Other runs may find the document absent; both are acceptable.

**Helper:** `isWriteInterruptError()`.

**Reject:** Panic; hang; silent success while proxy disabled; requiring the document to exist or not exist; no recovery.

---

## **DisconnectDuringTransactionCommit**

**Toxiproxy test #14 — Network cut during transaction Commit (outcome unknown)**

**Goal:** Validate that `Commit()` returns a **clean transport error** (category **A**, no panic/deadlock) when the proxy is disabled **while commit is in flight**. The commit outcome may be **unknown** — the transaction could already be committed on the server.

**Why keep this scenario:** Same unknown-outcome problem as **#13**, but for the critical commit boundary. Drivers must not hang or panic; callers must not treat a network error on `Commit` as a definitive abort.

**Source:** `toxiproxy_writes_test.go` (`TestToxiproxy_DisconnectDuringTransactionCommit`).

### **Execution order**


| **Step** | **Action**                                                                                                  |
| -------- | ----------------------------------------------------------------------------------------------------------- |
| **1**    | Connect through Toxiproxy; create database and collection                                                   |
| **2**    | `BeginTransaction` (write collection); `CreateDocument` inside txn — **must succeed**                       |
| **3**    | Add **downstream** `latency` **10000 ms**                                                                   |
| **4**    | Start `txn.Commit()` in a goroutine                                                                         |
| **5**    | Wait **500 ms**; `proxy.Disable()`                                                                          |
| **6**    | `Commit` **must return error** (≤ 30 s) — **checkpoint: commit interrupted**                                |
| **7**    | `proxy.Enable()`; remove toxic; wait for `Version()` — **checkpoint: recovery**                             |
| **8**    | Best-effort `Abort` (ignore errors); optionally read doc and **log** visibility — **do not fail** on either |


### **Checkpoints — acceptable errors**


| **#** | **Checkpoint**     | **What happens**              | **Accept categories** | **Notes**                |
| ----- | ------------------ | ----------------------------- | --------------------- | ------------------------ |
| —     | Prep               | Begin + in-txn insert succeed | —                     | **Required**             |
| **1** | Commit interrupted | `Commit()` after `Disable()`  | **A**                 | Clean network error only |
| **2** | Recovery           | `Version()` succeeds          | —                     | **Required**             |
| —     | Commit outcome     | Doc visible or not            | —                     | **Informational only**   |


### **Observed Go v2 reference run (HTTP ingress, kind/WSL2)**


| **Subtest** | **Failing API**               | **Example error** | **Category** | **After recovery (informational)**                                      |
| ----------- | ----------------------------- | ----------------- | ------------ | ----------------------------------------------------------------------- |
| HTTP/1      | `PUT …/_api/transaction/<id>` | `EOF`             | **A**        | Doc **visible**; best-effort Abort: `transaction was already committed` |
| HTTP/2      | `PUT …/_api/transaction/<id>` | `unexpected EOF`  | **A**        | Doc **visible**; best-effort Abort: `transaction was already committed` |


```text
# HTTP/1
Put "http://127.0.0.1:17001/_db/<db>/_api/transaction/<id>": EOF
commit outcome is unknown after mid-flight disconnect (txn may or may not be committed)
post-recovery Abort (best-effort): while trying to abort transaction …: transaction was already committed
after recovery: document txn_commit_cut_… is visible (commit likely succeeded before cut)

# HTTP/2
Put "http://127.0.0.1:17001/_db/<db>/_api/transaction/<id>": unexpected EOF
commit outcome is unknown after mid-flight disconnect (txn may or may not be committed)
post-recovery Abort (best-effort): while trying to abort transaction …: transaction was already committed
after recovery: document txn_commit_cut_… is visible (commit likely succeeded before cut)
```

**Suite timing (this run):** total ~9.02 s (HTTP/1 ~1.99 s, HTTP/2 ~7.02 s).

### **HTTP/1 vs HTTP/2 — same fault, different message**


| **Fault**             | **HTTP/1 (this run)** | **HTTP/2 (this run)** |
| --------------------- | --------------------- | --------------------- |
| Mid-flight commit cut | `EOF`                 | `unexpected EOF`      |


**Why Abort says “already committed”:** That is expected when the server finished `Commit` before the client saw the error. It confirms the unknown-outcome story — do **not** treat it as a test failure.

**Helper:** `isWriteInterruptError()`.

**Never / Reject:** Panic; deadlock/hang on `Commit`; treating network error as “definitely aborted”; requiring a specific committed/aborted outcome; no recovery after `Enable()`.

---

# **Running tests (Go reference)**

**Other drivers should replicate the same scenarios and error categories using their own test framework**, while reusing `run-driver-tests.sh` and preferably `test/toxiproxy.sh`.

```bash
# One-time: create kind cluster + ingress
bash ./deploy/kubernetes/run-driver-tests.sh setup-kind

# Run Go resiliency tests (HTTP Ingress — CI default)
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency

# Resiliency HTTPS Ingress only (local / optional)
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency-tls

# Resiliency end-to-end TLS: Ingress + ArangoDB (local / optional)
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency-e2e-tls

# Toxiproxy tests (HTTP Ingress — CI default)
unset TESTOPTIONS   # avoid accidental -test.run from other suites
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy

# Toxiproxy HTTPS Ingress only (local / optional)
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy-tls

# Toxiproxy end-to-end TLS: Ingress + ArangoDB (local / optional)
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy-e2e-tls
```


| **Setting**         | **Value**                                                                                                         |
| ------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **Coordinators**    | **3** (resiliency) / **1** (Toxiproxy)                                                                            |
| **Endpoint env**    | Harness exports `TEST_ENDPOINTS_OVERRIDE` (Go remaps to `TEST_ENDPOINTS`)                                         |
| **Endpoint value**  | Resiliency: `http(s)://arangodb.local`. Toxiproxy: `http(s)://127.0.0.1:17001`                                    |
| **Host header**     | Toxiproxy: send `Host: $TEST_INGRESS_HOST` (`arangodb.local`) on every request                                    |
| **Auth**            | `TEST_AUTHENTICATION[_OVERRIDE]=basic:root:rootpw`                                                                |
| **Deployment name** | `arangodb-driver-tests`                                                                                           |
| **Namespace**       | `default`                                                                                                         |


**Harness wiring:** see [Harness env contract (all drivers)](#harness-env-contract-all-drivers). Overview slides (optional): `deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html`. Fuller runner notes: `v2/tests/k8s-tests.md`, `deploy/kubernetes/README.md`.

---

# **Implementing in your driver — checklist**

1. **Reuse shared infra** — `run-driver-tests.sh` for the cluster; reuse or copy `test/toxiproxy.sh` to start Toxiproxy. Write language-specific admin helpers + error classifiers only.
2. **Connect through ingress / Toxiproxy** — same endpoints and auth as in [Running tests](#running-tests-go-reference).
3. **Run each scenario** — match the steps in Part A/B exactly (same fault, same API calls, same timing).
4. **Classify errors by category (A–E)** — do not hard-code one string.
5. **Test HTTP/1 and HTTP/2 separately** if your driver supports both.
6. **Log the full error in test output** — helps compare across drivers.
7. **Assert recovery** — same client instance works after fault is removed.
8. **Never accept:** hang past timeout, panic/crash, cursor completing after kill, data corruption.

### **Suggested helper names (map to categories)**


| **Category**          | **Suggested helper name**  | **Used in**                                       |
| --------------------- | -------------------------- | ------------------------------------------------- |
| **A + B + C + D**     | `isTransientOutageError()` | **Active workload tests (version loop, inserts)** |
| **A + B + C + D + E** | `isCursorKilledError()`    | **Cursor interrupt + dead cursor + close**        |
| **A**                 | `isConnectionError()`      | **Toxiproxy connection tests**                    |
| **B**                 | `isTimeoutError()`         | **Toxiproxy timeout tests** (Go: `isDriverTimeoutError`) |
| **A or B**            | `isIntermittentNetworkError()` | **Toxiproxy partial packet loss (#9)**        |


**Go reference helpers:** `isResiliencyTransientError`, `isCoordinatorKillInterruptedError`, `isDeadCursorError` in `v2/tests/network_fault_error_util_test.go`.

---

# **Appendix — Timing budgets**


| **Operation**                       | **Max wait**                |
| ----------------------------------- | --------------------------- |
| **Failover probe after kill**       | **90 s**                    |
| **Ingress rollout ready**           | **6 min**                   |
| **Coordinator pods ready**          | **10 min**                  |
| **Cluster recovery after kill**     | **5 min for version check** |
| **Version loop interval**           | **100 ms**                  |
| **Insert loop interval**            | **50 ms**                   |
| **Per-request timeout (workloads)** | **10 s**                    |
| **Cursor open**                     | **2 min**                   |
| **Cursor fail after kill**          | **90 s**                    |


---

# **Appendix — Go source files**


| **Area**                 | **Files**                                                                                                                                            |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| **K8s infrastructure**   | `deploy/kubernetes/run-driver-tests.sh`, `deploy/kubernetes/README.md`                                                                                |
| **This reference**       | `deploy/kubernetes/documentation/driver-resiliency-reference.md` (authoritative; internal)                                                           |
| **Overview slides**      | `deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html` (optional)                                                                       |
| **Toxiproxy process**    | `test/toxiproxy.sh` (shared start/stop + create proxy — reuse in other drivers)                                                                      |
| **Resiliency tests**     | `v2/tests/resiliency_loadbalancer_test.go`, `resiliency_failover_test.go`, `resiliency_ingress_restart_test.go`, `resiliency_coordinator_test.go`   |
| **K8s helpers**          | `v2/tests/resiliency_coordinator_k8s_helper_test.go`, `resiliency_util_test.go`                                                                       |
| **Workload helpers**     | `v2/tests/resiliency_workload_helper_test.go`, `resiliency_coordinator_workload_helper_test.go`                                                       |
| **Error classification** | `v2/tests/network_fault_error_util_test.go`                                                                                                          |
| **Toxiproxy tests**      | `v2/tests/toxiproxy_*_test.go`                                                                                                                       |
| **Toxiproxy Go helpers** | `v2/tests/toxiproxy_helper_test.go` (per-language; admin API only)                                                                                   |
| **Setup docs**           | `v2/tests/k8s-tests.md`                                                                                                                              |


