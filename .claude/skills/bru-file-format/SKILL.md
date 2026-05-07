---
name: bru-file-format
description: Reference for Bruno .bru file format, collection structure, and Go parsing patterns. Use when building tools that read, parse, or generate Bruno API collection files, or when working with bck_transaction/bck_notification/bck_material collections.
---

# Bruno .bru File Format

## Block structure

`.bru` files use a block-based plain-text format: `name { ... }` or `name:subtype { ... }`.

Two types of blocks:
- **KV blocks**: each line is `key: value` (or `~key: value` for disabled lines)
- **Raw blocks**: content between braces is a single raw string (`body:*`, `script:*`, `tests`, `docs`)

## Key blocks

```
meta { name: ..., type: http, seq: 1 }
get/post/put/delete/patch { url: ..., body: none|json|..., auth: none|inherit|awsv4|bearer }
headers { Key: Value, ~Disabled: value }
params:query { key: value, ~disabled: value }
body:json { <raw JSON> }
auth { mode: inherit|awsv4|none|bearer }
auth:awsv4 { accessKeyId: ..., secretAccessKey: ..., sessionToken: ..., service: ..., region: ... }
vars { k: v }
vars:pre-request { k: v }
script:pre-request { <JS code> }
tests { <JS code> }
settings { encodeUrl: true, timeout: 30000 }
```

Variable interpolation: `{{var_name}}`. Disabled line prefix: `~`.

## Collection directory structure

```
collection-root/
├── bruno.json              # { "version": "1", "name": ..., "ignore": [...] }
├── collection.bru          # collection-level auth, vars, scripts
├── environments/
│   ├── Local.bru           # vars { key: value }
│   └── Prod.bru
├── folder-name/
│   ├── folder.bru          # folder-level meta, auth (mode: inherit common)
│   └── request.bru
```

## Auth inheritance

Resolution order: request → folder.bru → parent folder.bru → collection.bru.
Walk up until first non-`inherit` auth block is found.

## Parsing in Go

Hand-rolled line-based parser is sufficient — no PEG/grammar needed.

Key insight: distinguish raw blocks (`body:*`, `script:*`, `tests`, `docs`) from KV blocks **before** parsing lines. Raw blocks collect everything between braces as a single string.

**Critical edge case**: in raw blocks, only track `"`-strings and `//` comments; never treat `'` as a string delimiter. `docs` blocks contain natural language with apostrophes that would break single-quote string tracking.

```go
type BruDoc struct {
    Blocks []Block
}
type Block struct {
    Name    string // e.g. "headers", "body"
    SubType string // e.g. "json", "awsv4" (from "body:json")
    Raw     bool
    Content string // for raw blocks
    Lines   []Line // for kv blocks
}
type Line struct {
    Disabled bool
    Key      string
    Value    string
}
```

## Variable scope (priority order, lowest to highest)

collection vars → env vars → folder chain (root→leaf) → request vars/pre-vars → runtime overrides

Use a depth cap (e.g. 16) to guard against cycles in `{{var}}` substitution.

## Collection loader

- Walk directory with `filepath.WalkDir`
- Read `bruno.json` for ignore patterns
- Always skip: `bin/`, `build/`, `node_modules/`, `.git/`
- Sort requests/folders by `meta.seq` then name
- Environment key = filename without `.bru` extension
- `model.AuthAWSv4` is the struct; `model.AuthModeAWSv4` is the string constant (avoid name collision)

## AWS SigV4 in Go

Use `aws-sdk-go-v2`:
```go
import (
    "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
    "github.com/aws/aws-sdk-go-v2/credentials"
)

signer := v4.NewSigner()
creds := aws.Credentials{
    AccessKeyID:     accessKeyID,
    SecretAccessKey: secretAccessKey,
    SessionToken:    sessionToken,
}
err = signer.SignHTTP(ctx, creds, req, payloadHash, service, region, time.Now())
```

Payload hash for empty body: `sha256("")` = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

## Pre-request JS scripts

Satispay collections use `script:pre-request` to extract AWS creds from a `my_aws_credentials` env var. These require a JS engine (`bru.setVar`, `bru.getEnvVar` API). In tools without a JS engine, skip silently and show a status hint: "script:pre-request skipped — set AWS creds directly in env file".

## Corpus facts (Satispay collections, verified 2026-05-06)

- `bck_transaction/src/main/resources/api/`: 207 requests, 81 folders, 4 envs (Local/Test/Staging/Prod), collection-level `auth:awsv4`
- `bck_notification/src/main/resources/api/`: similar structure
- `bck_material/core/core1/collections/satispay-api/`: many sub-collections
- Build mirrors under `bin/main/api/` and `build/resources/main/api/` — must be excluded
- Total corpus (all three): 1080 `.bru` files
