return {
  {
    "yetone/avante.nvim",
    opts = {
      provider = "claude", -- Evita la comprobación inmediata de copilot.lua al arrancar
    },
  },

  -- Orquestador de código y agentes con Gemini
  {
    "olimorris/codecompanion.nvim",
    dependencies = {
      "nvim-lua/plenary.nvim",
      "nvim-treesitter/nvim-treesitter",
    },
    cmd = { "CodeCompanion", "CodeCompanionChat", "CodeCompanionActions" },
    keys = {
      { "<leader>aa", "<cmd>CodeCompanionChat Toggle<cr>", desc = "Toggle AI Chat (Gemini)" },
      { "<leader>ap", "<cmd>CodeCompanionActions<cr>", desc = "AI Actions Palette" },
    },
    opts = {
      log_level = "DEBUG",
      adapters = {
        gemini = function()
          return require("codecompanion.adapters").extend("gemini", {
            env = {
              api_key = "GEMINI_API_KEY",
            },
            schema = {
              model = {
                default = "gemini-3.1-pro", -- Ajusta al modelo disponible en tu plan
              },
            },
          })
        end,
      },
      strategies = {
        chat = {
          adapter = "gemini",
          keymaps = {
            send = {
              modes = { n = "<CR>", i = "<C-s>" },
            },
          },
        },
        inline = {
          adapter = "gemini",
        },
      },
    },
  },

  -- Acceso instantáneo a agy-cli en terminal flotante
  {
    "folke/snacks.nvim",
    opts = {},
    keys = {
      {
        "<leader>ag",
        function()
          Snacks.terminal.toggle("agy", {
            cwd = LazyVim.root(),
            win = {
              position = "float",
              width = 0.85,
              height = 0.85,
              border = "rounded",
            },
          })
        end,
        desc = "Lanzar Antigravity CLI (agy)",
      },
    },
  },
}
