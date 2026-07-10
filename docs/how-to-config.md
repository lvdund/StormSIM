# How to Configure StormSim

StormSim is driven entirely by a single YAML file. This guide walks through **every
configuration block**, what each field means, valid values, and how to combine them.

You can pass a config with `-c <path>`:

```bash
sudo ./bin/emulator -c config/config.yml
sudo ./bin/emulator -c config/free5gc.yml --pcap capture.pcap
```

Three reference configs ship in `config/`:
- `config/config.yml` - minimal, single-UE example.
- `config/free5gc.yml` - multi-gNB/UE scale test against free5GC.
- `config/open5gs.yml` - multi-gNB/UE scale test against Open5GS.

> If you get stuck, run `./bin/emulator --config-help` for an inline cheat sheet.

---

## Table of Contents

1. [File Structure Overview](#1-file-structure-overview)
2. [`gnodeb` - Emulated gNodeBs](#2-gnodeb--emulated-gnodebs)
3. [`defaultUe` - Baseline UE Profile](#3-defaultue--baseline-ue-profile)
4. [`amfif` - AMF / 5G Core Endpoints](#4-amfif--amf--5g-core-endpoints)
5. [`scenarios` - What to Run](#5-scenarios--what-to-run)
6. [`rlink` - Radio-Link Impairment](#6-rlink--radio-link-impairment)
7. [`remote` - OAM Server](#7-remote--oam-server)
8. [`logging` & `loglevel`](#8-logging--loglevel)
9. [Full Annotated Example](#9-full-annotated-example)
10. [Recipes](#10-recipes)

---

## 1. File Structure Overview

A config has **eight top-level blocks**:

```yaml
gnodeb:     # gNB interfaces + identity list          (required)
defaultUe:  # baseline UE subscription/security       (required)
amfif:      # AMF endpoint(s) to connect to           (required)
scenarios:  # UE groups + event sequences             (required)
rlink:      # optional radio-link delay/loss          (optional)
remote:     # OAM HTTP server for the client          (optional)
logging:    # buffered logs + sizes                   (optional)
loglevel:   # global log verbosity                    (optional)
```

Loading rules:
- The file is parsed strictly; an unknown field or a type mismatch (e.g. quoting a
  number) is **fatal** at startup.
- Hostnames in `amfif`, `gnodeb.controlif.ip`, and `gnodeb.dataif.ip` are
  **DNS-resolved at startup**; only IPv4 results are accepted.
- Defaults: if `logging.ueLogBufferSize` / `gnbLogBufferSize` are `0`, they fall back
  to **50 / 100** entries respectively.
- Log level is forced to `INFO` during startup, then switched to your `loglevel` value
  just before scenarios execute.

---

## 2. `gnodeb` - Emulated gNodeBs

```yaml
gnodeb:
  controlif:           # base N2 / NGAP control plane
    ip: "192.168.1.10"
    port: 9487
  dataif:              # base N3 / GTP-U data plane
    ip: "192.168.1.10"
    port: 2152
  listGnbs:            # one or more gNB identities
    - gnbid: "000001"
      tac: "000001"
      plmn:
        mcc: "208"
        mnc: "93"
      slicesupportlist:
        - sst: "01"
          sd: "010203"
    - gnbid: "000002"
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist:
        - sst: "01"
          sd: "010203"
```

### Field reference

| Field | Type | Meaning |
|-------|------|---------|
| `controlif.ip` / `controlif.port` | string / int | Base N2 listening endpoint. Port is **auto-incremented per gNB** (gNB i uses `port + i`). |
| `dataif.ip` / `dataif.port` | string / int | Base N3 GTP-U endpoint. Also auto-incremented per gNB. |
| `listGnbs[].gnbid` | string | 6-digit hex-style gNB ID (`"000001"` … `"000020"`). Referenced from `scenarios.gnbs`. |
| `listGnbs[].tac` | string | Tracking Area Code (hex string, 6 digits). |
| `listGnbs[].plmn.mcc` | string | Mobile Country Code (e.g. `"208"`). |
| `listGnbs[].plmn.mnc` | string | Mobile Network Code (e.g. `"93"`). |
| `listGnbs[].slicesupportlist[].sst` | string | Slice/Service Type (hex string). |
| `listGnbs[].slicesupportlist[].sd` | string | Slice Differentiator (hex string). **Optional** (Open5GS examples omit `sd`). |

### How multiple gNBs share the base ports
gNB index `i` (0-based) listens on:
- N2: `controlif.port + i`
- N3: `dataif.port + i`

So with `controlif.port: 9487`, gNB `"000001"`→9487, `"000002"`→9488, … Make sure all
those ports are free and reachable by the AMF.

> **Handover needs ≥ 2 gNBs.** `XnHandover` and `N2Handover` events require at least
> two entries in `listGnbs`, and the scenario group must be able to move between them.

---

## 3. `defaultUe` - Baseline UE Profile

All spawned UEs inherit these fields. The `msin` is **auto-incremented** across groups
so each UE gets a unique IMSI.

```yaml
defaultUe:
  msin: "0000000000"
  key:  "8baf473f2f8fd09487cccbd7097c6862"
  op:   "8e27b6af0e692e750f32667a3b14605d"
  # opc: "..."           # alternative to op (Operator Key Code)
  amf:  "8000"
  sqn:  "00000000"
  dnn:  "internet"
  routingindicator: "0000"
  hplmn:  { mcc: "208", mnc: "93" }
  snssai: { sst: 01, sd: "010203" }
  integrity: { nia0: false, nia1: false, nia2: true,  nia3: false }
  ciphering: { nea0: true,  nea1: false, nea2: true,  nea3: false }
  timers:
    t3510: false
    t3511: false
    t3502: false
    t3512: false
    t3516: false
    t3519: false
    t3520: false
    t3580: false
    t3582: false
  delay: 100
```

### Authentication keys

| Field | Type | Meaning |
|-------|------|---------|
| `msin` | string | 10-digit MSIN; the subscription identifier suffix. Auto-incremented per UE. |
| `key` | hex string | Permanent key **K** (16 bytes / 32 hex chars). |
| `op` | hex string | Operator variant - use **either** `op` **or** `opc`. |
| `opc` | hex string | Operator Key Code (derived). Takes precedence over `op` if both set. |
| `amf` | hex string | Authentication Management Field (2 bytes). Typically `"8000"`. |
| `sqn` | hex string | Initial sequence number (6 bytes). Usually `"00000000"`. |

> These **must match** the subscriber profile provisioned in your 5G Core's UDM/ARPF,
> otherwise authentication will fail.

### Identity & slice

| Field | Type | Meaning |
|-------|------|---------|
| `dnn` | string | Data Network Name requested in PDU session establishment (e.g. `"internet"`). |
| `routingindicator` | string | 4-digit routing indicator for SUCI/SUPI formatting (`"0000"`). |
| `hplmn.mcc` / `hplmn.mnc` | string | Home PLMN of the subscriber. |
| `snssai.sst` | string/int | Requested slice SST. |
| `snssai.sd` | string | Requested slice SD. **Optional.** |

### Security algorithms

Booleans - set `true` to advertise the algorithm in the UE Security Capability.

```yaml
integrity:   # 5G-IA (integrity)
  nia0: false    # null integrity (test only)
  nia1: false    # 128-EIA1
  nia2: true     # 128-EIA2 (most common)
  nia3: false    # 128-EIA3
ciphering:   # 5G-EA (encryption)
  nea0: true     # null ciphering (useful for debugging NAS in the clear)
  nea1: false    # 128-EEA1
  nea2: true     # 128-EEA2
  nea3: false    # 128-EEA3
```

> **Tip:** enable `nea0` + `nia2` when debugging so NAS is unencrypted but still
> integrity-protected; disable `nea0` for realistic security tests.

### Timers

Each timer can be independently enabled (`true`) or disabled (`false`). When disabled,
the UE will not start/stop that timer for the corresponding procedure. If **omitted
entirely**, all timers default to `true` (`DefaultTimerConfig`).

| Timer | Default duration | Procedure |
|-------|------------------|-----------|
| `t3510` | 15 s  | Registration request guard |
| `t3511` | 10 s  | Registration re-attempt (≤ 5) |
| `t3502` | 12 min | Registration backoff (after 5 failures) |
| `t3512` | 54 min | Periodic registration update |
| `t3516` | 30 s  | RAND/RES* storage lifetime |
| `t3519` | 60 s  | SUCI freshness |
| `t3520` | 15 s  | Authentication response/failure guard |
| `t3580` | 16 s  | PDU session establishment re-attempt (≤ 5) |
| `t3582` | 16 s  | PDU session release re-attempt (≤ 5) |

> **Scale tip:** for large stress tests (thousands of UEs), disabling timers avoids
> runaway retransmission storms if the Core is slow. The shipped `free5gc.yml` and
> `open5gs.yml` set all timers to `false`.

### Other

| Field | Type | Meaning |
|-------|------|---------|
| `delay` | int (ms) | Inter-UE stagger: time between assigning an event to consecutive UEs **within the same group**. `0` = fire all at once. |

> There are also `protectionScheme`, `homeNetworkPublicKey`, `homeNetworkPublicKeyID`
> fields on `UeConfig`. These are parsed but **SUCI concealing is not yet active**
> (Null-Scheme is used). See the main README roadmap.

---

## 4. `amfif` - AMF / 5G Core Endpoints

```yaml
amfif:
  - ip: "192.168.1.110"
    port: 38412
```

A list of AMF SCTP/N2 endpoints the gNBs will connect to and run `NG Setup` against.

| Field | Type | Meaning |
|-------|------|---------|
| `ip` | string | AMF N2 IP **or** a hostname (resolved at startup; IPv4 only). |
| `port` | int | AMF N2 SCTP port. Standard 5G is **38412**. |

Every gNB in `listGnbs` connects to **all** listed AMFs. For a single-AMF core
(free5GC/Open5GS default), provide one entry.

---

## 5. `scenarios` - What to Run

A list of **UE groups**. Each group spawns `nUEs` UEs attached to the listed gNBs and
executes its `ueEvents` in order.

```yaml
scenarios:
  - nUEs: 100
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2
```

### Group fields

| Field | Type | Meaning |
|-------|------|---------|
| `nUEs` | int | Number of UEs to create in this group. |
| `gnbs` | string[] | gNB IDs (from `gnodeb.listGnbs[].gnbid`) the group can use. **At least one must exist**, else the emulator exits fatally. |
| `ueEvents` | list | Ordered list of events to apply to every UE in the group. |

### Event fields (`ueEvents[]`)

```yaml
ueEvents:
  - event: "RegisterInit Event"     # the trigger name (required)
    delay: 2                        # seconds to wait before this event (optional)
    register_type: 0                # 0: Initial, 1: Emergency
    deregister_type: 0              # 0: normal, 1: switch-off
    pdu_session_type: 0             # 0: Initial, 1: Emergency
    number_pdu_sessions: 1          # 1..16
    params: [1, 2, 3]               # opaque integer params (optional)
```

| Field | Applies to | Meaning |
|-------|------------|---------|
| `event` | all | Trigger event name (see table below). **Required.** |
| `delay` | all | Seconds to wait before executing this step (relative to previous step). |
| `register_type` | `RegisterInit` | `0` Initial, `1` Emergency. |
| `deregister_type` | `DeregistraterInit` | `0` normal re-registration allowed, `1` switch-off. |
| `pdu_session_type` | `PduSessionInit` | `0` Initial, `1` Emergency. |
| `number_pdu_sessions` | `PduSessionInit` | How many PDU sessions to create (max 16). |
| `params` | all | Free-form integer params forwarded to the event handler. |

### Available trigger events

| `event` value | Effect |
|---------------|--------|
| `RegisterInit Event` | Start registration |
| `DeregistraterInit Event` | Start deregistration |
| `ServiceRequestInit Event` | Service request |
| `PduSessionInit Event` | Establish PDU session(s) |
| `DestroyPduSession Event` | Release PDU session(s) |
| `XnHandover Event` | Xn-based handover (needs ≥ 2 gNBs; UE must be `Registered`) |
| `N2Handover Event` | N2-based handover (needs ≥ 2 gNBs; UE must be `Registered`) |
| `IdleInit Event` | Move UE to idle |
| `Terminate Event` | Graceful UE teardown |
| `Kill Event` | Force-kill UE |

> **Event name strings must match exactly** (including the ` Event` suffix and the
> quirky spelling `DeregistraterInit`). They come from `pkg/model/event-state.go`.

### How events are dispatched within a group
- For **non-handover** events: each UE in the group receives the event, spaced by
  `defaultUe.delay` ms (the inter-UE stagger).
- For **handover** events: only UEs currently in the `Registered` state are handed
  over, and the group rotates among the gNBs listed in `gnbs`.

### MSIN auto-increment
After each group is created, `defaultUe.msin` is incremented by that group's `nUEs`.
So group 0 gets MSINs `0000000000`…`0000000099`, group 1 gets `0000000100`…, etc. You
only set the **starting** MSIN.

### Complex / multi-group scenarios
Because `scenarios` is a **list**, you can combine several groups to build richer
behaviour. Each group runs concurrently with its own gNB set and ordered event list.
A common pattern is to register UEs on two different gNBs, then hand one group over:

```yaml
scenarios:
  # Group 0: 5000 UEs on gNB 1, full registration + a PDU session
  - nUEs: 5000
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2

  # Group 1: 5000 UEs on gNB 2, register then Xn-handover
  - nUEs: 5000
    gnbs: ["000002"]
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"
        delay: 3
      - event: "XnHandover Event"
        delay: 10
```

For **long-running / repeating** behaviour (e.g. periodic cycles of
XnHandover → IdleInit → ServiceRequest → Deregister → Register), the in-tree `loop()`
helper in `internal/scenarios/custom-scenarios.go` shows how to drive a periodic event
loop that re-dispatches events to groups over time. Uncomment / adapt it in the
scenario runner to enable looping phases.

---

## 6. `rlink` - Radio-Link Impairment

```yaml
rlink:
  delay_ms: 50      # 0 = disabled; 50 = random 0–50 ms added per NAS message
  loss_ratio: 0.05  # 0.0 = none; 0.05 = 5% drop probability
```

| Field | Type | Meaning |
|-------|------|---------|
| `delay_ms` | int | Max random delay added to each UE↔gNB NAS message. `0` disables. |
| `loss_ratio` | float | Probability in `[0.0, 1.0]` of dropping a NAS message. `0.0` disables. |

> **Note:** the scenario runner currently starts with impairment **disabled** during
  the initial phase. The `rlink` block documents the intended config for the loop phase
  (see `internal/scenarios/custom-scenarios.go`). Tune there if you want impairment
  active during a specific phase.

---

## 7. `remote` - OAM Server

```yaml
remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000
```

| Field | Type | Meaning |
|-------|------|---------|
| `enable` | bool | Start the HTTP OAM server that `bin/client` talks to. |
| `ip` | string | Bind address. `"0.0.0.0"` = all interfaces. |
| `port` | int | Listen port. The client defaults to **4000**. |

When disabled, the emulator logs `=========== not remote server ==========` and no
client can connect. See [docs/how-to-use-oam-client.md](how-to-use-oam-client.md) for
the full client command set.

---

## 8. `logging` & `loglevel`

```yaml
logging:
  bufferlog: false
  ueLogBufferSize: 256
  gnbLogBufferSize: 256
loglevel: info
```

| Field | Type | Meaning |
|-------|------|---------|
| `logging.bufferlog` | bool | Keep per-UE / per-gNB ring-buffer logs (queryable via client `logs`). Disable for max scale (saves memory). |
| `logging.ueLogBufferSize` | int | Ring-buffer entries per UE. Defaults to **50** if `0`. |
| `logging.gnbLogBufferSize` | int | Ring-buffer entries per gNB. Defaults to **100** if `0`. |
| `loglevel` | string | Global verbosity: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`. |

> For 10k-UE runs, set `bufferlog: false` and `loglevel: error` or `warn` - the
> shipped `free5gc.yml`/`open5gs.yml` do exactly this.

---

## 9. Full Annotated Example

```yaml
# ───────────── gNodeBs ─────────────
gnodeb:
  controlif: { ip: "192.168.1.10", port: 9487 }   # N2 base port; +1 per gNB
  dataif:    { ip: "192.168.1.10", port: 2152 }   # N3 base port; +1 per gNB
  listGnbs:
    - gnbid: "000001"
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [ { sst: "01", sd: "010203" } ]
    - gnbid: "000002"                              # 2nd gNB → enables handover
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [ { sst: "01", sd: "010203" } ]

# ───────────── UE profile ─────────────
defaultUe:
  msin: "0000000000"
  key:  "8baf473f2f8fd09487cccbd7097c6862"
  op:   "8e27b6af0e692e750f32667a3b14605d"
  amf:  "8000"
  sqn:  "00000000"
  dnn:  "internet"
  routingindicator: "0000"
  hplmn:  { mcc: "208", mnc: "93" }
  snssai: { sst: 01, sd: "010203" }
  integrity: { nia0: false, nia2: true }
  ciphering: { nea0: true,  nea2: true }
  timers: { t3510: false, t3512: false }           # other timers default to true
  delay: 100                                       # 100ms stagger between UEs

# ───────────── 5G Core AMF ─────────────
amfif:
  - ip: "192.168.1.110"
    port: 38412

# ───────────── Scenarios ─────────────
scenarios:
  - nUEs: 500
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"
        delay: 2
  - nUEs: 500
    gnbs: ["000002"]
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"
        delay: 3
      - event: "XnHandover Event"
        delay: 10                                  # HO group 1 → group 0's gNB

# ───────────── Impairment ─────────────
rlink:
  delay_ms: 0
  loss_ratio: 0.0

# ───────────── OAM + logging ─────────────
remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000
logging:
  bufferlog: true
  ueLogBufferSize: 256
  gnbLogBufferSize: 256
loglevel: info
```

---

## 10. Recipes

### Minimal single UE
```yaml
gnodeb:
  controlif: { ip: "192.168.1.10", port: 9487 }
  dataif:    { ip: "192.168.1.10", port: 2152 }
  listGnbs:
    - gnbid: "000001"
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [ { sst: "01", sd: "010203" } ]
defaultUe:
  msin: "0000000000"
  key: "8baf473f2f8fd09487cccbd7097c6862"
  op:  "8e27b6af0e692e750f32667a3b14605d"
  amf: "8000"
  sqn: "00000000"
  dnn: "internet"
  hplmn:  { mcc: "208", mnc: "93" }
  snssai: { sst: 01, sd: "010203" }
  integrity: { nia2: true }
  ciphering: { nea0: true }
amfif:
  - ip: "192.168.1.110"
    port: 38412
scenarios:
  - nUEs: 1
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
remote: { enable: true, ip: "0.0.0.0", port: 4000 }
loglevel: info
```

### Scale test: 10 000 UEs
- Define many gNBs (e.g. 20) so port+N2 load is spread.
- Spread UEs across multiple scenario groups (one per gNB) rather than one giant group.
- Disable timers (`timers: { t3510: false, … }`) and buffered logs
  (`bufferlog: false`), set `loglevel: warn` or `error`.
- Use small `delay` (e.g. 50–100 ms) to avoid a thundering herd at registration.

```yaml
scenarios:
  - nUEs: 500
    gnbs: ["000001"]
    ueEvents: [ { event: "RegisterInit Event" } ]
  - nUEs: 500
    gnbs: ["000002"]
    ueEvents: [ { event: "RegisterInit Event" } ]
  # … repeat for gNBs 000003 … 000020 (20 × 500 = 10 000)
```

### Handover test
- Provide **≥ 2 gNBs**.
- Put UEs on one gNB, register + establish a PDU session, then issue `XnHandover`
  (or `N2Handover`). Only `Registered` UEs will move.

```yaml
scenarios:
  - nUEs: 50
    gnbs: ["000001", "000002"]          # group can use both
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"
        delay: 3
      - event: "XnHandover Event"
        delay: 10
```

### Debug-friendly NAS (unencrypted)
```yaml
defaultUe:
  ciphering: { nea0: true }   # null ciphering → NAS visible in PCAP
  integrity: { nia2: true }   # keep integrity so the Core accepts it
```
Then run with `--pcap debug.pcap` and open in Wireshark.

### Disable the OAM client
```yaml
remote: { enable: false }
```
