# Configuration Reference

Complete YAML configuration schema for StormSIM.

## File Location

Default: `config/config.yml`

## Top-Level Structure

```yaml
gnodeb:           # gNodeB configuration
defaultUe:        # Default UE profile
amfif:            # AMF interface endpoints
scenarios:        # Test scenarios and UE behaviors
remote:           # Remote API server settings
testconf:         # Testing and fuzzing options
loglevel:         # Logging level
```

---

## gnodeb

gNodeB interface and identity configuration.

```yaml
gnodeb:
  controlif:
    ip: "127.0.0.20"      # N2 interface IP (required)
    port: 9487            # N2 interface port (required)
  dataif:
    ip: "127.0.0.20"      # N3 interface IP (required)
    port: 2152            # N3 interface port (default: 2152)
  listGnbs:               # gNodeB list (required, at least one)
    - gnbid: "000008"     # gNodeB ID, 6 digits (required)
      tac: "000001"       # Tracking Area Code (required)
      plmn:               # PLMN configuration (required)
        mcc: "208"        # Mobile Country Code
        mnc: "93"         # Mobile Network Code
      slicesupportlist:   # Network slices (optional)
        - sst: "01"       # Slice/Service Type
          sd: "010203"    # Slice Differentiator
```

| Field        | Type   | Required | Description                        |
| ------------ | ------ | -------- | ---------------------------------- |
| controlif.ip | string | Yes      | N2 interface IP for NGAP signaling |
| controlif.port | int    | Yes      | N2 interface port                  |
| dataif.ip    | string | Yes      | N3 interface IP for GTP-U traffic  |
| dataif.port  | int    | No       | N3 port (default: 2152)            |
| listGnbs     | array  | Yes      | List of gNodeB configurations      |

### gNodeB Entry

| Field            | Type   | Required | Description                        |
| ---------------- | ------ | -------- | ---------------------------------- |
| gnbid            | string | Yes      | 6-digit gNodeB identifier          |
| tac              | string | Yes      | Tracking Area Code                 |
| plmn.mcc         | string | Yes      | Mobile Country Code (3 digits)     |
| plmn.mnc         | string | Yes      | Mobile Network Code (2-3 digits)   |
| slicesupportlist | array  | No       | Supported network slices           |

---

## defaultUe

Default UE authentication and network profile.

```yaml
defaultUe:
  msin: "0000000000"                         # MSIN (auto-incremented for multiple UEs)
  key: "14b23ceb27e95eb732a3f9d602f551c4"    # 128-bit K (hex)
  opc: "a3e3c63de23b66dc6a8ae0272b44906c"    # Operator Code (hex)
  amf: "8000"                                # Authentication Management Field
  sqn: "00000000"                            # Sequence Number
  dnn: "internet"                            # Data Network Name
  routingindicator: "0000"                   # Routing Indicator
  hplmn:
    mcc: "208"
    mnc: "93"
  snssai:
    sst: 01
    sd: "010203"
  integrity:
    nia0: false
    nia1: false
    nia2: true
    nia3: false
  ciphering:
    nea0: true
    nea1: false
    nea2: true
    nea3: false
```

| Field             | Type    | Required | Description                                    |
| ----------------- | ------- | -------- | ---------------------------------------------- |
| msin              | string  | Yes      | MSIN, auto-incremented (e.g., 0000000001, 0000000002) |
| key               | string  | Yes      | 128-bit permanent subscription key (32 hex chars) |
| opc                | string  | Yes      | Operator variant algorithm code (32 hex chars) |
| amf               | string  | Yes      | Authentication Management Field (4 hex chars)  |
| sqn               | string  | Yes      | Sequence Number for replay protection          |
| dnn               | string  | Yes      | Data Network Name for PDU sessions             |
| routingindicator  | string  | Yes      | Routing Indicator (4 digits)                   |
| hplmn.mcc         | string  | Yes      | Home PLMN MCC                                  |
| hplmn.mnc         | string  | Yes      | Home PLMN MNC                                  |
| snssai.sst        | int     | Yes      | Slice/Service Type (1-255)                     |
| snssai.sd         | string  | No       | Slice Differentiator (6 hex chars)             |

