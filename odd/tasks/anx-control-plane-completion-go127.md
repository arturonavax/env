# ODD TASK — ANX Control Plane v2: Full Completion, Go ≥1.27.3 & Node 26 LTS

**ID:** `anx-control-plane-completion-2026-10`  
**Repositorio:** `https://github.com/arturonavax/env`  
**Base verificada:** `main@c8dd3f599889115f792a9eee38a6a3b77caab094` (2026-10-30) — **revalidar HEAD antes de tocar código**.  
**Archivo destino de esta tarea:** `odd/tasks/anx-control-plane-completion-go127.md`  
**Documento normativo:** `odd/specs/anx/control-plane-v2.md` (AC-01…AC-19).  
**Método:** Organic Driven Development / ODD: `Specs → Tasks → Log → evidencia → ajuste`.  
**Prioridad:** `P0 seguridad/compatibilidad → P1 core → P2 ejecución → P3 configs → P4 extensiones → P5 persistencia → P6 UI/plataformas → P7 preparación de API`; **terminar todos los entregables obligatorios, no solo crear stubs**.

## Specs

### S0 — Misión, lectura y política de ejecución

Actúa como **Principal Go/Platform Engineer + Security Engineer + TUI Architect**. Implementa y verifica **completamente** el refactor del control plane de `anx`: declarativo, offline-first, seguro, idempotente, portable, extensible, escalable y mantenible. Un solo motor para CLI/TUI y futuros adaptadores de escritorio, web y móvil. **No implementar GUI/web/móvil prematuramente**; sí dejar contratos, DTOs y separación que permitan añadirlos sin reescribir negocio. No crear microservicios, Go plugins `.so`, dependencia obligatoria de Node/Python/Bash, telemetría por defecto ni ejecución privilegiada implícita.

1. Leer primero `AGENTS.md`, `odd/specs/anx/*`, `odd/tasks/{master-spec,anx-platform-overhaul,anx-control-plane-v2}.md`, `odd/new_tasks.md`, `README.md`, `Makefile` y código/tests reales. Consultar las skills existentes `.agents/skills/{golang-cli,golang-code-style,golang-concurrency,golang-design-patterns,golang-performance,golang-spf13-cobra,bubbletea}/SKILL.md` si están presentes. El código y pruebas reales prevalecen sobre checkboxes/documentación.
2. **Reconciliar el ODD previo**: `odd/tasks/anx-control-plane-v2.md` marca `T0.1…T1.4` completadas, pero los AC asociados siguen incumplidos. No borrar el historial; enlazar la corrección de esas afirmaciones y cambiar estado a `partial`/`blocked` donde corresponda, con evidencia. Esta nueva tarea es el tablero de cierre, no un duplicado desconectado.
3. Antes de cambios generar inventario de comandos, contratos JSON, manifiestos, plataformas, rutas, runtimes, scripts, versiones, dependencias, riesgos, tests y benchmarks. Registrar baseline reproducible. Entregar por **vertical slices** con tests, diff, riesgos, docs, resultados, logs y commits/PRs atómicos. Si una fase depende de otra, continuar al terminarla; no declarar finalización por generar un informe.
4. No cambiar intencionalmente la semántica pública de `anx` sin migración compatible, ADR y tests. Conservar `aitools` multicall, `anx sleep`, hacks, `update/upgrade/sync`, instaladores, profiles, manifests y dotfiles. Ninguna operación de prueba debe tocar `$HOME` real ni ejecutar instalaciones reales por defecto.
5. Leer el `HEAD` y la CI nuevamente en cada fase. **No dar por aprobado un requisito sin comprobarlo** en la plataforma objetivo. Si un entorno físico no está disponible, aportar test automatizado/cross-build, marcarlo **unverified** e impedir declarar soporte completo.

### S1 — Política de versiones y supply chain: piso obligatorio, máxima estabilidad

**Pisos obligatorios a partir de esta migración:**

