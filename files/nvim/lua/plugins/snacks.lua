local exclusions = require("config.exclusions")

return {
  {
    "folke/snacks.nvim",
    opts = {
      scroll = { enabled = false },
      picker = {
        -- Exclusiones globales para todos los pickers de búsqueda
        exclude = exclusions.picker_exclude,
        sources = {
          explorer = {
            -- El explorador de archivos muestra ABSOLUTAMENTE TODO:
            -- ignored = true: muestra archivos y carpetas en .gitignore (node_modules, .env, etc.)
            -- hidden = true: muestra archivos ocultos (dotfiles)
            -- exclude = {}: anula exclusiones para que las carpetas vendor sigan visibles
            ignored = true,
            hidden = true,
            exclude = {},
          },
          files = {
            exclude = exclusions.picker_exclude,
          },
          smart = {
            exclude = exclusions.picker_exclude,
          },
          grep = {
            exclude = exclusions.picker_exclude,
          },
          grep_word = {
            exclude = exclusions.picker_exclude,
          },
          grep_buffers = {
            exclude = exclusions.picker_exclude,
          },
        },
      },
    },
  },

  -- Compatibilidad y seguridad para neo-tree en caso de ser activado
  {
    "nvim-neo-tree/neo-tree.nvim",
    optional = true,
    opts = {
      filesystem = {
        filtered_items = {
          visible = true,
          hide_dotfiles = false,
          hide_gitignored = false,
        },
      },
    },
  },
}
