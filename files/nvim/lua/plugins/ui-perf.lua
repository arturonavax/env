return {
  -- Desactiva el cálculo animado de indentación de mini.indentscope
  {
    "nvim-mini/mini.indentscope",
    opts = {
      draw = {
        animation = function()
          return 0
        end,
      },
    },
  },
  -- Evita refrescar la barra de estado en cada evento del cursor
  {
    "nvim-lualine/lualine.nvim",
    opts = {
      options = {
        refresh = {
          statusline = 1000,
          tabline = 1000,
          winbar = 1000,
        },
      },
    },
  },
}