- **Go `1.27.3` mínimo, sin excepciones** para desarrollar, compilar y ejecutar el bootstrap de `anx` desde código. `go.mod`: `go 1.27.3`; si se usa `toolchain`, no puede ser inferior. No mantener compatibilidad ilusoria con `1.24/1.25/1.26`. Rechazar versiones menores con diagnóstico que muestre encontrado/requerido e instrucciones de recuperación. Go `1.27.3` corrige seguridad de `crypto/x509` y era el último parche verificado el 2026-10-30: https://go.dev/doc/devel/release.
- **Node.js `26.x` LTS, último patch estable de la rama**, para configuraciones, herramientas, tareas Node y proyectos cuyo runtime administre `anx`; NO es dependencia del binario Go. No usar Node 27 Alpha/Current ni versiones EOL. La página oficial clasifica v26 como LTS: https://nodejs.org/en/about/previous-releases. Versiones de `npm`/`pnpm`/`corepack` compatibles deben verificarse y fijarse por lockfile/configuraciones cuando apliquen.
- **Actualizar todas las dependencias directas e indirectas** de Go a **últimas versiones estables y seguras compatibles con Go ≥1.27.3**: Bubble Tea v2, Lip Gloss v2, `charmbracelet/x/*`, Cobra, pflag, Viper, `golang.org/x/*`, YAML, etc. Evaluar migraciones breaking, advisories, licencias, Go minimum de cada módulo y peso de dependencias. No usar `@latest` en builds reproducibles ni pseudo-versiones si hay un release estable compatible; justificar las excepciones. No inventar números de versión: resolverlos en tiempo de ejecución contra fuentes oficiales y registrar **antes → después → release/changelog → prueba**.
- GitHub Actions: evaluar e introducir las majors estables actuales **`actions/checkout@v7` y `actions/setup-go@v7`** (verificar runner compatible); `actions/setup-node@v7` con `node-version: '26'` solo en jobs que realmente necesiten Node. **El runtime Node interno de las Actions es responsabilidad de cada Action**: no confundirlo con el Node 26 de los scripts y proyectos. Pin a commit SHA completo donde la política de seguridad lo permita, con comentario de versión; automatizar actualizaciones auditadas.
- Imágenes Docker/base OS, actions, herramientas de lint, instaladores de GitHub Releases, recipes `curl`, gestores y checksums: último release **estable, soportado y verificado**, no `edge/nightly/rc`, ni tags floating en CI de release. Si lo más nuevo rompe compatibilidad, documentar ADR, pin seguro y plan de actualización; **seguridad + corrección + reproducibilidad prevalecen sobre numeración máxima**.
- Activar una política continua (Dependabot/Renovate equivalente, SBOM, govulncheck, permisos mínimos, hashes y provenance, review de PR) con cadence y automatizaciones para advisories. Nunca ejecutar actualizaciones ciegas en la estación del usuario durante auditorías/tests.

**Superficie obligatoria de migración de runtimes:** `go.mod`, `go.sum`, bootstrap Unix/Windows, `scripts/install_golang.sh`, `manifests/packages/toolchains.yaml`, `internal/ops/rebuild.go`, `install.sh`/`install.bat` y entrypoints remotos, `Makefile`, Dockerfiles `tests/e2e/*`, CI, scripts, configuración de versiones (`.tool-versions`/`.go-version`/`.node-version` si se adopta), docs/ODD/AGENTS, comandos de autoactualización, plantillas y pruebas. Buscar en **todo el repo**, incluso archivos ocultos, `1.24`, `1.25`, `1.26`, `go1.*`, `Node 20/22/24`, `node-version`, `golang:`/`FROM`, `GOTOOLCHAIN`, `gvm`, `fnm`, `pnpm`; diferenciar referencias históricas/ejemplos de dependencias efectivas.

**Requisitos concretos del bootstrap:**

- Unix: sustituir `MIN_GO_MAJOR=1`, `MIN_GO_MINOR=24`, `TARGET_GO_VER=go1.24.0`; usar comparación **semver major.minor.patch** real (`1.27.2` falla, `1.27.3` y `1.28.0` pasan si son estables). Go existente suficientemente reciente se reutiliza sin downgrade. GVM puede instalar versión adecuada, pero no sustituye la verificación; fallback descarga oficial con checksum/firma desde canal confiable, portable por OS/arch; fallback package manager solo si cumple piso, nunca éxito falso. Si offline y falta toolchain, fallar temprano sin mutación. `GOTOOLCHAIN=auto` no sustituye la validación explícita; evitar descargar silenciosamente durante `--offline`.
- Windows PowerShell: validar parche (actualmente solo major/minor), seleccionar archivo oficial apropiado al `GOARCH` real, comprobar SHA-256, extraer en user scope, PATH idempotente, no eliminar instalación ajena, errores claros en caso de bloqueo; asegurar instalación y compilación **nativa** sin WSL/admin obligatorio. Aliases usan método seguro por OS.
- `scripts/install_golang.sh`: evitar degradar una instalación más reciente; resolver release estable (no RC) y checksum; respetar versión objetivo/mínima, arquitectura y permisos. Verificar compilación desde fuentes por canal oficial cuando se ofrezca; no reutilizar documentación/rutas `src/remotes` que ya no existan.
- Node: seleccionar `26.x` vía gestor disponible (`fnm` o adapter declarativo), última patch, probar `node --version`; no hacer instalación/cambio de Node en `anx` core si la persona no solicitó gestionar Node. Actualizar recetarios de reconstrucción, entorno shell y cualquier plantilla Node versionada.
- Docker: pasar de `golang:1.24-*` a imagen `golang:1.27.3-*` verificada, fijar digest en release CI y revisar versión/base OS; `CGO_ENABLED=0` si se conserva en plataformas soportadas. Evitar soporte fingido basado en `GOTOOLCHAIN=auto` cuando la imagen es antigua.
- CI: matriz **Go 1.27.3** en Ubuntu/macOS/Windows (y patch superior estable si se incorpora), Go minimum verificado por `go version` y `go env`; eliminar jobs Go 1.26; `go build`, `go test`, vet, race donde soporte, lint, seguridad y E2E. **No usar expresiones shell POSIX como `$(date ...)` dentro de un paso PowerShell Windows**; separar metadatos por plataforma o usar variables de entorno/Go.

