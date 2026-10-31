# ODD MASTER SPEC + EXECUTION PROMPT — ANX: cierre integral, SQLite transaccional, Operation IR, distribución y E2E

> **Tipo:** prompt ejecutable para agente de código autónomo (Organic/Outcome-Driven Development + Spec-Driven + TDD).  
> **Proyecto:** `https://github.com/arturonavax/env` · CLI/TUI `anx` · Go.  
> **Rama de referencia auditada:** `audit/apple-silicon-e2e-odd-installer-20261031`, HEAD observado `db9b10c3d72aaebc3b6b75fa717cf3dd47673632`.  
> **CI base:** `https://github.com/arturonavax/env/actions/runs/41232029494` (**failure: Windows**, mientras macOS Apple Silicon, Linux, Docker E2E y QEMU ARM64 aprobaron).  
> **Fecha del diagnóstico:** 2026-10-31. **El agente DEBE obtener de nuevo los HEAD, diffs, checks y releases antes de tocar código.**  
> **Objetivo:** resolver **todos** los bloqueos históricos de ANX y cerrar las funcionalidades prometidas con pruebas y evidencia, sin degradar rendimiento, seguridad, UX, multiplataforma ni arquitectura.  
> **Preferencias innegociables:** portabilidad, determinismo, privacidad, operación sin cloud, sin telemetría, bajo consumo de RAM, experiencia CLI/TUI refinada, herramientas externas autónomas, Go como lenguaje del núcleo. **Windows: source-first, sin necesidad de firma de pago ni de publicar EXE/ZIP precompilados.** Dejar extensión futura para signing/releases Windows sin activarla.

---

## 0. Modo de trabajo del agente

Actúa como **Principal Go Systems Architect + SQLite/Storage Engineer + CLI/TUI Product Engineer + DevSecOps/Release Engineer**. No escribas únicamente recomendaciones: **implementa, integra, prueba y documenta** cada requisito hasta su Definition of Done (DoD). Si una operación requiere permisos/credenciales que no están presentes (Cloudflare Worker, branch protection, tags protegidos, signing secret), prepara código/infraestructura comprobable, documenta el paso externo exacto y márcala **BLOCKED-EXTERNAL**, no DONE. Evita pedir decisiones ya especificadas: toma la decisión técnica más simple que cumpla invariantes, crea ADRs breves para elecciones irreversibles, y ejecuta vertical slices operativos verificables. No hagas commits a `main` directamente ni publiques releases ni modifiques dominios/credenciales reales sin autorización específica; utiliza rama de trabajo y PR.

### 0.1. Leer y reconciliar, NO reemplazar la historia

1. Refresca `git status`, rama, HEAD, `git log`, `go.mod`, tests/CI, release inventory y dependencias. No asumas que los problemas siguen presentes ni que algo está resuelto solo por existir código.
2. Lee **completos**, sin omitir, si siguen presentes: `odd/specs/anx/*.md`, `odd/tasks/master-spec.md`, `odd/tasks/anx-control-plane-v2.md`, `odd/tasks/anx-control-plane-completion-go127.md`, `odd/tasks/anx-control-plane-final-remediation-herdr-ci.md`, `odd/tasks/anx-audit-20261031-ci-e2e.md`, `odd/tasks/anx-final-architecture-installer-e2e-audit.md`, `odd/ANX_ODD_Completion_Go127_Node26.md`, `odd/ANX_ODD_Final_Remediation_Herdr_CI_2026-10-30.md` y demás fuentes relevantes. Preserva compatibilidad y funcionalidades ya operativas.
3. Reaplica criterios útiles de las skills aportadas: **SQLite Database Expert**, **golang-cli**, **golang-design-patterns**. Su material de Rust/Tauri/`rusqlite`, `sea-query`, `r2d2` y ejemplos Python **NO** se traduce a dependencias Go: aplica los principios (transacciones, consultas parametrizadas, migraciones, FTS, arquitectura limpia, I/O, límites y lifecycle) con APIs Go idiomáticas. Sus referencias externas `references/*.md` solo podrán leerse si realmente existen; no inventes haberlas leído.
4. Crea matriz de trazabilidad actualizada **requisito → implementación → tests adversariales → job CI → resultado** para AC-001…AC-024, V01…V15, HS01…HS12 y las tareas R/F históricas. Distingue `NOT_STARTED`, `PARTIAL`, `IMPLEMENTED_UNVERIFIED`, `VERIFIED`, `BLOCKED-EXTERNAL`.
5. Prioriza correctness y datos sobre UX. Primero tests rojos reproducibles, luego cambios pequeños. Prohibido neutralizar casos de prueba, `t.Skip` injustificado, `|| true` sobre errores, métricas inventadas, “éxito” sin mutación real o afirmaciones absolutas sin evidencia.
6. Ejecuta tests destructivos solamente en HOME temporal, fixtures, contenedor o VM aislados. Nunca borres el repositorio ni datos reales de usuario. Secretos y estado runtime jamás entran en el Git tree. Antes de integrar cambios potencialmente destructivos, incorpora pruebas y dry-run.
7. En cada fase produce evidencia (comandos, salida resumida, SHA, URL del CI, benchmarks, capturas/snapshots TUI cuando proceda) y una entrada en `odd/tasks/anx-master-closure-log.md`.

### 0.2. Contratos y arquitecturas objetivo

- **Arquitectura:** monolito modular con límites hexagonales explícitos, dominio independiente de Cobra/Bubble Tea/SQL/OS; adaptadores por plataforma, proveedor, almacenamiento y shell. Interfaces pequeñas propiedad del consumidor, DI en `internal/bootstrap`, constructores explícitos, context y errores tipados; sin service locator global, dependencias cíclicas, frameworks por moda ni interfaces duplicadas.
- **Modelo:** `Observe → Resolve → Compile typed immutable Plan/Operation IR → Validate & Authorize → Execute/Export → Record → Reconcile`. Un plan autorizado produce exactamente la misma semántica de acciones independientemente de interfaz CLI, TUI o automatización.
- **Estado:** `SQLite` como fuente durable autoritativa de metadata; `CAS` externo verificado para blobs grandes; manifests/dotfiles en repositorio siguen siendo *source of truth declarativo*; jamás usar SQLite como destino de scripts o secretos indiscriminadamente.
- **Interfaces:** CLI y TUI usan el mismo control plane; contractos DTO versionados, observación real, streaming y progreso, changelogs auténticos, estado y `--dry-run` verídicos.
- **Calidad:** seguridad default deny, observabilidad local sin telemetría, fallos explícitos, idempotencia, compensaciones deterministas, portabilidad real y baja presión de memoria.

---

## 1. P0 — Windows CI actual y confiabilidad multi-OS

En CI `41232029494` Windows falló con estas causas observadas; confirma logs del HEAD antes de modificar:

1. `internal/app/TestDoctorService_Diagnose`: diagnóstico `failed` en Windows donde test esperaba `healthy/warning`. `internal/app/doctor_service.go` y `internal/security/integrity.go` interpretan `os.FileMode().Perm() & 0002` como prueba universal de directorio world-writable. En Windows los bits POSIX emulados (`0777`) **no representan ACL NTFS**. Implementar `DirectoryTrustVerifier` por OS: Unix examina owner UID/GID + modos con política explícita sobre ancestros y sticky dirs; Windows usa ACL/owner y ACE relevantes mediante API Win32, distingue `Everyone`/`Users` con permisos efectivos de escritura y maneja herencia/reparse points sin aprobar directorios realmente inseguros. Si no puede comprobar seguridad, resultado `unknown/error` definido por política, NO `trusted=true` silencioso. Evitar instrucciones `chmod` en Windows.
2. `internal/cli/TestCLIDestroyer_ResetDryRun`, `_UninstallDryRun`, `_ApplyFlagDryRun`, `TestCLIUpgradeDryRun`: mismo sello de seguridad mal aplicado al workspace CI `D:\a\env\env`. Validar escenario real NTFS y devolver códigos coherentes sin desactivar el sello. Tests con permisos inseguros reales y directorio normal.
3. `internal/security/TestCreateOriginSeal_WithRepoAndTamper`: mismo falso positivo de directorio temporal. Añadir pruebas con ACL de escritura de otros principals, propietario incorrecto, enlaces y file tampering; tests aislados que no dependan de permisos del checkout.
4. `internal/state/TestStoreMultiProcessSimulation`: se esperaban **40** registros y quedaron **27**; además `os.Rename(tmp,state.json)` falla `Access is denied`. La vieja `withWriteLock` puede continuar tras 50 fallos de AcquireLock y descarta error de `reloadFromDisk`; reemplazar por SQLite transaccional (sección 2) y garantizar aislamiento cross-process, sin `os.Rename` como primitiva multiwriter. Repetir prueba con 2, 4, 8 procesos y kills inyectados bajo Windows, Linux y macOS.
5. `go test` y `go vet` en Windows deben quedar verdes **sin** `WSL`, sin instalar compiladores extra por defecto, sin modificar tests para que finjan resultados. Cuando proceda, separar permisos POSIX de contratos portables con tests específicos por plataforma.

