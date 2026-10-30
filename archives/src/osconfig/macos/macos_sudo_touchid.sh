#!/bin/bash
# Inspiration: https://gist.github.com/RichardBronosky/31660eb4b0f0ba5e673b9bc3c9148a70
# This script is ready to copy-paste in whole, or just the line above (without the leading #)
#
# Use TouchID for sudo on modern MacBook Pro machines
# This script adds a single line to the top of the PAM configuration for sudo
# See: https://apple.stackexchange.com/q/259093/41827 for more info.
#
# Run: curl -fsSL "https://env.arturonavax.dev/macos_sudo_touchid.sh" | bash

if [[ "$(uname -s)" != "Darwin" ]]; then
	echo "!!! This script is for MacOS"

	exit 1
fi

# CommandLineTools
curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash

# Homebrew
if [[ "$(command -v brew)" == "" ]]; then
	/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi

brewbin="/usr/local/bin/brew"
[[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
[[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"

if [[ "$(command -v brew)" == "" ]]; then
	echo "!!! Homebrew installation error occurred." && exit 1
fi

brew reinstall pam-reattach

if [[ "$(command grep '^[^#]*auth       sufficient     pam_tid.so' /etc/pam.d/sudo)" == "" ]]; then
	sudo bash -eu <<'EOF'
  file=/etc/pam.d/sudo
  # A backup file will be created with the pattern /etc/pam.d/.sudo.1
  # (where 1 is the number of backups, so that rerunning this doesn't make you lose your original)
  bak=$(dirname $file)/.$(basename $file).$(echo $(ls $(dirname $file)/{,.}$(basename $file)* | wc -l))
  cp $file $bak
  awk -v is_done='pam_tid' -v rule='auth       sufficient     pam_tid.so' '
  {
    # $1 is the first field
    # !~ means "does not match pattern"
    if($1 !~ /^#.*/){
      line_number_not_counting_comments++
    }
    # $0 is the whole line
    if(line_number_not_counting_comments==1 && $0 !~ is_done){
      print rule
    }
    print
  }' > $file < $bak
EOF
fi

path_reattach="pam_reattach.so"
[[ "$(sysctl -n machdep.cpu.brand_string)" == *"Apple"* ]] && path_reattach="/opt/homebrew/lib/pam/pam_reattach.so"

if [[ "$(command grep "^[^#]*auth       optional       $path_reattach" /etc/pam.d/sudo)" == "" ]]; then
	sudo bash -eu <<'EOF'
  file=/etc/pam.d/sudo
  path_reattach="pam_reattach.so"
  [[ "$(sysctl -n machdep.cpu.brand_string)" == *"Apple"* ]] && path_reattach="/opt/homebrew/lib/pam/pam_reattach.so"
  bak=$(dirname $file)/.$(basename $file).$(echo $(ls $(dirname $file)/{,.}$(basename $file)* | wc -l))

  cp $file $bak
  awk -v is_done='pam_reattach' -v rule="auth       optional       $path_reattach" '
  {
    # $1 is the first field
    # !~ means "does not match pattern"
    if($1 !~ /^#.*/){
      line_number_not_counting_comments++
    }
    # $0 is the whole line
    if(line_number_not_counting_comments==1 && $0 !~ is_done){
      print rule
    }
    print
  }' > $file < $bak
EOF
fi