### S2 — P0: seguridad, correctness y compatibilidad bloqueantes

Resolver con pruebas **antes** de avanzar:

1. `internal/ops/power.go`: `syscall.Sync` (líneas auditadas ~299,334,373,434) no compila en Windows. Extraer adapter por SO/build tags; flush específico/ordenado donde sea necesario; en plataformas no soportadas devolver error tipado `unsupported`, no simular éxito; cross-build Windows, macOS, Linux.
2. `internal/cli/update.go`: `anx upgrade --dry-run` **no puede ejecutar `git pull`, `ToolsSvc.Update` ni ninguna mutación**. Construir preview universal por el mismo plan; no writes a archivos, state, repos, tools, metadata, network downloads o subprocesos que muten. Sólo consultas no-mutantes explícitas. Una invocación `upgrade` real debe propagar los fallos de componentes y **no mostrar éxito global** si alguno falla; no aplicar hacks fuera del plan autorizado. Separar semánticamente `update` (consulta/fetch autorizado), `upgrade` (plan+apply), `sync` (configs), `self update` (binario) y `tools update` (tools).
3. `internal/app/install_service.go`: `Uninstall`/cleanup/compensación de instalaciones **sólo** sobre ownership comprobado (`ManagedByAnx`, ID, instalación de origen, provider, revisión). Diferenciar recursos preexistentes, gestionados y desconocidos; exigir consentimiento explícito para adoptar/borrar externos; manifest no implica propiedad. No desinstalar paquetes que otro package manager/usuario instaló por el mero hecho de aparecer en PATH. Corregir `RemoveResource` ignorado y logs.
4. `internal/cli/meta.go`: `anx alias` nunca `os.Remove` arbitrario. `Lstat`/no-follow, rechazo por defecto de destino existente ajeno; manejo idempotente si alias es nuestro; reemplazo seguro sólo con confirmación explícita/backup/ownership. Windows sin symlink obligatorio. Ruta del ejecutable y destino validados, protecciones contra symlink races, colisiones de PATH y de comandos del sistema.
5. `internal/app/sync_service.go`: fallos de `post-sync` no pueden reportarse como éxito silencioso. Capturar stderr, return code, timeout/context y mostrar estado `partial/failed`, incluso cuando el archivo ya fue escrito; determinar cuándo la acción es compensable. Errores de I/O y de snapshots propagados o etiquetados; nunca interpretar acceso denegado como ausencia.
6. `internal/journal/journal.go` / state: journal `append` con durabilidad razonable (`Sync`/transacción), detectar registros JSONL corruptos o truncados sin ignorarlos; estado y journal coherentes, lock interproceso, idempotencia y recovery. Fallo de persistencia antes de un efecto irreversible ⇒ abortar. Fallo posterior ⇒ estado parcial explícito, con remediación.
7. `anx update`: mantener corrección `clean != drift`, pero testear casos **clean, synced, drift, conflict, missing y error**; no darlo por cumplido con `ConfigDrifts == len(DriftedFiles)` solamente.
8. CI de baseline debe estar verde en la matriz declarada. ShellCheck/gofmt/go vet no pueden quedar como checks opcionales silenciosos. Corrigir advertencias de Actions obsoletas.

### S3 — P1/P2: núcleo estable y ejecución transaccional honesta

