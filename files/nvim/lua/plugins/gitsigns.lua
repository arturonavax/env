return {
  "lewis6991/gitsigns.nvim",
  opts = {
    current_line_blame = false, -- Activarlo a demanda con el toggle
    current_line_blame_opts = {
      virt_text = true,
      virt_text_pos = "right_align", -- Mantiene el blame visible pegado al borde derecho
      delay = 300,
      ignore_whitespace = false,
    },
    current_line_blame_formatter = " <author>, <author_time:%Y-%m-%d> • <summary>",
    preview_config = {
      border = "rounded",
      style = "minimal",
      relative = "cursor",
      row = 0,
      col = 1,
    },
  },
  keys = {
    -- Popup flotante completo con line wrap habilitado
    {
      "<leader>gb",
      function()
        require("gitsigns").blame_line({
          full = true,
        }, function(win_id)
          if win_id and vim.api.nvim_win_is_valid(win_id) then
            vim.wo[win_id].wrap = true
          end
        end)
      end,
      desc = "Git Blame Popup (Wrapped)",
    },
  },
}
