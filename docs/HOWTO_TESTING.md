# How to Test 5G Core Networks

This guide covers running tests for conformance, performance, and reliability validation.

## Run a Basic Conformance Test

**Goal**: Verify single UE can complete registration and PDU session establishment.

```yaml
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
      - event: "DestroyPduSession Event"
        delay: 10
      - event: "DeregistraterInit Event"
        delay: 12
```

```bash
sudo ./bin/emulator -c config/conformance.yml
```

**Expected**: All procedures complete within 15 seconds with no errors.

## Run a Load Test

**Goal**: Measure registration throughput and latency at scale.

Create `config/load-test.yml`:

```yaml
scenarios:
  - nUEs: 1000
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
```

```bash
sudo ./bin/emulator -c config/load-test.yml
```

**Monitor metrics:**
- Registration latency (target: <5 seconds per UE)
- Success rate (target: >99%)
- Core CPU/memory usage

**Scale progression**: 100 → 500 → 1000 → 2000 → 5000 → 10000 UEs

## Run a PDU Session Throughput Test

**Goal**: Test session establishment performance under load.

```yaml
scenarios:
  - nUEs: 500
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2
```

**Monitor:**
- Session setup latency
- UPF resource utilization
- GTP tunnel creation rate

## Run a Handover Test

**Goal**: Validate Xn and N2 handover procedures.

**Prerequisite**: Configure multiple gNBs.

```yaml
gnodeb:
  listGnbs:
    - gnbid: "000008"
      tac: "000001"
      plmn: {mcc: "208", mnc: "93"}
    - gnbid: "000009"
      tac: "000001"
      plmn: {mcc: "208", mnc: "93"}

scenarios:
  - nUEs: 10
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2
      - event: "XnHandover Event"
        delay: 5
      - event: "DeregistraterInit Event"
        delay: 10
```

**Expected:**
- Handover completion time <500ms
- Handover success rate >95%
- Session continuity maintained

For N2 handover, replace `XnHandover Event` with `N2Handover Event`.

## Run Multiple UE Groups

**Goal**: Simulate different UE behaviors simultaneously.

```yaml
scenarios:
  - nUEs: 50
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
  - nUEs: 30
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
      - event: "XnHandover Event"
        delay: 5
```

This creates 50 UEs that only register, plus 30 UEs that complete full lifecycle with handover.

## Run a Stress Test

**Goal**: Test core resiliency under signaling storm.

```yaml
scenarios:
  - nUEs: 500
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "DeregistraterInit Event"
        delay: 1
      - event: "RegisterInit Event"
        delay: 2
```

**Monitor:**
- Core stability
- Recovery time after failures
- Memory leak indicators

## Run a Long-Duration Test

**Goal**: Identify long-term stability issues.

```yaml
scenarios:
  - nUEs: 200
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 5
      - event: "XnHandover Event"
        delay: 300
      - event: "DeregistraterInit Event"
        delay: 600
      - event: "RegisterInit Event"
        delay: 900
```

**Duration**: Run for 24-48 hours.

**Monitor:**
- Memory usage trends
- Performance degradation over time
- Error frequency in logs

## Use the REST API for Real-Time Control

Enable the API:

```yaml
remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000
```

**Trigger UE registration:**

```bash
curl -X POST http://localhost:4000/ues/1/register
```

**Check UE status:**

```bash
curl http://localhost:4000/ues
```

**Get metrics:**

```bash
curl http://localhost:4000/metrics
```

## Collect Metrics During Tests

StormSIM provides built-in metrics:

| Category       | Metrics                                    |
| -------------- | ------------------------------------------ |
| UE State       | Registration status, session count         |
| Timing         | Registration latency, session setup time   |
| Network        | SCTP connections, NGAP messages, GTP tunnels |
| Errors         | Authentication failures, protocol errors   |

Sample metrics table output:

| State       | Msg Recv            | Time        | Event         |
| ----------- | ------------------- | ----------- | ------------- |
| deregistered | -                   | -           | register init |
| deregistered | identity req        | 0:31:000014 | -             |
| registered   | registration accept | 0:31:000034 | -             |

## Tune System for Large-Scale Tests

Before running tests with >1000 UEs:

```bash
sudo sysctl -w net.core.rmem_max=33554432
sudo sysctl -w net.core.wmem_max=33554432
sudo sysctl -w net.ipv4.tcp_rmem='4096 65536 33554432'
sudo sysctl -w net.ipv4.tcp_wmem='4096 65536 33554432'
sudo sysctl -w net.core.somaxconn=10240
sudo sysctl -w net.core.netdev_max_backlog=10000
```

## Verify Success Criteria

| Test Type     | Success Criteria                                  |
| ------------- | ------------------------------------------------- |
| Conformance   | All procedures complete, no protocol violations   |
| Performance   | Latency <5s, success rate >99%, CPU <80%          |
| Handover      | Completion <500ms, success rate >90%              |
| Stress        | System stable, recovery <30s, no memory leaks     |