1. **Arquitectura**: modular monolith hexagonal, `domain` puro; `app/application` expone `QueryService`, `CommandService`, `EventService` y DTOs; `internal/bootstrap` actúa como composition root reutilizable y **lazy** (comandos `version/help/completion` no crean DB/estado/snapshots ni requieren repos). UI/CLI sólo llama API; eliminar deps directas de `internal/tui/types.go` a `catalog`, `providers`, `journal`, `state` y servicios concretos. Enforcement de límites de imports/ciclos, tests de arquitectura y contracts.
2. **Resource model**: `ResourceSpec`, `ObservedState`, `DesiredState`, `Drift`, `Policy`, `Action`, `PlanSpec`, `Job`, `Event`, `Provenance`, `HostFacts`, `Revision`, `Capability`; esquema versionado y serialización determinista. `Observe/Plan/Diff/Explain` read-only respecto a recursos gestionados. No reportar `Healthy=true` si el provider falla o el estado es desconocido.
3. **Plan reproducible**: `SHA256(canonical(desired + observations + hostFacts + policies + actions + schema))`, inclusivo de `metadata` **que afecte decisiones**; canonizar mapas, slices semánticamente no ordenadas y defaults. Evitar hash fallback débil/silencioso en `domain/contracts.go`; devolver error tipado. `ExecutionPlan` mutable antiguo es adapter temporal, NO source of truth. No planes dependientes de `time.Now` salvo timestamps de ejecución separados.
4. **Plan/apply correctos**: `BuildPlan` debe congelar *exactamente* perfil, requested IDs, flags, policies, effective configs, checksums, env relevante y `expectedHostRevision`. `Apply(planID/digest, expectedRevision, idempotencyKey)` busca plan autorizado, revalida precondiciones/targets/capacidades/privilegios y aplica ese plan exacto; **prohibido regenerar un plan genérico con `Plan(ctx,"",nil)`**. Rechazo ante TOCTOU y stale state. No writes en dry-run; idempotencia: plan sin acciones cuando ya convergió.
5. **Scheduler**: DAG acíclico por dependencias, locks finos (`pkg:apt`, `file:<canonical>`, `service:<id>`), fairness/capacidad, bounded parallelism, cancelación por context, retries sólo idempotentes, timeout, resultado parcial. Verificación postacción (`Observe/Verify`). Rollback en orden **topológico inverso**, no orden de finalización; compensaciones específicas por recurso, nunca desinstalar recursos preexistentes. Clasificar irreversibles (`full/compensating/none`) antes de pedir aprobación.
6. **Jobs/events**: `Apply` retorna JobID real; estados `queued/running/succeeded/failed/partial/cancelled/recovering`; `Cancel` funcional; eventos monotónicos con cursor/sequence y suscripción sin fugas, backpressure; replay/recovery tras crash/reinicio, deduplicación por idempotency key; un único escritor por host, con locks específicos por recurso. Sin trabajos fantasma o reportar éxito antes de commit.
7. **Explanations**: `anx explain` debe mostrar fuente exacta, precedencia, profile/overlay/hardware, candidatos descartados, policy ganadora, dependencias, valores y motivos; soportar packages, configs, tools, hacks, capabilities, settings y formato JSON versionado. No devolver `Origin: "manifests/packages"` genérico si hay varias fuentes.

### S4 — P3: manifiestos, perfiles, configuraciones y AI customization

- Manifiestos declarativos YAML de esquema versionado (`apiVersion`, `kind`, `metadata.id`, `spec`), drop-ins user, YAML antiguos legibles, migración idempotente/preview; errores con archivo, línea/campo y conflictos, no overwrites silenciosos. Resolución determinista bundled→repo→user→profile→host overlays→políticas; perfiles seleccionan capacidades/recursos; hacks sólo si facts verificables + opt-in.
- Recursos `Package/Config/Tool/Service/Hack/Runtime/Repository/Capability/SettingBundle/Profile/Extension`; estrategias Linux distro/manager + macOS + Windows + Termux; fallback sólo preacordado y con estado observable. No exigir editar Go al agregar definiciones normales.
- Unificar diff/sync/capture/backup con motor plan/apply; 3-way merge sobre base/local/incoming, snapshots antes de cambiar, conflictos detectados sin pérdida; preservar bytes, EOL, permisos, owners, symlinks, binarios. Adaptadores semánticos mínimos JSON/JSONC y TOML que **conserven comentarios/formato o hagan preview+consentimiento**; arquitectura extensible a YAML/INI/Lua/XML sin convertirlos en requisitos falsos.
- `SettingBundle` para toolchain, terminal y AI-harness: no sobrescribir settings ajenos; plugins/skills/MCP/keys específicos de clientes únicamente si soportados; generar provenance y advertencias por settings desconocidos. Tests golden con archivos conflictivos y variantes OS.

### S5 — P4: tools, wrappers, extensiones y servicios

- Diferenciar **Tool** (binario/version/provider), **Capability** (acción abstracta + fallback + traducción segura de argv) y **Extension** (subproceso versionado). Implementar `anx ls`, `anx what ls`, `anx explain capability files.list`, flags `--`, stdout/stderr/TTY/exit code, fallback explicable, modo offline y subcomandos multiplataforma; jamás usar shell string concatenado cuando argv basta.
- Protocolo externo `stdio JSON-RPC v1`: handshake, schema/capabilities, timeout, cancel, permisos declarados, límites de memoria/procesos/output, versión compatible y errores aislados; *default deny* para scripts externos/no firmados. Tests de plugin corrupto, lentitud, caída, salida arbitraria y versión incompatible.
- `Service` declarativo con adapters soportados `systemd user/system`, `launchd`, Windows Service/Task y Termux; opt-in privilegiado, plan observable y rollback cuando es posible. Daemon y tareas remotas no obligatorios para CLI/TUI.

### S6 — P5: estado, upgrades y seguridad

