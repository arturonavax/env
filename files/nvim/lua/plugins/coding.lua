local exclusions = require("config.exclusions")

return {
  -- Autocompletado (blink.cmp)
  {
    "saghen/blink.cmp",
    opts = function(_, opts)
      opts.sources = opts.sources or {}
      opts.sources.providers = opts.sources.providers or {}

      -- Excluye directorios y archivos pesados en el autocompletado de rutas (path)
      opts.sources.providers.path = vim.tbl_deep_extend("force", opts.sources.providers.path or {}, {
        opts = {
          show_hidden_files_by_default = false,
        },
        transform_items = function(ctx, items)
          return vim.tbl_filter(function(item)
            return not exclusions.is_excluded_name(item.label)
          end, items)
        end,
      })

      -- Optimización de alto rendimiento: limita el autocompletado de buffers a ventanas visibles
      -- del proyecto, descartando instantáneamente buffers que pertenezcan a node_modules o vendors
      opts.sources.providers.buffer = vim.tbl_deep_extend("force", opts.sources.providers.buffer or {}, {
        opts = {
          get_bufnrs = function()
            return vim
              .iter(vim.api.nvim_list_wins())
              :map(function(win)
                return vim.api.nvim_win_get_buf(win)
              end)
              :filter(function(buf)
                if vim.bo[buf].buftype ~= "" then
                  return false
                end
                local name = vim.api.nvim_buf_get_name(buf)
                return name ~= "" and not exclusions.is_path_excluded(name)
              end)
              :totable()
          end,
        },
      })

      return opts
    end,
  },

  -- Búsqueda y reemplazo en masa (grug-far)
  {
    "MagicDuck/grug-far.nvim",
    opts = function(_, opts)
      opts = opts or {}
      opts.engines = opts.engines or {}
      opts.engines.ripgrep = opts.engines.ripgrep or {}
      local current = opts.engines.ripgrep.extraArgs or ""
      if not current:find("!node_modules", 1, true) then
        opts.engines.ripgrep.extraArgs = (current .. " " .. exclusions.rg_extra_args):gsub("^%s+", "")
      end
      return opts
    end,
  },
}
