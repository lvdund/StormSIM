# StormSIM Internals

Deep dive into the internal architecture for developers contributing to or extending StormSIM.

## FSM Framework

The finite state machine framework (`internal/common/fsm/`) provides async event-driven state transitions.

### Core Types

```go
type FSM struct {
    CurrentState State
    transitions  map[State]map[EventType]Transition
    eventChan    chan Event
    nextEvent    Event
}

type Transition struct {
    To     State
    Action func(Event) error
}

type Event interface {
    Type() EventType
}
```

### Event Flow

```
SendEvent() → eventChan → Worker picks event → FSM dispatch → State transition → Action callback
```

### Event Dispatch Rules

**Critical**: Never call `SendEvent()` from within an FSM callback. Use `SetNextEvent()` instead.

```go
// WRONG - Recursive dispatch
func onRegistrationAccept(event Event) error {
    fsm.SendEvent(PduSessionInitEvent{})  // DANGER: Recursive
    return nil
}

// CORRECT - Chained event
func onRegistrationAccept(event Event) error {
    fsm.SetNextEvent(PduSessionInitEvent{})  // Safe: Executes after callback
    return nil
}
```

**Why?** `SendEvent()` writes to a channel that the current goroutine is reading from. Calling it from within a callback creates a potential deadlock or recursive call chain.

### FSM Configuration

```go
fsm := fsm.NewFSM(
    MmDeregistered,
    fsm.Config{
        MmDeregistered: {
            RegisterInitEvent: {
                To:     MmRegistering,
                Action: onRegisterInit,
            },
        },
        MmRegistering: {
            RegistrationAcceptEvent: {
                To:     MmRegistered,
                Action: onRegistrationAccept,
            },
        },
    },
)
```

### Entry/Exit Actions

```go
fsm.Config{
    MmRegistered: {
        Entry: func() { startKeepAliveTimer() },
        Exit:  func() { stopKeepAliveTimer() },
        Events: { ... },
    },
}
```

---

## 5GMM State Machine

Mobility management states (`internal/core/uecontext/statemachine_5gmm.go`).

### State Diagram

```
                    ┌─────────────────────┐
                    │                     │
          ┌────────►│   MM_DEREGISTERED   │◄────────┐
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         RegisterInit Event              │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │  MM_REGISTERING     │         │
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │    Registration Success            Deregister Event
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │─────────┘
          │         │   MM_REGISTERED     │
          │         │                     │
          │         └─────────────────────┘
          │                    
          │         Deregistration Complete
          │                    
          └─────────────────────
```

### States

| State              | Description                              |
| ------------------ | ---------------------------------------- |
| MM_DEREGISTERED    | Initial state, no NAS connection         |
| MM_REGISTERING     | Registration procedure in progress        |
| MM_REGISTERED      | Successfully registered with network      |
| MM_SERVICE_REQUEST | Service request procedure active          |
| MM_PAGING          | Responding to network paging              |
| MM_DEREGISTERING   | Deregistration in progress                |

### Key Files

```
internal/core/uecontext/
├── statemachine_5gmm.go   # State definitions and transitions
├── handle_n1mm.go         # NAS-MM message handlers
├── triggers.go            # Event trigger functions
└── timer_manager.go       # NAS timers (T3510, T3511, etc.)
```

---

## 5GSM State Machine

Session management states (`internal/core/uecontext/statemachine_5gsm.go`).

### State Diagram

```
                    ┌─────────────────────┐
                    │                     │
          ┌────────►│   SM_PDU_INACTIVE   │◄────────┐
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │      PduSessionInit Event           Release Event
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │   SM_PDU_PENDING    │         │
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │    PduSessionAccept Event               │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │─────────┘
          │         │    SM_PDU_ACTIVE    │
          │         │                     │
          │         └─────────────────────┘
          │                    
          │    Release Complete
          │                    
          └────────────────────
```

### States

| State              | Description                              |
| ------------------ | ---------------------------------------- |
| SM_PDU_INACTIVE    | No PDU session established               |
| SM_PDU_PENDING     | Session establishment in progress         |
| SM_PDU_ACTIVE      | Session active, user plane available     |
| SM_MODIFYING       | Session modification in progress          |
| SM_RELEASING       | Session release in progress               |

