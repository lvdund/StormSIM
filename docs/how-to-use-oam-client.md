# How to Use the StormSim OAM Client

This document describes **how to use the OAM `client`** to inspect a running emulator
and trigger events live, command by command, with sample output. It also covers the
**metrics and logs** the backend exposes. It does **not** describe internal
architecture.

StormSim exposes a **runtime OAM API** over HTTP (default `0.0.0.0:4000`) when
`remote.enable: true` in `config.yml`. The `bin/client` binary is the front-end for
that API. Both sides speak the same command vocabulary organized into **three nested
contexts**:

| Context | Prompt | What it covers |
|---------|--------|----------------|
| **root** (`stormsim`) | `stormsim>` | Whole-emulator views: list UEs/gNBs, global delay & HO stats, state histogram |
| **UE** (`ue:<msin>`) | `stormsim:<msin>>` | A single emulated UE |
| **gNB** (`gnb:<gnbId>`) | `stormsim:<gnbId>>` | A single emulated gNodeB |

You drill into a context with `select-ue` / `select-gnb` and return with `exit`.

---

## 1. Metrics & logging overview

Before the command reference, here is what the OAM layer exposes conceptually.

### What gets measured

StormSim records the **delay of every tracked state transition** (5GMM and 5GSM), plus
an end-to-end **registration procedure** duration. For each transition it reports:

- **Count** - number of samples
- **Mean**
- **Standard deviation**
- **Percentiles**: P1, P5, **P25**, **P50 (median)**, **P75**, P95, P99
- **Min / Max**

> The **median** is the reported P50; the **interquartile range (IQR)** is P75 − P25.

Additional collectors:
- **State histogram** - per-second counts of UE MM/SM states (Deregistered /
  Registered, PduSessionInactive / Active).
- **Handover delay** - Xn handover count and mean Path-Switch-Request → Ack delay.
- **Worker-pool stats** - submitted / waiting / dropped / completed tasks.

### Per-UE / per-gNB logs

When `logging.bufferlog: true` in `config.yml`, each UE and gNB keeps a **ring-buffer
log** queryable via the `logs` command (with `--level` and `--last` filters). Disable
buffered logs (`bufferlog: false`) for maximum scale.

All of the above are surfaced through the commands in §3–§6 below.

---

## 2. Common output conventions

Two renderers are used everywhere:

### Text table (`Formatter.RenderTable`)
Printed to the terminal by default. Uses Go `text/tabwriter` (2-space padding). Always
has:
- a **title line** (e.g. `=== UE Context Info ===`)
- a **header row**
- a **separator row** of `-` dashes under the headers
- the data rows

Example shape:
```
=== UE Context Info ===
Key   Value
---   -----
Name  0000000000
Gnb   000001
```

### CSV export (`Formatter.WriteCSV`)
Most commands accept a `--file <path>` (UE/gNB contexts) or `-f` / `--file` (root
contexts) flag to dump the same data to CSV instead of printing a table.
- If the path is literally `auto`, a timestamped filename is generated:
  `YYYYMMDD-HHMMSS-<context>-<command>.csv` (root context only).
- For `states` (root) the default filename is `states-tracker-YYYYMMDD-HHMMSS.csv`
  when no path is given.
- CSV is RFC-4180 compliant via Go's `encoding/csv`.

### Error / empty results
When there is nothing to show, the server returns a JSON `Error` field and the client
prints it as:
```
Error: No UE contexts found
```

### Watch mode (`-w` / `--watch`, `-n` / `--interval`)
A **client-side** loop (the server deliberately does not block its handler). It:
- moves the cursor home (`\033[H`), re-issues the command, clears to end of screen
  (`\033[J`), then waits `interval`;
- default interval is **1s**; pass e.g. `-n 500ms` or `--interval 2s`;
- stop with **Ctrl+C** (clears the screen and exits).

`--watch` cannot be combined with `--file`.

---

## 3. Root (emulator) context - commands & output

### `list-ue`
List all emulated UE MSINs.

**Flags:** `--level` (filter logs by level), `--state` (only UEs whose MM state
*contains* the string), `--not-state` (exclude), `--last N` (only the last N).

Valid `--state` / `--not-state` tokens (substring match): `Deregistered`,
`DeregistrationInitiated`, `AuthenticationInitiated`, `RegisteredInitiated`,
`Registered`.

```
stormsim> list-ue --state Registered
msin: - 0000000000
 - 0000000001
 - 0000000002
```

### `list-gnb`
List all emulated gNB IDs.

```
stormsim> list-gnb
[
  "000001",
  "000002"
]
```
(Returned as pretty-printed JSON.)

### `select-ue --msin <msin>`
Enter a UE context. No data output, just a prompt change:
```
stormsim> select-ue --msin 0000000000
stormsim:0000000000>
```
Error if the MSIN is not found:
```
Error: UE with id=0000000123 not found
```

### `select-gnb --gnbId <gnbId>`
Enter a gNB context.
```
stormsim> select-gnb --gnbId 000001
stormsim:000001>
```

### `delay` - global state-transition delay statistics
Aggregates delay samples **across all UEs** for every tracked transition, plus an
end-to-end **Registration procedure** row.

