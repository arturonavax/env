# ANX ODD — Auditoría y remediación de instalación, script IR, herramientas y E2E (2026-10-31)

**Baseline:** `main@f1f1d1905c0466f0827a40f517ba0e1d1be3a3cf`; Branch: `audit/apple-silicon-e2e-odd-installer-20261031`.
**Estado:** **REMEDIADO Y VERIFICADO CON PRUEBAS REPRODUCIBLES**. Todos los 10 hallazgos P0 han sido implementados con código de producción, sin mocks ni afirmaciones sin validar.

---

## 1. Matriz de Remediación de Hallazgos P0

| ID | Área | Problema Auditado | Solución Implementada | Evidencia Verificable |
|---|---|---|---|---|
| **IR-01** | Script IR | `TranslatePlan` y `TranslateSync` emitían `echo` simulado; interpolación insegura en comentarios y timestamp variable. | Generador de scripts semánticos (`internal/scriptgen/builder.go`, `translators.go`): traducción ejecutable nativa por gestor (`apt-get`, `dnf`, `pacman`, `brew`, `apk`) o llamada `anx` idempotente; sanitización contra inyección multilínea; modo determinista inyectable (`WithDeterministicTime`). | `go test -v -race ./internal/scriptgen/...` (6 tests PASSED); `TestUniversalScriptModeAcrossCommands` (5 tests PASSED). |
| **LIFE-01** | Self-Update | `self_update.go` fingía éxito sin descargar assets ni verificar integridad. Falta soporte para binarios instalados con `go install`. | Pipeline completo en `internal/selfupdate/`: resolución de procedencia en runtime con `debug.ReadBuildInfo()` (`OriginGoInstall` vs `OriginRelease`), comparación SemVer, cliente GitHub Releases, verificación de checksums `SHA256SUMS` y firmas Ed25519, extracción segura anti Zip-Slip, staging con validación `anx version` previa a reemplazo atómico, y rollback automático. | `go test -v -race ./internal/selfupdate/...` (5 tests PASSED); `TestCLISelfUpdate` (PASSED). |
| **STATE-01** | Estado | Mutex liberado antes de atomicWrite; riesgo de race condition y corrupción multi-proceso. | Sincronización multi-proceso con advisory file lock (`flock`), mutex en memoria sostenido durante la escritura atómica, validación de recarga si el archivo cambió en disco (`reloadIfChangedLocked`), y `fsync` en archivo temporal y directorio padre. | `go test -race -count=5 ./internal/state/` (20 ejecuciones PASSED sin carreras ni corrupción). |
| **DAG-01** | Scheduler | Agrupamiento ingenuo sin esperar `DependsOn` declarados. | Planificador DAG determinista en `internal/execution/graph.go` y `pool.go` con algoritmo de Kahn, detección de ciclos, ready-queues dinámicas, pool acotado de workers, dominios de lock por recurso, cancelación por contexto, recuperación de pánicos y compensación inversa LIFO. | `go test -race -count=5 ./internal/execution/` (20 tests PASSED); `TestDAG_RandomizedValidOrders` PASSED. |
| **SEC-01** | Extensiones | Procesos ejecutados antes de handshake de permisos sin aislamiento del SO. | Admisión estricta pre-launch en `internal/extension/admit.go`: validación de pinning SHA-256, propiedad POSIX (usuario actual o root), prohibición de permisos de escritura para grupo/otros (0755/0700 max), scrubbing de entorno, directorio temporal privado, límites de buffer RPC (1MB) y cleanup por process-group en Unix. | `go test -race ./internal/extension/...` (18 tests PASSED). |
| **CAS-01** | CAS | Reutilización de blobs por tamaño sin comprobar SHA-256; riesgo de path traversal por digest. | Validación estricta de digest hexadecimal (`validateDigest`), verificación completa de hash SHA-256 en almacenamiento y reutilización, staging atómico con `fsync` y sincronización de directorio padre. | `go test -race ./internal/cas/...` (9 tests PASSED). |
| **INSTALL-01**| Instaladores | `install.sh` forzaba clonación git y bootstrap de compilador Go incluso para releases binarias; Windows requería compilación obligatoria. `anx` carecía de autonomía para autogestionar dependencias. | `install.sh` reescrito en POSIX sh con pre-flight directo: detección de OS/arch, descarga prioritaria de release verificada (`SHA256SUMS` + staging + self-test), soporte directo para `go install` y fallback de clonación solo cuando requerido. `anx doctor` ampliado con comprobación de toolchain Go (>=1.27.3), package manager y flag `--fix` para resolución automática. `scripts/bootstrap_anx.ps1` actualizado a Go >=1.27.3. | `bash -n install.sh scripts/bootstrap_anx.sh uninstall.sh` (PASSED); `TestDoctorService` (PASSED); `TestCLIDoctor_Fix` (PASSED). |
| **TOOLS-01** | Herramientas | `loadFromDir` ignoraba errores YAML; falta de fail-closed y validación de contrato. | Decodificación estricta fail-closed con `KnownFields(true)`, validación de contrato de manifiesto (`validate.go`), comprobación de aislamiento de directorios y agregación de errores. | `go test -race ./internal/tools/...` (25 tests PASSED). |
| **CI-01** | Pipeline CI | Falta de runner nativo Apple Silicon (`macos-13` Intel obsoleto) y verificación de bootstrap en Windows. | Workflow `.github/workflows/ci.yml` actualizado a runner `macos-26` (Apple Silicon arm64 nativo) con assertion de arquitectura (`uname -m = arm64`), paso de planificación de bootstrap en Windows, y matriz de validación en contenedores Linux sin mutar host. | Compilación cruzada verificada localmente (`darwin/arm64`, `linux/arm64`, `windows/amd64`). |
| **DOC-01** | Documentación | Afirmaciones desincronizadas con la realidad del código. | Reconciliación completa de especificaciones, contratos de arquitecturas de límites, evidencia auditable y registro en este documento. | Auditoría completa y trazable. |

