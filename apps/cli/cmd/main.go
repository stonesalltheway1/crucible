// Command crucible is the Phase-1 CLI.
//
// Commands implemented:
//
//   crucible version
//   crucible health                                 — pings the control plane
//   crucible task new --description "..."           — submits a task
//   crucible task list                              — lists tasks
//   crucible task get <id>                          — shows task + plan + budget
//   crucible plan show <task_id>                    — alias for task get, plan-focused
//   crucible plan approve <task_id>                 — approves the plan
//   crucible plan reject  <task_id> --reason "..."  — rejects the plan
//   crucible budget show  <task_id>                 — live budget snapshot
//
// Configuration:
//   CRUCIBLE_ENDPOINT (default http://localhost:8080)
//   CRUCIBLE_TENANT   (default "single-tenant")
package main

import (
	"fmt"
	"os"

	"github.com/crucible/cli/internal/cmd"
)

func main() {
	if err := cmd.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
