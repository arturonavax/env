return {
  "Wansmer/symbol-usage.nvim",
  event = "LspAttach",
  opts = {
    vt_position = "above", -- Posición encima de la definición (estilo JetBrains)
    kinds = {
      vim.lsp.protocol.SymbolKind.Function,
      vim.lsp.protocol.SymbolKind.Method,
      vim.lsp.protocol.SymbolKind.Interface,
      vim.lsp.protocol.SymbolKind.Struct,
    },
    references = { enabled = true },
    implementation = { enabled = true },
    definition = { enabled = false },
    text_format = function(symbol)
      local fragments = {}

      if symbol.references then
        local usage = symbol.references == 1 and "reference" or "references"
        table.insert(fragments, string.format("%s %s", symbol.references, usage))
      end

      if symbol.implementation then
        local impl = symbol.implementation == 1 and "implementation" or "implementations"
        table.insert(fragments, string.format("%s %s", symbol.implementation, impl))
      end

      return table.concat(fragments, " | ")
    end,
  },
}