**Flags:** `--domain MM|SM` (filter), `-f` (export per-second snapshot CSV),
`--file <path>` (CSV path override).

Table columns: `Domain, From, To, Event, Count, MeanMs, StdDevMs, P1Ms, P5Ms, P25Ms,
P50Ms, P75Ms, P95Ms, P99Ms, MinMs, MaxMs`.

> P50 = median. IQR = P75 − P25.

```
stormsim> delay
=== Global State Delay Statistics ===
Domain  From             To               Event                 Count  MeanMs  StdDevMs  P1Ms  P5Ms  P25Ms  P50Ms  P75Ms  P95Ms  P99Ms  MinMs  MaxMs
------  ----             --               -----                 -----  ------  --------  ----  ----  -----  -----  -----  -----  -----  -----  -----
MM      Deregistered     RegisteredInitiated InitRegistration   10000  12      4         6     7     9      11     14     20     28     5      35
MM      RegistrationProcedure Completed       RegistrationE2E    9987   145     22        98    110   130    142    158    185    210    90     240
SM      PDUSessionActivePending PDUSessionActive EstablishmentAccept 9500 38 9 20 24 32 37 44 55 65 18 70
```

CSV export (`-f`) writes one row per (domain, from, to, event) with columns:
`timestamp, domain, from, to, event, count, mean_ms, stddev_ms, p1_ms, p5_ms, p25_ms,
p50_ms, p75_ms, p95_ms, p99_ms, min_ms, max_ms`.

### `ho-delay` - handover delay statistics
Per-second **Xn handover count** and mean **Path-Switch-Request → Ack** delay.

**Flags:** `-f` / `--f <path>` → CSV. Without `-f` prints only the latest snapshot as a
2-row table.

```
stormsim> ho-delay
=== Handover Delay (PathSwitch Req→Ack) ===
Metric                  Value
------                  -----
Total Completed Handovers 8420
Overall Mean Delay        47 ms
```

CSV columns: `timestamp, ho_count, delay_mean`.

### `states` - UE state histogram (CSV only)
Per-second counts of UE MM/SM states. **Always writes CSV** (no table mode).

**Flags:** `--f <path>` (default `states-tracker-YYYYMMDD-HHMMSS.csv`).

Columns: `Timestamp, Deregistered, Registered, PduSessionInactive, PduSessionActive`.

```
stormsim> states
State snapshots written to states-tracker-20260701-103045.csv
```
```csv
Timestamp,Deregistered,Registered,PduSessionInactive,PduSessionActive
10:30:45,0,10000,0,9987
10:30:46,0,10000,0,9987
```

---

## 4. UE context - commands & output

Enter with `select-ue --msin <msin>`.

### `info`
One UE's current context as a key/value table.

**Flags:** `--file <path>` (CSV).

Fields: `Name` (msin), `Gnb` (gnbId), `Hplmn`, `Snssai`, `RanNgapId`, `AmfNgapId`,
`MMstate`, `ActiveSessions`.

```
stormsim:0000000000> info
=== UE Context Info ===
Key            Value
---            -----
Name           0000000000
Gnb            000001
Hplmn          208/93
Snssai         01/010203
RanNgapId      1
AmfNgapId      1
MMstate        Registered State
ActiveSessions 1
```

### `ssinfo`
PDU sessions owned by this UE. Empty → `Error: No session found`.

**Flags:** `--file <path>`.

Columns: `Id, Dnn, Snssai, State, Address`.

```
stormsim:0000000000> ssinfo
=== UE Sessions Info ===
Id Dnn       Snssai    State             Address
-- ---       ------    -----             -------
1  internet  01/010203 PDUSessionActive State 10.60.0.1
```

### `stats`
Lightweight per-UE summary: MM/SM transition counts.

**Flags:** `--file <path>`, `-w/--watch`, `-n/--interval`.

```
stormsim:0000000000> stats
=== Event/Message processing time, state of UE ===
Stat
----
State delay summary:
MM transitions: 12
SM transitions: 4
```

### `logs`
Buffered per-UE log history (requires `logging.bufferlog: true`). Newest entries first.

**Flags:** `--last N` (default 50, 0 = all), `--level` (trace/debug/info/warn/error/
fatal/panic), `--file <path>`.

Columns: `Time, Level, Message`. Time format `HH:MM:SS.mmm`.

```
stormsim:0000000000> logs --last 3 --level info
=== UE Logs ===
Time          Level  Message
----          -----  -------
10:30:45.012  info   Send RegistrationRequest
10:30:45.140  info   Receive RegistrationAccept
10:30:45.141  info   Send PduSessionEstablishmentRequest
```

### `delay`
Per-UE **individual** transition records (not aggregated). One row per recorded
transition; `Count` is always 1 here.

**Flags:** `--last N` (default 50), `--domain MM|SM`, `--file <path>`.

Columns: `Domain, From, To, Event, Count, DelayMs`.

