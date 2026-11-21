local exclusions = require("config.exclusions")

return {
  {
    "mason-org/mason.nvim",
    opts = function(_, opts)
      if type(opts.ensure_installed) ~= "table" then
        return
      end
      local bin_aliases = {
        delve = "dlv",
        ["rust-analyzer"] = "rust-analyzer",
      }
      local go_pkgs = {
        goimports = true,
        gofumpt = true,
        gomodifytags = true,
        impl = true,
        ["golangci-lint"] = true,
        delve = true,
      }
      local has_go = vim.fn.executable("go") == 1
      local has_npm = vim.fn.executable("npm") == 1
      local has_python_venv = nil
      local function check_python_venv()
        if has_python_venv ~= nil then
          return has_python_venv
        end
        if vim.fn.executable("python3") == 1 then
          vim.fn.system({ "python3", "-c", "import venv, ensurepip" })
          has_python_venv = (vim.v.shell_error == 0)
        else
          has_python_venv = false
        end
        return has_python_venv
      end
      local npm_pkgs = {
        ["nomicfoundation-solidity-language-server"] = true,
        solhint = true,
        prettier = true,
      }
      local python_pkgs = {
        sqlfluff = true,
      }

      local filtered = {}
      for _, pkg in ipairs(opts.ensure_installed) do
        local bin = bin_aliases[pkg] or pkg
        local already_present = vim.fn.executable(bin) == 1
        local missing_runtime = (go_pkgs[pkg] and not has_go)
          or (npm_pkgs[pkg] and not has_npm)
          or (python_pkgs[pkg] and not already_present and not check_python_venv())
        if not already_present and not missing_runtime then
          table.insert(filtered, pkg)
        end
      end
      opts.ensure_installed = filtered
    end,
  },
  {
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
        mason = (vim.fn.executable("gopls") == 0 and vim.fn.executable("go") == 1),
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
        mason = (vim.fn.executable("rust-analyzer") == 0 and vim.fn.executable("cargo") == 1),
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
  },
}
