# Test fixtures

Recorded command output used by tests instead of running real tools.

## Layout

    testdata/<tool>/<version>/

- `<tool>` is `t3` or `tailscale`.
- `<version>` is the version of that tool the output was recorded from
  (t3 fixtures are tagged by the t3 version, tailscale fixtures by the
  tailscale version).
- Each version directory has a `fixtures.json` that maps a command name plus
  its arguments to the recorded `stdout` (or a `stdout_file` next to it),
  `stderr` and `exit_code`. `runner.LoadFake` replays it.

## Sanitizing rule

Fixtures are committed, so they must never hold real identities or secrets.
Replace them with these placeholders before committing:

- `example-tailnet.ts.net` for the tailnet name, and
  `*.example-tailnet.ts.net` for hosts inside it (for example a device
  name followed by `.example-tailnet.ts.net`).
- `REDACTED` or `EXAMPLE-TOKEN` for the value of a pairing token in a pairing URL fragment (also when the fragment is URL-encoded).
- `tskey-...-REDACTED` or `tskey-...-EXAMPLE-TOKEN` for Tailscale keys.

Anything else is rejected: any other name ending in .ts.net, a pairing token
with another value, a Tailscale key not ending in one of the placeholders, and
any run of 32 or more key-like characters that mixes letters and digits
(hex or base64 keys, node keys, UUIDs with letters).

## Checking

    scripts/check-fixtures.sh [dir ...]

It defaults to `testdata`, prints each finding as `path:line: rule: match`,
and exits 1 on findings (2 if a directory is missing or a file cannot be scanned). `mise run check` runs
it too.