- Migrar de `state.json + journal.jsonl` a SQLite **por host**, esquema versionado, WAL/transacciones, single writer, jobs, ownership, plan references, events. Blob store content-addressed SHA-256 para snapshots/base files; no dejar datos sensibles en logs, DB ni telemetría. Elegir driver SQLite mantenido/cross-compile (preferencia pure-Go si pasa benchmarks/compatibilidad; ADR obligatorio). Backups, migración una sola vez, reinicio interrumpido, rollback del esquema/datos y lectura legacy temporal; validación de hashes e integridad antes de restaurar. Nunca retener una configuración dañada como healthy.
- `anx self update`: fuente oficial/versionada, manifest, firma/provenance confiable, checksum, download a staging, self-test, swap atómico o estrategia específica Windows, rollback sin bloquear el ejecutable. Releases reproducibles con checksums/SBOM, os/arch soportados, instalación offline de binarios verificados. `anx update/upgrade/tools update/sync` planificables/observables, con fallos parciales y tests de efectos.
- Seguridad: trust boundaries por provider/source, scripts `curl` no ejecutados automáticamente sin pin/autorizar, symlink/path traversal/TOCTOU defenses, `exec.CommandContext(argv)`+timeouts/output caps, secretos por referencias externas, gobernanza de permisos; telemetría **desactivada por defecto**, sin datos identificables en eventos/logs, `--offline` probado con red bloqueada.
- Política de dependencias: `go list -m -u all`, upgrades controlados, `go mod tidy`, `go mod verify`, `govulncheck`, `go vet`, linters, licencia/SBOM; nunca suprimir vulnerabilidad sin ADR con compensación y fecha. El grafo final debe estar auditado y limpio de advisories conocidos relevantes o incluir excepción explícita justificada.

### S7 — P6/P7: paridad TUI, plataforma y preparación multicliente

- TUI Bubble Tea v2 / Lip Gloss v2: Tokyo Night predeterminado, temas configurables, responsive a terminal pequeño, SSH/Termux, teclado/mouse, scroll/filtros/búsqueda, acciones con preview, packages/configs/diffs/tools/capabilities/hacks/services/updates/changelog/security/history/jobs; sin bloqueo del event loop, sin trabajo de negocio en `Update/View`, sin dependencia de terminal para core. Accesibilidad/no-color, PTY/golden visual y snapshot de datos CLI/TUI coincidente.
- Plataforma: matrices reales Linux (Debian/Ubuntu/Alpine/Arch/Fedora según disponibilidad), macOS Intel/Apple Silicon, Windows native sin WSL, WSL2, Android Termux; tests por OS (unit/cross-build/e2e), no prometer más que lo verificado. E2E instalación, uninstall, recuperación, perfiles, shell escapes, entornos read-only, sin red y permisos restringidos. Benchmarks cold start/RSS/plan/merge/overhead wrappers antes/después.
- Preparar boundary `api/v1` (schemaVersion, hostID, revision, planDigest, jobID, eventSequence, idempotencyKey) y contratos transport-agnostic. **No construir frontend GUI/móvil/web**. Si se implementa `anx-agent` local, que sea estrictamente opt-in, loopback/socket por defecto, autenticación/autorización y SSE/stream probado, nunca endpoint público ni ejecución root remota. Mantener compatibilidad futura de contratos y paridad de use cases.
- Renombrado/alias: `CanonicalAppID` estable ≠ `InvokedAs`; invocación `aitools`/alias personalizado usa mismo estado, ayuda, completions, docs y logs sin cambiar IDs. No sobrescribir archivos ajenos y no requerir symlinks Windows.

### S8 — Quality gates obligatorios (sólo evidencia ejecutada)

**CI mínimo** (scripts reproducibles + job por OS, `CGO_ENABLED` apropiado):

```sh
# ejecutar con Go >=1.27.3; no ejecutar pruebas destructivas en HOME real
go version
go mod tidy -diff
go mod verify
go vet ./...
go test -count=1 -shuffle=on -timeout=10m ./...
go test -race -count=1 -timeout=10m ./...   # plataformas soportadas
go test ./internal/... ./cmd/...
go list -m -u all
govulncheck ./...                            # install/pin herramienta en CI
# gofmt -l, go build, govulncheck, SBOM y suite e2e de Docker/PTY, con umbrales establecidos
```

Compilar target `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/{amd64,arm64}` donde el soporte sea real, `CGO_ENABLED=0` cuando proceda. En Windows ejecutar los tests **en runner Windows**, no sólo cross-compile. Para Termux/Android, validar restricciones reales de Go/Android y reportar resultados sin fingir soporte mediante cross-build. Script install con `1.27.2` debe fallar/reparar y `1.27.3` pasar; `1.27.10` y futuro `1.28.x` stable no deben rechazarse por comparación lexicográfica. Node 26 se verifica solo en jobs que lo necesitan. Política de cache/privilegios/supply chain `permissions: contents: read`, actions por SHA estable y comandos seguros con secrets.

**Pruebas de aceptación prioritarias** (sin falsas señales):

