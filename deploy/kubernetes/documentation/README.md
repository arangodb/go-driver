# Shared K8s driver test infrastructure — documentation

> **Internal use only.** Materials in this folder are for ArangoDB **driver maintainers** adopting the shared kind + ingress-nginx + kube-arangodb test runner. They are **not** for external / end-user documentation.

Slide deck and scenario reference for onboarding other ArangoDB drivers onto the shared runner (integration, resiliency, Toxiproxy).

Open the deck in a browser:

```bash
# from repo root
xdg-open deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html
# or: open / start depending on OS
```

Navigate with **← →**, **Space**, or the on-screen buttons.

## Using these files in other drivers

Other ArangoDB driver teams can treat the materials in this folder as a **complete reference** to implement the same resiliency and Toxiproxy tests in their language:

1. **[`driver-resiliency-reference.md`](driver-resiliency-reference.md)** — **authoritative** Part A (#0–#8) and Part B (#1–#14) steps, error categories A–F, harness env contract, timing budgets, and observed Go v2 messages (**required**)
2. **[This deck](driver-k8s-shared-infra-demo.html)** — optional overview of architecture, env contract slides, wrapper pattern, and adoption checklist

The deck alone is not enough to implement complete scenario tests. Prefer the reference markdown (including the harness env section) as the source of truth when porting scenarios.

## Contents

| Slides | Topic |
|--------|--------|
| 1–3 | Goal: reuse infra, rewrite language tests |
| 4–5 | Architecture + `run-driver-tests.sh` commands |
| 6–7 | Env contracts (runner exports; drivers read/forward) |
| 8–9 | Part A resiliency scenarios + error categories A–F |
| 10–11 | Toxiproxy env (defaults vs driver work) + Part B scenarios |
| 12–14 | Wrapper pattern, adoption checklist, close |

Supporting docs:

- [../README.md](../README.md) — runner contract
- [driver-resiliency-reference.md](driver-resiliency-reference.md) — scenarios (internal)
- [../../../v2/tests/k8s-tests.md](../../../v2/tests/k8s-tests.md) — Go v2 Make targets