### Security Algorithms

| Algorithm | Description        | Recommendation        |
| --------- | ------------------ | --------------------- |
| nia0/nea0 | Null (no security) | Testing only          |
| nia1/nea1 | SNOW 3G            | Optional              |
| nia2/nea2 | AES                | Recommended           |
| nia3/nea3 | ZUC                | Optional              |

---

## amfif

AMF endpoint configuration for core network connectivity.

```yaml
amfif:
  - ip: "127.0.0.8"       # AMF N2 interface IP
    port: 38412           # AMF N2 port (standard: 38412)
  - ip: "192.168.1.100"   # Additional AMF (optional)
    port: 38412
```

| Field | Type   | Required | Description           |
| ----- | ------ | -------- | --------------------- |
| ip    | string | Yes      | AMF N2 interface IP   |
| port  | int    | Yes      | AMF N2 port (default: 38412) |

Multiple AMFs supported for failover and inter-AMF scenarios.

---

## scenarios

UE groups and event sequences.

```yaml
scenarios:
  - nUEs: 100              # Number of UEs in this group
    gnbs: ["000008"]       # gNB IDs this group uses
    ueEvents:              # Event sequence
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
      - event: "PduSessionInit Event"
        delay: 2
        pdu_session_type: 0
```

### Scenario Entry

| Field     | Type     | Required | Description                          |
| --------- | -------- | -------- | ------------------------------------ |
| nUEs      | int      | Yes      | Number of UEs in this group          |
| gnbs      | []string | Yes      | gNB IDs (must exist in gnodeb.listGnbs) |
| ueEvents  | array    | Yes      | Sequence of events to execute        |

### Event Entry

| Field   | Type   | Required | Description                              |
| ------- | ------ | -------- | ---------------------------------------- |
| event   | string | Yes      | Event name (see [REFERENCE_EVENTS.md](./REFERENCE_EVENTS.md)) |
| delay   | int    | Yes      | Delay in seconds after previous event    |

Additional parameters depend on event type (see [REFERENCE_EVENTS.md](./REFERENCE_EVENTS.md)).

---

## remote

REST API server configuration.

```yaml
remote:
  enable: true            # Enable API server
  ip: "0.0.0.0"          # Bind address
  port: 4000             # Server port
```

| Field  | Type    | Required | Description               |
| ------ | ------- | -------- | ------------------------- |
| enable | bool    | Yes      | Enable/disable API server |
| ip     | string  | Yes      | Bind IP address           |
| port   | int     | Yes      | Server port               |

---

## testconf

Fuzzy testing and conformance configuration.

```yaml
testconf:
  enableFuzz: true
  5gmm:
    states:
      - "Registered State"
      - "Deregisterd State"
    events:
      - "RegisterInit Event"
      - "DeregistraterInit Event"
  5gsm:
    states:
      - "PDUSessionActive State"
      - "PDUSessionInactive State"
    events:
      - "PduSessionInit Event"
      - "DestroyPduSession Event"
```

| Field      | Type   | Required | Description                    |
| ---------- | ------ | -------- | ------------------------------ |
| enableFuzz | bool   | No       | Enable fuzzy testing (single UE only) |
| 5gmm       | object | No       | 5GMM state machine test config |
| 5gsm       | object | No       | 5GSM state machine test config |

---

## loglevel

Logging verbosity.

```yaml
loglevel: info
```

| Level  | Description                    |
| ------ | ------------------------------ |
| debug  | Detailed debugging information |
| info   | General operational messages   |
| warn   | Warning conditions             |
| error  | Error conditions               |
| fatal  | Fatal errors (process exits)   |
| panic  | Panic-level errors             |
