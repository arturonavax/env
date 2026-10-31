# ANX — ODD FINAL DE CIERRE INTEGRAL Y CERTIFICACIÓN (Go 1.27.3, SQLite, Operation IR, Windows, instalador, CI/E2E)

> **Tipo:** prompt ejecutable para agente de código autónomo (Organic/Outcome-Driven Development + Spec-Driven + TDD).  
> **Proyecto:** `https://github.com/arturonavax/env` · CLI/TUI `anx` · Go.  
> **Rama de referencia auditada:** `audit/apple-silicon-e2e-odd-installer-20261031`, HEAD comprobado `1051ec2a934db5190bba8b298f6150f05396d852`.  
> **CI base:** `https://github.com/arturonavax/env/actions/runs/41240251525` (**11/12 jobs exitosos**; el único fallo es Windows por cierre de SQLite y política ACL, no por pérdida de registros).  
> **Fecha del diagnóstico:** 2026-10-31. **El agente DEBE obtener de nuevo los HEAD, diffs, checks y releases antes de tocar código.**  
> **Objetivo:** resolver **todos** los bloqueos históricos de ANX y cerrar las funcionalidades prometidas con pruebas y evidencia, sin degradar rendimiento, seguridad, UX, multiplataforma ni arquitectura.  
> **Regla de interpretación:** ESTE documento sustituye en autoridad operativa a los prompts ODD históricos en lo que estos contradigan el HEAD auditado; los antiguos permanecen como trazabilidad. **No reimplementar módulos ya existentes ni cambiar tests para esconder fallos.** La expresión «100 %» significa que los requisitos verificables quedan implementados y los gates ejecutados aprobados; Cloudflare DNS, publicación real de releases y Termux físico se marcan `BLOCKED-EXTERNAL` hasta su validación real.  
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

## 1. P0 INMEDIATO — Cerrar el ÚNICO job rojo de Windows de manera definitiva

