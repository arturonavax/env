# ODD — Auditoría de cierre y CI/E2E multiplataforma (2026-10-31)

**Repositorio:** `arturonavax/env` · **Base auditada:** `main@d8ca78d74318c7e51f1712b1fc49c624cc0f6352` · **CI de referencia:** https://github.com/arturonavax/env/actions/runs/41203872117 · **Bloqueante:** Windows rojo, E2E anterior omitido.  
**Regla de cierre:** ninguna casilla se marca realizada hasta haber validado CI en SHA actual, pruebas positivas/negativas y evidencias; no confundir implementación parcial con aceptación de AC/V del control-plane v2.

## Specs y hallazgos comprobados

1. **CI Windows:** `TestHS11_PreservesPermissions` en `internal/configsync/capture_security_test.go` asume que NTFS representa `0o111`; el test debe verificar bytes en todos los sistemas y permisos POSIX solo donde existen. `TestMultiStrategyDispatcherGitHubTarballFallbackToScript` en `internal/providers/providers_test.go` falla por detección/ejecución de artefactos `.cmd`: el instalador `internal/providers/github_release.go` forzaba `.exe` incluso tras `.cmd`, y `probeBinaryVersion` no invocaba `cmd.exe /d /c`. No suprimir estos tests. Detectar regresiones de extensiones `.exe/.bat/.cmd`, PATH, espacios y unicode. La prueba debe demostrar instalación, reporte de versión, fallback y desinstalación.
2. **E2E omitida:** el job `e2e` tenía `needs: [test, lint]`; la falla Windows impidió ejecutar Debian y Alpine. Separar dependencias por gates para que un fallo Windows no suprima los diagnósticos Linux. Todo gate sigue siendo obligatorio para merge.
3. **E2E Linux eficientes:** construir un solo binario `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`; publicar artifact efímero; ejecutar **cinco contenedores nativos** sin reconstruir Go en cada imagen: Debian, Ubuntu, Fedora, Alpine, Arch. Cada runtime debe comprobar `/etc/os-release`, versión CLI, ayuda, `plan` JSON, `status` JSON y `sync --dry-run`; `--network none`, `--read-only`, `--cap-drop ALL`, non-root, recursos acotados, `/tmp` temporal y sin cambios al HOME real. Esto es un smoke E2E, **no** sustituye E2E profundo de bootstrap e instalación con mock-server; mantener Debian/Alpine full lifecycle como job independiente.
4. **Linux ARM64 ≠ Termux:** usar `GOOS=linux GOARCH=arm64 CGO_ENABLED=0` y QEMU/Ubuntu ARM64 como gate de compatibilidad ISA Linux; `tests/e2e/scripts/test_termux_native.sh` es un harness específico de Android/Termux con `PREFIX` y `uname -m=aarch64`; validarlo en un dispositivo Android Termux real, emulador Android viable o runner Android independiente. Prohibido reportar QEMU Linux ARM64 como prueba de Android Bionic/Termux. Sin hardware Android real, estado `not verified`.
5. **Estado de tareas aún no cerrado:** `internal/state/store.go` continúa usando `state.json` (no SQLite/WAL), `internal/bootstrap/bootstrap.go` conecta ese store; `internal/cli/self_update.go` imprime éxito sin descargar ni reemplazar binario; `internal/execution/pool.go` sigue agrupando tareas por lotes en vez de planificar ejecución DAG completa; `internal/app/control_plane.go` construye/ejecuta planes sin repositorio durable de planes autorizados ni revisión durable de host; `internal/extension/host.go` arranca subprocess antes de verificar permisos declarados, por lo que el handshake no es aislamiento OS. `internal/cas/cas.go` reutiliza blobs comprobando tamaño, no contenido, en fast-path. Todas estas observaciones requieren revisión/fix con pruebas de fallo.
6. **Supply chain y documentación:** `README.md` todavía menciona Go ≥1.24, aunque `go.mod` exige 1.27.3. Los módulos Go no han cambiado respecto al HEAD auditado anterior. La rama de CI migra actions checkout/setup-go/setup-node a v7 publicados; auditar compatibilidad, pinning SHA y mínimos privilegios, y añadir scan de secretos/vulnerabilidades, SBOM y pruebas de release antes de certificar T02/T99.
7. **Seguridad dotfiles:** `files/herdr/config.toml` es la única plantilla permitida; preservar exclusión absoluta de sesiones/sockets incluso si cambian nombres. `capture --commit` debe stagear únicamente el Changeset validado, no `git add files/`, ejecutar secret scanner de forma fail-closed y garantizar `--dry-run` sin escrituras/checkout/commit/push. Revisar `internal/configsync/capture.go`, git staging y symlinks con tests adversariales.
8. **Autonomía de pruebas:** tests de la matriz no deben ejecutar `apply`, instalaciones de paquetes, curl o scripts de terceros sobre el runner real. Los tests de acciones mutantes quedan dentro de contenedores con mock server, con userns/no privilegios y repositorio de prueba aislado.

