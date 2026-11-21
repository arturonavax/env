return {
  {
    "ledger/vim-ledger",
    ft = "ledger",
    init = function()
      vim.g.ledger_bin = "hledger"
      vim.g.ledger_is_hledger = true
      vim.g.ledger_fuzzy_account_completion = 1

      vim.filetype.add({
        extension = {
          journal = "ledger",
          hledger = "ledger",
          ledger = "ledger",
          ldg = "ledger",
        },
      })
    end,
  },
}