### Key Files

```
internal/core/uecontext/
├── statemachine_5gsm.go   # State definitions and transitions
└── handle_n1sm.go         # NAS-SM message handlers
```

---

## Worker Pools

Worker pools (`internal/common/pool/`) manage goroutines for different event types.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Worker Pool System                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │  MmWorkerPool   │    │  SmWorkerPool   │    │  GnbWorkerPool  │         │
│  │  (5GMM Events)  │    │  (5GSM Events)  │    │  (NGAP Events)  │         │
│  │                 │    │                 │    │                 │         │
│  │  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │         │
│  │  │ MmWorker  │  │    │  │ SmWorker  │  │    │  │ GnbWorker │  │         │
│  │  │ (gorount.)│  │    │  │ (gorount.)│  │    │  │ (gorount.)│  │         │
│  │  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │         │
│  │       ...       │    │       ...       │    │       ...       │         │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘         │
│                                                                              │
│  ┌─────────────────┐                                                        │
│  │  SctpWorkerPool │                                                        │
│  │  (SCTP I/O)     │                                                        │
│  └─────────────────┘                                                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pool Types

| Pool           | Purpose            | Events Processed              |
| -------------- | ------------------ | ----------------------------- |
| MmWorkerPool   | 5GMM state machine | Registration, Deregistration  |
| SmWorkerPool   | 5GSM state machine | PDU Session events            |
| GnbWorkerPool  | NGAP handling      | NG Setup, UE Context mgmt     |
| SctpWorkerPool | SCTP I/O           | Message send/receive          |

### Auto-Sizing Algorithm

When `maxPool=0`, pools auto-size based on hardware:

```
totalPool = min(max(numCPU * 1500, nUEs * 3 + nGnbs * 100), 50000)
sctpWorkers = totalPool / 6
mmWorkers = (totalPool - sctpWorkers) * 0.40
smWorkers = (totalPool - sctpWorkers) * 0.35
gnbWorkers = (totalPool - sctpWorkers) - mmWorkers - smWorkers
```

### Hardware Sizing

| Hardware               | Pool Size | Max UEs   |
| ---------------------- | --------- | --------- |
| 2 cores, 4GB RAM       | ~3,000    | 500       |
| 4 cores, 8GB RAM       | ~6,000    | 1,500     |
| 8 cores, 16GB RAM      | ~12,000   | 3,000     |
| 16 cores, 32GB RAM     | ~24,000   | 6,000     |
| 32 cores, 64GB RAM     | ~48,000   | 12,000    |

---

## RLink (Virtual Radio)

The virtual radio (`internal/transport/rlink/`) simulates the UE-gNB interface.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       RLink Manager                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌─────────────┐      ┌─────────────┐                     │
│   │  UE Tx Chan │─────►│  gNB Rx Chan│                     │
│   │  (per UE)   │      │  (per gNB)  │                     │
│   └─────────────┘      └─────────────┘                     │
│                                                              │
│   ┌─────────────┐      ┌─────────────┐                     │
│   │ gNB Tx Chan │─────►│  UE Rx Chan │                     │
│   │  (per gNB)  │      │  (per UE)   │                     │
│   └─────────────┘      └─────────────┘                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Message Types

Defined in `pkg/model/rrc.go`:

| Message               | Direction   | Purpose                    |
| --------------------- | ----------- | -------------------------- |
| RrcSetupRequest       | UE → gNB    | Initial connection request |
| RrcSetup              | gNB → UE    | Connection configuration   |
| RrcSetupComplete      | UE → gNB    | Connection confirmation    |
| RrcReconfiguration    | gNB → UE    | Parameter update           |
| RrcRelease            | gNB → UE    | Connection release         |
| HandoverCommand       | gNB → UE    | Handover instruction       |
| HandoverComplete      | UE → gNB    | Handover confirmation      |

---

## SCTP/NGAP Transport

SCTP and NGAP handling (`internal/transport/sctpngap/`).

### Key Components

| Component             | Purpose                              |
| --------------------- | ------------------------------------ |
| Connection Manager    | Establishes/maintains SCTP connections |
| NGAP Encoder          | Serializes NGAP messages             |
| NGAP Decoder          | Parses incoming NGAP messages        |
| Message Router        | Dispatches to appropriate handlers   |