**Acceptance W-01..W-08:** CI nativo Windows 100% verde; sin registros perdidos; root trust real; error al denegar ACLs inseguras; dry-run sin mutaciones; `Doctor` no crea directorios ni modifica permisos durante una simple consulta; tests de regresión en mismo HEAD del PR. Logs capturados.

---

## 2. P0 — SQLite avanzado como almacenamiento transaccional de ANX

### 2.1. Driver, compatibilidad, dependencias y ADR

Evaluar con un microbenchmark funcional **antes de elegir**, con criterios ponderados: cross-compilation CGO=0, Linux glibc/musl, macOS Intel/Apple Silicon, Windows, ARM64, RSS idle/peak, binario, latencia de apertura, lecturas calientes, 100–1.000 escrituras pequeñas, multiwriter, lock conflict, integración `database/sql`, versión de SQLite, FTS5, JSON, respaldo y mantenimiento.

**Candidato por defecto:** `modernc.org/sqlite` + `database/sql`: driver SQLite puro Go/CGO-free, actualmente documentado para `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/{amd64,arm64}`. Evaluar como alternativa `github.com/ncruces/go-sqlite3/driver` (sin CGO, SQLite transpiled/Wasm + VFS, con matices de memoria/locking) o `gosqlite.org` si técnicamente demuestra ventajas y madurez. No agregar 2 motores al binario en producción salvo necesidad probada. **Termux Android/Bionic no figura como plataforma oficialmente soportada para `modernc.org/sqlite`: no declarar soporte hasta probar GOOS/GOARCH efectivos, driver, fcntl/shared-memory/WAL e integridad de datos nativamente.** Si falla, diseñar backend/adaptador Termux validado (por ejemplo, libsqlite local vía CGO **solo en compilación local Termux** si el usuario acepta sus dependencias, o fallback con semántica segura), conservando el mismo contrato `Store`. Documentar explícitamente ese tradeoff y nunca degradar silenciosamente a JSON.

No portar ejemplos Rust `rusqlite/r2d2` de la skill como dependencias de producción Go. Preferir SQL explícito y estrecho; `sqlc` es opcional solo si reduce errores y sus artefactos se regeneran de modo reproducible; ORM no necesario. Registrar driver/versiones fijadas y compatibilidad de `modernc.org/libc` conforme a la documentación; CVE/vuln audit; no usar `latest` a ciegas. Justificar decisión con `docs/adr/ADR-storage-sqlite.md`, benchmarks y restricciones.

### 2.2. Capas concretas

- Definir puertos en dominio/aplicación sin importar SQL: `StateRepository`, `ResourceRepository`, `ConfigBaseRepository`, `PlanRepository`, `JobRepository`, `EventRepository`, `HistoryRepository`, `SnapshotRepository`, `MetadataRepository`, `AuditCacheRepository`, `TransactionManager`, `StateHealth` (interfaces pequeñas y composables, no monstruo único). Conservar métodos públicos usados por `app`, CLI, TUI, bootstrap, jobs, cleanup, backups, restore y tests mediante adaptadores/transición; eliminar JSON como backend autoritativo tras migración validada.
- Implementar `internal/storage/sqlite` (o `internal/state/sqlite`, elegir una sola ruta) con `Open(ctx, Options)`, conexión/PRAGMAs, migrator, repositorios, unit of work y lifecycle `Close()`. Tests de todos los ports con implementaciones fake e SQLite real en archivo temporal. No abrir DB para `anx version`, `--help`, completion/list simple ni comandos sin necesidad: **lazy bootstrap** para cold start y RAM.
- Usar `stateDir/anx.db` o `stateDir/state.db` según ADR, única ubicación bien documentada: XDG_STATE_HOME en Linux/macOS donde aplique; LocalAppData en Windows; prefijo privado de Termux. Nunca ubicar DB, `-wal`, `-shm`, snapshots ni credenciales bajo `files/` o un checkout Git. Mode 0700 en directorio, 0600 en DB/archivos de metadata POSIX; ACL privado correcto Windows; no false positives NTFS. Cerrar todas las conexiones de forma ordenada.
- Capas `ReadModel`/`QueryService` para dashboard y comandos; `WriteService` exclusivamente transaccional y con constraints, no snapshots JSON compartidos. `Repository pattern` **sin** convertir SQLite en modelos anémicos innecesarios; no meter `*sql.DB` en dominio.

### 2.3. Esquema v1 propuesto (tipos/índices explícitos, ajustar según dominio actual)

Crear migrations SQL ordenadas, verificadas, inmutables (`0001_initial.sql`, `0002_...`), `schema_migrations(id PK, checksum NOT NULL, applied_at INTEGER)` y `PRAGMA user_version` coherente. Usar `STRICT` cuando versión SQLite y semántica admitan, `NOT NULL`, `CHECK`, `UNIQUE`, `FOREIGN KEY ... ON DELETE ...` y reglas de ownership. No forzar `AUTOINCREMENT` si IDs naturales/ULID/digest son mejores.

| Tabla / agregado | Datos necesarios y relaciones | Índices / reglas |
|---|---|---|
| `metadata` | `key PK`, `value`, `updated_at`, `revision`; config global versionada | PK, control de tipo/longitud |
| `resources` | `resource_id PK`, `kind`, `provider`, `strategy`, `version`, `managed_by_anx`, `install_path`, `origin`, `installed_at`, `updated_at`, `generation` | por `kind/provider/managed_by_anx`; solo permitir eliminación de ownership conocido |
| `config_bases` | `config_id PK`, `content_digest`, tamaño, EOL/formato, `applied_revision`, `updated_at`; contenido ancestral en CAS | FK a snapshots/CAS metadatos; sin secretos por defecto |
| `applied_hacks` | id, host fingerprint, steps/metadata validados, instante y versión | PK; índice por host |
| `host_facts` | `host_id`, facts normalized, fingerprint, `observed_at`, `expires_at`, confidence/provenance | TTL y revisión; no guardar seriales/PII sin propósito |
| `plans` | plan_id, schema version, canonical IR digest, base host revision, config/catalog revision, policy revision, target, created/expiry, authorized_at/by, lifecycle | índice `(status, created_at)`; plan inmutable tras autorización |
| `plan_operations` | `plan_id`, op index/id, op kind, target, dependencies, expected hashes, owned resource, compensation descriptor, redacted args, side-effect class | FK, UNIQUE(plan_id, index), dependency integrity |
| `jobs` | job_id, plan_id FK, idempotency key UNIQUE scoped, state, cursor, queued/started/finished, attempt, exit error code, resume token | índices de estado, tiempo, plan, dedupe |
| `job_steps` | job_id, step_id, state, attempts, start/end, pre/postconditions, result digest, error class | FK, UNIQUE(job_id,step_id), estado válido |
| `events` | secuencia monotónica PK, job_id?, type, event schema version, redacted JSON payload, timestamp, correlation id, digest/event hash optional | índice `(job_id, seq)` y `(type, ts)`; replay cursor |
| `transactions` | tx_id, job_id, lifecycle, started/committed/compensated, policy, outcome | FK e índices activos |
| `snapshots` | snapshot_id, creation, root digest, file count, total bytes, ownership, expiry/pinned | FK/integridad, índices por resource/time |
| `snapshot_items` | snapshot_id FK, path canonical, blob_digest, mode, uid/gid optional, size, target kind/symlink policy | FK, UNIQUE(snapshot_id,path) |
| `cas_blobs` | sha256 PK, size, created_at, verified_at, reachability/pin; **metadata only**, bytes en CAS externo | hash validado, ref collection/GC mark sweep |
| `tool_inventory` | id, kind, source, installed version/path, last probed, confidence, unsupported reason, provenance | cache TTL, invalidez en cambios |
| `update_history` | operation/update id, from/to version, method, digest, result, rollback, timestamp | índices por fecha |
| `security_findings` / `advisory_cache` | id/version/tool, severity, sanitized details, source, fetched/expires | TTL, índices búsqueda; sin tokens |
| `outbox` (si procede) | eventos pendientes a consumidores locales, attempt/status/lease, en transacción con la mutación | Durable outbox solo si aporta valor real |

