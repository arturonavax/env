#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_ui_box.sh
# Unit tests for dynamic terminal UI box drawing and normalization library.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/ui_box.sh"

echo "  Running Dynamic UI Box & Border Normalization Tests..."

# Test 1: Width Detection & Clamping
w_clamped_min="$(ui_box_width 70 50 100)"
if ((w_clamped_min >= 50 && w_clamped_min <= 100)); then
	pass "ui_box_width clamped within valid min/max bounds: $w_clamped_min"
else
	fail "ui_box_width failed to clamp within bounds: $w_clamped_min"
fi

# Test 2: ANSI Escape Stripping and Visual Length
styled_text="$(printf '\033[1;34mHello\033[0m \033[32mWorld!\033[0m')"
plain_text="$(ui_strip_ansi "$styled_text")"
assert_equals "Hello World!" "$plain_text" "ui_strip_ansi strips color codes cleanly" &&
	pass "ui_strip_ansi stripped complex ANSI formatting"

vis_len="$(ui_visible_length "$styled_text")"
assert_equals "12" "$vis_len" "Visual length of formatted text is exactly 12" &&
	pass "ui_visible_length accurately calculated visual width ignoring ANSI"

# Test 3: Repeating Character Generator
rep_line="$(ui_repeat_char "═" 15)"
assert_equals "═══════════════" "$rep_line" "ui_repeat_char produced 15 double-line glyphs" &&
	pass "ui_repeat_char generated accurate glyph sequence"

# Test 4: Box Border Glyphs (Rounded & Double Styles)
top_rounded="$(ui_box_top 30 rounded "")"
assert_contains "$top_rounded" "╭" "Rounded top contains ╭" &&
	assert_contains "$top_rounded" "╮" "Rounded top contains ╮" &&
	pass "ui_box_top rendered rounded corner glyphs"

bot_rounded="$(ui_box_bottom 30 rounded "")"
assert_contains "$bot_rounded" "╰" "Rounded bottom contains ╰" &&
	assert_contains "$bot_rounded" "╯" "Rounded bottom contains ╯" &&
	pass "ui_box_bottom rendered rounded bottom glyphs"

top_double="$(ui_box_top 30 double "")"
assert_contains "$top_double" "╔" "Double top contains ╔" &&
	assert_contains "$top_double" "╗" "Double top contains ╗" &&
	assert_contains "$top_double" "═" "Double top contains ═" &&
	pass "ui_box_top rendered double-line glyphs"

# Test 5: Dynamic Box Line Padding & Alignment
box_line="$(ui_box_line "Centered" 30 rounded "" "" "center")"
assert_contains "$box_line" "│" "Box line includes vertical border" &&
	assert_contains "$box_line" "Centered" "Box line includes text payload" &&
	pass "ui_box_line formatted centered text with border alignment"

# Test 6: Horizontal Separator Rules
sep="$(ui_box_separator 20 "─" "")"
assert_equals "────────────────────" "$sep" "ui_box_separator drew exact 20-column rule" &&
	pass "ui_box_separator drew accurate rule"

# Test 7: Python UIBox Class Validation
py_uibox_out="$(python3 -c '
import os, importlib.machinery, importlib.util
path = os.path.abspath("files/ai/aitools/aitools")
loader = importlib.machinery.SourceFileLoader("aitools", path)
spec = importlib.util.spec_from_loader("aitools", loader)
aitools = importlib.util.module_from_spec(spec)
loader.exec_module(aitools)
UIBox = aitools.UIBox

s = "\033[1;31mTest\033[0m"
assert UIBox.visible_len(s) == 4, f"visible_len failed: {UIBox.visible_len(s)}"
w = UIBox.get_width(60, 40, 80)
assert 40 <= w <= 80, f"get_width failed: {w}"
top = UIBox.top(40, style="rounded")
assert "╭" in top and "╮" in top, f"top failed: {top}"
line = UIBox.line("Hello", 40)
assert "│" in line and "Hello" in line, f"line failed: {line}"
print("PY_UIBOX_OK")
')"
assert_equals "PY_UIBOX_OK" "$py_uibox_out" "Python UIBox class passed all assertions" &&
	pass "Python UIBox class unit assertions verified successfully"