### NGAP Procedures

| Procedure                    | Direction   | Trigger                    |
| ---------------------------- | ----------- | -------------------------- |
| NG Setup                     | gNB → AMF   | Initial connection         |
| Initial UE Message           | gNB → AMF   | New UE registration        |
| Uplink NAS Transport         | gNB → AMF   | NAS message from UE        |
| Downlink NAS Transport       | AMF → gNB   | NAS message to UE          |
| PDU Session Resource Setup   | AMF → gNB   | Session establishment      |
| PDU Session Resource Release | AMF → gNB   | Session release            |
| Handover Preparation         | gNB → AMF   | N2 handover initiation     |
| Handover Resource Allocation | AMF → gNB   | Handover target setup      |

---

## Concurrency Patterns

### Thread Safety

| Pattern       | Usage                    | Location         |
| ------------- | ------------------------ | ---------------- |
| sync.Map      | UE Pool, AMF Pool        | gnbcontext/      |
| sync/atomic   | ID generation, counters  | uecontext/       |
| Channels      | Event dispatch, RLink    | fsm/, rlink/     |
| Per-UE mutex  | Non-hot paths only       | uecontext/       |

### Critical Constraint: No Mutexes in Hot Paths

Hot paths are code executed for every event. Using mutexes here degrades performance:

```go
// AVOID in hot paths
type UeContext struct {
    mu    sync.Mutex  // BAD: Contention point
    state State
}

// PREFER
type UeContext struct {
    state atomic.Value  // GOOD: Lock-free
}
```

### Per-UE Goroutine Model

Each UE processes events in its own goroutine:

```go
func (ue *UeContext) run() {
    for event := range ue.eventChan {
        ue.fsm.Dispatch(event)
    }
}
```

Benefits:
- No shared state between UEs
- Natural parallelism
- No head-of-line blocking

---

## Delay Tracking

Built-in latency measurement at protocol boundaries.

### Measurement Points

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ NAS Tx  │───►│ NGAP Tx │───►│ SCTP Tx │───►│ AMF Rx  │
│ t0      │    │ t1      │    │ t2      │    │ t3      │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
```

| Segment        | Calculation  |
| -------------- | ------------ |
| NAS→NGAP       | t1 - t0      |
| NGAP→SCTP      | t2 - t1      |
| SCTP→AMF       | t3 - t2      |
| End-to-end     | t3 - t0      |

---

## Error Handling Patterns

### Current Anti-Pattern (Technical Debt)

```go
// AVOID: Fatal on protocol errors
func handleNgapMessage(msg []byte) {
    if err := decode(msg); err != nil {
        log.Fatal("decode failed")  // BAD: Crashes process
    }
}
```

### Preferred Pattern

```go
// PREFER: Graceful error handling
func handleNgapMessage(msg []byte) error {
    if err := decode(msg); err != nil {
        log.Errorf("decode failed: %v", err)
        return err  // GOOD: Let caller handle
    }
    return nil
}
```

---

## Extending StormSIM

### Adding a New Event

1. Define event type in `pkg/model/events.go`:

```go
type MyNewEvent struct {
    Param1 string
}

func (e MyNewEvent) Type() EventType {
    return MyNewEventType
}
```

2. Add state transition in appropriate FSM:

```go
// In statemachine_5gmm.go or statemachine_5gsm.go
MyState: {
    MyNewEventType: {
        To:     NextState,
        Action: onMyNewEvent,
    },
}
```

3. Implement handler:

```go
func onMyNewEvent(event Event) error {
    e := event.(MyNewEvent)
    // Handle event
    return nil
}
```

### Adding a New Configuration Option

1. Add field to `pkg/config/config.go`:

```go
type Config struct {
    // ... existing fields
    MyNewOption string `yaml:"myNewOption"`
}
```

2. Update YAML parsing and usage in relevant components.

### Adding a New API Endpoint

1. Add handler in `monitoring/oambackend/`:

```go
func (s *Server) handleMyNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}

// Register in routes
router.HandleFunc("/api/mynew", s.handleMyNewEndpoint).Methods("GET")
```