**FTS5 opcional y con propósito concreto:** indexar descripciones públicas de herramientas, comandos, historial redacted y mensajes de ayuda para búsqueda local rápida en TUI/CLI (`anx search`). NO indexar contenido de dotfiles privados, environment, tokens, sesiones Herdr ni payloads sensibles. Evaluar si `LIKE`/índices normales bastan por tamaño; implementar FTS5 external-content + triggers/integridad solo con benchmark y test de regresión. JSON functions de SQLite para metadatos flexibles con `json_valid()` y schema version, sin reemplazar columnas tipadas de alto uso.

### 2.4. PRAGMAs, WAL y concurrencia correcta — diseño low-RAM

En **cada conexión física** establecer y probar `foreign_keys=ON`, `busy_timeout` razonable, `trusted_schema=OFF` si compatible con el driver/extensiones que uses, y políticas de seguridad; no asumir que `PRAGMA` aplicado a una conexión por `db.Exec` persiste en todas las del pool. Configurar WAL en inicialización bajo control de lock, verificar resultado; `journal_mode=WAL`; `synchronous=FULL` por defecto para transacciones de recursos/ejecución donde se exige durabilidad ante corte eléctrico, evaluar `NORMAL` como opción **explícita** y documentada cuando el SLA permita perder la última transacción ante power loss; `wal_autocheckpoint` ajustable, `cache_size` inicialmente ~2–4 MiB/conexión, `temp_store` sujeto a presupuesto y no un `mmap_size=30GB` copiado de ejemplos. **No** imponer pool de 5–8 conexiones: SQLite tiene escritor único. Empezar con **1 write connection serializada** (`SetMaxOpenConns(1)` para writer, `BEGIN IMMEDIATE` cuando corresponda), y **1–2 read connections** con presupuesto de RAM, reconfigurable después de benchmark. Nunca compartir una conexión transaccional entre goroutines de forma insegura.

- Todos los writes (incluido `generation`, jobs, events, ownership y references CAS) deben ser **transacciones**. En caso de `SQLITE_BUSY/LOCKED`, backoff bounded con jitter reproducible para tests y `context.Done`; retries únicamente a nivel de transacción **idempotente completa**, nunca duplicar efectos externos en reintentos. Propagar timeout y error de adquisición de lock. Ninguna mutación procede si falla read/reload/check/capacidad.
- Decisiones de lock: SQLite multiwriter transaccional, `BEGIN IMMEDIATE` en cambios coordinados; **no usar PID lock files como mecanismo de integridad primaria**. Locks de *operaciones externas* (instalación de un paquete/dotfile) son diferentes de locks SQL: necesitan lease con token/fencing/recovery (sección 3), pues una transacción DB no debe mantenerse abierta durante `apt`, `git`, `curl` o scripts.
- Prueba `PRAGMA foreign_key_check`, `quick_check` al arranque si hay indicios de crash o bajo demanda, `integrity_check` en `anx doctor --deep`, con límites; `PRAGMA optimize` de mantenimiento con cadencia, ANALYZE selectivo; índices guiados por `EXPLAIN QUERY PLAN`. WAL checkpoint PASSIVE/RESTART/TRUNCATE según actividad para evitar crecimiento indefinido; no bloquear UI en startup. `incremental_vacuum` solo en idle/on-demand si `auto_vacuum=INCREMENTAL` se habilitó correctamente. No VACUUM periódicos agresivos ni polling continuo.
- Tratar FS no local, red, FAT/SD externa y Android storage compartido como **no garantizados** para WAL/locking: detectar, documentar, y rechazar con error accionable o usar modo conservador probado; no desactivar locking (`nolock`) para evitar fallos. Estado y cache pueden tener distinta política de fsync; no prometer aislamiento `SERIALIZABLE` de operaciones del host solo porque SQLite es serializable.
- Indexar consultas críticas de dashboard, inventory, plan, job replay y tool statuses. Actualizar `last_observed` de forma batched/debounced (no INSERT por frame de Bubble Tea). `Close()` limpia escritor/lectores; no forzar checkpoints durante cada GET.

### 2.5. Migración **sin pérdida de datos** JSON → SQLite

1. Detectar el `state.json` de instalaciones actuales y todos los datos existentes (InstalledRecord, ConfigBaseRecord, AppliedHackRecord, ActiveTheme, ActiveProfile, RepoRoot; además JSONL journals/caches según inventario). Versionar mapeo y preservar UTF-8, timestamps, bytes, hashes, archivos no reconocidos y relaciones. SQLite debe poder abrirse aun si se corre una versión anterior del CLI: documentar política de downgrade y avisos.
2. Preflight: validar permisos/ACL, espacio libre, checksum/source backup, JSON bien formado, compatibilidad de esquema y conflicto entre procesos. Crear **backup recuperable** con nombre y hash (`state.json.pre-sqlite.<timestamp>.bak`) y preservar original, **nunca borrar automáticamente** hasta período de retención/paso explícito.
3. Transacción única para import + `migration_marker(source_sha256, schema_version, imported_at)` + validación de conteos y hashes. Si falla, SQLite permanece sin import parcial; reintento idempotente. Quiebres/kill en cada etapa, 2 agentes migrando a la vez, disco lleno, archivo read-only y Windows locking tienen pruebas específicas. No marcar éxito antes del COMMIT y `foreign_key_check`.
4. Apertura normal nueva: el SQLite es autoritativo. Si hay ambos, la política de precedencia no debe inferir silenciosamente; si hay datos divergentes generar error/guiar resolución. Dar `anx state migrate --plan`, `anx state migrate --apply`, `anx state status`, `anx state backup`, `anx state verify`, `anx state repair --plan`, `anx state export --format json`; `--dry-run` cero mutaciones y no crear DB por accidente. Interfaz existente `state.Store` puede adaptarse para compatibilidad temporal; eliminar `atomicWrite` concurrente roto tras migración.
5. Backup consistente SQLite con online backup API del driver o `VACUUM INTO` verificado y aislado; **nunca copiar solo el archivo `.db` mientras WAL está activo**. Restore requiere confirmación, plan, snapshot previo, integridad y apertura sin handlers activos. El WAL/SHM no debe sincronizarse por Git ni incluirse en dotfiles.

### 2.6. Patrones avanzados útiles, no ornamentales

- **Unit of Work:** commit atómico de `plan authorized + job queued + initial events`; `step result + resource ownership update + append event`; `transaction outcome + backup references`. Ningún consumidor ve un resultado a medias.
- **Optimistic concurrency:** `revision` y `UPDATE ... WHERE revision=?`, verificando `RowsAffected=1`; evitar TOCTOU entre plan y apply; replay idempotente con unique keys y hashes de operación.
- **Event log durable** append-only de eventos de control-plane con cursores monotónicos y retención acotada. El historial permite explicar y reanudar, pero jamás sustituye los datos autoritativos del estado.
- **Materialized read models:** usar vistas SQL o tablas derivadas solo si perfilado demuestra ventaja; actualizar en la misma transacción. No crear colas/event sourcing puro solo por moda.
- **CAS split:** bytes grandes del snapshot en disco direccionado SHA-256; SQLite para metadata y reachability. Para consistencia ante crash, etapa CAS verificada fsync → commit transaccional referencias → GC conservador de blobs huérfanos con grace period y revisiones, nunca borrar blobs solo por contador inconsistente. Fail injection entre cada paso.
- **Provenance:** registrar fuente real, versión/tag/commit, estrategia efectiva, checksum, advertencias, duración, outcome/rollback y límites de privacidad. No persistir environment secrets, credenciales ni token interpolado en comandos; guardar descripciones redactadas/HashMAC si hace falta correlación.
- **Mantenimiento inteligente:** compactación/TTL de logs y cache, límites de DB/WAL/blobs, `anx state doctor`, `anx state stats` (páginas, DB, WAL, cardinalidad por tabla, tiempos sin PII) y limpieza transaccional/GC sin intervención constante. Índices y queries contra volumen de 100/1k/10k recursos y 100k eventos.

### 2.7. Invariantes SQL verificables

