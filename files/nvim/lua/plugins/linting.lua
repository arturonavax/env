return {
  "mfussenegger/nvim-lint",
  opts = {
    -- Elimina "InsertLeave" y "BufReadPost"
    events = { "BufWritePost" },
  },
}
