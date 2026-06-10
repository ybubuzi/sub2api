# Deploy Sub2API Candidate

## Objective

Push the committed code to GitHub, verify GitHub workflows with gh, then prepare and verify a new sub2api candidate on bubuzi.cc without touching the running sub2api-kiro service or switching production traffic. Final overwrite/switch remains user-controlled.

## Constraints

- Only handle sub2api; do not modify sub2api-kiro.
- Do not change nginx production routing until user decides.
- Fix CI failures before server image validation.
- Use ssh-skill for remote commands.