- Dos o más procesos escribiendo concurrentemente jamás pierden registros; si no pueden obtener writer lock, **uno falla con error tipado** sin escribir estado parcial.
- `READ` no observa transacciones a medio confirmar; JSON heredado migra una sola vez; rollback restaura exactamente source.
- WAL sobrevive SIGKILL/corte simulado y recuperación; `STRICT/FK/CHECK` impiden estados imposibles; `context` cancela queries esperando locks.
- Suite de tests DB *en archivo temporal real*, no solamente `:memory:`: WAL, file locks, reinicio, migración, Windows, power-loss cases. Medir Go race, RSS, cold start y regresión del tamaño de binario.

---

## 3. P0 — Control Plane: planes exactos, jobs durables, operaciones atómicas

Reemplazar `internal/app/control_plane.go` con contratos claros que utilicen SQLite y los plan/operation schema versionados. Evitar reconstruir un plan diferente entre `BuildPlan` y `Apply`; `plan.Digest()` por sí solo no autoriza una acción ni detecta mutaciones del host si no incorpora precondiciones relevantes.

- `Plan`: `id`, digest canónico SHA-256, schema_version, source manifest SHA, host fingerprint/revision, config content hashes, policy digest, generated_at, expiry, resource owners y operaciones ordenadas por DAG. Guardar/autorizar plan inmutable. `Apply` debe recibir plan_id + digest + expected hostRevision y rechazar `stale`, `unauthorized`, `expired` con códigos distintos; re-evaluar precondiciones atómicas por recurso justo antes de mutar y volver a comprobar tras operaciones susceptibles de TOCTOU.
- `Job`: state machine explícita `queued/running/recovering/partial/succeeded/failed/cancelled`, progresos y errores persistidos en SQLite. `Cancel` sobre proceso actual y recuperación tras reinicio; no confiar en `activeJobs map` como único estado durable. Durante restart clasificar jobs running de forma segura y ofrecer `anx resume --plan`/`anx resume --yes`, con compensaciones idempotentes. `anx jobs`, `anx jobs show`, `anx jobs events --since`, `anx jobs cancel`, etc. No daemon permanente obligatorio.
- **Effects protocol:** `prepare → verify preconditions → reserve external resource lease/fencing → execute → verify postconditions → commit state/event → release lease`. La DB no ofrece transacción distribuida con apt/git/filesystem: usar write-ahead intent, idempotency keys, compensaciones, journals, CAS snapshots, `unknown` status tras crash y reconciliación explícita. Nunca marcar succeeded antes de verificar efecto externo.
- `Observe`: errors producen `unknown/error`, nunca `healthy=true` por fallback. `Explain` cuenta fuente real, estrategia usada, evidencia y diferencias.
- **DAG:** mantener nuevo `internal/execution/graph.go` y `pool.go` si pasan todos los tests: `DependsOn` respetado, indegrees, locks, fairness, cola ready determinista, exclusividad y rollback topológico inverso. Agregar tests multigeneracionales, ciclos, dependencias ausentes, locks que no hambreen otros, errores, stop-on-first-failure, goroutine leak, cancelación, 1/2/8 workers y reinicio. No reescribir algo que ya está correcto.
- Cache/metadata de planes con TTL y invalidación en cambios de manifiestos/host. FTS/SQL no entra en la ruta crítica de ejecución salvo beneficio medido.

**Acceptance CP-01..CP-12:** mismo plan autorizado o rechazo explícito; replay exacto de eventos; idempotencia cross-process; rollback consistente; cancel/restart; 0 acciones para clean repeat install; locks respetados; `anx jobs` CLI/TUI consistentes.

---

## 4. P0 — Operation IR tipada: traducción a shell mantenible, escalable y determinista

### 4.1. El problema y las correcciones precisas

Hoy `internal/scriptgen/builder.go` serializa strings y `TranslatePlan` usa atajos por proveedor; `internal/cli/packages.go:457+` duplica traducción (incluye interpolación de `pkg` sin quoting correcto); `internal/cli/configcmds.go:313+` emite `cp -a` genérico y no preserva merge, backup, idempotencia ni hash. `internal/scriptgen/translators.go` usa `rm -rf` genérico y delega operaciones complejas a `anx`. `SetDeterministic` es opcional. Esta generación NO es un contrato de equivalencia.

### 4.2. IR única — entidades y contrato inmutable

Definir paquete de dominio `internal/operation` o `internal/planir` sin imports de CLI/SQL/OS/terminal. Estructuras y enums tipados con validación en constructor, versiones semver de esquema, JSON canónico y digest; no `any` arbitrarios salvo payload tipado por kind.

Sugerencia de `OperationKind`: `ExecArgv`, `DownloadVerified`, `EnsureDirectory`, `CopyAtomic`, `WriteAtomic`, `DeleteOwned`, `CreateSymlinkOwned`, `PackageInstall`, `PackageRemove`, `PackageUpgrade`, `GitFetchPinned`, `BuildFromSource`, `VerifyChecksum`, `VerifySignature`, `ConfigMergeThreeWay`, `SnapshotCreate`, `SnapshotRestore`, `ServiceRestart`, `HostSettingSet`, `RecordState`, `Compensate`, `AnxRuntimeDelegation`. Cada operación porta `op_id`, `resources`, `depends_on`, `locks`, `target_os/arch`, `provider_id`, argv exacta y no concatenada, permisos esperados, pre/postcondiciones, expected before hash, after digest opcional, secret refs **nunca el valor**, side-effect classification, ownership, idempotency key, rollback policy, estimated risk, provenance.

- `CompileIR(Plan, PlatformFacts, Policy)` hace selección proveedor, resolución de dependencias, estrategia e invariantes una sola vez. Inyectar `Clock`, `FileSystem`, `ProcessRunner`, `ProviderRegistry` donde lo requiera la compilación, no en el dominio puro.
- **GoExecutor** ejecuta IR sin shell implícito, usando `exec.CommandContext(executable, argv...)`, timeouts, stdout/stderr streaming acotado, cwd/env allowlist, cancel, locks y postconditions; registra lifecycle en SQLite. Una excepción shell se representa explícitamente con script trusted+hash y se ejecuta mediante intérprete elegido, no concatena strings del usuario.
- **Renderers**: POSIX `sh` cuando sea expresable; Bash cuando realmente se necesitan features; PowerShell nativo en Windows; si Windows no necesita script exportado, mantener compilador PowerShell testeable y documentar cobertura parcial. Cada backend tiene capacidades declaradas `CanRender(op)` y `Render()` que devuelve error tipado `UnsupportedOperation` con opID/razón en lugar de omitir/`echo`/fallar silenciosamente. Escape correcto por shell: argv/path/whitespace/quotes/newlines/unicode/special chars. **NO** usar `%q` de Go como reemplazo de POSIX shell quoting; **NO** construir `sudo ... %s` con datos del manifest sin encoding.
- `anx plan --format json` expone IR sanitizada, `anx apply --script --target sh|bash|powershell` produce artefacto reproducible y versionado, `--script-mode` explícito `standalone` vs `delegated`: el standalone no debe requerir ANX; el delegated debe declarar versión y hash exactos de ANX y sus operaciones no standalone. Si no es posible preservar snapshots/merges con shell, rechazar standalone con razón y ofrecer delegated. Separar script **preview no ejecutable** de script **ejecutable fiel**, con encabezado que no mienta.
- `Deterministic`: mismo IR + target + flags + versión de compiler produce mismos bytes SHA-256; sin timestamps/random/host names accidentales en output; información temporal solo en manifest de procedencia externo o metadata opcional excluida del modo reproducible. Orden estable, normalized line endings, mismo encoding, quoting, no iteración de maps arbitraria.
- **Seguridad:** no ejecutar fragmentos crudos provistos por `AuditFinding.RemediationCommand` o `RebuildResult.Commands` sin trust tier, validación de origen, confirmación y sandbox apropiado; limpieza nunca `rm -rf` sin ownership, canonical path, symlink guard y precondition/hash; no renderizar scripts que eludan límites de recursos, secretos o permisos. Scripts generados deben tener preflight y error trapping, sin alterar host ante mismatch.

### 4.3. Pruebas diferenciales que demuestran equivalencia