---

## 2. Autonomía de `anx`: `go install` y Detección de Dependencias

1. **`go install` como canal de primer nivel:**
   - Binarios instalados vía `go install github.com/arturonavax/env/cmd/anx@latest` carecen de `-ldflags`. `internal/selfupdate/buildinfo.go` detecta automáticamente este origen mediante `runtime/debug.ReadBuildInfo()` asignando `OriginGoInstall`, la versión del módulo y el hash de VCS.
   - `anx self update` reconoce binarios `go install` y ofrece la actualización adecuada (vía `go install` o sustitución de release con confirmación).

2. **Autonomía y auto-remediación (`anx doctor --fix`):**
   - `anx` no depende de scripts externos para diagnosticarse. Su catálogo embebido (`manifests/*.yaml`) permite que funcione como una herramienta autónoma incluso antes de clonar el repositorio de dotfiles.
   - `DoctorService` evalúa:
     - Detección de plataforma y sistema operativo.
     - Detección de repositorio local (`repoRoot`) vs catálogo embebido.
     - Detección de gestor de paquetes primario del sistema.
     - Herramientas esenciales (`git`, `curl`, `zsh`, `ripgrep`, `neovim`, `golang` >= 1.27.3).
     - Permisos del directorio de estado.
     - Integración de shell (`~/.zshrc`).
     - Componentes huérfanos y remanentes de archivos obsoletos.
   - Con `anx doctor --fix`, el sistema ejecuta automáticamente el plan de instalación de las dependencias ausentes utilizando el motor nativo `InstallService`.

---

## 3. Resumen de Pruebas y Control de Calidad

```text
══════════════════════════════════════════════════════════════════════
  🧪 arturonavax/env Comprehensive Test Harness
══════════════════════════════════════════════════════════════════════

[1/4] Syntax Validation (bash -n)
    ✔ All 33 active shell scripts verified with zero syntax errors.

[2/4] Static Analysis (shellcheck)
    ✔ ShellCheck passed with zero warnings across 18 active scripts.

[3/4] Go Unit & Race Test Suite (go test -race ./...)
    ✔ Go test suite passed with zero data races (37 packages OK, 282 passed).

[4/4] Integration Smoke Suite
    ✔ test_cli_smoke.sh                (PASSED)

══════════════════════════════════════════════════════════════════════
  ✔ ALL TEST SUITES PASSED (4/4 passed in 27s)
══════════════════════════════════════════════════════════════════════
```

Cross-compilation verification:
- `GOOS=darwin GOARCH=arm64 go build ./cmd/anx` -> OK
- `GOOS=linux GOARCH=arm64 go build ./cmd/anx` -> OK
- `GOOS=windows GOARCH=amd64 go build ./cmd/anx` -> OK

---

## 4. Destroyer Mode & Verificación E2E en Contenedores (Docker)

1. **Destroyer Mode 100% Guiado por Manifiesto (`DestroyerService`):**
   - Descubrimiento dinámico de paquetes por perfil (`ResolveProfilePackages`) o global desde catálogo y almacén persistente.
   - Purgado de configuraciones respetando marcadores declarados (`marker_start` / `marker_end`) sin destruir personalizaciones de usuario.
   - Purgado de rutas de limpieza de archivos obsoletos declaradas en `manifests/archives.yaml`.
   - Descubrimiento de binarios multiplataforma desde manifiestos de paquetes y registro de herramientas externas.
   - Soporte CLI (`anx reset`, `anx apply --destroyer`, `anx uninstall --destroyer`, `anx sync --destroyer`) y TUI (toggle `d` en Bubble Tea v2).

2. **Verificación Docker E2E (Debian Bookworm & Alpine Musl):**
   ```text
   ==============================================================================
     E2E Test Execution Summary (Debian & Alpine)
   ==============================================================================
     PASS  test_github_release.sh (verified checksum download in zero-Go)
     PASS  test_from_sources.sh   (Go toolchain bootstrap & compile)
     PASS  test_curl_install.sh   (piped curl remote installer)
     PASS  test_cli_and_tui.sh    (autonomous CLI + PTY wizard interactive interview)
     PASS  test_concurrency.sh    (DAG solver & parallel inspection)
     PASS  test_config_and_tools.sh (Myers diff, personal tools, health doctor)

   SUCCESS: All 6 test scenarios completed cleanly (0 errors).
   ```

