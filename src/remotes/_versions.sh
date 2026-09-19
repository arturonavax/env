#!/bin/bash
# This file centralizes the locked/pinned versions of core tools and dependencies.
# Every variable uses the ${VAR:-default} pattern so it can be overridden at runtime
# (e.g. NVIM_VERSION=v0.13.0 ./install.sh editor).

# Terminal & Editors
export GHOSTTY_VERSION="${GHOSTTY_VERSION:-1.3.1}"
export NVIM_VERSION="${NVIM_VERSION:-v0.12.5}"
export TMUX_VERSION="${TMUX_VERSION:-3.7c}"

# Node & Package Managers
export PNPM_VERSION="${PNPM_VERSION:-12.4.2}"

# Go Tools (compiled locally)
export GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.13.2}"
export LAZYGIT_VERSION="${LAZYGIT_VERSION:-v0.44.1}"
export GOPLS_VERSION="${GOPLS_VERSION:-v0.23.0}"
export GOFUMPT_VERSION="${GOFUMPT_VERSION:-v0.12.0}"
export DLV_VERSION="${DLV_VERSION:-v1.27.2}"
export SHFMT_VERSION="${SHFMT_VERSION:-v3.8.0}"
export ACTIONLINT_VERSION="${ACTIONLINT_VERSION:-v1.7.1}"
export FX_VERSION="${FX_VERSION:-v35.0.0}"
export AIR_VERSION="${AIR_VERSION:-v1.61.7}"
export GOSEC_VERSION="${GOSEC_VERSION:-v2.22.1}"

# Rust Tools (compiled locally with native CPU flags)
export STYLUA_VERSION="${STYLUA_VERSION:-0.20.0}"
export SHELLHARDEN_VERSION="${SHELLHARDEN_VERSION:-4.3.1}"

# Fonts
export NERD_FONTS_VERSION="${NERD_FONTS_VERSION:-v3.5.1}"

# AI Tools
export AGY_VERSION="${AGY_VERSION:-1.2.7}"