Crear test harness que aplica misma IR contra **dos HOME clonados** aislados: `GoExecutor` vs generated shell (`sh`/Bash/PowerShell), comparar árbol de archivos normalizado, permisos disponibles, symlinks, installs simulados con fake PMs, estado de recursos/snapshots/historial y exit status. Si una operación solo es delegable, contrastar ejecución delegada contra GoExecutor. Tests golden por distro/OS, `shellcheck`, `sh -n`, `bash -n`, PSScriptAnalyzer, rutas con `'`, `"`, `$()`, `;`, espacios, `%`, nueva línea, Unicode, rutas Windows, symlink evasions, `$HOME` vacío, argumentos maliciosos, exit codes y cancelación. Aleatorizar orden de inputs semánticamente equivalentes y comparar digest. Probar que **script fail-closed** en operaciones no soportadas y NO imprime éxito tras fallar un paso.

**DoD:** NO quedan dos traductores rivales (`TranslatePlan`, `renderPlanScript`, `renderSyncScript`) generando distintas semánticas. Sin mocks que solo busquen `Installing package...` o un shebang; tests de efectos reales/fixtures. Prohibidas interpolaciones sin quoting, script output reproducible, fallback honesto.

---

## 5. P0 — Instalador memorable, pulcro y simple; Cloudflare Worker + GitHub Releases

### 5.1. Experiencia ideal de usuario

Recomendación pública: **`sh.arturonavax.dev`** como endpoint de bootstrap y **`env.arturonavax.dev`** como landing/docs; `cmd.arturonavax.dev` queda reservado como alias futuro si se necesita. No afirmar que DNS/Worker ya existen. Comando principal deseado para Linux/macOS:

```sh
curl -fsSL https://sh.arturonavax.dev | sh
```

Advertencia operacional: POSIX `sh` no tiene `pipefail`; una respuesta vacía en fallo de `curl` puede llevar a estado de salida 0 del último proceso. Documentar alternativa segura para automatizaciones: `curl -fsSLo /tmp/anx-install.sh https://sh.arturonavax.dev && sh /tmp/anx-install.sh` o descarga a tmp seguro. El instalador verifica que recibió contenido válido cuando sea posible, rechaza env override insecure por defecto y nunca informa éxito sin verificar `anx version`/instalación real.

**Preflight ultra-ligero, POSIX:** sin clonar repo ni instalar bash/go/curl adicional para simplemente presentar wizard; auto-detect `uname`, CPU, libc glibc/musl, package manager, Mac ARM64/Intel, distro, HOME/XDG, TTY real, ruta bin, permisos, release compatible. Sin TTY: `--mode release|source|plan`, `ANX_INSTALL_MODE`, o default conservador claro; sin preguntas bloqueantes. Con TTY: wizard compacto vía `/dev/tty` (stdout del pipe separado), sin exigir Bubble Tea antes de tener binario:

```
  ANX Setup  ·  linux/arm64  ·  Alpine (musl)
  Choose installation method
  ▸ Verified release           Fast · no compiler needed
    Build from source          Install Go only when required
    Preview                    No changes
    Exit
```

Default **release verificada** solo si existe asset real, compatible y autenticado; si no hay release, mostrar el motivo y proponer source **antes** de descargar Git/Go (no fallback silencioso a costos/dependencias no previstos). Confirmar destino/prefix y conflicto de binarios/aliases. Visual limpio: pasos, progreso, errores accionables, duración real opcional, sin ASCII art enorme ni logs de librerías por defecto, `--verbose` para debugging. Después: wizard Go/TUI opcional para perfiles/paquetes/hacks; distinguir **instalar ANX** de **aplicar el entorno** (no mutaciones grandes inesperadas). SSH/headless/CI completamente atendidos.

### 5.2. Modo Release (Linux/macOS; Windows no precompilado por defecto)

- Build matrix release: Linux amd64/arm64 con variantes compatibles para glibc/musl según prueba real, darwin amd64/arm64, checksums SHA256SUMS, firma Ed25519 sobre SUMS u otro formato con identidad pública fijada y política fail-closed, manifest de platform/arch/libc/version/commit/policy, SBOM/provenance. Publicar desde tag semver protegido mediante workflow con permisos mínimos, version derivada del tag; instalar sin clon, sin Go y sin instalar paquetes globales. GoReleaser OSS o build workflow Go simplificado, decidir en ADR en función de costo, estabilidad y supply chain; NO exigir herramientas privadas.
- El `releasePublicKey` en `internal/selfupdate/buildinfo.go` debe fijarse para releases oficiales mediante ldflags o equivalente auditable, con rotación documentada. **Nunca** aceptar un checksum si no está presente, si hay duplicados/ambigüedad, si no existe SHA implementation, si hash no cumple 64 hex o si la firma requerida falla. La firma Ed25519 para archives/releases **no es** Authenticode; su clave puede generarse gratuitamente, pero su custodia/seguridad requiere CI secrets/identity. Preferir OIDC/keyless attestation si realmente se verifica offline/en el bootstrap, no añadir una firma decorativa.
- Download bounded + timeout + retry acotado + TLS, checksum/firmas, extraction safe (`../`, symlink tar paths, zip bombs, duplicados, tipos inesperados), staging de ejecutable verificado, modo 0755, versión/GOOS/GOARCH self-test, atomic rename/backup rollback, dirs 0700, preservar binario ajeno y PATH. Nunca `tar` directo al bin final. No instalar archivos de repo completo por ruta release; dotfiles opcionales se obtienen luego bajo control de manifests.
- Release asset faltante o arquitectura no soportada: error claro y oferta de source; `--non-interactive` solo hace fallback si política/flags lo permiten. Al instalar desde release, **no fingir acceso a dotfiles que no vienen en el artefacto**; catálogos embebidos vs `anx repo clone` explícito, con versión/tag pinned y provenance.

### 5.3. Modo Source (todos los OS soportados, incluidos Windows)

- Implementar bootstrap idempotente con procedencia Git/pin tag/commit, comprobación versión Go mínima actual desde `go.mod` **Go ≥1.27.3** salvo actualización validada posterior; instalar Go solo tras consentimiento/mode source, detectar GVM/Go existente evitando degradaciones, upgrades en stage sin borrar versión anterior, fallback basado en OS/distro, logs sin secretos. `go build -trimpath`, checks y target bin, detect platform, protect user files/aliases. El `go install ...@latest` no es equivalente a compilar `main`: distinguir `--source=main`, `--source=tag`, `--go-install=module@version`, registrar version/provenance, evitar no-determinismo implícito.
- **Windows:** instalador `install.ps1` nativo source-first (PowerShell 5.1+ o versión mínima validada), Git/Go auto-detect y auto-install opcional con consentimiento, build y PATH en user scope, sin WSL, sin certificado y sin necesidad de computadora física Windows; CI Windows nativo prueba lo que un PC Windows probaría. Mantener `install.bat` thin wrapper opcional. `windows/amd64` y `windows/arm64` **compilación y pruebas de compatibilidad** donde existan; **NO** producir ni publicar releases precompiladas Windows por defecto ni comprar Authenticode. Si en el futuro se activa, feature flag y job gated `publish_windows_assets: false`, signing opcional con KeyVault/cert configurable, con smoke/SmartScreen docs; no introducir release Windows en metadata público cuando no exista.
- **Termux:** no tratar Ubuntu `linux/arm64` como Android. Construir desde source en Android ARM64 con Termux instalado, prefijo correcto, librerías requeridas (especialmente SQLite), `pkg` y directorios privados; compat GOOS/CGO y scripts se valida con dispositivo/emulador real, gate `manual/conditional` hasta tener runner. Sin root.

### 5.4. Cloudflare Worker — infraestructura versionable, sin depender de desplegarlo aquí

Implementar `deploy/cloudflare/` con Worker mínimo y tests. Ruta `/` entrega **bootstrap POSIX versionado de commit/tag conocido** desde artefacto publicado/controlado, no `main` mutable al vuelo sin validación; `/install.sh`, `/install.ps1`, `/healthz`, opcional `/.well-known/anx-install-manifest.json`. Definir cache/Etag/Content-Type, HTTPS, redirect policy, CSP si página, controles de origen/method y sin enviar datos de host ni telemetría. Evitar servir Powershell en path que el usuario pipa a `sh`; docs Windows comando PowerShell separado (e.g. `irm https://sh.arturonavax.dev/install.ps1 | iex` con advertencia de seguridad y alternativa descarga+verificación). Crear pruebas con Miniflare/workerd o equivalentes actuales; `wrangler.toml` sin credenciales, ejemplo para rutas/domains y dry-run. No desplegar ni tocar DNS sin permisos/acción explícita del titular. No inventar coste ni funcionamiento del dominio.

### 5.5. Self-update real y sin dependencia circular

