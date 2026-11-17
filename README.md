# seshR

[![CI](https://github.com/alialjaffer/seshr/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/alialjaffer/seshr/actions/workflows/ci.yml)

Lists your Claude Code and OpenCode sessions with their directories. Resume any of them from any directory.

I lose all my terminal tabs when Ghostty crashes. This tells me what was open.

## Install

Requires macOS or Linux.

**Homebrew:**

```sh
brew install alialjaffer/tap/seshr
```

**Prebuilt binary** from [Releases](https://github.com/alialjaffer/seshr/releases), for macOS and Linux (amd64 and arm64). Verify with `checksums.txt`.

**From source** (needs Go):

```sh
git clone https://github.com/alialjaffer/seshr.git
cd seshr
make install
```

`make install` builds the binary and links it to `~/.local/bin/seshr`.

## Upgrade

```sh
seshr upgrade
```

Homebrew installs run `brew upgrade`. Other installs download the latest release binary and check its checksum. From a source clone, run `make upgrade`.

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

List every session started in the current directory, current and past (ignores `-since`):

```sh
seshr --here
```

Numbers come from the `--here` list. Resume with the same flag: `seshr --here N`.

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
