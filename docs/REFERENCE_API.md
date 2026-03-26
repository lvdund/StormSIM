# REST API Reference

StormSIM provides a REST API for real-time control and monitoring.

## Enable the API

```yaml
remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000
```

Base URL: `http://<ip>:<port>`

---

## UE Endpoints

### List All UEs

```
GET /ues
```

**Response:**

```json
[
  {
    "id": 1,
    "msin": "0000000000",
    "state": "Registered",
    "gnb": "000008",
    "sessions": 1
  },
  {
    "id": 2,
    "msin": "0000000001",
    "state": "Deregistered",
    "gnb": "000008",
    "sessions": 0
  }
]
```

---

### Get UE Details

```
GET /ues/{id}
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Response:**

```json
{
  "id": 1,
  "msin": "0000000000",
  "imsi": "208930000000000",
  "state": "Registered",
  "gnb": "000008",
  "sessions": [
    {
      "id": 1,
      "state": "Active",
      "dnn": "internet",
      "slice": {
        "sst": 1,
        "sd": "010203"
      }
    }
  ],
  "security": {
    "integrity": "nia2",
    "ciphering": "nea0"
  }
}
```

---

### Trigger UE Registration

```
POST /ues/{id}/register
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Request Body (optional):**

```json
{
  "register_type": 0
}
```

**Response:**

```json
{
  "status": "accepted",
  "ue_id": 1,
  "event": "RegisterInit"
}
```

---

### Trigger UE Deregistration

```
POST /ues/{id}/deregister
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Request Body (optional):**

```json
{
  "deregister_type": 0
}
```

**Response:**

```json
{
  "status": "accepted",
  "ue_id": 1,
  "event": "DeregistraterInit"
}
```

---

### Establish PDU Session

```
POST /ues/{id}/session/create
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Request Body (optional):**

```json
{
  "pdu_session_type": 0
}
```

**Response:**

```json
{
  "status": "accepted",
  "ue_id": 1,
  "event": "PduSessionInit"
}
```

---

### Release PDU Session

```
POST /ues/{id}/session/release
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Response:**

```json
{
  "status": "accepted",
  "ue_id": 1,
  "event": "DestroyPduSession"
}
```

---

### Trigger Handover

```
POST /ues/{id}/handover
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | int    | UE identifier   |

**Request Body:**

```json
{
  "type": "xn",
  "target_gnb": "000009"
}
```

| Field       | Type   | Values        | Description          |
| ----------- | ------ | ------------- | -------------------- |
| type        | string | "xn", "n2"    | Handover type        |
| target_gnb  | string | gNB ID        | Target gNodeB ID     |

**Response:**

```json
{
  "status": "accepted",
  "ue_id": 1,
  "event": "XnHandover"
}
```

---

## gNB Endpoints

### List All gNBs

```
GET /gnbs
```

**Response:**

```json
[
  {
    "id": "000008",
    "tac": "000001",
    "plmn": {
      "mcc": "208",
      "mnc": "93"
    },
    "ues": 50,
    "amf_connected": true
  }
]
```

---

### Get gNB Details

```
GET /gnbs/{id}
```

**Parameters:**
| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| id        | string | gNB ID (6 digits) |

**Response:**

```json
{
  "id": "000008",
  "tac": "000001",
  "plmn": {
    "mcc": "208",
    "mnc": "93"
  },
  "slices": [
    {
      "sst": 1,
      "sd": "010203"
    }
  ],
  "ues": 50,
  "amf_connected": true,
  "amf_address": "127.0.0.8:38412"
}
```

---

## Metrics Endpoints

### Get Simulation Metrics

```
GET /metrics
```

**Response:**

```json
{
  "ues": {
    "total": 100,
    "registered": 95,
    "deregistered": 5
  },
  "sessions": {
    "total": 90,
    "active": 88,
    "inactive": 2
  },
  "procedures": {
    "registrations": {
      "success": 100,
      "failed": 0,
      "avg_latency_ms": 234
    },
    "pdu_sessions": {
      "success": 90,
      "failed": 0,
      "avg_latency_ms": 156
    },
    "handovers": {
      "success": 10,
      "failed": 0,
      "avg_latency_ms": 89
    }
  }
}
```

---

### Get Delay Measurements

```
GET /delays
```

**Response:**

```json
{
  "ue_1": {
    "nas_to_ngap_ms": 5,
    "ngap_to_sctp_ms": 3,
    "sctp_to_amf_ms": 12,
    "end_to_end_ms": 20
  },
  "ue_2": {
    "nas_to_ngap_ms": 4,
    "ngap_to_sctp_ms": 2,
    "sctp_to_amf_ms": 11,
    "end_to_end_ms": 17
  }
}
```

---

### Get Procedure Statistics

```
GET /stats
```

**Response:**

```json
{
  "registration": {
    "total": 100,
    "success": 98,
    "failed": 2,
    "success_rate": 0.98,
    "latency": {
      "min_ms": 150,
      "max_ms": 500,
      "avg_ms": 234,
      "p50_ms": 220,
      "p95_ms": 380,
      "p99_ms": 450
    }
  },
  "pdu_session": {
    "total": 90,
    "success": 90,
    "failed": 0,
    "success_rate": 1.0,
    "latency": {
      "min_ms": 100,
      "max_ms": 300,
      "avg_ms": 156,
      "p50_ms": 150,
      "p95_ms": 250,
      "p99_ms": 280
    }
  }
}
```

---

## Log Endpoints

### Access Log Streams

```
GET /logs
```

**Query Parameters:**
| Parameter | Type   | Description               |
| --------- | ------ | ------------------------- |
| ue_id     | int    | Filter by UE ID (optional) |
| level     | string | Filter by level (optional) |

**Response:** Server-sent events stream

---

## Error Responses

All endpoints return standard error format:

```json
{
  "error": "UE not found",
  "code": 404,
  "details": "UE with id 999 does not exist"
}
```

| Code | Description              |
| ---- | ------------------------ |
| 400  | Bad request              |
| 404  | Resource not found       |
| 500  | Internal server error    |
| 503  | Service unavailable      |