```
stormsim:0000000000> delay --last 5 --domain MM
=== UE State Delay ===
Domain  From           To                    Event               Count  DelayMs
------  ----           --                    -----               -----  -------
MM      Deregistered   RegisteredInitiated   InitRegistration    1      13
MM      AuthenticationInitiated SecurityModeInitiated SecurityModeCommand 1 24
MM      RegisteredInitiated Registered RegistrationAccept 1 18
MM      RegisteredInitiated Registered ConfigurationUpdate 1 6
MM      RegistrationProcedure Completed RegistrationE2E 1 142
```

### `ps-create` - trigger a PDU session (runtime event)
Enqueues a `PduSessionInit` event on this UE with the given slice/DNN.

**Required flags:** `--dnn`, `--sst` (int), `--sd` (string).

```
stormsim:0000000000> ps-create --dnn internet --sst 1 --sd 010203
Session estalishment triggered
```
On failure: `Error: Fail to trigger session establishment`.

### `exit`
Return to the root context.
```
stormsim:0000000000> exit
stormsim>
```

---

## 5. gNB context - commands & output

Enter with `select-gnb --gnbId <gnbId>`.

### `info`
gNB identity and endpoints.

**Flags:** `--file <path>`.

Fields: `Name` (gnbId), `Plmn` (mcc/mnc), `Snssai` (sst/sd), `NgapAddr` (N2 IP:port),
`GtpAddr` (N3 IP:port).

```
stormsim:000001> info
=== GNB Context Info ===
Key      Value
---      -----
Name     000001
Plmn     208/93
Snssai   01/010203
NgapAddr 192.168.1.10:9487
GtpAddr  192.168.1.10:2152
```

### `logs`
Buffered per-gNB log history (same shape as UE `logs`).

**Flags:** `--last N` (default 50), `--level`, `--file <path>`.

```
stormsim:000001> logs --last 2
=== gNB Logs ===
Time          Level  Message
----          -----  -------
10:30:42.001  info   Receive NGSetupResponse
10:30:45.012  info   Send InitialUEMessage
```

### `list-amf`
AMF(s) this gNB is connected to.

**Flags:** `--file <path>`.

Columns: `Name, Address, State, PLMNs, Slices`.

```
stormsim:000001> list-amf
=== AMF Contexts ===
Name      Address           State   PLMNs   Slices
----      -------           -----   -----   ------
amf-001   192.168.1.110:38412 Connected 208/93  01/010203
```

### `count-ue`
Number of UEs currently anchored at this gNB. Supports watch.

**Flags:** `--file <path>`, `-w/--watch`, `-n/--interval`.

```
stormsim:000001> count-ue
=== UE Count ===
Key              Value
---              -----
Number of UE in gnb 5000
```

### `list-ue`
MSINs of UEs anchored at this gNB.

**Flags:** `--file <path>`.

```
stormsim:000001> list-ue
=== UE Contexts ===
MSIN
----
0000000000
0000000001
```

### `release-ue` - trigger UE context release (runtime event)
Sends a UE-context-release trigger toward the gNB for the given UE.

**Required flag:** `--msin`.

```
stormsim:000001> release-ue --msin 0000000000
Ue release triggered
```
On failure: `Error: Fail to trigger ue release`.

### `exit`
Return to the root context.

---

## 6. Client global commands

These exist only in the **client** (not the server vocabulary):

| Command | Effect |
|---------|--------|
| `help` | List all currently available commands (depends on which context you're in). |
| `clear` | Clear the screen. |
| `exit` / `quit` | At root: exit the client. Inside a UE/gNB context: go back to root. |

### Two client invocation modes

**Interactive** (no args):
```
$ ./bin/client
stormsim> help
stormsim> list-ue
```

**Single-shot** (args on the command line) - runs one command and exits, against
`localhost:4000`:
```
$ ./bin/client list-ue
$ ./bin/client delay -f
$ ./bin/client delay --domain MM
```
Single-shot supports the same `--watch` / `--interval` flags; with `--watch` it loops
until Ctrl+C.

---

## 7. Runtime event triggering - summary

The OAM layer isn't read-only. Two commands actively push events into the live
emulator:

| Command | Context | Event injected | Effect |
|---------|---------|----------------|--------|
| `ps-create --dnn --sst --sd` | UE | `PduSessionInit` | Establish a new PDU session for that UE |
| `release-ue --msin` | gNB | UE context release | Tear down the UE's RAN context |

Everything else (`info`, `logs`, `delay`, `states`, `ho-delay`, `count-ue`, …) is a
read-only probe of in-memory emulator state.

---

## 8. Output quick-reference

| Want… | Use |
|-------|-----|
| Median (P50) registration time across all UEs | `delay` → row `RegistrationE2E`, column `P50Ms` |
| Tail latency (95th/99th) | `delay` → `P95Ms` / `P99Ms` |
| Per-UE individual transition times | (UE context) `delay` |
| Per-second CSV for plotting | root `delay -f`, `states`, `ho-delay -f` |
| Live-refreshing table | append `-w` / `--watch` (with optional `-n <dur>`) |
| A single UE's debugging trail | (UE context) `logs --level debug --last 200` |
| How many UEs finished | root `list-ue --state Registered` or gNB `count-ue -w` |