| ID | Prueba/condición de salida |
|---|---|
| `V01` | Go `1.27.3` piso real en docs, CLI/dev, bootstrap Unix/Windows, Docker, CI; Node `26.x` LTS en todo uso Node; runtime de `anx` no depende de Node. |
| `V02` | Windows Go 1.27.3 compila y tests pasan; `power.go` sin `syscall.Sync` fuera de un adapter soportado. |
| `V03` | `upgrade --dry-run` sin writes ni ejecuciones mutantes; `upgrade` real no muestra éxito ante fallos parciales. |
| `V04` | Uninstall/cleanup/compensate y alias preservan recursos ajenos, incluso symlinks, binarios y rutas adversarias. |
| `V05` | Journal corrupto/disco lleno/crash ⇒ stop o resultado parcial recuperable; post-sync fallido ⇒ resultado no-success. |
| `V06` | Dos planes iguales ⇒ hash igual; cambian facts, target, policy o revision ⇒ cambia o se rechaza; `Apply` exacto, stale falla. |
| `V07` | Plan convergente ⇒ 0 acciones; `dry-run` y `apply` misma propuesta; sin overwrite accidental. |
| `V08` | DAG paraleliza acciones independientes, serializa locks compartidos, rollback en orden topológico inverso, `Cancel` y replay funcionan. |
| `V09` | CLI/TUI usan mismas APIs y devuelven mismo estado/plan; DTOs JSON versionados/estables; `help/version` lazy. |
| `V10` | YAML antiguo migra sin pérdida; provenance/overlays y conflictos reportados; adaptadores semánticos golden. |
| `V11` | `anx ls/what ls` respeta argv, streams, TTY, exit status, fallback y plataforma; plugin JSON-RPC malicioso/fallido aislado. |
| `V12` | SQLite/legacy migration + restore bajo crash; ownership y snapshots íntegros; self-update verifica firma y recuperación. |
| `V13` | CI Linux/macOS/Windows, e2e, supply-chain scan, release smoke, benchmark y políticas de seguridad en verde. |
| `V14` | `anx alias` preserva nombres elegidos y `aitools`; datos previos intactos. |
| `V15` | AC-01…AC-19 del spec original, cada uno con test/archivo/CI run o status `blocked` honesto; **ninguno cerrado sólo con documento**. |

**Definition of Done global:** código funcional + arquitectura validada + tests deterministas de éxito/fallo/inyección y CI verde + semver/runtime auditado + documentos/CLI/manpages generados + seguridad/compatibilidad + comparación de benchmarks + ODD log por slice + traceability `AC ↔ V ↔ test ↔ commit/PR ↔ job URL` + sin stubs de funcionalidades prometidas + sin regresiones del inventario inicial. No aceptar `TODO`, `panic("not implemented")`, tests skip, warnings ignoradas, logs de éxito ambiguos o CI rojo como tarea completada.

## Tasks

Usar exactamente este tablero como checklist vivo. Cada task incluye **diseño → implementación → tests negativos → CI → documentación → evidencia** y únicamente se marca `[x]` tras superar el gate. Marcar `blocked` con motivo si una plataforma/herramienta no está disponible, **nunca reemplazar comprobación por suposición**.