**HECHOS OBSERVADOS, NO HIPÓTESIS:** última ejecución [`41240251525`](https://github.com/arturonavax/env/actions/runs/41240251525), SHA exacto `1051ec2a934db5190bba8b298f6150f05396d852`: Ubuntu/macOS Apple Silicon ARM64/5 Linux Docker smoke/Deep E2E/ARM64 QEMU/lint aprobaron; job `Test (windows-latest, Go 1.27.3)` falló. Logs del job `124652479804`. `internal/storage/sqlite` y `TestStoreMultiProcessSimulation` **aprobaron** en Windows con 2/4/8 instancias. No presentar el incidente histórico `27/40` como el fallo presente.

### W0-A — Base de datos ANX abierta al limpiar `t.TempDir()` (bloqueante P0)

El error literal es `testing.go:1619: TempDir RemoveAll cleanup: unlinkat ...\anx.db: The process cannot access the file because it is being used by another process.`. Aparece en `internal/app` (9 tests), `internal/hacks/TestPlanHackAndApplyHackIsolatedExecution`, `internal/state/TestStorePersistenceAndConfigBases`, `TestStoreConcurrentInProcessWrites`. **Causa confirmada por código:** existe `(*state.Store).Close()` -> `(*sqlite.DB).Close()` (cierra pools reader/writer), pero los fixtures `internal/app/app_test.go:newFixture`, `internal/state/state_test.go` y otros abren `OpenStore()` y no registran cierre, y `internal/cli/root.go:ExecuteCLI` tampoco cierra el `AppContext` creado por `BuildAppContext`/`BuildLazyAppContext`.

**Implementación obligatoria:**

1. Inventariar absolutamente todos los puntos `OpenStore`, `sqlite.Open`, `bootstrap.NewContainer`, `BuildAppContext`, `BuildLazyAppContext`; identificar owners por instancia y handles asociados. Cada `Open` exitoso requiere exactamente un owner responsable de `Close()`, sin dobles cierres ni goroutines vivas. Mantener conexiones abiertas durante la operación; nunca cerrarlas prematuramente para satisfacer tests.
2. `internal/bootstrap.Container`: método `Close() error` idempotente, que cierra Store, journal/snapshots/streams/otros closers si abren recursos, detiene actividades asíncronas, espera goroutines y agrega errores con `errors.Join`; aplicar cleanup de construcción parcial con LIFO si falla cualquier fase. No convertir el container en un service locator global; lifetime por invocación CLI/TUI.
3. `internal/cli.AppContext`: owner explícito del container/función `closeFn`, método `Close()` idempotente que no cierre recursos prestados por tests; inyectar `Close()` desde `BuildAppContext` sin filtrar detalles de SQL al dominio. `ExecuteCLI` debe usar `defer` a continuación de crear el contexto; conservar orden correcto de cancelación, `rootCmd.ExecuteContext(ctx)`, flush/cierre de DB y mapeo de exit code. En caso de fallo crítico de cierre, la CLI **no** imprime éxito, pero no se anula el error original.
4. Tests que construyen Store directamente: inmediatamente después de un Open válido añadir `t.Cleanup(func(){ if err:=store.Close(); err!=nil { t.Errorf(...) } })`, incluyendo tests que abren **más de una instancia** (`store` + `reopened`, `finalStore`, cada instancia concurrente). En `internal/app/app_test.go:newFixture` registrar cleanup **antes** de volver de fixture. Revisar `internal/hacks`, `internal/cli`, `internal/bootstrap`, `internal/storage/sqlite`, tests de seguridad y rutas de error; fixtures no deben depender de GC, finalizers, `time.Sleep`, ni retrasar `RemoveAll`.
5. Asegurar cierre antes de que `t.TempDir` intente borrar: Go `t.Cleanup` ejecuta callbacks LIFO; como `TempDir` registra su cleanup al crear, la llamada `t.Cleanup(store.Close)` posterior es correcta. Evitar `t.Parallel` y shared globals que retengan Store después del test; `database/sql` permite conexiones adicionales en reader pool: asegurar filas, transacciones y `*sql.Conn` cerrados en todo path.
6. Pruebas de regression Windows: abrir/cerrar/reabrir + `os.RemoveAll` de directorios con `anx.db`, `anx.db-wal`, `anx.db-shm` inmediatamente tras Close; iterar cientos de veces, con busy queries y cancelación, sin fugas. Validar que `Close` reporta errores, es idempotente y que invocaciones CLI repetidas no mantienen locks.
7. Ejecutar tests con `-count=5`, `-shuffle=on` y Go race cuando esté disponible. **Prohibido** ocultar mediante `t.Skip`, `GODEBUG`, `GC`, sleeps, cambios de `TempDir` a permanent dirs o deshabilitar la limpieza.

### W0-B — Evaluación NTFS / security seal: evitar falsos positivos, conservar fail-closed

`internal/cli/TestCLIDestroyer_ResetDryRun`, `_UninstallDryRun`, `_ApplyFlagDryRun`, `TestCLIUpgradeDryRun` devuelven exit 6 por el sello: checkout `D:\a\env\env` marcado `TrustDegraded` por ACE de `Users` (SID `S-1-5-32-545`) y raíz de volumen `D:\` con propietario `NETWORK SERVICE` (`S-1-5-20`) como no confiable. Código relevante: `internal/security/trust_windows.go:VerifyDirectory` examina **todos** los ancestros hasta raíz; `evaluateWindowsEntry` evalúa propietarios de ancestros igual que propietario del leaf. Un ancestro propiedad de cuenta de servicio NO basta por sí solo para demostrar que otro usuario puede reemplazar el origen.

1. Conservar el verificador real Win32 `GetNamedSecurityInfo`/owner/DACL. Separar *trust del origen* (leaf: owner, DACL, modificación de manifests y archivos ejecutables) de *integridad de ruta* (ancestros: capacidad efectiva de borrar, renombrar, reparentar/reemplazar o redirigir el subárbol). No tratar todos los owner SIDs de todo el volumen como propietarios equivalentes al leaf.
2. Evaluar los derechos **efectivos** relevantes, teniendo en cuenta ACE allow/deny en orden de Windows, flags de herencia, grupos/token, derechos genéricos mapeados, reparse/junction/symlink y permisos de supresión/renombrado sobre el hijo. Evitar simplificaciones que ignoran denials de grupos distintos o confunden `Users:Read` con permiso de modificar código. Preferir comprobación mediante la API Windows de AccessCheck/Authz o una política conservadora documentalmente justificada, no confiar solamente en `os.FileMode` ni en máscara ACE ingenua.
3. Diferenciar `Trusted`, `Degraded` seguro por entorno compartido sin reemplazo efectivo, `Untrusted` y `Unknown`; registrar razón exacta, principal y derecho. Ningún bypass universal para GitHub Actions/`CI=true`/`D:\` ni flag secreto de tests que convierta cualquier ACL en segura. Ante inseguridad real, seguir bloqueando `apply`/scripts y mostrar remedios Windows reales (no `chmod`).
4. Tests reales Windows con `icacls` y fixtures aislados: owner de cuenta de servicio en ancestro pero leaf inaccesible a terceros; `Users` lectura; Everyone modify; Authenticated Users modify; ACL heredada; deny ACE; NULL DACL; reparse point; capacidad de reemplazar hijo con acceso en padre; rutas UNC/espacios/caso; fallos API -> Unknown. Restaurar ACL tras cada prueba, no cambiar permisos del checkout global.
5. Tests CLI no deben depender del ACL no controlable de workspace ni de `%TEMP%` salvo que estén comprobando esa política expresamente. Crear fixtures de repo/manifest con ACLs definidas; inyectar `DirectoryTrustVerifier` **solo** en unit tests que prueban lógica de consumo; reservar pruebas de seguridad Windows reales para checker nativo.

### W0-C — Gates obligatorios después de arreglar causas

```powershell
go test -count=1 -v ./internal/state/... ./internal/storage/sqlite/...
go test -count=1 -v ./internal/app/... ./internal/hacks/... ./internal/cli/... ./internal/security/...
go test -count=5 -shuffle=on -timeout 20m ./...
go vet ./...
go build ./cmd/anx
```

Además: test paralelo que abre/cierra 20+ Store y confirma que puede borrar inmediatamente toda DB en Windows; Windows source installer E2E contra HOME/PATH aislados sin releases precompiladas ni Authenticode. Reejecutar GitHub Actions **en el SHA nuevo** y verificar 12/12 jobs o la matriz que lo sustituya. No basta re-run del mismo commit rojo. El job Windows debe correr sin WSL ni computadora Windows del usuario; GitHub-hosted Windows runner es suficiente.

**Aceptación W01–W12:** sin `TempDir RemoveAll cleanup`, sin locks persistentes, 0 registros perdidos, tests ACL negativos eficaces, no bypass universal, `Close()` de producción y fixtures, compilación Windows source, `go test/vet` verde y logs verificables.

---

## 2. P0 — SQLite avanzado como almacenamiento transaccional de ANX

### 2.1. Estado actual confirmado y política de consolidación (NO REESCRIBIR LO QUE YA PASÓ)

Ya existen `internal/storage/sqlite/{db,options,migrations,migrate,models,repositories}.go`, `internal/state/store.go` como adapter SQLite, `docs/adr/ADR-storage-sqlite.md`, CLI `anx state`, migración JSON a SQLite, WAL, pools writer/readers y `modernc.org/sqlite@v1.61.1`; `TestMigrateJSONToSQLite`, `TestStoreMultiProcessSimulation` y suite `internal/storage/sqlite` **aprobaron en Windows** en el run actual. No introducir otra DB, no volver a escribir desde cero los repositorios ni agregar `mattn/go-sqlite3` por moda. Validar/conservar compatibilidad actual CGO=0 para Linux glibc/musl, macOS ARM64, Windows build; Termux Bionic sigue sin prueba nativa y NO debe declararse soportado.

**Auditar de forma específica y remediar:**

- `internal/storage/sqlite/db.go` usa DOS pools `database/sql` y DSN `file:%s?...`: verificar codificación URI de ruta Windows (espacios, `#`, `?`, Unicode, letras de unidad, UNC); usar DSN correctamente escapado conforme al driver, sin concatenar una ruta arbitraria en la parte de query. PRAGMAs `foreign_keys`, `busy_timeout`, `synchronous` aplicados por conexión realmente obtenida y después de reopen, WAL, `journal_size_limit`, checkpoint medido y no agresivo, cancelación de lock, cada `rows.Close`/`tx.Rollback`.
- `OpenStore` crea `anx.db` incluso para comandos triviales si no hay lazy path: preservar arranque rápido de `version/help/completion` sin DB, startup del dashboard y memoria reducida; instrumentar *cold start* y RSS reales. Cerrar pools al terminar toda invocación.
- Integridad y recuperación: crash injection entre commit y efecto externo, checkpoint, migración, backups consistentes con DB abierta, multiproceso Windows/Linux/macOS, DB corrupta/read-only/disk full, idempotencia de migración JSON, recuperación sin borrar estado ni sobrescribir DB dañada.
- `docs/adr/ADR-storage-sqlite.md` declara rendimientos concretos (1.04 ms, 48.3 ms/10k writes, ~12.3 MiB): exigir comando, CPU/OS, iteraciones, benchstat y entorno reproducible antes de tratarlos como benchmarks representativos; documentar si son medidas de un solo host.
- Cerrar todos los métodos silenciosos de `state.Store` que descartan error SQL por contratos legacy: introducir API `(..., error)` en las rutas de consistencia crítica, adaptando consumidores; si wrappers de lectura histórica deben conservar compatibilidad, no usarlos para seguridad ni para confirmar éxito/estado mutado.

**Reglas SQLite:** SQL parametrizado, FK y constraints, migraciones con checksum e inmutabilidad, `WriteTx` tipado/Unit of Work, locks transaccionales e idempotency keys, pool con límite medido y operaciones sin retener conexiones durante descargas o comandos externos. Incluir pruebas de migración con DB antigua, WAL activo, lentitud, conexiones leak y rate of locks. Nada de caché histórica en RAM cuando SQLite puede leer paginado.

### 2.2. Capas concretas

- Auditar e introducir SOLO los puertos de dominio que falten, sin importar SQL: `StateRepository`, `ResourceRepository`, `ConfigBaseRepository`, `PlanRepository`, `JobRepository`, `EventRepository`, `HistoryRepository`, `SnapshotRepository`, `MetadataRepository`, `AuditCacheRepository`, `TransactionManager`, `StateHealth` (interfaces pequeñas y composables, no monstruo único). Conservar métodos públicos usados por `app`, CLI, TUI, bootstrap, jobs, cleanup, backups, restore y tests mediante adaptadores/transición; eliminar JSON como backend autoritativo tras migración validada.
- Consolidar la única ruta ya creada `internal/storage/sqlite` con `Open(ctx, Options)`, conexión/PRAGMAs, migrator, repositorios, unit of work y lifecycle `Close()`. Tests de todos los ports con implementaciones fake e SQLite real en archivo temporal. No abrir DB para `anx version`, `--help`, completion/list simple ni comandos sin necesidad: **lazy bootstrap** para cold start y RAM.
- Usar `stateDir/anx.db` o `stateDir/state.db` según ADR, única ubicación bien documentada: XDG_STATE_HOME en Linux/macOS donde aplique; LocalAppData en Windows; prefijo privado de Termux. Nunca ubicar DB, `-wal`, `-shm`, snapshots ni credenciales bajo `files/` o un checkout Git. Mode 0700 en directorio, 0600 en DB/archivos de metadata POSIX; ACL privado correcto Windows; no false positives NTFS. Cerrar todas las conexiones de forma ordenada.
- Capas `ReadModel`/`QueryService` para dashboard y comandos; `WriteService` exclusivamente transaccional y con constraints, no snapshots JSON compartidos. `Repository pattern` **sin** convertir SQLite en modelos anémicos innecesarios; no meter `*sql.DB` en dominio.

### 2.3. Auditar y completar el esquema v1 YA PRESENTE (no duplicar migraciones históricas)

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

**Estado actual confirmado:** `ControlPlaneService.BuildPlan` ya hace `SavePlan` en SQLite, y `Apply` llama a `SavePlan`, `CreateJob`, `AppendEvent` y `UpdateJobState`; PERO **todos sus errores se descartan con `_ =`**, se autoriza al mismo tiempo que se aplica, `PlanDigest` es opcional, y `InstallService.Apply` ejecuta vía `internal/execution.Engine` en lugar de `operation.GoExecutor`. El resultado puede haber modificado la máquina mientras el registro durable falla. Ver [`internal/app/control_plane.go`](https://github.com/arturonavax/env/blob/1051ec2/internal/app/control_plane.go), [`internal/app/install_service.go`](https://github.com/arturonavax/env/blob/1051ec2/internal/app/install_service.go). Este es un P0 de integridad, NO una mejora cosmética.

**Cerrar contrato transaccional:**

1. Un comando mutante sin plan autorizado debe seguir un flujo explícito de autorización local; no persistir como `authorized_by=user` solo porque el proceso conoce el UID. Para automatización usar capability/grant limitado y registrable, no elevar a trusted con `--yes` fuera del contrato.
2. `BuildPlan`: validar IR, guardar digest y snapshot/revision de manifests/host, expirar, observar mutaciones y **propagar** errores de DB. No guardar operaciones incompletas como plan ejecutable. Un plan inmutable no puede reemplazarse bajo el mismo ID.
3. Antes de cambiar host: transacción atómica `AuthorizePlan + CreateJob(queued) + AppendEvent(authorized|queued)` y validar relectura de plan/digest/revisión; ante fallo de DB, **cero side effects**. Evitar `_ =` tanto al iniciar como al finalizar jobs. Usar compare-and-swap SQL (`WHERE status=... AND generation=...`) para evitar doble apply cross-process.
4. Como filesystem/APT/red no son transacciones SQLite, usar *intent/event journal* durable previo a cada efecto externo, postconditions y reconciliación/compensación en crash; si commit final de estado falla después de efecto externo, marcar estado `unknown/needs_reconcile` y **nunca** imprimir `succeeded` ni mentir con DB; preservar evidencia y ruta de reparación.
5. Cancelación real en proceso y al reiniciar, jobs orphan/stale, event sequence monotonic, cursor por step, idempotency key, retention/GC; `anx jobs show/events/cancel/resume` deben leer la misma base SQLite usada por CLI/TUI, distinguir offline/readonly.
6. No duplicar journaling, snapshots ni CAS. Convertirlos en adaptadores del mismo protocolo de efectos. Exigir tests con SQLite fallando exactamente antes/durante/después de cada mutación y comprobar que no hay estado engañoso.

**No reescribir el CP entero si ya existe; resolver estos problemas demostrados por pruebas.**

- `Plan`: `id`, digest canónico SHA-256, schema_version, source manifest SHA, host fingerprint/revision, config content hashes, policy digest, generated_at, expiry, resource owners y operaciones ordenadas por DAG. Guardar/autorizar plan inmutable. `Apply` debe recibir plan_id + digest + expected hostRevision y rechazar `stale`, `unauthorized`, `expired` con códigos distintos; re-evaluar precondiciones atómicas por recurso justo antes de mutar y volver a comprobar tras operaciones susceptibles de TOCTOU.
- `Job`: state machine explícita `queued/running/recovering/partial/succeeded/failed/cancelled`, progresos y errores persistidos en SQLite. `Cancel` sobre proceso actual y recuperación tras reinicio; no confiar en `activeJobs map` como único estado durable. Durante restart clasificar jobs running de forma segura y ofrecer `anx resume --plan`/`anx resume --yes`, con compensaciones idempotentes. `anx jobs`, `anx jobs show`, `anx jobs events --since`, `anx jobs cancel`, etc. No daemon permanente obligatorio.
- **Effects protocol:** `prepare → verify preconditions → reserve external resource lease/fencing → execute → verify postconditions → commit state/event → release lease`. La DB no ofrece transacción distribuida con apt/git/filesystem: usar write-ahead intent, idempotency keys, compensaciones, journals, CAS snapshots, `unknown` status tras crash y reconciliación explícita. Nunca marcar succeeded antes de verificar efecto externo.
- `Observe`: errors producen `unknown/error`, nunca `healthy=true` por fallback. `Explain` cuenta fuente real, estrategia usada, evidencia y diferencias.
- **DAG:** mantener nuevo `internal/execution/graph.go` y `pool.go` si pasan todos los tests: `DependsOn` respetado, indegrees, locks, fairness, cola ready determinista, exclusividad y rollback topológico inverso. Agregar tests multigeneracionales, ciclos, dependencias ausentes, locks que no hambreen otros, errores, stop-on-first-failure, goroutine leak, cancelación, 1/2/8 workers y reinicio. No reescribir algo que ya está correcto.
- Cache/metadata de planes con TTL y invalidación en cambios de manifiestos/host. FTS/SQL no entra en la ruta crítica de ejecución salvo beneficio medido.

**Acceptance CP-01..CP-12:** mismo plan autorizado o rechazo explícito; replay exacto de eventos; idempotencia cross-process; rollback consistente; cancel/restart; 0 acciones para clean repeat install; locks respetados; `anx jobs` CLI/TUI consistentes.

---

## 4. P0 — Operation IR tipada: traducción a shell mantenible, escalable y determinista

### 4.1. Estado actual preciso: la IR existe, pero NO es aún la única autoridad

**Conservar** `internal/operation/{types,compiler,executor,renderer,escape}.go`: ya definen tipos, `CompileIR`, `GoExecutor`, `ShRenderer`, `BashRenderer`, `PowerShellRenderer`, hashing y quoting, con tests; NO crear paquete paralelo `planir`. **Corregir defectos concretos:**

- `internal/operation/compiler.go`: `OpID = fmt.Sprintf("step-%d-%s", step.ID, step.ResourceID)` mientras `DependsOn` usa `"step-%d"`: **referencias inválidas**. Crear mapping `stepID -> opID` construido y validado antes de resolver edges; rechazar duplicados/missing/cycles, asegurar orden topológico estable, tests con 3+ niveles de dependencia y operaciones con mismo ID distinto resource.
- `internal/operation/executor.go:ExecutePlan` recorre `plan.Operations` en orden sin verificar `DependsOn`; la existencia de `DependsOn` en JSON no lo hace un scheduler DAG. Debe verificar preflight DAG y ejecutar con la misma política de orden/locks/idempotencia y journal; no abrir un segundo scheduler distinto de `internal/execution`.
- `internal/app/install_service.go:Apply` continúa usando `execution.Engine` y `buildTask`; aún **no** consume la misma IR emitida por export. Diseñar un único flujo `CompileIR -> Validate -> Authorize -> Execute` y export `CompileIR -> Validate -> Render`; reutilizar el engine existente como orquestador si es sólido. Todo plan que el CLI/TUI llama `apply` ejecuta exactamente la IR autorizada; no permitir rutas legacy paralelas.
- `internal/cli/packages.go:renderPlanScript` sigue llamando al traductor legacy `scriptgen.TranslatePlan`, que no garantiza equivalencia ni siempre usa `operation`. `internal/cli/configcmds.go:renderSyncScript` sigue emitiendo `cp -a`, lo cual ignora merge 3-way, hashes, snapshots, ownership y staging. `internal/scriptgen/builder.go` sigue construyendo strings Bash. Consolidar hacia renderers de `internal/operation`; compat wrapper solo si usa la misma IR, sin divergencia.
- `OpAnxRuntimeDelegation` en `CompileIR` puede generar `anx apply --requested <resource>` / `anx sync`, que puede replanificar/recursar/diferir del plan original; no emitir fallback ejecutable inseguro. Standalone debe rechazar op no traducible; delegated debe pasar plan id+digest de una IR aprobada y comprobar versión/runtime y *no recursar*.
- `GoExecutor.ExecuteOp` actualmente hace `os.Remove(op.TargetPath)` antes de crear symlink, sin verificar ownership/fingerprint; `OpDeleteOwned` tampoco demuestra ownership. Añadir guardias de tipo, basename, canonical paths, symlink/reparse traversal, hash preconditions, staging atómico, backups/compensación, y no borrar archivos ajenos. `writeAtomic` debe comprobar `Close`/Sync/rename y preservar directorios/metadata según políticas.
- Contrastar qué `OperationKind` realmente ejecuta y renderiza cada backend; si compilador produce un kind que `GoExecutor` no maneja, devolver error tipado, no éxito silencioso. Convertir la matriz de capacidad por plataforma en datos comprobados.

**DoD adicional IR-P0:** cero invocaciones de traductores ad hoc en rutas productivas, DAG references consistentes y validadas, el mismo digest cruza Plan→Apply→Render, pruebas diferenciales con HOME clonado/PMs fake, fallo seguro de operaciones no soportadas y tests de rutas con quotes/secrets/Unicode.

### 4.2. IR única — entidades y contrato inmutable

Completar y normalizar el paquete ya existente `internal/operation`; no crear un segundo paquete `internal/planir`. Mantener el contrato de tipos independiente de CLI/SQL/OS/terminal. Estructuras y enums tipados con validación en constructor, versiones semver de esquema, JSON canónico y digest; no `any` arbitrarios salvo payload tipado por kind.

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

**Estado actual:** `install.sh` ya tiene menú numerado release/source/preview, flags/env; existe `deploy/cloudflare/{src/index.ts,wrangler.jsonc}` y sus tests; Windows PowerShell source-first, y `selfupdate` real con tests. No pedir crear estas piezas desde cero. **Todavía** no hay assets publicados (`v1.0.0` de 2023 no tiene ninguno), no se ha probado el dominio desplegado, ni el E2E Termux Android real.

**Bloqueos concretos del instalador:**

- `install.sh:79` condiciona menú a `[ -t 0 ] && [ -t 1 ]`. En el comando público `curl -fsSL https://sh.arturonavax.dev | sh`, stdin es una **tubería**, por lo que normalmente NO hay prompt aunque stdout sea TTY. Implementar lectura **exclusivamente desde `/dev/tty`** si existe terminal de control; el código shell del pipe sigue llegando por stdin de la shell. Guardar/restaurar fd si se necesita. Test con pseudo-TTY ejecutando el pipeline real y comprobar los cuatro caminos y timeout/cancelación; sin TTY jamás bloquear esperando input.
- `install.sh:137-154` **permite** continuar la instalación aunque no haya `want_sum` o `computed_sum` en un entorno sin `sha256sum/shasum`; en ese caso el mensaje de “verified release” es falso. Rechazar si falta checksum exacto, hay múltiples/duplicados, formato ≠ SHA-256 hex de 64 chars o no hay implementación de cálculo. Un SHA256SUMS descargado del mismo origen NO establece por sí solo procedencia/autenticidad del proveedor: política de firma Ed25519 fijada o attestation **efectivamente verificada** cuando esté disponible; sin garantía, no etiquetar como verified.
- `install.sh:67-75` mapea arquitecturas desconocidas a `amd64`, peligroso; OS/arch/libc no soportado debe ser error explícito. En Alpine, resolver variante musl real. Android Termux no es `linux/arm64` genérico.
- `install.sh:155` extrae tarball remoto con `tar -xzf` sin prevalidación de entradas: bloquear path traversal, symlinks, hardlinks, special files, sobreescrituras, zip bomb, y no instalar sobre archivo ajeno. Staging + atomic replace + rollback ante falta de espacio, señal o binario inválido.
- `--mode=plan` / preview **solo imprime** que no modifica datos, pero no ofrece plan útil; entregar plan verdadero sobre prerequisitos, origen, asset/version, destino, permisos, operaciones previstas, sin instalar y sin red oculta si offline. Separar instalar binary vs configurar dotfiles/paquetes.
- Cloudflare Worker actualmente tiene `DEFAULT_BRANCH=main`; el trabajo nuevo está en rama. Endpoint público no puede asumir código no mergeado; empaquetar origen por tag/digest conocido y verificar content type/status/caché/redirect. El Worker existente debe preservar métodos/UA correctos, no servir accidentalmente PowerShell a `| sh`, no aceptar remote source inyectado y probar 404/origin compromise/redirect loops.
- Windows debe permanecer **source-only por defecto**, sin release ZIP/EXE ni Authenticode de pago; Windows GitHub Actions compila y prueba sin hardware físico. Un futuro `windows-assets` job queda desactivado, habilitable solo con elección explícita. No vincular la publicación Linux/macOS a éxito de signing Windows.


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

**Últimas verificaciones obligatorias:** aunque `internal/extension/admit.go` valida hashes y condiciones antes de lanzar, **no constituye por sí solo sandbox**; el código nativo puede acceder a HOME/red según permisos del proceso. Diferenciar *trusted tool* vs *untrusted extension* y declarar sandbox/denial por OS sin falsas promesas. Comprobar que los eventos de SQLite, scripts exportados y diagnósticos aplican redacción de secretos y no almacenan env arbitrario. Mantener la allowlist Herdr, incluso para archivos nuevos que Git todavía no ignore. Revisar `GoExecutor` DeleteOwned/SymlinkOwned por permisos y borrados de terceros (P0).


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

**CI observado en `41240251525`:** 11 éxitos, Windows rojo. `macos-26` es **Apple Silicon ARM64 REAL** y pasó. Cinco distros smoke Debian/Ubuntu/Fedora/Alpine/Arch pasaron, Linux ARM64 QEMU pasó y E2E integración Debian/Alpine también. Optimizar *después* de sanar Windows: revisar assertions de smoke frente al deep E2E y quitar solo duplicaciones genuinas; NO eliminar coverage de libc/gestores/instalador para mejorar métricas artificialmente. `tests/e2e/scripts/test_concurrency.sh` continúa ejecutando planes secuencialmente, por lo que **no es una prueba de concurrencia**; sustituir por fixture Go multiworker/multiprocess y renombrar si se conserva smoke. Agregar `deploy/cloudflare` Node test al CI (hoy no es job explícito), y tests de scripts/PTY/firmas/release assets contra server local mock; E2E Windows source installer separado de `go test` para cobertura real. Probar el mismo checkout final, no PR previo.


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

## 11. Plan final de ejecución por vertical slices, sobre el código que ya existe

**Orden obligatorio por integridad; cada slice debe entregar cambios y tests reales. NO reimplementar fases S02–S08 que ya tienen código si basta remediarlas.**

| Slice | Urgencia | Cambio incremental exacto | Gate/evidencia |
|---|---|---|---|
| **F00 Baseline** | P0 | `git fetch`, confirmar branch SHA/logs/ODD, matriz de capacidad+riesgo de datos, bench antes, tests repro rojos | SHA, URL CI, diff, lista de owners de DB |
| **F01 Windows DB lifetime** | P0 | Cerrar cada Store/DB en fixtures, bootstrap, CLI, failure/cancel, rows/Tx, test `RemoveAll` inmediato | Windows `-count=5` sin error de cleanup; store multiwriter 2/4/8 pasa |
| **F02 Windows ACL real** | P0 | Separar root/ancestros, derechos efectivos, fixture aislado ACLs, seguridad fail-closed | Windows security + CLI dry-run verdes, 5 negativos ACL |
| **F03 SQLite correctness** | P0 | Pool/DSN Windows, SQL errors, WAL, replay/backup, migration, low-RAM, observability | CPU/RSS/cold-start medidos; crash/SQL fail tests |
| **F04 CP durable & jobs** | P0 | Prohibir `_ = db.*` en críticos; atomic plan/job/event authorization, compare-and-swap, crash recovery, fail before external effect | fault injection cada escritura y tests cross-process |
| **F05 IR canonical** | P0 | Resolver OpIDs y DependsOn, executor real de IR y renderer único, ownership, no recursive `anx apply`, golden/differential | `GoExecutor` vs `sh/bash/PowerShell` same effects; deterministic digests |
| **F06 Seguridad & datos** | P0 | Herdr allowlist; `capture` precise staging, secret scanning, sandbox claims, CAS/rollback, no unsafe symlink delete | HS01–HS12; negativas de rutas/ACL/claims |
| **F07 Instalador** | P0/P1 | `/dev/tty` en `curl|sh`, checksums/firma fail-closed, NO arch fallback, tar safe, preview verdadero, consent | PTY, source install, signed mock release, hostile tar tests |
| **F08 Releases & update** | P1 | Publicador reproducible Linux/macOS, firma anclada, manifest/checksums, self-update rollback; Windows source-first | tagged prerelease dry-run + mock E2E; publish real autorizado |
| **F09 Cloudflare Worker** | P1 | Revisar router actual+tests, versionado de origin y CI `node --test`, configurar DNS únicamente autorizado | edge test; dominio real `BLOCKED-EXTERNAL` sin acceso |
| **F10 Tools/plugins** | P1 | Contracts externos versionados, tests autónomos sin ANX, aislamiento honesto, gestión de lifecycle | ejecutar tool externo independiente; negative sandbox |
| **F11 CLI/TUI parity** | P1 | Jobs/DB health/dashboard y control plane únicos, salida JSON/stdout y errors/secret hygiene, docs | PTY/pseudoterminal, resize, snapshots, RAM |
| **F12 CI quality** | P1 | Reducir E2E duplicados medidos, Windows source E2E, Mac arm64, 5 distros, supply-chain, Termux conditional | CI verde todos los requeridos SHA final |
| **F13 Closure** | Gate | Actualizar trazabilidad AC/V/HS + README/ADR y evidencia, abrir PR o actualizarlo | no P0/P1 abiertos; externos precisos y desbloqueables |

**Criterio de continuidad:** después de cada F01/F02 empujar cambios a rama de trabajo y ejecutar Windows nativo; resolver los nuevos fallos antes de seguir con refinamientos. CI completo obligatorio al final, prohibido fusionar con cualquier job requerido rojo. Si no hay permisos para disparar ejecución, la existencia de commit en rama no cuenta como CI aprobado.

---

## 12. Contract tests y escenarios negativos obligatorios

**Nuevos invariantes de regresión sobre SHA 1051ec2:** (a) `Close()` antes de `TempDir` y sin handles en Windows; (b) ninguna mutación externa si `SavePlan`/`CreateJob`/`AppendEvent`/`UpdateJobState` previa falla; (c) no se puede ejecutar una IR con DependsOn apuntando a ID inexistente; (d) un script generado sin permisos de ownership falla sin borrar archivos ajenos; (e) `curl | sh` con TTY real muestra selector y con stdin pipe no se come el script; (f) checksum ausente/repetido/sin herramienta no se considera validado; (g) `%TEMP%` y `D:\` ACL no se confunden con trust del manifest; (h) `modernc.org/sqlite` cierra WAL/SHM antes de cleanup; (i) fallos de DB después de side effects dan `needs_reconcile`, no `succeeded`; (j) v1.0.0 sin assets nunca devuelve instalación release exit 0.


**SQLite/Windows:** 2, 4, 8 procesos escriben N registros únicos; todo registro exactamente una vez; file permission error; busy timeout; SIGKILL entre intent/commit; database locked; corruption + restore; migración interrumpida; misma mtime entre writes; read during WAL checkpoint; DB backup/restore while active; full disk; paths con espacio/Unicode; NTFS ACL seguro/inseguro; restart job; no `Access denied` por os.Rename JSON.

**IR:** same IR byte-for-byte script 10 runs; malicious provider/paths/argv; invalid plan digest; unsupported op; untrusted raw command; duplicate IDs/cycles; shell effects vs Go effects; Bash/POSIX/PowerShell; 3-way merge conflict; secure cleanup (foreign symlinks); secrets never echoed.

**Install:** valid signed release; missing asset; corrupt sum; bad/missing signature; tar traversal; broken binary; low disk; non-TTY with/without flags; TTY selector/cancel; missing Go/Git; Linux musl; mac ARM64; Windows source; PATH conflict; source=main vs tag; offline; rollback after failure; `curl` error with no false success; release absent explicit switch; Cloudflare mocked response.

**Tools:** manifest invalid/unknown key/version; standalone binary not on PATH; no Git dependency; external over built-in priority; unknown executable path; Windows batch; version parsing; network attempt blocked when offline; tool update return status correct; no transitive import from ANX `internal/` in standalone tool.

**Privacy/security:** Herdr socket/session.json NEW name file leak, glob bypass, Git staging pre-staged unrelated file, symbolic/hard link attack, symlink cycles, dotfile with token, ACL writable by untrusted users, plugin reading $HOME secrets if sandbox claimed, malformed JSON RPC, event payload containing access key, unsafe rollback.

**Performance:** `go test -bench=... -benchmem` on driver/read/write, baseline vs after for cold start `version`/`help`/`status`, dashboard idle RAM and CPU, 100/1k/10k resources and 100k events with DB size, WAL and p95 times, startup w/o DB for help. **Report actual measured host/resources and variance;** choose thresholds from baseline with documented SLA; do not impose unrealistic global `<25MB` if impossible with selected runtime without redesign. Focus on working at 8GB-RAM laptops and Android as real use case.

---

## 13. Definition of Done final (estricto)

**Interpretación exacta de «100 %»:** exige `VERIFIED` en todas las tareas cuyo entorno controla el agente. Para lo externo (dominio Cloudflare, publicación release real, firma secreta, prueba nativa Termux física) el código y las pruebas de mock pueden estar `VERIFIED-LOCAL`, pero el requisito operativo queda `BLOCKED-EXTERNAL` con paso y responsable; nunca redondear esas casillas a DONE. Nunca confundir que 11/12 jobs pasaron con que ANX pasó íntegro.

**Prioridades de aceptación que no pueden omitirse:**

- [ ] **WIN-LIFE:** toda DB abierta por API/test/CLI cierra antes de cleanup y causa de error original conservada, sin handles colgantes.
- [ ] **WIN-ACL:** ACL real de repo/ancestros, sin depender de workspace CI; ACL insegura sigue bloqueando.
- [ ] **CP-ATOMIC:** ninguna llamada crítica `_ = db.*`; pre-mutación SQL atómica; tras fallo de persistencia postefecto estado `needs_reconcile` y recovery.
- [ ] **IR-UNIFIED:** ID/DependsOn válidos, un executor de producción + renderers misma IR, no lógica separada ni comandos recursivos.
- [ ] **INSTALL-P0:** selector funciona exactamente en `curl | sh` con terminal, release con firma+checksum efectiva, tar seguro y preview real.
- [ ] **E2E-WIN:** pruebas de instalación PowerShell desde fuente en Windows nativo y DB close/ACL; no exigen binarios Windows precompilados ni firma pagada.
- [ ] **SECURITY-P0:** `DeleteOwned`, `SymlinkOwned`, captura Herdr, ACL, plugins, rollback y scripts no cruzan frontera de archivos ni secretos.


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
- Repo/CI de partida: <https://github.com/arturonavax/env/tree/audit/apple-silicon-e2e-odd-installer-20261031> · <https://github.com/arturonavax/env/actions/runs/41240251525>.

**Instrucción final:** partiendo del SHA comprobado `1051ec2a`, NO repitas módulos ya implementados. Aplica las correcciones y las capacidades faltantes como una implementación real, no como otro documento con promesas; prioriza las invariantes de datos y seguridad, conserva el resultado verde de macOS Apple Silicon, resuelve Windows de forma definitiva, y demuestra los resultados con evidencias reproducibles antes de declarar ANX listo.

---

## 15. Comandos de auditoría, verificación y entrega exacta para el agente

Ejecutar **en una rama de trabajo distinta de `main`**, desde el repositorio local del agente; no asumir que el SHA sigue idéntico al de esta auditoría:

```bash
git status --short
git branch --show-current
git rev-parse HEAD
git log -5 --oneline
rg -n 'OpenStore\(|sqlite\.Open\(|BuildAppContext\(|BuildLazyAppContext\(|NewContainer\(|\.Close\(' internal cmd
rg -n '_ = (db|store|tx)\.|SavePlan\(|CreateJob\(|AppendEvent\(|UpdateJobState\(' internal
rg -n 'TranslatePlan\(|renderSyncScript\(|GoExecutor\(|CompileIR\(|ExecuteTasks\(' internal
rg -n 't\.TempDir\(|t\.Cleanup\(' internal/app internal/cli internal/hacks internal/state internal/storage
rg -n '\[ -t 0 \]|sha256sum|shasum|want_sum|computed_sum|tar -xzf|detect_os_arch' install.sh
rg -n 'GetNamedSecurityInfo|ownerIsTrusted|evaluateWindowsEntry|evaluateDACL|DACL|SID' internal/security
rg -n 'CI|Termux|macos-26|ubuntu-24.04|E2E|test_concurrency' .github/workflows tests scripts
```

**Orden de verificación local y CI (adaptar comandos al shell del host):**

```bash
gofmt -w ./internal ./cmd
go mod verify
go vet ./...
go test -count=1 -timeout 20m ./...
go test -race -count=1 -timeout 20m ./...  # plataformas donde sea soportado y costo razonable
CGO_ENABLED=0 go build -trimpath -o /tmp/anx-verify ./cmd/anx
sh -n install.sh
bash -n scripts/bootstrap_anx.sh
# shellcheck <scripts; no ignorar warnings sin causa>
# pruebas Cloudflare Node (desde deploy/cloudflare según package.json)
# pruebas differential IR + SQLite fault injection + CLI PTY
# Windows PowerShell y macOS ARM64 se prueban en jobs nativos, no se simulan con GOOS
```

**No aplicar `gofmt -w` ciegamente si el árbol tiene cambios del usuario: detectar y preservar modificaciones no propias.** En Windows usar archivos `.exe` y rutas del temp adecuados, no `/tmp`. No declarar `-race` equivalente a procesos cross-machine; añade stress real de varias instancias.

**Releases y cuentas:** no publicar una release pública ni modificar DNS/Workers/branch protection sin autorización del titular. Crear la automatización y pruebas herméticas y documentar los comandos de activación; Windows `source-first` sin Authenticode es un requisito definitivo, no un bloqueo.

**Entregables finales obligatorios del agente:**

1. Código Go real compilable + migraciones forward-only nuevas (nunca editar SQL ya aplicado), tests Go cross-platform, test shell/PTY y Docker/CI actualizados.
2. PR con commits incrementales, diff explicable y sin secretos; CI del **último commit exacto** con links de jobs y ausencia de `TempDir RemoveAll` / falsos ACL positives.
3. `odd/tasks/anx-master-closure-log.md` actualizado: `requirement_id | baseline | change | test | job_url | SHA | status | evidence`. Reconciliar old claims `100% Passed` con CI efectivo.
4. `docs/adr/` actualizar con tradeoffs y evidencias para SQLite, autorizar plan, scripts standalone/delegated, security policy NTFS, instalación y firma.
5. Documentación usable: Linux/macOS curl+sh seguro, Source/Release reales, PowerShell Windows, Termux conditional, `anx state`, `anx jobs`, self-update sin asset, troubleshooting Windows; shell completions.
6. Tabla final `VERIFIED`, `PARTIAL`, `BLOCKED-EXTERNAL`, `FAILED` sin afirmar 100 % cuando queda un gate requerido rojo o sin prueba. Si detectas requisito contradictorio, documenta ADR y conserva semántica segura.

**Prioridad máxima:** no fusionar prematuramente, no comprar firma Windows, no introducir precompilados Windows por defecto, no debilitar seguridad para pasar CI y no pedir al usuario repetir requisitos ya especificados. **Implementa, mide, ejecuta y demuestra.**