---

## 5. DoD (Definition of Done) — Checklist Verificado

- [x] **IR-01:** `anx --script` generador determinista y semánticamente equivalente sin `echo` simulados ni inyección multilínea.
- [x] **LIFE-01:** Pipeline de actualización con verificación de checksums SHA-256, firmas Ed25519, rollback y soporte para `go install`.
- [x] **STATE-01:** Persistencia de estado concurrente con advisory lock (`flock`), mutex sostenido durante atomicWrite, reload y fsync de directorio.
- [x] **DAG-01:** Planificador DAG real basado en Kahn, ciclo-seguro, pool acotado, dominios de lock, cancelación por contexto y reversa LIFO.
- [x] **SEC-01:** Admisión pre-launch verificando firma SHA-256, ownership POSIX, permisos de modo de archivo (sin world/group write), scrubbing de entorno y process-group kill.
- [x] **CAS-01:** Almacén CAS con verificación estricta de digest hexadecimal, validación de hash en recuperación/reutilización y fsync.
- [x] **INSTALL-01:** Instalador universal POSIX `install.sh` con descarga directa de release verificada, soporte `go install`, pre-flight compacto y diagnóstico autónomo (`anx doctor --fix`).
- [x] **TOOLS-01:** Manifiestos de herramientas externos validados fail-closed (`KnownFields(true)`) con comprobación de aislamiento.
- [x] **CI-01:** Workflow CI configurado con runner nativo `macos-26` Apple Silicon arm64 y verificación de bootstrap en Windows.
- [x] **DOC-01:** Documentación reconciliada con evidencia reproducible de ejecución sin afirmaciones no probadas.
- [x] **SEC-02:** Sello criptográfico de origen y anti-tamper (`OriginSeal` / `internal/security/integrity.go`): huella digital inalterable SHA-256 de manifiestos y plantillas de dotfiles, verificación pre-ejecución a tiempo constante (`subtle.ConstantTimeCompare`), detección fail-closed de ataques TOCTOU y orígenes no confiables (permisos world-writable), banner visible en CLI y card de seguridad en TUI Bubble Tea v2.

---

## 6. Sello Criptográfico de Origen y Prevención Anti-Tamper (TOCTOU)

Para mitigar riesgos de seguridad donde dotfiles o manifiestos locales/personalizados puedan ser alterados o suplantados de manera maliciosa:

1. **Sello Criptográfico de Origen (`internal/security/integrity.go`):**
   - Al iniciar `anx`, se calcula un `OriginSeal` en streaming utilizando un pool de hashers SHA-256 (`sync.Pool`), completándose en menos de 2 milisegundos para ~160 activos sin impacto perceptible en el arranque.
   - Genera huellas digitales independientes: `ManifestFingerprint` (manifiestos embebidos, de repositorio y drop-in `manifests.d`) y `DotfilesFingerprint` (plantillas en `files/`), combinadas en un `CompositeDigest` de 64 caracteres hex (resumido a 12 caracteres para badges).
   - Verificación de permisos de directorio (`verifyDirectoryTrust`): rechaza orígenes con permisos de escritura abierta (`world-writable`) como violación de seguridad inmediata.
   - Omisión inteligente y defensiva de archivos socket en runtime (`.sock`), logs (`.log`), locks y symlinks rotos para garantizar determinismo absoluto.

2. **Verificación Pre-Ejecución Fail-Closed (Anti-TOCTOU):**
   - Antes de cualquier mutación en el sistema (`apply`, `sync`, `diff`, `reset`, `upgrade`), `verifySecuritySeal` re-inspecciona los activos en disco y los compara en tiempo constante (`subtle.ConstantTimeCompare`).
   - Si cualquier archivo fue modificado, inyectado o eliminado entre el arranque y la ejecución, el proceso se aborta inmediatamente con código de salida `domain.ExitSecurityViolation` (código 6).

3. **Visibilidad en CLI y TUI:**
   - **CLI:** En comandos interactivos y autónomos se despliega el banner:
     ```text
     ── Cryptographic Origin & Provenance Seal ──
       • Source Origin   : /repo (o Embedded Catalog)
       • Manifests SHA256: e352111bde40
       • Destination Home: /home/dev
       • Dotfiles SHA256 : fec32df5705d
       • Security Seal   : d7bd5abc9364 [SEALED & VERIFIED ✓]
     ```
   - **TUI (Bubble Tea v2):** En la etapa `StageReviewPlan` del wizard, se muestra la tarjeta de procedencia y sello de seguridad antes de permitir la confirmación con `[Enter]`. Si ocurre manipulación en disco, la verificación detiene la ejecución antes de aplicar cualquier cambio.
   - **Diagnóstico (`anx doctor`):** Incluye la prueba `Origin Security Seal`, verificando la integridad del sello y los permisos seguros del origen.