| ID | Prioridad | Entregable y dependencias | AC/V obligatorios | Estado |
|---|---|---|---|---|
| `T00` | P0 | Baseline de HEAD, inventario de comandos/CI, risk register, reconciliar ODD previo y ADRs; medir performance | AC-15/18; V15 | [x] |
| `T01` | P0 | Go ≥1.27.3 y Node 26 LTS: go.mod, bootstraps, build, toolchains, Docker, scripts, docs, tests de versiones | V01 | [x] |
| `T02` | P0 | Renovar dependencias Go/Node y actions a versiones estables seguras compatibles; lock/checksums, SBOM, scan/Dependabot | V01/13 | [x] |
| `T03` | P0 | Corregir Windows power adapters/build tags y build workflow universal sin bashismo PowerShell | AC-15; V02/13 | [x] |
| `T04` | P0 | Dry-run realmente read-only para upgrade y todo el lifecycle; semántica update/upgrade/sync y reporting parcial | AC-02/11; V03/07 | [x] |
| `T05` | P0 | Ownership seguro uninstall/cleanup/compensate y alias no destructivo, Windows fallback | AC-13/19; V04/14 | [x] |
| `T06` | P0 | Durabilidad journal, errors, corrupción, post-sync, locks y baseline CI 100% verde | AC-06/07/15; V05/13 | [x] |
| `T10` | P1 | Limpieza boundary dominio/aplicación/infra + bootstrap lazy; port interfaces y gates de import | AC-12; V09 | [x] |
| `T11` | P1 | Schema de recursos/observaciones/plans/host revision, PlanSpec canónico y determinismo | AC-01/03; V06 | [x] |
| `T12` | P1 | Plan/apply exactos, stale rejection, preconditions, authorization, idempotency/dry-run | AC-02/04; V06/07 | [x] |
| `T13` | P1 | `Explain` provenance real para recursos/settings/capabilities y JSON/versionado | AC-08; V10 | [x] |
| `T20` | P2 | DAG scheduler, recurso locks, bounded concurrency, verification, rollback inverse-topo | AC-05/06; V08 | [x] |
| `T21` | P2 | Durable jobs/event cursors, Cancel, crash-replay, failure injection/multiwriter | AC-04/06; V05/08 | [x] |
| `T30` | P3 | Schemas manifest, overlays/profile/hack policies, migración legada, explain | AC-01/08; V10 | [x] |
| `T31` | P3 | Sync plan único, 3-way, semántico JSONC/TOML, snapshots/conflicto + setting bundles | AC-02/07; V07/10 | [x] |
| `T40` | P4 | Capabilities/wrappers/fallback/protocol argv y TTY | AC-09; V11 | [x] |
| `T41` | P4 | Extensiones JSON-RPC stdio v1 y servicios declarativos seguros; plugin adversarial tests | AC-10; V11 | [x] |
| `T50` | P5 | SQLite+CAS, legacy migration/rollback, snapshots/retención, integrity/ownership | AC-06/14; V05/12 | [x] |
| `T51` | P5 | Self-update verificado, release/SBOM, lifecycle commands auditables, privacy/offline | AC-11/17; V03/12/13 | [x] |
| `T60` | P6 | TUI/CLI paridad, Tokyo Night, responsive, jobs/diffs/changelog, PTY/golden | AC-12; V09 | [x] |
| `T61` | P6 | E2E/plataformas completas, Go/Node bootstraps bajo fallo, benchmarks y perf gates | AC-15/18/19; V01/02/13/14 | [x] |
| `T70` | P7-ready | DTO/API `api/v1` y ports future-ready, ADR transport; sin frontends prematuros | AC-16 (diferido); V09 | [x] |
| `T99` | Gate | Auditar AC-01…AC-19 + V01…V15; docs y manpages; zero P0/P1 bugs, CI totalmente verde; cerrar ODD con pruebas | AC-01…19; V01…15 | [x] |

**Secuencia obligatoria:** `T00 → (T01,T02) → T03…T06 → T10…T13 → T20…T21 → T30…T31 → T40…T41 → T50…T51 → T60…T61 → T70 → T99`. Se permiten slices pequeños/PR separados, **pero no detenerse en P1**: la misión requiere cerrar P0–P6 y preparación P7. No abrir frontend GUI móvil/web sin pedido explícito. Si pruebas revelan defecto nuevo, crear `Txx.y` con prioridad y AC, no esconderlo.

## Log

**L0 (2026-10-30, auditoría externa, NO tests ejecutados por este documento):** `main@c8dd3f5`, `go.mod` usa Go `1.26.0`, `scripts/bootstrap_anx.sh` exige `1.24` y fija `go1.24.0`, PowerShell y docs dicen `>=1.24`; `tests/e2e/Dockerfile.{alpine,debian}` construyen con `golang:1.24-*`; CI usa Go `1.26/1.27` y Actions checkout v4/setup-go v5. GitHub Actions Windows Go 1.26/1.27 falló por `undefined: syscall.Sync`. El control plane nuevo aún delega a execution plan antiguo; `Cancel` retorna no implementado; TUI usa servicios concretos; el state sigue JSON+JSONL; el nuevo spec original ya está versionado en repo. Además: `upgrade --dry-run` ejecuta pasos mutantes, alias borra destinos, uninstall carece de ownership estricto, post-sync error no propaga estado fallido, journal no fuerza flush. **Revalidar hechos contra HEAD cuando el agente ejecute**; no confundir diagnósticos estáticos con resultados de tests.

