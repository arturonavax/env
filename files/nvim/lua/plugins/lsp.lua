local exclusions = require("config.exclusions")

return {
  "neovim/nvim-lspconfig",
  init = function()
    -- Evita que cualquier observador de archivos de LSP intente vigilar
    -- carpetas masivas de dependencias (node_modules, vendor, etc.) a nivel de Neovim
    if vim.lsp and vim.lsp._watchfiles and vim.lsp._watchfiles._watchfunc then
      local orig_watchfunc = vim.lsp._watchfiles._watchfunc
      vim.lsp._watchfiles._watchfunc = function(path, opts, callback)
        if exclusions.is_path_excluded(path) then
          return function() end
        end
        return orig_watchfunc(path, opts, callback)
      end
    end
  end,
  opts = {
    servers = {
      -- Configuración global aplicada a todos los servidores LSP
      ["*"] = {
        capabilities = {
          workspace = {
            -- Desactiva el registro dinámico de observadores de archivos para evitar
            -- que los servidores sature Neovim vigilando miles de archivos en node_modules
            didChangeWatchedFiles = {
              dynamicRegistration = false,
            },
          },
        },
      },

      -- Go: excluye node_modules, vendor y directorios pesados del análisis
      gopls = {
        settings = {
          gopls = {
            hints = {
              parameterNames = true,
              constantValues = true,
              compositeLiteralFields = true,
            },
            ["build.directoryFilters"] = exclusions.gopls_directory_filters,
          },
        },
      },

      -- TypeScript / JavaScript (vtsls)
      vtsls = {
        settings = {
          typescript = {
            tsserver = {
              watchOptions = {
                excludeDirectories = exclusions.lsp_exclude_dirs,
              },
            },
            preferences = {
              includePackageJsonAutoImports = "off",
            },
          },
          javascript = {
            tsserver = {
              watchOptions = {
                excludeDirectories = exclusions.lsp_exclude_dirs,
              },
            },
            preferences = {
              includePackageJsonAutoImports = "off",
            },
          },
        },
      },

      -- TypeScript / JavaScript (ts_ls / tsserver alternativo)
      ts_ls = {
        settings = {
          typescript = {
            tsserver = {
              watchOptions = {
                excludeDirectories = exclusions.lsp_exclude_dirs,
              },
            },
          },
          javascript = {
            tsserver = {
              watchOptions = {
                excludeDirectories = exclusions.lsp_exclude_dirs,
              },
            },
          },
        },
      },

      -- Python (pyright / basedpyright)
      pyright = {
        settings = {
          python = {
            analysis = {
              exclude = exclusions.lsp_exclude_dirs,
              ignore = exclusions.lsp_exclude_dirs,
            },
          },
        },
      },
      basedpyright = {
        settings = {
          python = {
            analysis = {
              exclude = exclusions.lsp_exclude_dirs,
              ignore = exclusions.lsp_exclude_dirs,
            },
          },
        },
      },

      -- Rust (rust-analyzer)
      ["rust-analyzer"] = {
        settings = {
          ["rust-analyzer"] = {
            files = {
              excludeDirs = exclusions.dirs,
            },
          },
        },
      },
    },
  },
}