## Tasks verificables

| ID | Prioridad | Condición de salida | Estado en base auditada |
|---|---|---|---|
| W01 | P0 | Windows `TestHS11` portable y test de archivo ejecutable POSIX en Linux/macOS; 3 OS verdes | ❌ |
| W02 | P0 | Windows `.cmd` instalado con nombre intacto; detección de versión y fallback; suite Windows verde | ❌ |
| E01 | P0 | E2E Debian/Alpine completos ejecutan aun si Windows falla y reportan resultados | ❌ |
| E02 | P1 | Matriz smoke Debian, Ubuntu, Fedora, Alpine, Arch en containers no privilegiados y aislados | ❌ |
| E03 | P1 | Pipeline compila 1 binario compartido; caché de Go; tiempo/coste medidos | ❌ |
| E04 | P1 | Linux ARM64 comprobado realmente; Android Termux ARM64 nativo separado y verificado | ❌ |
| E05 | P1 | Pruebas profundas Docker no simulan éxito por `SKIP` silenciosos y cubren paquete real por distro | ❌ |
| A01 | P0 | SQLite/WAL + migración JSON rollback, integridad y multiwriter bajo fallo | ❌ |
| A02 | P0 | Self-update autentica release, compara versiones, verifica checksums/firma y aplica/recupera | ❌ |
| A03 | P0 | Plan autorizado exacto, stale revision, job durable y Cancel/replay | ❌ |
| A04 | P1 | Scheduler DAG real, concurrency/lock/failure rollback bajo escenarios adversariales | ❌ |
| A05 | P0 | Extensiones aisladas de permisos OS, handshake y límites reales | ❌ |
| A06 | P1 | CAS revalida SHA-256 en todos los paths y reproduce corrupción/carrera | ❌ |
| A07 | P1 | Documentación/versiones/acciones y dependencia seguro verificadas por evidencia | ❌ |
| A08 | P0 | Herdr allowlist + Changeset Git staging + secretos + symlink evasions + dry-run 0 mutations | Revalidar |
| Z99 | Gate | AC-01..AC-19/V01..V15, CI completo y E2E reproducibles en SHA actual; todos los P0/P1 resueltos | ❌ |

## Evidencia y cierre

- CI roja: https://github.com/arturonavax/env/actions/runs/41203872117/job/124544898045 ; errores lineales en tests `HS11` y `MultiStrategyDispatcher` (no es el fallo de scripts mock del run anterior).
- Matriz y parches propuestos en rama `fix/windows-ci-e2e-linux-matrix-20261031`. **Su existencia no implica que hayan aprobado**. Ejecutar workflow en PR y archivar URLs de los seis targets (5 distros + arm64) y tests Windows. Si los tests fallan, corregir la causa, no saltarlos.
- Dependencias de CI: `test`, `lint`, `e2e-build`, `e2e-smoke`, `e2e-arm64-linux`, `e2e`; todos requeridos para merge. Verificar que los jobs `needs` no se salten indebidamente.
- Reconciliar `odd/tasks/anx-control-plane-v2.md`, `odd/tasks/anx-control-plane-completion-go127.md` y `odd/tasks/anx-control-plane-final-remediation-herdr-ci.md`: cambiar `[x]` a `blocked`/`partial` donde no exista aceptación. No borrar bitácoras anteriores.
- Mantener benchmark de cold start/CPU/RAM, evitar duplicación entre Dockerfiles, pin images por digest tras validación y considerar Podman **solo si** la prueba operativa sobre GitHub Actions muestra una ventaja real; no reemplazar Docker sin evidencia.
