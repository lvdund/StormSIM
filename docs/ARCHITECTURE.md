# StormSIM Architecture

This document explains the high-level architecture and design of StormSIM.

## System Purpose

StormSIM is a high-performance 5G UE and gNodeB emulator designed for testing and benchmarking 5G Core networks (Open5GS, Free5GC). It simulates the radio access network (RAN) side of 5G, handling both control plane (NGAP/NAS) and user plane (GTP-U) protocols.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              5G Core (AMF/SMF/UPF)                          │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │    SCTP/NGAP (N2)         │
                    │    GTP-U (N3)             │
                    └─────────────┬─────────────┘
                                  │
┌─────────────────────────────────┴───────────────────────────────────────────┐
│                            StormSIM Emulator                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                        GnbContext (gNodeB)                               ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ ││
│  │  │   AMF Pool   │  │   UE Pool    │  │ NGAP Dispatch│  │  SCTP Conn  │ ││
│  │  │  (sync.Map)  │  │  (sync.Map)  │  │  (handlers)  │  │  Manager    │ ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘ ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                  │                                           │
│                    ┌─────────────┴─────────────┐                            │
│                    │   RLink (Virtual Radio)   │                            │
│                    │   UE↔gNB Channels         │                            │
│                    └─────────────┬─────────────┘                            │
│                                  │                                           │
│  ┌───────────────────────────────┴───────────────────────────────────────┐  │
│  │                    UeContext (per UE)                                 │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │  │
│  │  │  5GMM FSM    │  │  5GSM FSM    │  │ NAS Handler  │  │  Timers    │ │  │
│  │  │ (MmWorker)   │  │ (SmWorker)   │  │ (Encode/Dec) │  │  Engine    │ │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Observability Layer                                │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ Ring Buffer  │  │ Delay Track  │  │   Stats      │               │   │
│  │  │   Logger     │  │   (per UE)   │  │ (per proc)   │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │     OAM Backend           │
                    │     (REST API)            │
                    └───────────────────────────┘
```

## Core Components

### GnbContext

The gNodeB context manages all gNB-side functionality:

- **AMF Pool**: Manages connections to multiple AMFs using `sync.Map`
- **UE Pool**: Stores all UE contexts using `sync.Map`
- **NGAP Dispatch**: Routes incoming NGAP messages to appropriate handlers
- **SCTP Connection Manager**: Handles SCTP connections to AMFs

### UeContext

Each UE has its own context containing:

- **5GMM FSM**: Mobility management state machine
- **5GSM FSM**: Session management state machine
- **NAS Handler**: Encodes and decodes NAS messages
- **Timer Engine**: Manages NAS timers (T3510, T3511, etc.)

### RLink (Virtual Radio)

The virtual radio simulates the air interface between UE and gNB using Go channels:

- UE-to-gNB channel for uplink messages
- gNB-to-UE channel for downlink messages
- No actual RF simulation; purely for protocol layer testing

### Worker Pools

StormSIM uses dedicated worker pools for different event types:

| Pool           | Purpose            | Worker Type  |
| -------------- | ------------------ | ------------ |
| MmWorkerPool   | 5GMM events        | MmWorker     |
| SmWorkerPool   | 5GSM events        | SmWorker     |
| GnbWorkerPool  | NGAP events        | GnbWorker    |
| SctpWorkerPool | SCTP I/O           | SctpWorker   |

### Observability Layer

- **Ring Buffer Logger**: Per-UE log buffers for high UE counts
- **Delay Tracking**: Measures latency at protocol boundaries
- **Stats**: Aggregates procedure success/failure rates and latencies

## Protocol Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                        UE Side                                   │
├─────────────────────────────────────────────────────────────────┤
│  NAS (5GMM/5GSM)                                                 │
├─────────────────────────────────────────────────────────────────┤
│  RRC (Simulated via RLink)                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        gNB Side                                  │
├─────────────────────────────────────────────────────────────────┤
│  NGAP                                                            │
├─────────────────────────────────────────────────────────────────┤
│  SCTP (N2)                        │  GTP-U (N3)                  │
└───────────────────────────────────┴─────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        5G Core                                   │
│  AMF (N2)                         │  UPF (N3)                    │
└───────────────────────────────────┴─────────────────────────────┘
```

## Directory Structure

```
StormSIM/
├── cmd/                        # Application entrypoints
│   └── emulator/               # Main emulator binary
├── internal/
│   ├── core/                   # Core protocol handling
│   │   ├── uecontext/          # UE context, FSMs, NAS handlers
│   │   └── gnbcontext/         # gNB context, NGAP handlers
│   ├── common/                 # Shared infrastructure
│   │   ├── fsm/                # Async FSM framework
│   │   ├── pool/               # Worker pool management
│   │   ├── logger/             # Ring buffer logging
│   │   └── stats/              # Procedure statistics
│   ├── transport/              # Transport layer
│   │   ├── rlink/              # Virtual radio (UE↔gNB)
│   │   └── sctpngap/           # SCTP/NGAP transport
│   └── scenarios/              # UE orchestration
├── pkg/
│   ├── model/                  # 3GPP data models, events, states
│   └── config/                 # Configuration structures
├── monitoring/
│   ├── oambackend/             # REST API for control/metrics
│   └── gtp5g/                  # Kernel GTP-U tunnel management
└── config/                     # YAML configuration files
```

## Key Design Decisions

### Why Per-UE Goroutines?

Each UE context processes events in its own goroutine. This provides:

- **Isolation**: One slow UE doesn't block others
- **Simplicity**: No locking needed for per-UE state
- **Scalability**: Natural parallelism across CPU cores

### Why Worker Pools Instead of Unlimited Goroutines?

Worker pools prevent goroutine explosion under load:

- Bounded resource consumption
- Predictable memory usage
- Backpressure on event submission

### Why Virtual Radio (RLink)?

RLink eliminates real RF simulation complexity:

- Focus on protocol correctness, not PHY layer
- Fast message passing via Go channels
- Deterministic testing without RF variability

## Scalability Design

StormSIM is designed for 10,000+ concurrent UEs:

1. **sync.Map for pools**: Lock-free reads for UE/AMF lookup
2. **Ring buffer logging**: Fixed memory per UE regardless of log volume
3. **Worker pools**: Bounded goroutine count
4. **Channel-based coordination**: No mutexes in hot paths

## Critical Constraints

These constraints must never be violated:

| Constraint                      | Reason                                    |
| ------------------------------- | ----------------------------------------- |
| No recursive FSM events         | Causes stack overflow and deadlock        |
| No sync.Pool for SCTP reads     | Causes data corruption across workers     |
| No mutexes in hot paths         | Degrades performance under high concurrency |
| Use SetNextEvent, not SendEvent | Prevents recursive event dispatch         |

## Anti-Patterns and Technical Debt

| Issue                              | Location                        | Status   |
| ---------------------------------- | ------------------------------- | -------- |
| Race condition in ID generation    | `getRanUeId()`, `getUeTeid()`   | Known    |
| Excessive Fatal() calls            | Protocol handlers               | Known    |

## Further Reading

- [INTERNALS.md](./INTERNALS.md) - Deep dive into FSM framework, worker pools, and concurrency patterns
- [REFERENCE_CONFIG.md](./REFERENCE_CONFIG.md) - Configuration schema
- [REFERENCE_API.md](./REFERENCE_API.md) - REST API reference
