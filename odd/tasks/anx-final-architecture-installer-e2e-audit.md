# ANX ODD — Auditoría y remediación de instalación, script IR, herramientas y E2E (2026-10-31)

**Baseline:** `main@f1f1d1905c0466f0827a40f517ba0e1d1be3a3cf`; [CI verde](https://github.com/arturonavax/env/actions/runs/41210643807). Estado: **PARTIAL, NO CERRADO**. No alterar los ODD antiguos ni tratar casillas [x] como evidencia. Este documento requiere validación de CI nuevo.

## P0 — Fallos verificables

- **IR-01 (arquitectura):** `internal/scriptgen/builder.go:TranslatePlan` emite `echo 'Installing package...'` en instalaciones; `internal/scriptgen/translators.go:TranslateSync` emite echo en sincronización. Es un *audit transcript*, NO traducción ejecutable equivalente. `Render()` incorpora `time.Now()` por defecto y comentarios con interpolación no escapada: no garantiza contenido reproducible ni inyección segura. **Prohibir que --script se presente como alternativa operacional completa hasta rediseñarlo**.
- **LIFE-01:** `internal/cli/self_update.go` anuncia éxito de actualización sin descargar asset ni reemplazar binario. La release v1.0.0 (2023) no contiene assets. Implementar pipeline de release reproducible, checksum firmado/provenance, matriz de artefactos, resolver canal/versión, staging/verificación/rollback, update/check/dry-run reales, pruebas de red local.
- **STATE-01:** `internal/state/store.go` persiste JSON, no SQLite/WAL, y libera mutex antes de persistir snapshot. Riesgo de escrituras fuera de orden, multi-proceso y pérdida de datos. Implementar SQLite WAL transaccional o revisar requisito y demostrar alternativa bajo fallos.
- **DAG-01:** `internal/execution/pool.go` agrupa pasos no exclusivos contiguos y los lanza juntos, sin bloquear cada paso hasta cumplir todos sus DependsOn. Programación DAG real con colas ready, recursos locks, bounded pool, cancelación, rollback y tests de orden aleatorio.
- **SEC-01:** `internal/extension/host.go` ejecuta procesos antes del permission handshake: el handshake **no** es sandbox de SO. Aislamiento pre-launch y política verificable de filesystem/network/process, o documentar explícitamente trusted-only (sin venderlo como aislamiento).
- **CAS-01:** `internal/cas/cas.go` reutiliza blobs del mismo tamaño sin comprobar contenido. Revalidar hash, concurrencia, corrupción y error handling.
- **INSTALL-01:** `install.sh` obtiene repo/git antes de seleccionar modo; requiere bash aun para release; `scripts/bootstrap_anx.sh` intenta release de forma automática pero no existe selector preflight compacto release/source. No hay release assets oficiales disponibles. En Windows `install.ps1` siempre clona y compila Go; `scripts/bootstrap_anx.ps1` tiene texto Go>=1.24 obsoleto. El bootstrap usa `tar -xzf` directamente al DEST_BIN, sin comprobación de SHA/provenance/paths del tarball ni staging atómico; riesgo al instalar remoto.
- **TOOLS-01:** `internal/tools/registry.go` soporta manifests de tool externos y paths aislados, `runner.go` forwarding y `updater.go` estrategias. Pero `anx ai` está compilado en `internal/ai`, por lo que no es un proyecto externo plenamente desacoplado; registrar contrato de tool, compatibilidad de versiones y version probing por lenguaje, separación lifecycle, y pruebas stand-alone sin anx. Fallos YAML de overlay se ignoran en loadFromDir: no fail-closed.
- **CI-01:** `main` verde demuestra tests Go en ubuntu/macOS Intel/Windows; E2E full Debian/Alpine y smoke cinco distros; Linux arm64-QEMU ≠ Termux. Falta macOS arm64 nativo y Windows install E2E. `test_concurrency.sh` no ejercita concurrencia real; no contarlo como concurrencia. `e2e-build` y Alpine/Debian smoke añaden duplicación liviana; optimizar coste con métricas, no sacrificar cobertura.
- **DOC-01:** ODD y README presentan funciones no implementadas o afirmaciones incongruentes (SQLite, actualización, paths, Go mínimo); reconciliar especificación vs evidencia real; `main` protected=false al auditar.
 
## Arquitectura de ejecución & script — objetivo

Usar **una única representación inmutable y tipada de operaciones (Operation IR)** para `plan → validate/authorize → execute` y `plan → compile`. Operaciones con tipos como ExecArgv, PackageInstall, CopyFile, Mkdir, RemoveOwned, AtomicReplace, GitOp, MergeConfig, VerifyChecksum; cada op declara plataforma, proveedor, precondiciones, dependencias, ownership, idempotencia, compensación, side effects y secretos. Backend Go ejecuta operaciones con args sin shell; backends Bash POSIX/PowerShell traducen **solo las operaciones expresables semánticamente**, con quoting específico, allowlisted commands y validación static; operaciones que requieren Go runtime/snapshots/AST generan `unsupported` explícito o envuelven una ejecución anx versionada; nunca `echo` fingiendo éxito. Evitar dos implementaciones divergentes del negocio. Función determinista `Compile(IR, Target, Options)` con reloj inyectado o sin timestamp, sort estable y golden snapshots. Registrar manifest IR schema version, plan hash y provenance, checks por shellcheck, sh -n, PSScriptAnalyzer, run con mocks en contenedores, igualdad de efectos `native apply` vs `generated script` en entornos herméticos y tests adversariales de quoting/rutas/secrets.

## Diseño de instalador público refinado

**Nombre propuesto:** `sh.arturonavax.dev` para instalador HTTP y alias `env.arturonavax.dev` para documentación/redirección. Entry POSIX `curl -fsSL https://sh.arturonavax.dev | sh` (dominio NO configurado/verificado). El Cloudflare Worker debe servir script versionado desde publicación confiable (Git SHA/tag/digest), TLS, `Content-Type: text/x-shellscript`, caching apropiado, no ejecutar sustitución dinámica no auditada, modo offline/documentación. Al inicio, TTY: header compacto y pregunta Release (por defecto si hay asset soportado y verificado), Source, Plan-only, Quit; TTY via `/dev/tty` sin tragarse stdin del pipe; sin TTY: flags/env explícitos con decisión segura, nunca prompts ni mutación inesperada. Preflight detecta OS/arch/libc (Linux glibc/musl), distro/manager, Termux/Android Bionic vs Linux/aarch64, Mac darwin arm64/amd64, root/prefix/sudo, CLI dependencies. Solo descarga repo y Go en modo source. Releases con SHA256 verificable anclado a fuente autenticada y preferentemente firma/attestation + verificación de identidad, cobertura arm64/amd64 glibc/musl/macOS/Windows. Descarga `mktemp` + límites + tar safe extraction + validación de paths + staging/atomic switch/rollback + ownership, nunca overwrite a ciegas. En Windows una entrada PowerShell equivalente (documentada; no obligar CMD a pipe incompatible) soporta ambos modos. Interactive compact wizard y TUI completa permanecen etapas distintas.

Cloudflare Worker **especificación**, no desplegado: personal domain mapping, routing, source audit, logs sin PII, tests Content-Type/Cache-Control/security, cierre de acceso al origin si procede. Confirmar dominio/DNS/control antes de publicar receta final.

## E2E eficiente y SIN falsas garantías

| Gate | OS/target | Escenarios exclusivos |
|---|---|---|
| unit | Ubuntu 24.04 + macOS 26 ARM64 + Windows | `go test`/race donde compatible, vet, build, shell syntax |
| linux-native-unit | Fedora/Arch/Ubuntu | Aislamiento entre detectores/gestores y operaciones reales simuladas con fixture, no repetir todo `go test` |
| smoke-runtime | Ubuntu, Fedora, Arch, Debian, Alpine | CLI plan/status/dry-run/no-network; si Debian/Alpine full ya incluye smoke, eliminar esos dos jobs redundantes tras igualar assertions |
| full-lifecycle | Debian + Alpine | Fallback/release/source/curl, TTY/PTy, dotfiles/snapshots, bugs musl; cada escenario setup/teardown aislado |
| mac-arm | macos-26 (Apple Silicon) | darwin/arm64 verificado, tests race/build, instalador POSIX nativo, release signed, symlinks, TTY/PTy |
| win-e2e | Windows runner | PowerShell install release/source fixture, PATH, aliases .exe/.cmd, safe rollback, no WSL |
| linux-arm | Ubuntu ARM64 native preferible, QEMU solo fallback | ejecución binaria aarch64, no Android |
| termux-native | Dispositivo/emulador Android con Termux Bionic | `pkg`, PREFIX, Android aarch64, instalación completa, sin privilegios |
| supply-chain | Release tag/release assets | descarga verificada, checksums/provenance/attestation, rollback al fallo |
| script-IR | Diferencial por plataforma | effects equivalentes entre `apply` y script en fixture, quoting y escapes |

- El job Linux E2E debe informar sobre cobertura de cada escenario, costo/tiempos y artifact de logs; nunca `SKIP` silenciosos.
- Tests de concurrencia real, races y crash injection deben estar en Go/integration, no scripts E2E falsamente etiquetados.
- En CI, no ejecutar `apt/dnf/pacman` en runners host; sólo contenedores aislados.
- Tests en PR obligatorios y branch protection; pin acciones/imágenes por SHA/digest tras validar.
- Se adjunta cambio de runner a `macos-26` y gate `uname -m=arm64`; su prueba aún pendiente de CI.

## DoD innegociable

- [ ] Auditoría de logs y tests de cada gate por SHA de PR.
- [ ] `anx --script` generador determinista y **semánticamente equivalente** en subset soportado o fail-closed.
- [ ] Release assets publicados y verificados, instalador UX release/source en Linux/macOS/Windows y documentación Termux honesta.
- [ ] Tools 100% portables como proyectos independientes, sin depender del monolito para ejecución, versionado y distribución.
- [ ] SQLite/CAS/DAG/plug-in sandbox o un ADR que modifique explícitamente los requisitos con pruebas alternativas.
- [ ] Cero falsos E2E, cero duplicate testing injustificado, benchmarks y tests de migración.
- [ ] macOS Apple Silicon nativo verde.
- [ ] Windows installers/Termux verificados o marcados pendientes, nunca cerrados sin ejecución.
- [ ] Seguridad de captura, instalación y actualización bajo tests adversariales.
- [ ] Documentación y ODD reconciliados y cierre firmado con resultados reproducibles.

**No marcar completado por el mero hecho de abrir PR o porque la unidad compiló.**
