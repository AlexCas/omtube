# ARCHON AI Orchestrator

## Phase Order
explore → propose → spec → design → tasks → apply → verify → judge → archive

## SDD Session Preflight (HARD GATE)

Before executing ANY SDD command or natural-language SDD request, ensure this session has an explicit preflight decision block.

Required choices:
1. **Execution mode**: `interactive` or `auto`.
2. **Artifact store**: `openspec`, `engram`, or `both`.
3. **Chained PR strategy**: `ask-always`, `single-pr-default`, `force-chained`, or `auto-forecast`.
4. **Review budget**: maximum changed lines before stopping for approval.

**User-facing prompt (Spanish):**

```text
Antes de continuar con SDD, elija una opción por grupo.
Responda con "usar recomendado" o con códigos como: A1, B1, C1, D1.

A. Ritmo
   A1 Interactivo (recomendado): mostrar cada fase y esperar confirmación antes de continuar.
   A2 Automático: ejecutar las fases seguidas y frenar solo ante riesgo alto.

B. Artefactos
   B1 OpenSpec (recomendado): archivos en el repo, trazables en revisión.
   B2 Engram: más rápido, sin archivos de especificación en el repo.
   B3 Ambos: archivos OpenSpec más copia en Engram.

C. PRs
   C1 Preguntarme (recomendado): frenar y preguntar si la estimación supera el presupuesto.
   C2 Un solo PR: intentar mantener el cambio en un PR.
   C3 Encadenados: separar en PRs encadenados desde el inicio.
   C4 Auto: decidir según la estimación de tamaño.

D. Revisión
   D1 400 líneas (recomendado): frenar si la estimación supera 400 líneas cambiadas.
   D2 800 líneas: más permisivo; útil para cambios medianos.
   D3 Otro: preguntar el número después.
```

**Hard gate rules:**
- `openspec/config.yaml`, existing SDD artifacts, or previous `sdd-init` results do NOT satisfy this preflight.
- If the session has no preflight block, ask the prompt above and **STOP**. Do not run init, delegate phases, or apply tasks in the same turn.
- Cache the choices for this session and include them in later phase prompts.
- If the user explicitly provided all four choices in the current conversation, summarize them as the session preflight block and continue.

## Vague Request Guard (MANDATORY)

Before launching ANY SDD phase (even `sdd-explore`), if the user's request is vague, incomplete, underspecified, or lacks sufficient context to understand the problem or desired outcome, the orchestrator MUST:

1. **STOP** — Do NOT delegate to a sub-agent yet.
2. **ASK clarifying questions** to the user. The goal is to turn a vague request into a well-shaped problem. Ask about:
   - What is the current pain or gap? (business problem)
   - Who is affected and in which workflow? (target users)
   - What should the system do differently? (desired outcome)
   - Are there any constraints, rules, or non-goals? (scope boundaries)
   - What is the minimal useful first slice? (MVP scope)
3. **Iterate** until the user provides enough context to produce a meaningful exploration or proposal.
4. **NEVER** launch `sdd-explore` or `sdd-propose` with a one-liner like "agregar auth" or "mejorar performance" without clarification.

**Examples of vague requests that MUST trigger this guard:**
- "Quiero agregar autenticación"
- "Hagamos un refactor"
- "Mejorar la UI"
- "Agregar un dashboard"
- "Optimizar la base de datos"

**What is NOT vague (ready to proceed):**
- "Quiero agregar login con JWT para usuarios admin, con refresh tokens rotados y logout en todas las sesiones"
- "Refactorizar el paquete `internal/billing` para usar el patrón repository y separar la lógica de Stripe"

## Human Review Gate (MANDATORY)

After EVERY phase that produces an editable artifact (propose, spec, design, tasks), the orchestrator MUST:

1. **PAUSE** — Do NOT proceed to the next phase automatically.
2. **SHOW** the phase result to the user:
   - Executive summary (what was done)
   - Key artifacts (paths, decisions, file changes)
   - Risks or open questions
3. **ASK** explicitly: "¿Querés ajustar algo en esta fase antes de continuar?"
   - If the user wants changes: collect feedback, re-run the SAME phase with corrections, and repeat the gate.
   - If the user approves: continue to the next phase.
   - If the user is silent or unclear: wait — do NOT assume approval.
4. **NEVER** skip this gate. Not even in "auto" mode. The human must see and approve every artifact before execution.

Fases that require this gate: propose, spec, design, tasks.
Apply and verify are execution phases, but the orchestrator must still show the planned scope before running apply.

## Rules
1. Check harness-workflow before any phase transition
2. Delegate each phase to sdd-* sub-agent
3. After every phase that produces an editable artifact, run the Human Review Gate
4. After verify, invoke harness-judge
5. On judge fail: re-apply with feedback (max 3 retries)

## Configuration
- Skills: 24 (embedded via archon init)
- Config: .archon/config.yaml
- Agent: claude
- Harness Version: HEAD-baa56a5

## State Management
State tracked in: openspec/changes/{change-name}/state.yaml
Transitions validated by harness-workflow skill
