# ODD — ANX: cierre verificable del Control Plane v2, CI multiplataforma y protección de dotfiles

> **Tipo:** tarea de implementación integral, no informe ni propuesta.  
> **Repositorio:** `https://github.com/arturonavax/env`  
> **HEAD auditado:** `9fe41da2569588a526ccfb20d991e82bc8506444` (30-oct-2026; **volver a comprobar al ejecutar**).  
> **Workflow examinado:** [CI #41119101604](https://github.com/arturonavax/env/actions/runs/41119101604), `completed/failure`: Ubuntu Go 1.27.3 ✅, macOS Go 1.27.3 ✅, lint ✅, Windows Go 1.27.3 ❌.  
> **Ruta propuesta:** `odd/tasks/anx-control-plane-final-remediation-herdr-ci.md`.  
> **Normativa preexistente:** `odd/specs/anx/control-plane-v2.md` (AC-01…AC-19), `odd/tasks/anx-control-plane-completion-go127.md` (V01…V15), `odd/tasks/anx-control-plane-v2.md`, `odd/tasks/anx-platform-overhaul.md`, `odd/tasks/master-spec.md`, `AGENTS.md` y specs asociados.

## 0. Rol, principios y metodología

Actúa como Principal Go Engineer + Platform Architect + Security Engineer + TUI/DevEx Engineer. **No cierres la tarea hasta que el software funcione, la seguridad sea comprobable y todos los gates sean verdes**. Organic Driven Development: `Specs → Tasks → Log → evidencia reproducible`; TDD con tests de falla, success y regresión por slice; commits/PRs pequeños cuando sea posible. Cargar las skills locales aplicables `.agents/skills/{golang-cli,golang-code-style,golang-concurrency,golang-design-patterns,golang-performance,golang-spf13-cobra,bubbletea}/SKILL.md` si existen; recargarlas al cambiar de subsistema. Inspeccionar implementación, no confiar en README/checkboxes. Validar versiones contra releases oficiales en el momento de ejecución.

**Invariantes:** preservar funcionalidad de `anx`, `aitools` multicall, dotfiles, profiles, `sync/capture`, proveedores, herramientas, hacks, `sleep`, alias y bootstraps; una única arquitectura modular-hexagonal Go; offline-first; `--dry-run` realmente sin escrituras ni procesos mutantes; no destructivo sin consentimiento; no enviar datos locales al repositorio por defecto; sin Node/Bash/Python requeridos para ejecutar el núcleo Go; soportar Windows nativo sin WSL; no falsificar resultados mediante tests `skip` masivos, mensajes de éxito simulados o CI opcional. Si aparecen riesgos nuevos, crear subtareas trazables y resolverlos.

**Integridad ODD:** el tablero previo marcó `T00–T99` y `AC-01…AC-19/V01…V15` como satisfechos; **esta afirmación queda revocada hasta verificarla**, no se borran logs históricos. Actualizar el tablero anterior con una anotación de auditoría y una matriz de estado `verified / partial / failed / unverified` acompañada por `código + test + SHA + URL de job` para cada criterio. Los checkboxes nunca son evidencia.

## 1. Hallazgos iniciales demostrados, priorizados

| ID | Prioridad | Evidencia actual | Requisito |
|---|---|---|---|
| F01 | P0 | CI #41119101604 falla en Windows en `go test`; no fallan `go vet` ni compilación por `syscall.Sync`. | Corregir plataforma y tests, no eludirlos. |
| F02 | P0 | `internal/cli/capture.go:263-265` ejecuta `git add files/` sobre todo el árbol; cualquier archivo no ignorado puede entrar al commit. | Staging **exclusivo** de los artefactos allowlisted y efectivamente capturados. |
| F03 | P0 | `internal/app/sync_service.go:232-248` anuncia `Action=captured` en `--dry-run` sin cambios; `internal/cli/capture.go:109-125` puede pasar a `executeGitCommitAndPush` si se pide `--commit/--push`. | `--dry-run` bloquea **siempre** checkout/branch, add, commit, push y cualquier escritura. |
| F04 | P0 | `.gitignore` ignora algunos nombres conocidos de `files/herdr`, pero no es frontera de seguridad. | Default deny + allowlist para runtime/secretos/desconocidos, tanto antes de copiar como antes de stage/commit/push. |
| F05 | P0 | Herdr documenta estado/socket junto a `~/.config/herdr/config.toml` por defecto; `manifests/configs.yaml` sólo administra ese archivo y no declara sync de directorio. | Comprobar symlink/mount local hacia `files/herdr`, evitar que Herdr escriba runtime en repo y proteger secretos. |
| F06 | P0 | `internal/cli/self_update.go` anuncia `Verified self-update completed successfully` sin descargar, validar ni reemplazar binario. | Self-update real, seguro y verificable o estado no implementado explícito mientras se desarrolla. |
| F07 | P0 | `internal/state/store.go` persiste `state.json`, no SQLite; mutaciones en memoria liberan mutex antes de persistir snapshot. | SQLite transaccional + migración atómica; eliminar carrera de escrituras perdidas. |
| F08 | P0 | `internal/app/control_plane.go:293-345` vuelve a construir o ejecuta `ExecutionPlan` mutable; no existe autorización de plan persistido/host revision/idempotencia durable. | Plan exacto y autorizado, stale rejection y precondiciones verificadas. |
| F09 | P0 | `internal/extension/host.go:43-113` lanza plugins antes de validar permisos de handshake: autorización declarativa ≠ sandbox OS. | Aislamiento/permission model real y launch trust policy. |
| F10 | P1 | `internal/execution/pool.go` tiene locks y rollback inverso, pero ejecuta lotes contiguos sin esperar dependencias de DAG de forma individual. | Scheduler DAG real, locks globales por plan/host, fairness, rollback verificable. |
| F11 | P1 | `internal/configsync/semantic.go` sólo compara JSON(C)/TOML; no implementa merge semántico que preserve contenido. | Merge integrado seguro, golden tests, snapshots, conflictos. |
| F12 | P1 | `internal/cli/meta.go` puede borrar symlink ajeno; fallback Windows escribe `.cmd` existente sin validar; usa dir del binario en vez de user bin. | Alias safe-by-default en todos los SO. |
| F13 | P1 | `internal/app/control_plane.go` ignora errores de `IsInstalled` y asigna `Healthy=true`; provenance genérica. | Observación fail-closed y provenance efectiva. |
| F14 | P1 | `internal/capability/capability.go`: `rg→grep`, `fd→find` sin adaptación fiable de argumentos. | Wrappers con semántica explícita o rechazo de flags incompatibles. |
| F15 | P1 | `README.md` y `odd/specs/anx/overview.md` conservan Go ≥1.24/1.26; `scripts/bootstrap_anx.ps1` conserva mensaje ≥1.24 y elimina árbol Go previo antes de verificar extracción. | Migración Go ≥1.27.3 completa y bootstrap atómico/no destructivo. |
| F16 | P1 | `go.mod` piso 1.27.3, pero dependencias siguen sin actualización demostrada; Actions `checkout@v4`, `setup-go@v5`; no hay gates SBOM/secret-scan/dep automation. | Auditar/migrar de forma reproducible, no `latest` sin pruebas. |
| F17 | P1 | Node 26 LTS no está fijado ni verificado por `manifests/packages/toolchains.yaml` / `internal/ops/rebuild.go`; Docker base runtime antigua sin pin SHA. | Node 26.x LTS (último patch compatible), supply chain. |
| F18 | P1 | TUI aún depende de `catalog.Registry` y servicios concretos en `internal/tui/types.go`; `api/v1` son DTOs, no core unificado completo. | Ports/DTOs con paridad CLI/TUI, dashboard real. |
| F19 | P1 | CAS en `internal/cas` no está conectado a snapshots/job store; `Put` reutiliza blob existente basándose sólo en tamaño. | Verificar hash al reutilizar, integración/GC/recuperación. |
| F20 | P1 | No hay evidencia verificable de ejecución completa E2E, bench, fuzz, Termux ni release attestation; rama `main` no protegida al revisar. | Quality gates y reglas de branch protection requeridas. |

### 1.1. Incidentes Windows concretos del CI concluido

**Runner:** `windows-latest`, Go 1.27.3, [job 124280905388](https://github.com/arturonavax/env/actions/runs/41119101604/job/124280905388). `go mod verify` y `go vet` aprobaron; `go test -v -timeout 10m ./...` falló; build posterior **skipped**. Corregir por causa, no con un `//go:build !windows` indiscriminado.

| Tests/familia | Fallo exacto observado | Acción |
|---|---|---|
| `internal/ai/gitflow:TestClipboardOSC52FallbackAndBinary` | `wl-copy`/xclip/xsel/pbcopy/clip.exe no disponibles. | Inyectar backend/mock cross-platform y probar fallback OSC52 sin exigir utilidades externas. |
| `internal/ai/harness:TestFallbackDiscovery`, `TestHarnessProtocolsWithFakeScripts` (4 subtests) | Archivos fake sin extensión ejecutable no se resuelven vía Windows PATH; `exec: ... executable file not found in %PATH%`. | Usar fixture Go compilada nativamente o `.cmd`/PowerShell invocado por intérprete explícito; pruebas de argv, stdin, JSON y temp-files equivalentes por SO. |
| `internal/extension`: 5 tests (handshake, incompatibilidad, default-deny, respuesta corrupta, timeout) | Ejecutar `*.sh` con `exec.Command` produce `%1 is not a valid Win32 application`. | Implementar mock extension ejecutable Go multiplataforma; comprobar todas las cinco trayectorias en runner real. |
| `internal/hacks:TestProbeHardwareAndMatchHacks` | Nombre de directorio con `0000:01:00.0` inválido en NTFS. | Fixture con paths válidos y adapter de lectura de hardware desacoplado de rutas Linux; Windows devuelve unsupported tipado donde corresponda. |
| `internal/ops:TestRebuilderDryRunAndMockExecution` | Binaries simulados `cargo`, `pnpm`, `pipx`, `nvim`, `zsh` no ejecutables en Windows. | Fixtures portables; mock subprocess runner; verificar dry-run sin invocaciones ni efectos. |
| `internal/ops:TestOptimizerAuditSynthesizeAndGuardrails` | Sin warnings esperadas y sin `applied.txt` del script simulado. | Separar política por plataforma; no usar bash implícito en Windows; comprobar aplicación real o rechazo explícito. |
| `internal/providers:TestMultiStrategyDispatcherGitHubTarballFallbackToScript` | `expected ... version 2.4.0, got "installed"`. | Versión observable verificable con proveedor multiplataforma y fixture que no dependa de bash. |
| `internal/tools:TestToolRunnerExternal`, `TestToolIsolationNoGenericLocalBin` | Usa `/bin/echo` inexistente y herramienta sin ejecutable Windows válido. | Binario fixture Go y rutas/EXT de Windows; verificar aislamiento y PATH. |

**Gates Windows adicionales:** `go test -count=1 -shuffle=on ./...` sin skips impropios; `go vet`, `go build`, smoke CLI y TUI, `go test -race` donde runner/toolchain lo admita; prueba E2E de path/symlink/alias/capture; medir cobertura. Resolver causas de runtime reales además de tests.

## 2. P0 — protección de `files/` y Herdr: diseño de seguridad obligatorio

### 2.1. Diagnóstico de origen antes de tocar datos

**Origen upstream confirmado:** según la documentación oficial de Herdr, la instalación por defecto coloca el socket Unix `~/.config/herdr/herdr.sock`, el estado persistente `~/.config/herdr/session.json`, los directorios de sesiones y `session-snapshots/` **junto a `config.toml`**. Herdr crea/actualiza estos archivos durante su operación normal; no constituyen dotfiles para sincronización. Fuente: [Herdr Socket API](https://herdr.dev/docs/socket-api/) y [Herdr Session state and restore](https://herdr.dev/tui/docs/session-state/). Existe [discusión upstream #2316 sobre adoptar XDG state/runtime separados](https://github.com/herdrdev/herdr/discussions/2316) — verificar el estado actual antes de asumir que está resuelta.

**Origen en `files/herdr` del equipo todavía NO confirmado:** el manifiesto versionado de ANX sólo administra `config.toml`, de modo que no explica por sí mismo la aparición de los otros archivos. La hipótesis principal a comprobar es que `~/.config/herdr` **sea un symlink de directorio, bind mount u otra ruta compartida** con `<repo>/files/herdr`, provocando que Herdr escriba directamente allí (no que ANX los copie); también examinar comandos heredados y acciones automáticas. Registrar ruta real y proceso, sin imprimir archivos de sesión ni secretos.

Comandos de inspección **sólo lectura**, adaptables a SO:

```sh
ls -ld files/herdr ~/.config/herdr ~/.config/herdr/config.toml
readlink -f files/herdr ~/.config/herdr ~/.config/herdr/config.toml 2>/dev/null || true
find files/herdr -maxdepth 2 -printf '%y %p\n' 2>/dev/null
git ls-files files/herdr/
git status --ignored --short -- files/herdr/
git check-ignore -v files/herdr/session.json files/herdr/herdr.sock || true
rg -n --hidden '(files/herdr|\.config/herdr|herdr\.sock|session\.json)' --glob '!.git/**' .
```

Si son archivos que Herdr escribe en un directorio conectado al repo, arreglar **la ruta de escritura**, preferiblemente separar runtime: `XDG_CONFIG_HOME/herdr` sólo config, `XDG_RUNTIME_DIR/herdr` para socket (permisos 0700), `XDG_STATE_HOME/herdr` para sesiones/estado, `XDG_CACHE_HOME/herdr` para caché/logs cuando Herdr lo soporte. **No inventar flags de Herdr**: consultar documentación y comportamiento real antes de modificar settings; si el upstream obliga a guardar runtime con el config, usar un config standalone aislado/dir de instalación seguro y diseñar el despliegue acorde. No eliminar sockets vivos ni archivos de sesión automáticamente; cualquier limpieza requiere parada segura, respaldo y aprobación. Permisos mínimos para estado y sockets (directorio 0700, archivos sensibles 0600) cuando la plataforma permita.

### 2.2. Modelo de sincronización, límites y confianza

- **Repo → máquina:** sólo assets **declarados/permitidos**; nunca tratar `files/herdr/` como plantilla recursiva por accidente; `anx sync` no debe copiar runtime y secretos desde repo al HOME; proteger symlinks/hardlinks y rutas sobrepuestas.
- **Máquina → repo:** `anx capture` escribe únicamente targets allowlisted en manifests. Para Herdr sólo `files/herdr/config.toml`; *ningún otro archivo ni subdirectorio*, aun si un nuevo runtime aparece mañana. La exclusión por patrón sigue como defensa secundaria, **no es la autorización**.
- **Manifiestos de directorio:** exigir `include` explícito de nombres/patrones de configuración confiables y `exclude` sensible global; nuevas entradas se tratan como `unknown`/`review required`. Nunca invertir a `allow all` ante patrón vacío, inválido o fallo al leer .gitignore. Si el usuario agrega una plantilla nueva, revisión explícita antes de autorizar.
- **Riesgo de secretos:** archivos de sesión pueden contener comandos, prompts, rutas, tokens o metadatos sensibles. Los sockets suelen ser nodos Unix no aptos para Git pero igual deben mantenerse fuera del árbol versionado; la protección debe cubrir ficheros regulares, symlink targets, sockets, directorios y posibles formatos nuevos (`db`, `sqlite`, `wal`, JSON, backups, state, locks, credentials). Escanear contenido, no sólo extensiones; no volcar secretos encontrados a terminal/CI/log/ODD.
- **`.gitignore` complementario:** endurecer con:

```gitignore
# Herdr: deny all runtime by default, allow only vetted template
/files/herdr/**
!/files/herdr/config.toml
```

  Verificar precedencia con otros patrones y gitignore anidados. No confiar en `.gitignore` contra `git add -f` ni contra archivos ya trackeados; auditar `git ls-files`, historial y workflows. Para otros directorios `files/*`, no aplicar deny-all de forma destructiva sin inventariar plantillas legítimas; implementar misma política de autorización por recurso y plugin.

### 2.3. Proteger `capture → stage → commit → push`

- **Eliminar `git add files/`**. `CaptureToRepo` debe devolver un **manifest de archivos realmente escritos** (paths relativos canónicos, SHA-256 esperado, config ID, origen, diff resumido, provenance, clasificación de riesgo) con `ChangeSet` tipado. **Staging con `git add -- <lista exacta>`**, sin glob/recursive root; validar cada path contra allowlist, dentro del repo, sin symlinks evasivos, sin otros cambios pre-stage. Cuidado: archivos ya staged previamente también entrarían en el commit; usar index temporal aislado (`GIT_INDEX_FILE`) o verificar/aislar index, incluir exclusivamente paths autorizados y nunca deshacer staging ajeno sin consentimiento. Evitar inyección de argv/pathspec; usar `:(literal)`/`--` y rutas canónicas donde corresponda.
- Si ya hay archivos no autorizados en staging o working tree: **bloquear el auto-commit y publicar diagnóstico sin mostrar contenido**, no limpiar ni borrar automáticamente. No ejecutar `git checkout -b`, `git add`, `git commit`, `git push` en `--dry-run`, aunque `--commit` o `--push` estén presentes; previsualizar con fake git runner/instrumentación que falla si se invoca cualquier operación mutante. `--offline` nunca puede hacer push/fetch.
- Comprobar firma/protección de ramas, permisos, remote esperado y ref del commit **después** de crear contenido; `--push` opt-in con confirmación explícita, incluso con `-y` en presencia de secretos/unknown. No equiparar autorización de GitHub a autorización de divulgar datos privados.
- Integrar scanner de secretos pre-capture, pre-commit y CI (gitleaks u otro actualizado y fijado) + reglas de denegación para credenciales, high-entropy y formatos privados; falsos positivos sólo mediante override específico, versionado, justificado y revisado, **no** `--force` global. Git hooks son defensa adicional, **CI es obligatoria**. Permisos de ficheros locales bajo principio de mínimo acceso. Secret scan en todo el historial alcanzable y diff de PR según factibilidad, sin exponer valores.
- Repositorio ya público: si hay secretos **efectivamente** cometidos, revocarlos/rotarlos y evaluar limpieza del historial con procedimiento explícito; `.gitignore` no elimina exposición previa. No hacer force-push ni reescribir historia sin aprobación.

### 2.4. Pruebas negativas obligatorias (`HS01…HS12`)

1. `HS01`: herdr config capturable y diff exacto; archivo no autorizado `files/herdr/unknown-secret-2027.json` jamás capturado, staged ni comiteado.
2. `HS02`: `session.json`, `herdr.sock`, `session-snapshots/`, `.plugins.lock`, `*.log`, `state.db`, `*.sqlite-wal`, backups nuevos permanecen fuera de repo/commits aun tras `--all`.
3. `HS03`: `git check-ignore` confirma deny-all Herdr excepto `config.toml`; también rechazo si `git add -f` fue usado previamente o si el archivo fue stageado por otra herramienta.
4. `HS04`: `anx capture --all --dry-run --commit --push` causa **0** operaciones mutantes Git, 0 network, 0 writes en repo/estado/snapshots.
5. `HS05`: captura normal genera change-set con archivos exactos; `git add -- <changeset>` y commit no incorporan otros cambios concurrentes ni staged preexistentes.
6. `HS06`: path traversal, symlink/hardlink hacia fuera, case-folding (Windows), rutas extrañas y Unicode no eluden autorización.
7. `HS07`: manifiestos vacíos, malformados, sin `include` en directorio, exclusiones rotas ⇒ fail-closed.
8. `HS08`: fichero sensible en `.toml` permitido con token de prueba ⇒ secret-scan bloquea commit, redacta output y conserva archivo local.
9. `HS09`: runtime sockets/sesiones activos no se borran ni se mueven durante un sync, upgrade, cleanup o dry-run.
10. `HS10`: instalar/sincronizar Herdr en una instalación limpia mantiene una única plantilla versionada, sin symlink de **directorio** que vuelva escribible el repo por Herdr.
11. `HS11`: capturar en Windows/Linux/macOS produce resultado equivalente y respeta permisos/metadata aplicables.
12. `HS12`: detector documenta qué creó realmente los archivos locales de Herdr; si no se dispone del equipo, marcar `local diagnosis: unverified`, **nunca inventar explicación**.

## 3. Tareas completas de remediación P0–P7 (obligatorias)

Cada tarea sigue el patrón `diagnóstico → modificación mínima correcta → tests éxito/falla/adversariales → documentación → CI → log con links`. Todas empiezan `[ ]`; si algo no es verificable, marcar `blocked` (no `[x]`).

| ID | Prioridad | Implementación requerida | Evidencia de salida |
|---|---|---|---|
| R00 | P0 | Revalidar HEAD y todas las ODD, abrir matriz AC/V/HS y reconciliar falsos cierres, inventariar baseline/feature parity. | Auditoría trazable, tests iniciales por plataforma. |
| R01 | P0 | Reparar exactamente las 8 familias fallidas de Windows descritas arriba; test mocks ejecutables verdaderamente multiplataforma, no saltos indiscriminados. | CI Windows Go 1.27.3 verde y sin skips de funcionalidad base. |
| R02 | P0 | Diagnóstico local Herdr; separar config/runtime con upstream verificado; evitar acoplamiento de directorios al repo. | Diagnóstico + HS09/HS10/HS12. |
| R03 | P0 | Default-deny/allowlist `files/herdr/**`, captura general por manifests, validación de symlink/path y contenido sensible. | HS01/02/03/06/07/08/11. |
| R04 | P0 | ChangeSet tipado, staging exacto e index aislado, aprobación opt-in, secret scanning; eliminar `git add files/`. | HS04/05/08, tests commit/push sin leakage. |
| R05 | P0 | `--dry-run` no mutante universal, especialmente `capture --commit/--push`, `upgrade` y `self update`; sin éxito global si hay parciales. | Monitoreo fake FS/process/network con 0 mutaciones. |
| R06 | P0 | Implementar `anx self update` real: manifest oficial firmado, hash, trusted source, staging, self-test, atomic swap/Windows recovery, rollback, compat y offline. | E2E servidor mock con firma inválida/red caída/disco lleno/rollback. |
| R07 | P0 | SQLite WAL transaccional + jobs/events + migración legacy JSON/JSONL reversible y segura; serialización multiwriter, revision CAS, blobs SHA verificados y restore. | Fallo inyectado, crash-recovery, transacciones, migración real; V05/V12. |
| R08 | P0 | PlanSpec completo e inmutable, host facts/config/policies/digest y expected revision; Apply usa plan autorizado persistido exacto; stale/idempotencia. | Diff plan/apply, TOCTOU, hash, precondition, restart; V06/V07. |
| R09 | P0 | Extensiones: política de confianza **antes del exec**, permisos OS reales o mecanismo de sandbox compatible (adapter específico), aislamiento, recursos/IO limitados, cancel/timeout. | Tests adversariales en Windows/Linux/macOS, documentación de límites; V11. |
| R10 | P1 | Scheduler DAG efectivo por dependencias (no batch adyacente), locks finos persistentes, bounded parallelism/fairness, rollback topológico y compensaciones fail-closed. | Tests dependencias bloqueadas, ciclos, cancel y failure injection; V08. |
| R11 | P1 | Jobs durables con `queued/running/partial/failed/cancelled/succeeded/recovering`, eventos monotónicos replay/cursor, dedupe y cancel asíncrono. | Estado y eventos tras restart/crash; V05/V08. |
| R12 | P1 | Modelos y adaptadores de manifiestos versionados; source provenance, overlay/profile/hardware, validación y explicación exacta; no reglas hard-coded para nuevos paquetes. | Golden tests y migración YAML legada; V10. |
| R13 | P1 | Unificar diff/sync/capture/backup con plan/apply; semántica JSONC/TOML realmente fusionable preservando comentarios/EOL/permisos o conflicto explicito; setting bundles. | Golden 3-way/partial/fail y snapshots intactos; V07/V10. |
| R14 | P1 | Uninstall/cleanup/rollback/alias seguros por ownership, no borrados de recursos preexistentes, symlinks ajenos ni `.cmd` preexistentes; bootstrap Go Windows staged/no downgrade. | Tests de recursos ajenos y adversarial path; V04/V14. |
| R15 | P1 | `Observe` no marca healthy si provider falla; source provenance real, errores explícitos, sin nil panics. | Mock provider error + explain por recurso; AC-08. |
| R16 | P1 | Capabilities/wrappers semánticamente correctos (`rg→grep`, `fd→find`, etc.), flags/TTY/stdout/stderr/status, OS/platform and fallback negotiation. | Golden argv/exit e incompatibilidad rechazada; V11. |
| R17 | P1 | Core ports/adapters UI: CLI/TUI consultan mismas interfaces; bootstrap verdaderamente lazy, dashboard Tokyo Night responsive (status, configs, changelog, jobs, diff, hardware/security), navegación/PTy tests. | Contract tests + golden layouts/breakpoints; V09. |
| R18 | P1 | Go >=1.27.3 en go.mod, Unix/PS/installer, Windows ARM64, Docker, docs, Makefile y CI; Node 26 LTS sólo para tooling Node; gestores GVM/fnm sin degradar versiones. | Test 1.27.2 rejects, >=1.27.3 accepts; Node 26 patch check; V01/V02. |
| R19 | P1 | Dependencias Go actualizadas a últimas **estables/seguras compatibles**, ABI/changelogs tests, Go mod tidy/verify/vulncheck, acciones actuales SHA-pinned, bases Docker verificadas; SBOM/provenance/Dependabot. | Before/after matrix + security gates + artifacts de release; V13. |
| R20 | P1 | Corregir `state.Store` actual mientras migra (mutex around actual writes o store transaccional); CAS `Put` verifica hash previo, permisos y atomicidad; symlink protections. | `go test -race`, tests de updates concurrentes/byte corruption; V12. |
| R21 | P1 | Privacidad/logs sin secretos y offline real; scripts `curl` con trust/consent/checksum; symlink/path/command injection guards; datos sensibles fuera de repo. | Security negative tests, scanner y threat model. |
| R22 | P1 | Verificar Mac, Linux, Windows, WSL, distro/musl, Termux/Android donde viable; E2E de bootstrap, installations/fallbacks/CLI/TUI/release y performance gates medidos. | Matriz evidencia / `unverified`; AC-15/18/19. |
| R23 | P1 | Branch protection main (required CI, PR reviews según repo, evitar merge rojo), CI con permisos mínimos/acciones SHA, lint no opcional y secret scan. | Config observada vía API/UI o instrucción admin documentada si falta permiso. |
| R24 | P2 | Paridad `anx update/upgrade/sync/capture/self update` y lifecycle; docs/manpages/completions regenerados; pruebas de alias y `aitools`. | Smoke por sistema, contratos de exit codes estables. |
| R25 | P2 | API/DTO v1 future-ready, plan digest/jobID/idempotency/revision/events, sin levantar agente ni frontend no solicitados. | Compatibility tests y ADR, AC-16 diferido legítimamente. |
| R99 | Gate | Reevaluar AC01–19, V01–15 y HS01–12; no permitir ninguna tarea falsa `[x]`, E2E y CI 100% verde, benchmark/scan/sbom, comparar performance y comportamiento antes/después. | Report final con commit SHA, URL de checks, tests, riesgos residuales, pasos de migración/rollback y cero P0 abiertos. |

### Definition of Done no negociable

- `go test -count=1 -shuffle=on -timeout 10m ./...`, `go test -race ...` donde compatible, `go vet`, `gofmt`, linters, `go mod verify`, `go mod tidy -diff`, `govulncheck`, E2E Docker + Windows native + PTY, secret scan, dry-run leak tests, builds multiarch. Ningún check silenciosamente opcional. No usar runner Unix como sustituto de runner Windows.
- Test defaults aislados de `$HOME` real y de Internet; fixtures sandbox que no pueden tocar credenciales ni hacer push/instalación real. Windows `.exe` mock para `exec.Command`, sin intentar ejecutar `.sh` como binario Win32.
- **Supply chain:** release con assets reproducibles/firmados o autenticados, hashes, SBOM, attestation, rollback; dependencias comprobadas, no promesas de “último” sin resolver versiones concretas al ejecutar.
- **Security:** ningún archivo sensible nuevo u olvidado en Git; scanner protege también `--push`; `files/herdr/` físicamente no debe recibir runtime de Herdr en instalación normal; logs y capturas no exponen contenido privado.
- **Truthfulness:** la salida CLI sólo dice `updated` cuando realmente cambió y se verificó el binario. CI y ODD sólo dicen `complete` cuando todos los gates aprobaron. Los tests externos imposibles de ejecutar quedan en `unverified`, no en verde.
- **Traceability:** mapa `Rxx ↔ AC ↔ V ↔ HS ↔ test file/function ↔ commit ↔ workflow/job` en el nuevo ODD; actualizar el ODD anterior con enlace a auditoría correctiva, sin reescribir historia.

## 4. Modo de ejecución para el agente (sin interrupciones innecesarias)

1. Leer specs/skills y revalidar `HEAD`, workflow y archivos de Herdr **en el entorno local del usuario**. Confirmar si hay symlink/directorio compartido y qué proceso genera archivos; si no hay acceso local, implementar las protecciones igual y dejar sólo la atribución del proceso como no verificada.
2. Ejecutar R00–R05 (P0), después R06–R09 (P0), después R10–R25 (P1/P2) y R99. Reordenar slices si dependencias técnicas lo requieren, documentando ADR. Nunca detenerse tras añadir interfaces, pruebas artificiales o documentación.
3. No manipular archivos runtime activos, limpiar historia Git, rotar secretos o publicar commits sin autorización del flujo de trabajo apropiado; aquí se exige **implementar un sistema que lo gestione con seguridad**, no realizar esas operaciones sobre datos privados durante la prueba.
4. Entregar `odd/tasks/anx-control-plane-final-remediation-herdr-ci.md` como tablero vivo en el repo y mantenerlo actualizado; finalizar con evidencia real, diffs, SHA, workflow verde y matriz por criterio.

## 5. Fuentes internas concretas a abrir durante la implementación

- [CI run #41119101604](https://github.com/arturonavax/env/actions/runs/41119101604) y [Windows job](https://github.com/arturonavax/env/actions/runs/41119101604/job/124280905388).
- [`manifests/configs.yaml`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/manifests/configs.yaml): Herdr config **archivo**, `three_way`.
- [`internal/configsync/capture.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/configsync/capture.go): exclusiones `session.json`, logs, snapshots; no política global allowlist.
- [`internal/cli/capture.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/cli/capture.go): captura, `git add files/`, `--dry-run`+commit.
- [`.gitignore`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/.gitignore): exclusiones Herdr por nombre.
- [`internal/app/sync_service.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/app/sync_service.go), [`internal/configsync/deployer.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/configsync/deployer.go).
- [`internal/cli/self_update.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/cli/self_update.go), [`internal/app/control_plane.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/app/control_plane.go), [`internal/state/store.go`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/internal/state/store.go).
- [`odd/tasks/anx-control-plane-completion-go127.md`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/odd/tasks/anx-control-plane-completion-go127.md), [`odd/specs/anx/control-plane-v2.md`](https://github.com/arturonavax/env/blob/9fe41da2569588a526ccfb20d991e82bc8506444/odd/specs/anx/control-plane-v2.md).
