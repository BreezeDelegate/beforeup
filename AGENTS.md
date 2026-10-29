# BeforeUp project rules

BeforeUp exists to find a few host-side problems before a Docker Compose project starts. Keep the scope narrow and use Docker Compose's own resolution semantics.

- Keep the command read-only. Do not execute `up`, `down`, `pull`, `build` or mutate the Docker daemon.
- Do not print interpolated environment values or raw Docker Compose errors.
- A busy port is a warning, not proof of failure; attribute Docker port ownership and do not ignore inactive profiles.
- The comparison feature must only read running state; do not imply unchanged app or image behavior from matching ports and mounts.
- Never include interpolated bind-source paths or raw Compose error details in reports.
- Remote contexts must skip local host checks.
- Preserve both healthy and failing cases in tests.
- Use concise, factual English. Keep README.fr.md aligned.
- Public files must not contain a contact email address; use the issue tracker or https://vitrine.bar/.
- Never claim external adoption or test results without evidence.
- Keep Git author and committer metadata on the owner's GitHub noreply identity. Follow the owner's canonical workspace GitHub write rules; do not use an assistant connector for commits.
- Run tests and package validation locally on the owner's personal VPS.

- Comparing declared environment values reads secrets through Docker inspect. Keep values in memory; report only counts and never print the variable name, value or raw inspect output.
