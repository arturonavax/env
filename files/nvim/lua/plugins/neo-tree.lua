return {
  "nvim-neo-tree/neo-tree.nvim",
  opts = {
    enable_git_status = false, -- Desactiva git recursivo en el árbol (evita lag en monorepos)
    enable_diagnostics = false, -- No computar diagnósticos LSP sobre el árbol de archivos
    filesystem = {
      use_libuv_file_watcher = false, -- CRUCIAL: inotify/fs_events en carpetas grandes satura la CPU
      bind_to_cwd = true,
      follow_current_file = {
        enabled = true,
        leave_dirs_open = false, -- No mantiene abiertas ramas innecesarias
      },
      filtered_items = {
        visible = false,
        hide_dotfiles = false,
        hide_gitignored = true,
        -- "never_show" omite los directorios del escaneo del sistema de archivos por completo
        never_show = {
          ".git",
          "node_modules",
          ".venv",
          "venv",
          "target",
          "dist",
          "build",
          ".next",
          ".cache",
          "__pycache__",
        },
      },
    },
    window = {
      mappings = {
        -- Evita colisión de "a" accidental: exige presionar "A" (Shift+a) para crear archivo/directorio
        ["a"] = "none",
        ["A"] = "add",
        ["c"] = "none", -- Desactiva copy directo accidental
        ["C"] = "copy",
      },
    },
  },
}
