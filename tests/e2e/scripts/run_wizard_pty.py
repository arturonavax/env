#!/usr/bin/env python3
"""
Simulate interactive terminal session (PTY) for testing Bubble Tea v2 wizard.
Feeds navigation and confirmation keystrokes to advance through all wizard stages.
"""

import os
import pty
import select
import sys
import time

def run_wizard():
    bin_path = os.path.expanduser("~/.local/bin/anx")
    if not os.path.exists(bin_path):
        print(f"[ERR] anx binary not found at {bin_path}", file=sys.stderr)
        sys.exit(1)

    print("[INFO] Spawning anx wizard inside PTY...")
    master_fd, slave_fd = pty.openpty()

    pid = os.fork()
    if pid == 0:
        # Child process: inside slave PTY
        os.close(master_fd)
        os.setsid()
        os.dup2(slave_fd, 0)
        os.dup2(slave_fd, 1)
        os.dup2(slave_fd, 2)
        os.close(slave_fd)
        os.environ["TERM"] = "xterm-256color"
        os.environ["COLORTERM"] = "truecolor"
        os.execl(bin_path, "anx", "wizard")
        sys.exit(1)

    # Parent process
    os.close(slave_fd)
    output_chunks = []

    # Stages:
    # 1. Profile selection -> Enter
    # 2. Environment selection -> Enter
    # 3. Packages selection -> Enter
    # 4. Review Plan -> Enter
    # 5. Completed -> Enter to exit
    keystrokes = [
        b"\r",  # Select profile -> advance to Environment
        b"\r",  # Confirm environment -> advance to Packages
        b"\r",  # Confirm packages -> advance to ReviewPlan
        b"\r",  # Confirm plan -> execute / applying
        b"\r",  # Exit after completion
    ]

    start_time = time.time()
    step_idx = 0
    exit_code = 0

    while time.time() - start_time < 90:  # 90-second total timeout
        # Check if child process has already terminated
        w_pid, status = os.waitpid(pid, os.WNOHANG)
        if w_pid == pid:
            exit_code = os.waitstatus_to_exitcode(status)
            break

        r, _, _ = select.select([master_fd], [], [], 0.5)
        if r:
            try:
                data = os.read(master_fd, 1024)
                if not data:
                    break
                output_chunks.append(data)
            except OSError:
                break

        # Send next keystroke to advance stages
        if step_idx < 4:
            time.sleep(0.5)
            try:
                os.write(master_fd, keystrokes[step_idx])
                step_idx += 1
            except OSError:
                break
        else:
            # After plan confirmation, send Enter periodically to exit StageCompleted as soon as it finishes
            time.sleep(1.0)
            try:
                os.write(master_fd, b"\r")
            except OSError:
                break

    os.close(master_fd)
    if w_pid != pid:
        _, status = os.waitpid(pid, 0)
        exit_code = os.waitstatus_to_exitcode(status)

    full_output = b"".join(output_chunks).decode("utf-8", errors="replace")
    print(f"[INFO] Wizard PTY completed with exit code: {exit_code}")

    if "ANX SETUP WIZARD" not in full_output:
        print("[WARN] ANX SETUP WIZARD header not observed in output stream")

    if exit_code != 0:
        print(f"[FAIL] Wizard exited with non-zero code: {exit_code}", file=sys.stderr)
        sys.exit(exit_code)

    print("✓ Interactive Wizard PTY execution succeeded cleanly!")

if __name__ == "__main__":
    run_wizard()
