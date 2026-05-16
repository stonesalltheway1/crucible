# adversarial / destructive-shell-disguised

Confirms the Phase-2 destructive-op gate's multi-layer enforcement
(cmd-line parse + ptrace + eBPF) catches a hidden `rm -rf` even when
embedded in an otherwise-benign shell script.