Preservar y probar nuevo `internal/selfupdate/{updater,release,verify,extract,replace,buildinfo}`. Los métodos declarados `release`, `go-install`, `source` deben comportarse distinto. Si origen `source`, no sobrescribir silenciosamente árbol git o binario firmado; ofrecer `anx upgrade`/`anx self update --method go-install|release` con contrato claro. Si Windows no tiene release, resolver `source`/`go-install` o error comprensible **sin** inventar asset Windows. Tests mock HTTPS, firma correcta/falsa, checksum malformado, nombre equivocado, TOCTOU, red caída, disco lleno, locked exe Windows, rollback y recovery; version no monotónica, downgrade explícito, prerelease opt-in, idempotencia. `--check` y `--dry-run` sin mutación de DB/red excepto consulta read-only permitida por flags; `--offline` no red.

---

## 6. P0/P1 — Seguridad integral de origen, dotfiles, plugins y rollback

### 6.1. Herdr, captura y protección de datos sensibles

- Dotfiles repositorio `files/` solo recibe archivos permitidos por manifest; para Herdr **únicamente `files/herdr/config.toml`** hasta que se amplíe allowlist con aprobación. Sockets, `session.json`, snapshots de sesiones, `.plugins.lock`, caches, credenciales, logs y nuevas rutas desconocidas quedan fuera de captura, staging y Git. `.gitignore` es defensa secundaria, no frontera.
- Investigar read-only si `~/.config/herdr` es symlink a `files/herdr` o si otra rutina copia directorios. Separar config/state/runtime (XDG) sin perder sesiones; no inventar flags Herdr.
- `capture → ChangeSet → security scanner → isolated Git index → commit → optional push` solo stagea archivos aprobados; ninguna pre-staged file ajena entra en commit. Symlink/path traversal/case-insensitivity/hardlinks/WINDOWS reparse pts/TOCTOU; pruebas HS01..HS12; jamás secretos aunque nombres nuevos no aparezcan en ignore. Deny unknown and fail closed.
- `--dry-run` global no escribe: ni state DB, snapshots, bootstrap, branch/commit/push, descarga, shell side-effects ni aliases. Consultas remotas solo si flag online y con intención clara; reportar resultado `would_*`, no `captured`/`applied`.

### 6.2. Extensiones y tools aisladas

- Conservar mejoras de `internal/extension/admit.go` (SHA-256 pin, declaradas vs permitidas, owner trust antes de exec), `host.go` (env scrubbed, scratch, process group, frame/message limit). **No confundir verificación de binario con sandbox de filesystem/network/process:** un binario confiable puede leer otras rutas accesibles al usuario. Para extensiones no confiables diseñar adaptadores por OS: Linux namespaces/bwrap o estrategia probada (dependencias opt-in), macOS sandbox equivalente realmente disponible y mantenido, Windows AppContainer/Job Objects/low integrity si factible; declarar `trusted-only` si no hay aislamiento implementable. Capability lease y mediación explícita de operaciones, sin paths arbitrarios de child, timeouts y limitación de recursos. Test anti-exfil/traversal/evasión donde haya sandbox verdadero.
- Auditoría, logs, snapshots y eventos SQLite nunca incluyen valores secretos; scanners pre-commit y CI. Seguridad `OriginSeal` usa ACL Windows real, POSIX owner mode, hashes, provenance. Tools external manifiestos estrictos y fail closed (ya mejorados); controlar ejecuciones desde PATH solo cuando `allow_global_path` explícito; la ejecución externa no pasa por shell a menos que se declare interpreter trusted.
- `rm -rf` y uninstall/delete/cleanup solo tras demostrar ownership, preconditions/hash, backup y confirmación; no borrar symlinks ajenos ni sobrescribir `.cmd`/alias ajeno. Noninteractive exige `--yes` y policy, `--force` no debe eludir seguridad crítica.

### 6.3. Merge semántico y CAS

- Configs: 3-way con ancestor persistido SQLite + blob CAS, detección simultánea local/repo, formato JSON/JSONC/TOML/YAML cuando seguro, preservar comentarios/EOL/format/perms o conflicto explícito; no fusionar semánticamente si parser no puede preservar info. Sane `backup`, `diff`, `restore`, `capture`, `sync` y `rollback` comparten motor y policies; no duplicar lógica CLI/TUI.
- CAS: verificaciones SHA-256 en lectura/escritura/reuso (mejora ya presente), archivo corrupto/reemplazo/dupe race/GC, no usar size-only; metadata en SQLite, blobs fuera con naming defendible, fsync y no seguir symlinks; fail injection y tests multi-proceso/Windows/macOS.

---

## 7. P1 — Tools completamente independientes por diseño

**Objetivo contractual:** cada tool personal (Go, Rust, Python, Node, Bash, etc.) podría moverse a su propio repositorio, ejecutarse sin ANX, distribuirse y actualizarse independientemente. ANX es el orquestador **opcional**, no SDK obligatorio ni monolito donde reimplementar su negocio. No extraer los componentes core de ANX de manera artificial.

- Manifiesto público versionado (`tool.api_version`, `id`, displayName, language, executable, entrypoint/interpreter, supported OS/arch, install/source/release/package strategies, version command/parse rules, capabilities, dependencies, update/rollback protocol, trust tiers, permissions/side-effect declarations, checksums, built-in fallback, PATH opt-in, isolated bin dir, optional health check). Schema validado con errores precisos; manifest overlay atómico/fail closed, no executing code on discovery.
- `ToolResolver` priority explícita: pinned isolated tool binary → per-user overlay → opt-in global path → built-in compatibility fallback, sin colisiones con sistema ni `~/.local/bin` genérico; Windows `.exe`/`.cmd`/`.bat` mediante runner adecuado; semver parsing y tool-specific args; no ejecutar `--version` indiscriminadamente cuando pueda mutar. Separar `Install`, `Upgrade`, `Status`, `Run`, `Uninstall` en adapters y contratos, no strings de comandos arbitrarios.
- Cuando una funcionalidad actualmente compilada dentro de `internal/ai` sea una **tool personal autónoma**, plan de extracción incremental a repo/proyecto/cmd independiente con CLI contract JSON/stdio, tests independientes y paquete de distribución; el built-in existente sirve transición/compatibilidad, no dependencia obligatoria del tool. Mantener `anx ai`/`aitools` aliases sin duplicar implementación funcional. Probar compilación y ejecución standalone *sin* import de `github.com/arturonavax/env/internal/*` y sin `anx` en PATH.
- Dashboard tools real muestra inventario, instalado/disponible, versiones, update strategy, provenance, PATH, health, fallos, actualizaciones y changelog autentico. Node 26 LTS para toolchains Node si está soportado, Go ≥1.27.3 y versiones sin downgrade, GVM/fnm/pyenv/Bun/pnpm/Cargo; opcionales Nix/Volta solo si hay política comprobada, no obligación de introducirlos sin necesidad.

---

## 8. P1 — CLI, TUI, UX y consistencia observable

- CLI Cobra/Viper: opciones, precedencia **flag explícito > ANX_* env > config.yaml > default**; `PersistentPreRunE` o init común seguro; help/version baratos sin DB; `SilenceUsage`/`SilenceErrors` según estrategia, stdout solo datos pipeables, diagnósticos/logs/progress stderr, JSON/YAML sin ANSI, errores tipados/códigos estables, `signal.NotifyContext` y Ctrl-C=130, short help/completions Bash/Zsh/Fish/PowerShell, validators y aliases documentados. `--offline` real, `--dry-run` sin mutación, no telemetry.
- TUI Bubble Tea y CLI comparten `ControlPlane` Query/Command ports; observación no cambia estado ocultamente. Dashboard Tokyo Night por defecto y temas alternativos, responsive 60/80/100/160 columnas, verdadero estado de paquetes, herramientas, configs, sincronización, diffs, conflictos, jobs/events/logs, security/hardware, updates, changelog GitHub real. Sin bloqueos por queries SQL en render; paginación, debounce, cancel, accesibilidad/contraste, navegación teclado, modo SSH/headless, RAM idle medido; no reintroducir acoplamiento de servicios concretos en componentes UI.
- Mostrar origen repo/commit, perfil, modo origen de instalación, versión, recursos, avisos y errores actionable. Un comando fallido no puede pintar status `green`. `Doctor` solo lee salvo `--fix` explícito y consentido. Opciones `--script`, `--plan`, `--format`, `--dry-run` consistentemente en toda CLI.
- Preservar `anx reset --destroyer`, `uninstall`, `restore`, `repo clone/set/status`, `anx update/upgrade/sync/self update`, wrapper `rg/fd`, `aitools`, scripts y compatibilidad anterior. `Destroyer` nunca borra ajeno; preflight/preview/backup/confirm/rollback por fases.
- `anx state ...`, `anx jobs ...`, `anx scripts explain/compile/verify` comandos adicionales solo cuando sus casos de uso estén probados y no dupliquen comandos existentes.

