# terraform-azure-weka

## Guidelines

### Agentic flow

The main agent is an orchestrator. It should delegate work via the Task tool and
minimize direct tool use — direct tool use is acceptable only for 1-2 quick
checks to orient. Always set the model explicitly on tasks/subagents.

Delegation model, in order:

1. **Haiku task** — all codebase exploration, investigation, searching, and
   reading files. Even for complex debugging: haiku can read and trace code
   paths, and it is 10-20x cheaper than opus.
2. **Sonnet task** — code edits, test runs, build verification, deploy flows.
3. **Opus task** — only for complex plan generation that needs deep understanding
   of the codebase, and only on top of context a haiku task already gathered, or
   when sonnet is struggling with execution.

### Scope of a change

Fix what was asked, and only that. Unrelated problems you notice along the way
stay untouched — even obvious ones, even one-line ones. Mention them and offer
to open a ticket instead.

A change is in scope only if the requested fix does not work without it.

### After any code change

Run `/simplify` once the change is complete and working. Not after each edit —
after the change as a whole.

Then run the checks CI runs on every PR: `pre-commit run --all-files`
(`terraform_fmt`, `tflint`). For changes under `function-app/code`, also run
`go vet ./...` and `go test ./...` from that directory.

### Verify before reporting done

Run the checks above and read the output before saying a change works.
"Should work" is not a result, and neither is a run you launched but never
looked at.

When a check fails, say so and show the real output. When you skipped one —
a real `terraform apply` against Azure, no credentials, not reproducible locally — name which and
why. Silence reads as "it passed".

### Commit messages

One line. No body, no bullet list, no trailing explanation — if the change needs
more words than a single subject line, those words belong in the PR description,
the code, or a comment, not in the commit.

Existing history is the bar: `fix: reap halted instances during clusterization only`.

### Comments

Keep them minimal. Write a comment only when the code cannot be made obvious on
its own: a non-obvious Azure VMSS / Function App constraint, a deliberate omission, a workaround
whose reason isn't visible in the diff. Do not comment what the code already
says.

The `includeStopped` comment at `function-app/code/common/common.go:1768` is the bar — it carries information the reader
cannot recover from the code.

The test is what happens when the code is wrong. Comment what fails *silently*,
at apply time, or in only one environment — those cost a cycle to rediscover.
Say nothing about syntax, types, or API shape: `terraform validate`, `tflint` and the Go compiler reject those
instantly and loudly, so the comment buys nothing even when it is accurate.
