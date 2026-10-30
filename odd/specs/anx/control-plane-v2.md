# ODD — ANX Control Plane v2

> **Estado:** propuesta para implementación, no implementación realizada.  
> **Repositorio:** [arturonavax/env](https://github.com/arturonavax/env)  
> **Base auditada:** `main@b71fdca00a54db27be1067c8cb6bb3912245a0da` (30-10-2026).  
> **Destino propuesto:** `odd/specs/anx/control-plane-v2.md`.  
> **Método:** Organic Driven Design (ODD), integrado con el workflow «Organic Driven Development» ya presente en el repositorio: especificación viva → tareas verticales → evidencias/decisiones → reajuste.  
> **Prioridad:** evolución incremental; no reescritura total.

## 1. Objetivo y límites

Convertir `anx` en un **control plane local, declarativo, offline-first, modular y extensible** para administrar entornos de desarrollo: software, dotfiles, perfiles, configuraciones semánticas, servicios, hacks de hardware, runtimes, herramientas externas y capacidades de comandos.

**Un solo motor y múltiples adaptadores de presentación:** CLI Cobra, TUI Bubble Tea v2 y, posteriormente, GUI de escritorio, app móvil y consola web. Las interfaces jamás replicarán resolución, autorización, planificación ni ejecución. Un cliente móvil/web podrá administrar *un host que ejecute anx*; no se exige portar el ejecutor a iOS/Android. Termux seguirá siendo un host independiente soportado.

**No objetivos iniciales:** nube obligatoria, microservicios, clúster distribuido, plugins Go `.so`, API pública expuesta a Internet, reescritura de todos los scripts, automatización privilegiada sin consentimiento ni introducir una GUI prematuramente.

## 2. Auditoría de partida: preservar y corregir

**Preservar y reutilizar:** `internal/domain`, `app`, `manifest`, `catalog`, `providers`, `configsync`, `execution`, `journal`, `state`, `snapshot`, `security`, `hacks`, `ops`, `tools`, `ai`, `platform`, `theme`, `cli`, `tui`; los manifiestos y dotfiles en `manifests/` y `files/`; instaladores, tests y ODD existentes.

| Evidencia verificada en código | Brecha / decisión ODD |
|---|---|
| `internal/app/install_service.go`: `Plan` construye pasos lineales; todos los paquetes tienen `IsExclusive: true`; incorpora sincronizaciones sin comprobar siempre convergencia previa. | Planificador universal con **DAG de acciones**, locks específicos, observación, diffs, costo/riesgo e idempotencia real. |
| `internal/execution/pool.go`: lotes contiguos más barreras `Exclusive`; compensaciones en orden inverso de finalización. | Scheduler por dependencias y conflictos; rollback en **orden inverso topológico**, con resultados parciales explícitos. |
| `internal/domain/plan.go`: `ExecutionPlan` contiene `Steps` mutables; ID depende de `time.Now()`. | Separar `PlanSpec` inmutable de `ExecutionStatus`; hash determinista del contenido, sin timestamps. |
| `internal/app/install_service.go`: `journalAppend` y `recordInstalled` ignoran errores de persistencia. | **Fail-closed** antes de mutaciones críticas; journal fiable, errores propagados y recovery probado. |
| `internal/state/store.go`: `state.json` versionado y persistido con rename; `journal.jsonl` por separado. | Mantener compatibilidad de lectura y migrar progresivamente a almacén transaccional por host (SQLite), conservando snapshots externos. |
| `internal/app/sync_service.go`: merge de tres vías existente; algunas lecturas y `post-sync` descartan errores. | Plan de sync unificado, clasificación de conflictos, reporting de errores y adaptadores semánticos que preserven comentarios/formato. |
| `internal/cli/bootstrap.go`: inicializa estado, snapshots, catálogo y servicios dentro de `cli`. | Extraer `internal/bootstrap`/composition root reutilizable por CLI/TUI/daemon/API; lazy initialization para comandos baratos. |
| `internal/tui/model.go`: modelo usa servicios y modelos internos concretos. | Proyección de `QueryService`, `CommandService` y eventos públicos; sin I/O ni reglas de negocio en UI. |
| `internal/tools/runner.go`: ejecuta binarios y builtins; no existe contrato de plugins externos versionado. | Separar **Tool** (ejecutable gestionado) de **Extension** (integración protocolizada en subproceso). |
| `internal/cli/update.go`: `Diff` devuelve `clean`, pero el recuento compara con `synced`. | Corregir clasificación; pruebas de regresión para estado limpio/drift/conflicto. |
| `go.mod`: Go 1.26.0; `odd/specs/anx/overview.md` anuncia Go 1.27+; `README` documenta `cmd/aitools` aunque el árbol contiene `cmd/anx`. | Fijar una única fuente de verdad de compatibilidad y generar documentación/ayudas. |
| No hay archivos `.github/workflows/` en el árbol de `main` auditado. | CI multi-OS, pruebas de integración y gates arquitectónicos obligatorios. |

**Alcance de la evidencia:** inspección estática del árbol GitHub y de sus módulos críticos; no se ha ejecutado `go test`, benchmarks ni una instalación real. Un comportamiento descrito en README o en `odd/` **no cuenta como terminado** sin prueba ejecutable. La investigación `Deepsearch_anx.md` es insumo de diseño, no prueba del estado de ejecución.

## 3. Arquitectura innegociable

```text
CLI Cobra  TUI Bubble Tea  GUI Desktop  Mobile  Web/REST
    \          |              |           |      /
     └─────────┴───── Adapters / Transport ─────┘
                        │
             Application API v1 (Go contracts)
             Queries · Commands · Jobs · Events
                        │
             ANX Core / Modular Monolith
        Resolver → Observer → Planner → Policy
           → Reconciler → Scheduler → Executor
              │          │          │
       Resource Drivers  State/Journal  Extension Host
              │          │          │
       OS/Package/FS   SQLite+CAS    stdio JSON-RPC

  opcional después: anx-agent ⇄ local socket/HTTP API v1
```

**Reglas de importación:**

- `domain` solo Go estándar y modelos puros; **nunca** Cobra, Bubble Tea, Viper, `os/exec`, HTTP ni dependencias UI.
- `application` depende de contratos/ports, no de adaptadores concretos ni presentation.
- `adapters` implementan ports y contienen filesystem, DB, SO, red, subprocess y proveedores.
- `frontend/*` usa application API/DTOs; no importa drivers, repositorios ni almacenamiento.
- `cmd/anx` y futuro `cmd/anx-agent` solo hacen composición, señales y salida.
- Evitar `pkg/` como SDK público prematuro. Versionar contratos de wire en `api/v1` al introducir red; mantener la implementación en `internal/`.
- Añadir gate automático de ciclos y límites de imports, además de `go vet`.

**Estructura objetivo — mapa lógico, migración gradual (sin mover archivos por estética):**

```text
cmd/anx/                 # CLI + TUI en proceso
cmd/anx-agent/           # FUTURO: servidor/daemon opt-in
internal/domain/         # resource, plan, policy, platform, capability, job
internal/application/    # use cases, query/command services, DTO mapping
internal/engine/         # observe, resolve, plan, reconcile, execute, recover
internal/resources/      # package, config, tool, service, hack, runtime, repo
internal/adapters/       # pkgmgr, fs, semantic-config, service, platform, network
internal/extension/      # discovery, protocol v1, permissions, lifecycle
internal/persistence/    # JSON legacy, SQLite, CAS blobs, journal, migrations
internal/bootstrap/      # dependency wiring + runtime identity
internal/frontend/       # cli, tui, json (+ desktop/web adapters when needed)
api/v1/                  # FUTURO: OpenAPI/JSON Schema + generated clients
manifests/               # bundled definitions, no Go edit for normal additions
odd/specs/anx/           # decisiones, contratos y requisitos vivos
odd/tasks/               # vertical slices, verificación, log de evidencias
```

No imponer una interfaz gigante a todas las clases: el núcleo expone contratos pequeños y compone capacidades opcionales.

## 4. Contratos de dominio y consistencia

**Tipos estables:** `ResourceID`, `ResourceKind`, `ResourceSpec`, `DesiredState`, `ObservedState`, `Drift`, `Capability`, `Plan`, `Action`, `PolicyDecision`, `Job`, `Event`, `Snapshot`, `HostFacts` y `Revision`.

**Lifecycle universal:** `Discover/Observe → ResolveDesired → Diff → Plan → Authorize → Apply → Verify → Record → (Compensate/Restore)`. `Rollback` no es obligatorio para un provider irreversible: declarar `Reversibility=full|compensating|none` y avisar antes de aprobar.

**Invariantes:**

1. `observe`, `status`, `diff`, `explain` y `plan` no modifican los recursos gestionados; se permiten cachés/diagnósticos internos solo si están documentados.
2. El plan es serializable, inmutable y reproducible: `planDigest = SHA256(canonical(desired + observations + facts + policy + actions))`; IDs de ejecución y timestamps se guardan aparte.
3. `Apply(planDigest, expectedHostRevision)` rechaza plans desactualizados; revalida precondiciones/archivos antes de escribir para evitar TOCTOU.
4. Cada acción declara `dependsOn`, `locks`, `preconditions`, `privileges`, `risk`, `timeout`, `verify` y compensación posible.
5. Máximo **un escritor por host**; parallelismo acotado entre acciones sin dependencias ni locks compartidos. Locks: `pkg:apt`, `file:<canonical>`, `service:<id>`, `privilege:root`, etc.
6. `dry-run` produce el mismo árbol de acciones que `apply`; sin descargas/ejecuciones ocultas. Si algo no puede verificarse sin mutar, reportar `unknown`.
7. El journal registra intención **antes** del efecto, resultado después, durable flush/commit donde proceda; errores de estado nunca se silencian. Reanudar exige reconciliar el estado real, no repetir ciegamente.
8. Todas las operaciones son cancelables, con resultados parciales, eventos secuenciados y código de salida estable.
9. `uninstall`/`cleanup` opera solo sobre recursos de propiedad demostrada; **nunca** elimina un paquete preexistente por detectarlo en PATH.

**Application API v1 (conceptual, no atada a HTTP):**

```go
// Pseudocontratos; ubicación y firmas definitivas mediante ADR.
type Queries interface {
    Inventory(context.Context, InventoryFilter) (Inventory, error)
    Observe(context.Context, []ResourceID) ([]ObservedState, error)
    Explain(context.Context, ResourceID) (Provenance, error)
    Job(context.Context, JobID) (JobSnapshot, error)
}
type Commands interface {
    BuildPlan(context.Context, PlanRequest) (Plan, error)
    Apply(context.Context, ApplyRequest) (JobID, error)
    Cancel(context.Context, JobID) error
    Restore(context.Context, RestoreRequest) (JobID, error)
}
type Events interface {
    Subscribe(context.Context, EventCursor) (<-chan Event, error)
}
```

Para ejecución remota futura: DTOs compatibles y versionados (`schemaVersion`, `hostID`, `revision`, `planDigest`, `jobID`, `eventSequence`); comandos con `idempotencyKey`, autorización y `expectedRevision` para evitar ejecución duplicada/replay.

## 5. Manifiestos, perfiles, hacks y procedencia

**Mantener YAML para autoría**, structs Go tipados y schemas versionados (`apiVersion`, `kind`, `metadata.id`, `spec`). Evaluar CUE como validador/políticas **después** de un ADR de coste/dependencias; no bloquear el MVP. Migrar manifiestos existentes con un lector de compatibilidad y `anx manifest migrate`; no romper overlays existentes.

**Clases nativas:** `Package`, `Config`, `Tool`, `Service`, `Hack`, `Runtime`, `Repository`, `Capability`, `SettingBundle`, `Profile`, `Extension`. Scripts quedan como escape hatch declarado con permisos, hash, origen y confirmación.

**Resolución determinista:** bundled → repo → drop-ins usuario → perfiles → host overlays → políticas explícitas; prioridad/conflictos fijados por contrato y detectados antes de ejecutar. Un `Profile` selecciona recursos y preferencias; no duplica recetas de instalación. Un `Hack` es un overlay condicional por facts (DMI, GPU, distro, DE, memoria) con riesgo y opt-in, no un script implícito.

```yaml
apiVersion: anx.dev/v1
kind: Config
metadata:
  id: terminal.ghostty
spec:
  source: files/ghostty/config
  target: ${XDG_CONFIG_HOME}/ghostty/config
  strategy: semantic-or-three-way
  requires: [package.ghostty]
  settings:
    background-opacity: 0.97
  overlays:
    - when: {hardware: dell-precision-m6800}
      patch: {background-opacity: 1.0}
```

*Ejemplo de esquema objetivo; **no** es sintaxis actualmente admitida.*

**`anx explain` es requisito transversal:** para cualquier valor/paquete/estrategia/capacidad mostrar origen, condiciones, prioridades, candidatos excluidos, política ganadora, dependencias y motivo del resultado.

**Configs:** file/directory/marker/symlink/3-way conservados; agregar adaptadores semánticos JSON, JSONC, YAML, TOML, INI, Lua y XML **por demanda**, preservando comentarios/orden/formato cuando el parser lo permita. Si no, usar merge de archivo con preview; jamás formatear destructivamente sin aprobación. Base/local/incoming, hashes y conflicto explícito. Permisos, owner, symlink, EOL y archivos binarios forman parte del diff.

**AI customizer:** extensión sobre `SettingBundle` + adapters de cada harness (Codex, Claude, OpenCode, Pi, etc.) para keybindings, MCP, skills, prompts y settings; detectar capacidades/formatos por cliente; no forzar propiedades no soportadas ni reemplazar ajustes ajenos.

## 6. Tools, capabilities, extensiones y procesos

Separar claramente:

- **Tool:** ejecutable externo administrado, versión/proveedor/instalación/actualización/ubicación/estado.
- **Capability:** intención abstracta (`files.list`, `search.text`, `diff.structured`) resuelta a backend y traducción de argumentos; soportar `anx ls`, `anx what ls`, `anx explain capability files.list` con fallbacks y explicación. Preservar `--` y códigos/salida/TTY del backend.
- **Extension:** proceso externo opcional que participa en `Discover`, `Observe`, `Plan`, `Apply`, `Verify` mediante protocolo **stdio JSON-RPC v1**, handshake, versión, capabilities, timeouts, límites y eventos. No Go plugins compartidos ni ejecución arbitraria de manifests sin consentimiento.

**Daemons y servicios:** modelo `Service` con adapters systemd(user/system), launchd, Windows Services/Tasks y Termux, detección de soporte; daemon `anx-agent` **opt-in**, no requisito para CLI/TUI. Reinicio/reload solo tras verificación y aprobación. Nunca instalar componentes privilegiados globales sin plan explícito.

## 7. Frontends y API futura

- **CLI:** Cobra/Viper finos; texto/JSON estable, stderr para logs, no ANSI en JSON, exit codes, `--offline`, `--dry-run`, `--yes`, `--profile`, completions; `update` = consultar/fetch metadatos autorizado, `upgrade` = ejecutar upgrades planificados, `sync` = reconciliar dotfiles, `self update` = binario, `tools update` = herramientas desacopladas. Soportar nombre personalizado del comando (`anx alias`, `--bin-name`, symlink/renombrado): separar `CanonicalAppID` de `InvokedAs`, mantener almacenamiento y protocolos estables, y generar ayuda/completions según nombre invocado.
- **TUI:** Bubble Tea v2/Lip Gloss v2; Tokyo Night predeterminado y temas configurables. Dashboard overview, packages, configs/diffs, tools/capabilities, hacks/services, updates/changelog, seguridad, historial/jobs; scroll, filtros, búsqueda, comandos con preview; responsive/SSH/Termux y degradación para TTY reducido. Consumo exclusivo de Query/Command/Event API.
- **Desktop GUI futura:** Tauri/Wails/Electron es una **decisión futura (ADR)**; conectará al mismo servicio local o biblioteca via adaptador estrecho. Nunca incrustar lógica de provisión en JS/UI.
- **Mobile futura:** cliente remoto con identidad de dispositivo, permisos por host, notificaciones de jobs; sin asumir ejecución de instaladores en teléfono. Android Termux usa host CLI distinto.
- **Web futura:** API HTTP local + OpenAPI v1, SSE/event cursor para progreso, recursos/jobs/planes, RBAC, cancelación, idempotency y auditoría. Por defecto solo loopback/Unix socket/Windows named pipe. Acceso remoto solo habilitado explícitamente con autenticación fuerte, TLS y controles CSRF/CORS/origen, tasa y sesión; preferir túnel autenticado por SSH/Tailscale, jamás endpoint root público.
- **Multihost futura:** controlador gestiona agentes *por host*; cada agente conserva autonomía, su estado y locks. No introducir servidor central en v1.

## 8. Persistencia, rollback y seguridad

Migrar gradualmente `state.json` + `journal.jsonl` a **SQLite por host** (WAL, versionado de esquema, transacciones, única fuente de verdad para jobs/ownership/plan references/event cursor), con blobs de snapshots y bases de merge en almacenamiento content-addressed (`sha256`) fuera de DB. Evaluar driver pure-Go, footprint y compatibilidad Windows/Termux mediante ADR. Estado antiguo se importa de forma idempotente con backup y posibilidad de reversión. Nunca eliminar migraciones anteriores sin garantía.

Pre-write snapshots verificables por SHA-256; política de retención por cuotas configurable (la actual es 50 MB), nunca borrar un restore point asociado a job incompleto. Recovery clasifica `committed`, `interrupted`, `partially-compensated`, `manual-intervention-required`; no prometer transacciones ACID sobre `apt`, `pacman` o scripts. Cada acción registra si es reversible y cómo.

**Seguridad:** confianza por origen/proveedor; hashes **y autenticidad** de artefactos (firma/provenance cuando sea viable, no solo checksum junto al artefacto); scripts remotos deshabilitados por defecto o marcados de alto riesgo, pin/digest y aprobación; rutas normalizadas y resistentes a escapes por symlink; ejecución por argv con límites y context; secrets mediante referencias a keyring/gestor externo, sin serializar valores en manifests, logs, snapshots ni telemetría. `anx self update`: download → verify → staging → self-test → reemplazo seguro (en Windows, helper/reinicio) → recovery. Telemetría **off por defecto**, preview local y opt-in revocable; sin rutas, hostnames, argumentos ni contenido de archivos.

## 9. Migración ODD por vertical slices

| Fase | Entregable funcional | Gate de salida |
|---|---|---|
| **P0 — Baseline** | Congelar contrato actual, inventariar comandos/manifiestos/archivos, ADRs, matriz features × SO, pruebas de regresión, corregir `update`/errores descartados y docs desactualizadas. | Baseline reproducible; no pérdida de funciones; CI Linux/macOS/Windows; gofmt/vet/race; simulación de sync sin tocar `$HOME`. |
| **P1 — Core contracts** | `ResourceSpec/Observed/Plan/Job/Event`, ports pequeños, composition root independiente de CLI, `explain`, validación schemas y `Plan` canónico; adaptadores de compatibilidad. | CLI y TUI consumen mismos casos de uso; golden JSON/Plan, límites de import verificables. |
| **P2 — Engine fiable** | Planner DAG + locks + policies + scheduler; ejecución/verificación; journal durable; replay/resume, compensación y ownership. | Fallos inyectados, cancelación, crash/restart, dos procesos, idempotencia, carreras y garantías de no pérdida. |
| **P3 — Configs & overlays** | Merge 3-way fiable, provenance, profiles/hacks, patching tipado, `SettingBundle` y primer adapter semántico JSONC/TOML. | Conflictos y comentarios preservados; preview=apply; rollback probado. |
| **P4 — Ecosistema** | Capabilities/wrappers, tool/provider registry, plugins JSON-RPC v1, servicios/daemons declarativos, AI customizer. | Plugins incompatibles/sospechosos aislados; versiones/fallbacks, privilegios y outputs probados. |
| **P5 — Persistencia y lifecycle** | Migración SQLite + CAS, snapshot/restore, jobs/event cursor, update/upgrade/sync/self-update firmado, releases reproducibles. | Migración/restauración y reinicio interrumpido; releases verificados; ninguna regresión de commands. |
| **P6 — Plataforma & UI** | Dashboard completo/responsive, themes, accesibilidad, rutas multiplataforma, Termux, SSH, e2e reales. | Pruebas PTY/golden, VM Windows/macOS, Alpine/Ubuntu/Arch/Fedora y Termux según disponibilidad real. |
| **P7 — API/clients (cuando haya demanda)** | `anx-agent` local, OpenAPI v1 y SSE, SDK generado, GUI/web/móvil opcionales. | Tests de paridad local/remoto, authZ, idempotency/replay, aislamiento por host, API backwards compatibility. |

**Entrega por fase:** `Specs` (resultados/constraints, NFR, ADRs), `Tasks` (cortes verticales independientes), `Log` (commits, tests, benchmarks, deuda, decisiones). PRs pequeños; si cambia el diseño, actualizar spec antes del código afectado. No ejecutar P7 como requisito para terminar P1–P6.

## 10. Requisitos de aceptación verificables

| ID | Criterio de aceptación |
|---|---|
| **AC-01** | Un manifiesto válido se añade sin editar/recompilar Go; overlays inválidos producen errores de campo y origen. |
| **AC-02** | Con estado convergente, `anx plan` devuelve cero acciones; `anx apply` no escribe ningún recurso. |
| **AC-03** | Dos planners con iguales entradas producen el mismo `planDigest`, independientemente de hora/orden del mapa. |
| **AC-04** | Cambiar el destino tras `plan` causa rechazo por precondición; un plan vencido no se aplica silenciosamente. |
| **AC-05** | Las acciones independientes pueden concurrir; `apt`/misma ruta/servicio nunca se ejecutan en paralelo. |
| **AC-06** | Error de journal/state bloquea una mutación riesgosa; crash a mitad de operación permite recovery seguro y auditable. |
| **AC-07** | Config local y upstream modificados: `diff` muestra 3-way y conflicto sin pisar datos; preservar comentarios en adaptadores soportados. |
| **AC-08** | Perfil/hack explica *por qué* un setting gana; hack privilegiado nunca corre implícitamente. |
| **AC-09** | `anx ls` selecciona backend compatible y preserva args/stdout/stderr/exit code; `anx what ls` explica fallback. |
| **AC-10** | Plugin externo desactualizado/fallido no bloquea el núcleo ni puede adquirir privilegios por herencia implícita. |
| **AC-11** | `anx update` no interpreta `clean` como drift; update, upgrade, sync y self-update tienen tests de efectos. |
| **AC-12** | CLI y TUI muestran los mismos planes, estados y resultados; JSON estable, sin ANSI; sin TTY funciona. |
| **AC-13** | Instalación/desinstalación conserva recursos preexistentes y no borra dotfiles ajenos. |
| **AC-14** | Estado legacy migra y puede restaurarse; snapshots se verifican y no se purgan si respaldan un job abierto. |
| **AC-15** | Build/test/lint/race sobre plataformas soportadas; matriz de compatibilidad basada en resultados, no promesas. |
| **AC-16** | El servidor opcional arranca en local sin exposición pública; autenticación y autorización probadas antes de habilitar acceso remoto. |
| **AC-17** | Telemetría nula por defecto, red opcional, sin datos sensibles; modo `--offline` probado con red inaccesible. |
| **AC-18** | Benchmarks reproducibles de cold start, plan, inventario, memoria, 3-way y overhead de wrappers; sin regresión no justificada sobre baseline P0. |
| **AC-19** | Invocar el mismo binario con otro nombre conserva estado e identidad de aplicación; la ayuda/ejemplos/completions muestran el nombre elegido. |

**Definition of Done por slice:** código + test determinista (incluye casos de fallo) + revisión de seguridad + documentación CLI/manifest + contrato JSON + benchmark si aplica + entrada `Log` + compatibilidad con usuarios actuales. Una pantalla vacía, stub o test omitido **no** equivale a funcionalidad terminada.

## 11. Instrucción al agente de código

**Implementa incrementalmente este spec en `arturonavax/env`; no reemplaces los 15 documentos ODD existentes ni borres funcionalidades.** Antes de cada fase: leer `AGENTS.md`, `odd/tasks/master-spec.md`, `odd/tasks/anx-platform-overhaul.md`, los docs `odd/specs/anx/` y las skills Go/Cobra/Bubble Tea pertinentes; comparar lo declarado con código/ejecución y abrir ADR cuando haya conflicto. Crear `odd/tasks/anx-control-plane-v2.md` con secciones **Specs / Tasks / Log**, dependencias, responsables, PRs, evidencias y estado. Empezar exclusivamente por **P0 + P1**, priorizando los bugs verificables y la extracción de contratos; no crear GUI, web, daemon ni DB nueva antes de validar el núcleo. Mantener flags/comandos y migración retrocompatibles. No afirmar tests, plataformas ni funciones como aprobados sin ejecutarlos. Para cada slice entregar diff, comandos ejecutados, resultados, riesgos y criterios AC cubiertos.

---

**Referencias de auditoría:** [README](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/README.md) · [AGENTS](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/AGENTS.md) · [ODD original](https://github.com/arturonavax/env/tree/b71fdca00a54db27be1067c8cb6bb3912245a0da/odd) · [Plan actual](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/internal/app/install_service.go) · [Scheduler](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/internal/execution/pool.go) · [Estado](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/internal/state/store.go) · [Update](https://github.com/arturonavax/env/blob/b71fdca00a54db27be1067c8cb6bb3912245a0da/internal/cli/update.go).