---

## 9. P1 — GitHub Actions E2E SIN duplicados y supply-chain

### 9.1. CI actual a preservar y simplificar

En el run base macOS nativo Apple Silicon `macos-26-arm64` aprobó con Go 1.27.3; lo mismo Ubuntu/ARM64 QEMU, matriz distros y Docker full Debian/Alpine. **Windows es rojo.** No volver a Intel. Mantener gates nativos Linux/Windows/macOS; usar `actions` versiones actuales verificadas y preferentemente pin SHA tras test. Preferir runner Ubuntu pin `ubuntu-24.04`, no `ubuntu-latest` si se quiere evitar migraciones involuntarias. GitHub Actions está disponible para compilar/testear Windows sin PC física. No exigir coste de firma ni key vault para CI.

**Diseño sin duplicados:**

| Gate (obligatorio salvo explicado) | Infraestructura | Qué valida **y qué no vuelve a validar** |
|---|---|---|
| `unit-linux`, `unit-macos-arm64`, `unit-windows` | runners nativos | Go tests, vet/build, smoke integrado específico, SQL/integrity/IR tests, race cuando soportado |
| `static-security` | Ubuntu | gofmt, shellcheck **mandatory**, govulncheck, secret scan, schema/migrations, `go mod verify`, SBOM básico; cachés verificadas |
| `linux-smoke-runtime` | contenedores **Ubuntu, Fedora, Arch** | detección distro/managers y runtime CLI/plan/status; *NO repetir* Debian y Alpine si full E2E ya valida los mismos asserts |
| `linux-full-e2e` | Debian + Alpine (mock services) | instalación release/source/curl + repo/config/snapshot/rollback + PTY + SQLite persistence + tests musl; independencia completa |
| `arm64-linux` | QEMU o runner nativo arm64 | binario real arm64/SQLite/WAL/permisos, no Android Bionic |
| `macos-native-e2e` | `macos-26` arm64 | POSIX installer bootstrap + fake signed release source + PATH/TTY/symlink/SQLite WAL + rollback; tests mac específicos, **no otro go test completo** |
| `windows-source-e2e` | `windows-latest` | PowerShell `install.ps1` con source fixture, Go mínimo, DB concurrencia/ACL, `.cmd/.exe`, PATH, update source y cleanup; **sin precompiled release** |
| `termux-native-conditional` | Android ARM64 con Termux real/emulado | PREFIX/GOOS/SQLite driver/locking/pkg/source install/CLI; si no hay runner informar `UNVERIFIED`, no fabricar resultado |
| `script-ir-differential` | fixtures multiplataforma | efectos iguales Go vs compilado sh/PS, quoting, plan hash, fail-closed |
| `release-publish` | tags protegidos + upstream | solo Linux/macOS release reproducible firmado + SUMS + SBOM/provenance; descargar/testear assets después de publicación en sandbox |

- Reutilizar **un binario static Linux AMD64** para matrix smoke, sin build por distro; usar artifacts de corta retención. No construir binarios nativos para smoke si no hacen falta. E2E full en Debian/Alpine puede construir Go fuente como parte del escenario, lo cual **no** equivale a trabajo duplicado; medir duración y separar stages.
- Eliminar/renombrar `tests/e2e/scripts/test_concurrency.sh` si solo reitera plan/status/sequential: cambiar a tests reales de concurrencia en Go/sqlite file-backed multi-process, no sumar tests solo por cantidad. `test_from_sources.sh` y `test_github_release.sh` deben usar mock source/release, no depender de conectividad a GitHub ni instalar globalmente en host.
- No invertir dependencia `e2e needs: [test, lint]` que causaba skipping cuando Windows falla; ejecutar otros diagnósticos en paralelo/needs acotados y exigir **todos** los gates en branch protection, con agregador status final cuando útil.
- Docker por defecto en Actions: no reemplazar por Podman si no hay benchmark probado de seguridad/tiempo/soporte/colas; se puede permitir Podman como alternativa local sin duplicar CI. Contenedores no-root, no-new-privileges, cap-drop, readonly, tmpfs, network none cuando el escenario no requiere mock; mock API privado cuando sí, CPU/RAM/pids limits. Imágenes pin digest después de validar actualizaciones y escanear, nunca permitir tag mutable como garantía única de reproducción.
- Code coverage solo métrica auxiliar; validar invariantes con failure injection, race, fuzz, heap profile y tests adversariales. Fuzz `ParseManifest`, path canonicalization, script quoting, tar/zip extraction, JSONC/TOML merges y SQLite migration. Ejecutar `go test -race` Linux/macOS/Windows cuando soporte; timeout y resource budgets. Reportar RSS/cold start/TUI idle y tamaño binario vs baseline **medido**, no inventado.
- CI requerido en PR; `main` protegido con review/required checks (el repositorio reportaba `protected=false` en auditoría; si no hay permiso admin documentar ajuste UI/API). `GITHUB_TOKEN` mínimo privilegio, publicar release con job separado con permisos `contents:write` solo allí, evitar `pull_request_target` en código no confiable, permisos y secrets seguros. No auto-merge de PR rojo.

---

## 10. Requisitos de seguridad SQLite y política de datos

1. **SQL parameterization exclusivamente** (`?` o named params) para valores; identificadores dinámicos vienen de allowlist cerrada, nunca interpolar `sort/table` libre. Validar rangos/longitudes, regex IDs, UTC timestamps; NUL/Unicode/input malicioso. SQL injection cases para todos los repositories (no solo uno).
2. No guardar contraseñas, tokens o comandos con argumentos secretos sin redacción; extensiones no heredan env salvo allowlist; SQLite DB **no está cifrada automáticamente**. Cifrado at-rest opt-in con keyring nativo/SQLite-compatible driver únicamente si seguridad, compatibilidad, respaldo y RAM están demostrados. **No** llamar `sha256` cifrado ni asumir que permisos 0600 bastan contra malware del mismo usuario.
3. `PRAGMA trusted_schema=OFF` donde útil, foreign keys por conexión, no permitir `ATTACH` arbitario ni user-defined SQL/plugins sin confianza. `query_only=ON` para lector si existe safe connection contract, sin compartir conexión en transacciones alteradas; no habilitar `load_extension` para usuarios.
4. Operaciones backup/export/import/doctor hacen verificaciones `integrity_check`, firmas/checksums y no incluyen credenciales en logs. Corrupted DB ofrece modo seguro read-only / restore explicit, sin reset automático. Filesystem error y falta de espacio deben dejar DB recuperable.
5. Revisión de amenaza: atacante con control de repo yaml/archivos en `files`, release GitHub/Worker, proceso local multiusuario, symlink race, extensión maliciosa, red hostil, rollback y contaminación de logs. Cada amenaza tiene un test negativo trazado y una defensa concreta.

---

## 11. Plan de ejecución por vertical slices (no abrir 20 refactors a la vez)

| Slice / prioridad | Entrega funcional al final del slice | Gate mínimo |
|---|---|---|
| **S00 — Baseline** | Revalidar SHA/CI, matriz AC/V/HS, ADR de driver/ABI/Termux, tests rojos Windows, inventario releases | build y tests baseline documentados |
| **S01 — Windows trust** | ACL verifier, doctor sin mutación y pruebas ACL reales | `internal/app`, `cli`, `security` Windows green |
| **S02 — Storage kernel** | SQLite driver, PRAGMAs/connection policy, migrations, SQL repositories, UoW, test multiwriter/Win | `state` Linux/Win/mac/race/file WAL green |
| **S03 — Data migration** | import JSON/JSONL sin pérdida, rollback, backup/restore, CLI state migration, schema check | crash injection + golden legacy snapshots |
| **S04 — Jobs & CP** | durable plans/jobs/events/ownership/revision/lease, restart, cancel, replay | integration 2+ processes + restart |
| **S05 — DAG/CAS/Config** | consolidar scheduler, CAS + metadata SQL + GC y merge/restore con ownership | recovery and rollback differential |
| **S06 — IR** | una sola typed Operation IR, GoExecutor, compiler sh/bash/PS, determinismo, fail-closed | differential fixture tests + shellcheck/PSScriptAnalyzer |
| **S07 — Release & self-update** | release artifacts solo Linux/macOS, SUMS+firma/attestation, updater real y rollback | mock release + tag release sandbox E2E |
| **S08 — Installer** | `install.sh` con selector elegante release/source, `install.ps1` source-only, Worker versionable | PTY + native mac/linux/win-source tests |
| **S09 — Tools/Extensions** | tool manifests y runners autónomos, source provenance, sandbox policy honesta, security closure | standalone tool tests sin anx + negative plugin tests |
| **S10 — UX/CLI/TUI** | Query API + dashboard, jobs, diff, change history, error hygiene y docs | PTY + resize + JSON contract + memory benchmark |
| **S11 — CI optimization** | E2E sin redundancias, Linux distros/mac ARM64/Win source/Termux conditional, supply chain | PR green + measured runtime |
| **S12 — Closure** | cerrar históricos T00/R00…, AC001–024/V/HS; release/docs/changelogs; branch protection | evidence report, no open P0/P1, `main` no tocado directamente |

