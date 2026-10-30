-- =============================================================================
-- ~/.config/wezterm/wezterm.lua — arturonavax/env WezTerm Configuration
-- =============================================================================

local wezterm = require 'wezterm'
local config = wezterm.config_builder()

config.color_scheme = 'Tokyo Night'
config.font = wezterm.font('JetBrainsMono Nerd Font')
config.font_size = 12.0
config.enable_tab_bar = false
config.window_decorations = "RESIZE"
config.audible_bell = "Disabled"
config.default_prog = { 'zsh', '-ic', 'herdr' }

return config
