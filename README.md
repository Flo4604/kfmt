# kfmt [![](https://github.com/fatih/kfmt/workflows/build/badge.svg)](https://github.com/fatih/kfmt/actions)

Format Kubernetes byte quantities for humans.

- Zero dependencies (just Go stdlib)
- Parses Kubernetes `resource.Quantity` formats
- Designed for readability, not precision

## Why

Kubernetes `resource.Quantity` only outputs human-readable format (like `170Mi`, `12Gi`) when the value is **exactly** divisible into binary units:

```
1073741824 bytes  → "1Gi"       (exact)
178257920 bytes   → "170Mi"     (exact)
178255984 bytes   → "178255984" (arbitrary - not human readable)
```

Additionally, `resource.Quantity` preserves the scale at which values were originally set. A value from `df` output (in KiB) stays in KiB even when it would be more readable as GiB:

```
"spaceAvailable": "12075408Ki"  →  hard to read (actually ~11.5 GiB)
```

Many operators store byte values as raw integers or strings in status fields. This tool converts those values in-place to the most readable unit.

## Install

```bash
go install github.com/fatih/kfmt@latest
```

## Usage

Convert a single value:

```bash
$ kfmt 178255984
170MiB

$ kfmt 12075408Ki
11.5GiB
```

Convert fields in JSON:

```bash
$ echo '{"usedBytes": "178255984", "spaceAvailable": "12075408Ki"}' | kfmt --json-fields "usedBytes,spaceAvailable"
{"usedBytes": "170MiB", "spaceAvailable": "11.5GiB"}
```

Real-world example with kubectl, without and with `kfmt`:

```bash
$ kubectl get cr -o json | jq '.items[].status.storage'
{
  "currentSize": "12884901888",
  "usedBytes": "178255984",
  "spaceAvailable": "12365217792"
}

$ kubectl get cr -o json | jq '.items[].status.storage' | kfmt --json-fields "currentSize,usedBytes,spaceAvailable"
{
  "currentSize": "12.0GiB",
  "usedBytes": "170MiB",
  "spaceAvailable": "11.5GiB"
}
```

## Supported Formats

| Input | Type | Example |
|-------|------|---------|
| Raw bytes | integer or decimal | `178255984` → `170MiB` |
| Scientific | e-notation | `1.5e9` → `1.40GiB` |
| Binary (IEC) | Ki, Mi, Gi, Ti, Pi, Ei | `12075408Ki` → `11.5GiB` |
| Decimal (SI) | k, K, M, G, T, P, E | `1G` → `954MiB` |

Decimal values with suffixes are supported: `1.5Gi` → `1.50GiB`

## Precision

The output is rounded to fit the most readable unit:
- Values ≥100 show no decimal places (`100MiB`)
- Values ≥10 show 1 decimal place (`11.5GiB`)
- Values <10 show 2 decimal places (`1.50KiB`)

**Note:** Converting between units may lose precision. For example:
- `12075408Ki` (12,365,217,792 bytes) displays as `11.5GiB` (actual: 11.5146... GiB)
- Decimal SI units (K, M, G) are converted to binary IEC units (KiB, MiB, GiB), which are ~2.4% smaller

This tool is designed for human readability, not precision. Use the original values for calculations.

## Not Supported

This tool is for byte/storage values. The following Kubernetes quantity features are intentionally not supported:

- Milli (`m`), micro (`u`), nano (`n`) suffixes — not meaningful for bytes
- Negative values — bytes can't be negative
- Full arbitrary precision — we accept rounding for readability
