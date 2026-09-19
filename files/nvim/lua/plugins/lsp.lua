return {
  "neovim/nvim-lspconfig",
  opts = {
    servers = {
      -- Configuración global aplicada a todos los servidores LSP
      ["*"] = {
        capabilities = {
          workspace = {
            didChangeWatchedFiles = {
              dynamicRegistration = false,
            },
          },
        },
      },
      -- Puedes mantener aquí tus configuraciones específicas por servidor
      gopls = {
        settings = {
          gopls = {
            hints = {
              parameterNames = true,
              constantValues = true,
              compositeLiteralFields = true,
            },
          },
        },
      },
    },
  },
}
