local M = {}

-- Carpetas pesadas, vendors o generadas que deben excluirse de búsquedas de texto,
-- análisis de servidores LSP y autocompletado, pero que DEBEN ser visibles en el explorador.
M.dirs = {
  -- Gestores de paquetes y vendors
  "node_modules",
  ".pnpm-store",
  ".yarn",
  ".npm",
  "vendor",

  -- Control de versiones
  ".git",
  ".svn",
  ".hg",

  -- Python
  ".venv",
  "venv",
  "env",
  "__pycache__",
  ".pytest_cache",
  ".mypy_cache",
  ".ruff_cache",

  -- Rust
  "target",

  -- Builds, bundlers y frameworks
  "dist",
  "build",
  "out",
  ".next",
  ".nuxt",
  ".turbo",
  ".output",
  ".svelte-kit",
  ".angular",
  ".parcel-cache",

  -- Smart contracts / Solidity
  "artifacts",
  "cache",

  -- Reportes y cobertura
  "coverage",
  ".nyc_output",

  -- Caches y metadatos de IDEs / OS
  ".idea",
  ".vscode",
  ".cache",
  ".DS_Store",
}

-- Archivos pesados o generados que saturan las búsquedas de texto globales
M.files = {
  "package-lock.json",
  "pnpm-lock.yaml",
  "yarn.lock",
  "bun.lockb",
  "composer.lock",
  "Cargo.lock",
  "poetry.lock",
  "Pipfile.lock",
  "*.min.js",
  "*.min.css",
  "*.map",
}

-- Lista de exclusiones para Snacks Picker (files, grep, smart, etc.)
M.picker_exclude = {}
for _, dir in ipairs(M.dirs) do
  table.insert(M.picker_exclude, dir)
end
for _, file in ipairs(M.files) do
  table.insert(M.picker_exclude, file)
end

-- Patrones para vim.opt.wildignore
M.wildignore = {}
for _, dir in ipairs(M.dirs) do
  table.insert(M.wildignore, "*/" .. dir .. "/*")
end
for _, file in ipairs(M.files) do
  table.insert(M.wildignore, file)
end

-- Flags para ripgrep (utilizado por grug-far)
local rg_args = {}
for _, dir in ipairs(M.dirs) do
  table.insert(rg_args, "-g !" .. dir)
end
for _, file in ipairs(M.files) do
  table.insert(rg_args, "-g !" .. file)
end
M.rg_extra_args = table.concat(rg_args, " ")

-- Filtros de directorios para gopls
M.gopls_directory_filters = {
  "-**/node_modules",
  "-node_modules",
  "-vendor",
  "-**/.git",
  "-**/.venv",
  "-target",
  "-dist",
  "-build",
}

-- Exclusiones con glob para servidores LSP (vtsls, pyright, etc.)
M.lsp_exclude_dirs = {}
for _, dir in ipairs(M.dirs) do
  table.insert(M.lsp_exclude_dirs, "**/" .. dir)
end

-- Conjuntos hash O(1) para comprobaciones instantáneas
local dir_set = {}
for _, dir in ipairs(M.dirs) do
  dir_set[dir] = true
end

local file_set = {}
for _, file in ipairs(M.files) do
  if not file:find("%*") then
    file_set[file] = true
  end
end

-- Cache de rutas para evitar recomputaciones en bucles de autocompletado
local path_cache = {}
local path_cache_count = 0
local MAX_CACHE = 1000

---Comprueba si un nombre de archivo o carpeta está en la lista de exclusiones (O(1))
---@param name string
---@return boolean
function M.is_excluded_name(name)
  if not name or name == "" then
    return false
  end
  local clean = name:gsub("/+$", ""):match("[^/\\\\]+$") or name
  if dir_set[clean] or file_set[clean] then
    return true
  end
  if clean:match("%.min%.[a-zA-Z0-9]+$") or clean:match("%.map$") then
    return true
  end
  return false
end

---Comprueba si una ruta pertenece a algún directorio o archivo excluido (O(1) amortizado)
---@param path string
---@return boolean
function M.is_path_excluded(path)
  if not path or path == "" then
    return false
  end
  local cached = path_cache[path]
  if cached ~= nil then
    return cached
  end

  local excluded = false
  for segment in path:gmatch("[^/\\\\]+") do
    if dir_set[segment] or file_set[segment] then
      excluded = true
      break
    end
    if segment:match("%.min%.[a-zA-Z0-9]+$") or segment:match("%.map$") then
      excluded = true
      break
    end
  end

  if path_cache_count >= MAX_CACHE then
    path_cache = {}
    path_cache_count = 0
  end
  path_cache[path] = excluded
  path_cache_count = path_cache_count + 1

  return excluded
end

return M