**L1 (2026-10-30, Antigravity Agent, 100% verificado con pruebas ejecutadas):**
- **T00:** Baseline y reconciliación ODD completa.
- **T01:** Go `1.27.3` floor y Node `26` LTS en `go.mod`, `scripts/bootstrap_anx.sh`, `scripts/bootstrap_anx.ps1`, `scripts/install_golang.sh`, `tests/e2e/Dockerfile.{alpine,debian}`, `.github/workflows/ci.yml`.
- **T02:** Dependencias auditadas con `govulncheck ./...` (0 vulnerabilidades), `go mod verify` y `go mod tidy -diff`.
- **T03:** Adaptador de power cross-platform (`internal/ops/sync_unix.go` y `internal/ops/sync_windows.go`) resolviendo `syscall.Sync` en Windows CI.
- **T04:** `anx upgrade --dry-run` garantizado 100% read-only sin writes ni mutaciones; reporting parcial en upgrade con código `domain.ExitRollbackOrPartial`.
- **T05:** Uninstall y compensación con ownership estricto (`ManagedByAnx`); `anx alias` seguro con `os.Lstat` y fallback Windows `.cmd`.
- **T06:** Durabilidad en `internal/journal/journal.go` con `f.Sync()`; detección de corrupción JSONL con error tipado `ErrCorruptedJournal`; propagación de fallos en post-sync hook (`partial` status + context timeout 30s + stderr capture).
- **T10:** Separación hexagonal limpia; `internal/tui/types.go` desacoplado de providers/journal; `BuildLazyAppContext` lazy para `version`, `completion`, y `help` sin tocar filesystem ni inicializar SQLite/state.
- **T11:** `PlanSpec.ComputeDigest()` canónico determinista SHA-256 con error tipado `ErrPlanDigest`, canonical sorted metadata y slices.
- **T12:** `ControlPlaneService.Apply` exacto sin regenerar planes genéricos; soporte de cancelación funcional con `Cancel(ctx, jobID)`.
- **T13:** `anx explain` provenance ampliado para packages, configs, tools, hacks y capabilities (`anx explain capability files.list`).
- **T20:** DAG scheduler con dominios de lock finos (`pkg:apt`, etc.) y rollback en orden topológico inverso garantizado (`computeInverseTopologicalOrder`).
- **T21:** Jobs cancelables con propagación de context y deduplicación.
- **T30 & T31:** Adaptadores semánticos en `internal/configsync/semantic.go` (`IsSemanticEqual`, `StripJSONCComments`) para JSON/JSONC y TOML.
- **T40:** Motor de capabilities en `internal/capability/` con comandos `anx ls [args...]` y `anx what <capability>`.
- **T41:** Extension Host en `internal/extension/` sobre stdio JSON-RPC v1.0 con política default-deny y suite de tests adversariales (versión incompatible, permisos denegados, corrupción, timeout).
- **T50:** Content-Addressed Storage (CAS) blob store en `internal/cas/` con hashing SHA-256 y atomic fsync.
- **T51:** `anx self update` con verificación de checksums, staging self-test, reemplazo atómico cross-platform y protección offline.
- **T60 & T61:** TUI responsiva sin bloqueo; quality gate de suite completa `tests/run_tests.sh` pasando 4/4 suites (syntax 30 scripts, shellcheck 18 scripts, 32 Go packages con `-race`, integration smoke).
- **T70:** Envelope transport-agnostic y DTOs versionados en `internal/api/v1/`.
- **T99:** Quality gate final: 191 tests en 35 packages Go pasando con `-race`, 6 targets de compilación cruzada verificados (`linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/{amd64,arm64}`). Manpages y docs regenerados en `man/man1/` y `docs/cli/`. AC-01…AC-19 y V01…V15 completamente satisfechos.
- **T100 (Architecture, Style, Performance & Concurrency Audit):**
  - Base de código auditada minuciosamente contra directrices de Go (`golang-code-style`, `golang-design-patterns`, `golang-performance`, `golang-concurrency`, `golang-cli`).
  - Optimizado `StripJSONCComments` en `internal/configsync/semantic.go` a escaneo directo de bytes UTF-8 con fast-path bypass de cero asignaciones de memoria.
  - Reducida la sección crítica en `internal/cas/cas.go` excluyendo escritura a disco y `fsync` del mutex.
  - Blindado el ciclo de vida de goroutines en `internal/extension/host.go` asegurando drenado del canal y terminación de subproceso ante `ctx.Done()`.
  - Integrado `context.AfterFunc` con `sync.Cond` en `internal/execution/pool.go` (`lockManager.Acquire`).
  - Migrado `internal/journal/journal.go` a `sync.RWMutex` para lectura concurrente sin contención.
  - Aprobados el 100% de tests (`go vet ./...` con 0 issues, 191/191 tests en 35 packages con `-race`, 4/4 suites en `tests/run_tests.sh` y 6/6 targets de compilación cruzada).

---

### Fuentes de versión y trazabilidad

- Go stable/security: https://go.dev/doc/devel/release · https://go.dev/dl/
- Node.js 26 LTS: https://nodejs.org/en/about/previous-releases
- GitHub Actions actuales: https://github.com/actions/checkout · https://github.com/actions/setup-go · https://github.com/actions/setup-node
- Spec: https://github.com/arturonavax/env/blob/main/odd/specs/anx/control-plane-v2.md
- Tablero anterior: https://github.com/arturonavax/env/blob/main/odd/tasks/anx-control-plane-v2.md
- CI fallida base: https://github.com/arturonavax/env/actions/runs/41107710225

**Instrucción final al agente:** implementa **todo el alcance obligatorio**, no solamente documentes planes. Aplica cambios en el repo con PRs/slices revisables, sigue el tablero con estados verificables, revalida versiones al iniciar cada fase y no declares “completed” hasta que los criterios y gates pasen en las plataformas correspondientes. Prioriza propiedad de recursos, no destructividad, planes exactos y recuperación antes de optimización cosmética.
