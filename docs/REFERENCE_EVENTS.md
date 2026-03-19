# UE Events Reference

Complete reference for all UE events and their parameters.

## Event Syntax

Events are specified in the `ueEvents` array within scenarios:

```yaml
ueEvents:
  - event: "EventName"
    delay: 0              # Seconds after previous event
    parameter1: value1    # Event-specific parameters
    parameter2: value2
```

---

## Lifecycle Events

### RegisterInit Event

Initiate UE registration with the network.

```yaml
- event: "RegisterInit Event"
  delay: 0
  register_type: 0
```

| Parameter     | Type | Values                      | Default |
| ------------- | ---- | --------------------------- | ------- |
| register_type | int  | 0=Initial, 1=Mobility, 2=Periodic | 0       |

**Registration Types:**
| Value | Type       | Description                        |
| ----- | ---------- | ---------------------------------- |
| 0     | Initial    | First registration after power-on  |
| 1     | Mobility   | Registration due to mobility       |
| 2     | Periodic   | Periodic registration update       |

---

### DeregistraterInit Event

Initiate UE deregistration from the network.

```yaml
- event: "DeregistraterInit Event"
  delay: 10
  deregister_type: 0
```

| Parameter      | Type | Values                        | Default |
| -------------- | ---- | ----------------------------- | ------- |
| deregister_type | int  | 0=Normal, 1=Switch-off        | 0       |

**Deregistration Types:**
| Value | Type       | Description                        |
| ----- | ---------- | ---------------------------------- |
| 0     | Normal     | Graceful deregistration            |
| 1     | Switch-off | UE switching off (no response expected) |

---

### ServiceRequestInit Event

Initiate service request to resume connectivity after idle mode.

```yaml
- event: "ServiceRequestInit Event"
  delay: 5
```

No additional parameters.

---

### Terminate Event

Graceful UE termination (clean shutdown).

```yaml
- event: "Terminate Event"
  delay: 30
```

No additional parameters.

---

### Kill Event

Force UE termination (immediate shutdown).

```yaml
- event: "Kill Event"
  delay: 30
```

No additional parameters.

---

## Session Events

### PduSessionInit Event

Establish a PDU session for data connectivity.

```yaml
- event: "PduSessionInit Event"
  delay: 2
  pdu_session_type: 0
```

| Parameter       | Type | Values                        | Default |
| --------------- | ---- | ----------------------------- | ------- |
| pdu_session_type | int  | 0=Initial, 1=Emergency        | 0       |

**Session Types:**
| Value | Type      | Description                        |
| ----- | --------- | ---------------------------------- |
| 0     | Initial   | Standard PDU session establishment |
| 1     | Emergency | Emergency PDU session              |

---

### DestroyPduSession Event

Release an established PDU session.

```yaml
- event: "DestroyPduSession Event"
  delay: 10
```

No additional parameters.

---

### ModificationRequest Event

Request modification of PDU session parameters.

```yaml
- event: "ModificationRequest Event"
  delay: 15
```

No additional parameters. Used to update QoS, routing rules, or other session attributes.

---

## Mobility Events

### XnHandover Event

Perform handover between gNBs via Xn interface.

```yaml
- event: "XnHandover Event"
  delay: 5
```

**Requirements:**
- Multiple gNBs configured in `gnodeb.listGnbs`
- UE must be registered with active PDU session
- Target gNB selected automatically from configured list

No additional parameters.

---

### N2Handover Event

Perform handover between gNBs via N2 interface (through AMF).

```yaml
- event: "N2Handover Event"
  delay: 5
```

**Requirements:**
- Multiple gNBs configured
- AMF must support N2 handover procedures
- UE must be registered with active PDU session

No additional parameters.

---

## Event Timing

The `delay` parameter controls event sequencing:

```yaml
ueEvents:
  - event: "RegisterInit Event"
    delay: 0          # Execute immediately
  - event: "PduSessionInit Event"
    delay: 2          # Execute 2 seconds after registration
  - event: "DeregistraterInit Event"
    delay: 10         # Execute 10 seconds after PDU session init
```

**Timing Notes:**
- Delay is relative to the previous event's scheduled time, not completion time
- Events execute in order regardless of procedure completion
- Use sufficient delays to allow procedures to complete

---

## State Dependencies

Events have state requirements:

| Event                  | Required State                | Resulting State              |
| ---------------------- | ----------------------------- | ---------------------------- |
| RegisterInit           | Deregistered                  | Registering → Registered     |
| DeregistraterInit      | Registered                    | Deregistering → Deregistered |
| ServiceRequestInit     | Registered (idle)             | Connected                    |
| PduSessionInit         | Registered                    | Session Active               |
| DestroyPduSession      | Session Active                | Session Inactive             |
| XnHandover             | Registered + Session Active   | Registered (different gNB)   |
| N2Handover             | Registered + Session Active   | Registered (different gNB)   |
| Terminate              | Any                           | Terminated                   |
| Kill                   | Any                           | Terminated                   |

---

## Example Sequences

### Full Lifecycle

```yaml
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
    deregister_type: 0
```

### Handover Test

```yaml
ueEvents:
  - event: "RegisterInit Event"
    delay: 0
  - event: "PduSessionInit Event"
    delay: 2
  - event: "XnHandover Event"
    delay: 5
  - event: "N2Handover Event"
    delay: 10
  - event: "DeregistraterInit Event"
    delay: 15
```

### Rapid Re-registration (Stress Test)

```yaml
ueEvents:
  - event: "RegisterInit Event"
    delay: 0
  - event: "DeregistraterInit Event"
    delay: 1
  - event: "RegisterInit Event"
    delay: 2
  - event: "DeregistraterInit Event"
    delay: 3
```
