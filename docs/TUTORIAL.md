# StormSIM Tutorial

## Prerequisites

Before starting, ensure you have:

- Go 1.22.5 or later
- A running 5G core network (Free5GC or Open5GS)
- Linux system with SCTP kernel module (`sudo modprobe sctp`)
- Root/sudo access

## Build StormSIM

```bash
git clone <repository-url>
cd StormSIM-private
make
```

This produces:
- `bin/emulator` - Main emulator binary
- `bin/client` - OAM CLI client

## Your First Test

### Step 1: Configure the Emulator

Create `config/my-first-test.yml`:

```yaml
gnodeb:
  controlif:
    ip: "127.0.0.20"
    port: 9487
  dataif:
    ip: "127.0.0.20"
    port: 2152
  listGnbs:
    - gnbid: "000008"
      tac: "000001"
      plmn:
        mcc: "208"
        mnc: "93"
      slicesupportlist:
        - sst: "01"
          sd: "010203"

defaultUe:
  msin: "0000000000"
  key: "14b23ceb27e95eb732a3f9d602f551c4"
  opc: "a3e3c63de23b66dc6a8ae0272b44906c"
  amf: "8000"
  sqn: "00000000"
  dnn: "internet"
  routingindicator: "0000"
  hplmn:
    mcc: "208"
    mnc: "93"
  snssai:
    sst: 01
    sd: "010203"
  integrity:
    nia2: true
  ciphering:
    nea0: true
    nea2: true

amfif:
  - ip: "127.0.0.8"
    port: 38412

scenarios:
  - nUEs: 1
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
      - event: "PduSessionInit Event"
        delay: 2
        pdu_session_type: 0

remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000

loglevel: info
```

**Key configuration points:**
- `amfif` must match your 5G core's N2 interface
- `defaultUe.key` and `defaultUe.opc` must match subscriber data in your core
- `hplmn` (MCC/MNC) must match your core's PLMN configuration

### Step 2: Add Subscriber to Your Core

Add a subscriber in your 5G core with:
- IMSI: `208930000000000` (MCC + MNC + MSIN)
- Key: `14b23ceb27e95eb732a3f9d602f551c4`
- OPC: `a3e3c63de23b66dc6a8ae0272b44906c`
- AMF: `8000`
- SQN: `00000000`

### Step 3: Run the Test

```bash
sudo ./bin/emulator -c config/my-first-test.yml
```

### Step 4: Verify Success

You should see output similar to:

```
[INFO] SCTP connection established to 127.0.0.8:38412
[INFO] NG Setup successful
[INFO] UE 0000000000: Registration initiated
[INFO] UE 0000000000: Authentication successful
[INFO] UE 0000000000: Registration accepted
[INFO] UE 0000000000: PDU session established
```

Check via the API:

```bash
curl http://localhost:4000/ues
```

Expected response shows the UE in `Registered` state with an active PDU session.

## What Just Happened?

1. **NG Setup**: StormSIM established SCTP connection to the AMF and performed NG Setup procedure
2. **Registration**: The UE authenticated with the core using 5G-AKA
3. **Security Mode**: Integrity and ciphering algorithms were negotiated
4. **PDU Session**: A data session was established, creating a GTP-U tunnel

## Next Steps

- **Scale up**: Increase `nUEs` to test load (e.g., 100, 500, 1000)
- **Add scenarios**: See [HOWTO_TESTING.md](./HOWTO_TESTING.md) for advanced test configurations
- **Explore APIs**: Use the REST API at `http://localhost:4000` for real-time control
- **Understand the system**: Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system design

## Troubleshooting

### SCTP Connection Failed

```bash
sudo modprobe sctp
sudo ss -a | grep sctp  # Verify no existing connections
```

### Authentication Failures

1. Verify subscriber exists in 5G core database
2. Check key/opc match exactly (case-sensitive, no spaces)
3. Confirm PLMN (MCC/MNC) matches core configuration

### No Output

Set logging to debug:

```yaml
loglevel: debug
```
