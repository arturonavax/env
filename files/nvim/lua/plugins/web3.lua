return {
  -- 1. Resaltado de sintaxis
  {
    "nvim-treesitter/nvim-treesitter",
    opts = function(_, opts)
      if type(opts.ensure_installed) == "table" then
        vim.list_extend(opts.ensure_installed, { "solidity" })
      end
    end,
  },

  -- 2. Asegurar binarios en Mason
  {
    "mason-org/mason.nvim",
    opts = function(_, opts)
      opts.ensure_installed = opts.ensure_installed or {}
      vim.list_extend(opts.ensure_installed, {
        "nomicfoundation-solidity-language-server",
        "solhint",
      })
    end,
  },

  -- 3. Configuración del Servidor LSP de Solidity
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        solidity_ls_nomicfoundation = {},
      },
    },
  },

  -- 4. Formateo con Forge fmt (ultrarrápido)
  {
    "stevearc/conform.nvim",
    opts = {
      formatters_by_ft = {
        solidity = { "forge_fmt" },
      },
      formatters = {
        forge_fmt = {
          command = vim.fn.expand("~/.foundry/bin/forge"),
          args = { "fmt", "--raw", "-" },
          stdin = true,
        },
      },
    },
  },

  -- 5. Linter de seguridad y estilo
  {
    "mfussenegger/nvim-lint",
    opts = {
      linters_by_ft = {
        solidity = { "solhint" },
      },
    },
  },
}
