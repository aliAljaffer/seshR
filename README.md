# seshR

Lists your Claude Code and OpenCode sessions with their directories. Resume any of them from any directory.

I lose all my terminal tabs when Ghostty crashes. This tells me what was open.

## Install

Requires Go and macOS or Linux.

```sh
git clone https://github.com/aliAljaffer/seshR.git
cd seshR
make install
```

`make install` builds the binary and links it to `~/.local/bin/seshr`.

## Use

List sessions, newest first:

```sh
seshr
```

```
 1  claude   crashed 13h   ~/azm-dev-x  SMTP failures in dev-x
 2  claude   recent  14h   ~/gpu        AZM-OCR service deployment prep
 3  opencode recent  5d    ~/fai        Remaining improvements in docs
```

Resume session 2 in its own directory:

```sh
seshr 2
```

Change how far back "recent" sessions go (default 24h):

```sh
seshr -since 72h
```

List only sessions in the current directory:

```sh
seshr --here
```

Numbers match the full list, so `seshr N` resumes the same session with or without `--here`.

## States

| State | Meaning |
| --- | --- |
| `open` | The Claude Code process is still running. |
| `crashed` | Claude Code left a session file, but the process is gone. |
| `recent` | No process record. The session was updated within `-since`. |

OpenCode has no process record, so its sessions are always `recent`.

## Where it reads

- Claude Code: `~/.claude/sessions/*.json` and `~/.claude/projects/*/*.jsonl`.
- OpenCode: `~/.local/share/opencode/opencode.db`, read-only.

The title is the AI title. If there is none, it is the last prompt.

## Limits

- On Linux, a reused process ID can show a dead session as `open`.
- Claude Code may delete stale session files on its own. The `recent` rows cover that case.
- Windows is not supported.