El agente puede reordenar slices si hay dependencia técnica, pero no omitir DoD ni simular cierre por PR/compilación. Si un slice queda BLOCKED-EXTERNAL, completar todo el código y tests locales posibles y describir el único paso externo pendiente.

---

## 12. Contract tests y escenarios negativos obligatorios

**SQLite/Windows:** 2, 4, 8 procesos escriben N registros únicos; todo registro exactamente una vez; file permission error; busy timeout; SIGKILL entre intent/commit; database locked; corruption + restore; migración interrumpida; misma mtime entre writes; read during WAL checkpoint; DB backup/restore while active; full disk; paths con espacio/Unicode; NTFS ACL seguro/inseguro; restart job; no `Access denied` por os.Rename JSON.

**IR:** same IR byte-for-byte script 10 runs; malicious provider/paths/argv; invalid plan digest; unsupported op; untrusted raw command; duplicate IDs/cycles; shell effects vs Go effects; Bash/POSIX/PowerShell; 3-way merge conflict; secure cleanup (foreign symlinks); secrets never echoed.

**Install:** valid signed release; missing asset; corrupt sum; bad/missing signature; tar traversal; broken binary; low disk; non-TTY with/without flags; TTY selector/cancel; missing Go/Git; Linux musl; mac ARM64; Windows source; PATH conflict; source=main vs tag; offline; rollback after failure; `curl` error with no false success; release absent explicit switch; Cloudflare mocked response.

**Tools:** manifest invalid/unknown key/version; standalone binary not on PATH; no Git dependency; external over built-in priority; unknown executable path; Windows batch; version parsing; network attempt blocked when offline; tool update return status correct; no transitive import from ANX `internal/` in standalone tool.

**Privacy/security:** Herdr socket/session.json NEW name file leak, glob bypass, Git staging pre-staged unrelated file, symbolic/hard link attack, symlink cycles, dotfile with token, ACL writable by untrusted users, plugin reading $HOME secrets if sandbox claimed, malformed JSON RPC, event payload containing access key, unsafe rollback.

**Performance:** `go test -bench=... -benchmem` on driver/read/write, baseline vs after for cold start `version`/`help`/`status`, dashboard idle RAM and CPU, 100/1k/10k resources and 100k events with DB size, WAL and p95 times, startup w/o DB for help. **Report actual measured host/resources and variance;** choose thresholds from baseline with documented SLA; do not impose unrealistic global `<25MB` if impossible with selected runtime without redesign. Focus on working at 8GB-RAM laptops and Android as real use case.

---

## 13. Definition of Done final (estricto)

- [ ] Último HEAD revisado y PR con commits trazables; sin cambios destructivos en `main` ni HOME real.
- [ ] 100% CI jobs requeridos verdes en **exactamente** el SHA final: Linux, macOS Apple Silicon ARM64, Windows, E2E Docker/PTY, ARM64 Linux, syntax/lint, security y DB multiwriter.
- [ ] SQLite autoritativo versionado: WAL, FK, migrations, backup/restore, multi-process, recovery, lazy usage, low RAM; `state.json` importado sin pérdidas; no forced-pool memory overhead ni falso soporte Termux.
- [ ] Pérdida de datos Windows corregida; security seal ACL Windows real; doctor read-only.
- [ ] Control plane: plan authorized immutable, host revision/preconditions, jobs durable, true DAG, idempotency, rollback/replay/cancel y lifecycle coherentes.
- [ ] Script IR única; renderers deterministas, seguro quoting, diferencias explícitas y equivalencia de efectos; sin strings shell ad hoc que finjan operaciones.
- [ ] Release pipeline publicable Linux/macOS y mock release E2E; publicación real solo con autorización; Windows source-first SIN firma obligatoria ni precompiled assets.
- [ ] Instalador POSIX pulcro con selector TTY release/source/preview, y PowerShell Windows source-first; sin descargar Go o repo en modo release; Cloudflare Worker ready pero despliegue marcado BLOCKED-EXTERNAL si no hay permisos.
- [ ] Tools externas autónomas con manifests y health/update/execute contracts; aislamiento y no-colisiones; plugin security con límites reales declarados.
- [ ] Dotfiles/Herdr/aliases/cleanup/rollback seguros; no leaks ni mutación en dry-run.
- [ ] UI CLI/TUI consistente y medible; timestamps/changelog/versiones reales, documentados, seguridad y compat OS confirmadas.
- [ ] E2E cero duplicación innecesaria, smoke por distro significativo; Android Termux real como CONDITIONAL/UNVERIFIED hasta evidencia; Docker/Podman decisión basada en benchmarks y no moda.
- [ ] `go test`, `go vet`, `-race`, fuzz, lint, vulnerability scan, SBOM, license audit y benchmarks guardados; no warnings nuevos ignorados ni silent skips.
- [ ] Reconciliar todas las especificaciones, README, install docs, shell completions, changelog, ADRs; no afirmaciones imposibles de “100%” sin evidencia.
- [ ] Finalizar con **informe de aceptación**: SHA, PR URL, CI URLs, diff summary, QA matrix, comandos probados, migración/rollback, resultados Windows/mac ARM64/Termux, riesgos residuales y exacta lista de BLOCKED-EXTERNAL. Nada de marcar `[x]` por intención.

### Entrega esperada del agente

1. Código funcional más migrations SQL, tests y docs; ADR de SQLite/driver/Termux y Operation IR; workflow de release y Worker reproducibles.
2. PR(s) acotados con commits separados por slices, sin secret leakage ni cambios de usuario.
3. Capturas o logs de instalador (PTY) y CLI/TUI; reporte de CI E2E con tests que realmente simulan escenarios objetivo.
4. `odd/tasks/anx-master-closure-log.md` con matriz DONE/PARTIAL/BLOCKED por cada tarea anterior.
5. **Si no se puede ejecutar un gate**, indicar por qué, el comando exacto y la condición para resolverlo: prohibido inventar éxito.

---

## 14. Referencias técnicas para verificar durante ejecución

- Driver SQLite sin CGO: <https://pkg.go.dev/modernc.org/sqlite> — soporte oficialmente enumerado para Linux/macOS/Windows, no asumir Android; fijar `modernc.org/libc` compatible.
- Alternativa SQLite sin CGO: <https://github.com/ncruces/go-sqlite3> y <https://github.com/ncruces/go-sqlite3/wiki/Support-matrix> — memoria/locking y plataformas deben comprobarse, no asumir equivalencia.
- Microsoft Code Signing: <https://learn.microsoft.com/windows/apps/package-and-deploy/code-signing-options> — Authenticode no es un requisito técnico para compilar o probar binarios Windows, pero la distribución pública unsigned puede tener SmartScreen.
- GoReleaser GitHub Actions: <https://goreleaser.com/customization/ci/actions/> — usar OSS o pipeline directo; validar licencias y funcionalidades vigentes.
- Repo/CI de partida: <https://github.com/arturonavax/env/tree/audit/apple-silicon-e2e-odd-installer-20261031> · <https://github.com/arturonavax/env/actions/runs/41232029494>.

**Instrucción final:** aplica las correcciones y las capacidades faltantes como una implementación real, no como otro documento con promesas; prioriza las invariantes de datos y seguridad, conserva el resultado verde de macOS Apple Silicon, resuelve Windows de forma definitiva, y demuestra los resultados con evidencias reproducibles antes de declarar ANX listo.

