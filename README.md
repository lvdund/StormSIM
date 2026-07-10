# StormSim

> **Notice** - StormSim source code may be temporarily hidden until **10/2026** while the project completes full support for the planned specifications and features listed in [Future Work](#future-work). If you want to use StormSim during this period, use the published Docker image and follow [Simple way to run](#simple-way-to-run).

A scalable **UE** and **gNodeB** emulator for **testing, validating, and benchmarking 5G Core networks**. StormSim emulates thousands of UEs and gNodeBs that establish real NAS/NGAP signalling toward an actual 5G Core (AMF/SMF/UPF, e.g. [free5GC](https://github.com/free5gc/free5gc) or [Open5GS](https://github.com/open5gs/open5gs)). It is designed for stress-testing, regression validation, and performance benchmarking of the control plane.

> **Note** - StormSim emulates the **control plane** (UE ↔ gNB ↔ AMF) and sets up N3 data-plane tunnels via the [gtp5g](https://github.com/free5gc/gtp5g) kernel module of Free5gc. It is not a user-plane traffic generator.

---

## 📚 Documentation

| Document | Description |
| --- | --- |
| [docs/how-to-config.md](docs/how-to-config.md) | How to configure StormSim: every `config.yml` block, field reference, scenarios, recipes, and gotchas |
| [docs/how-to-use-oam-client.md](docs/how-to-use-oam-client.md) | How to use the OAM `client`: command reference with sample output and metrics |

---

## Features

### Scale

- Each UE and gNB runs on a shared worker pool and tested up to **10,000 UEs + gNodeBs in a single run** against a real 5G Core.  

### PCAP capture

- Capture signalling to a PCAP file with `--pcap`. StormSim attaches a live capture to the host interface that carries the gNodeB control plane (N2, which transports **N1 NAS**), so you get the full NAS/NGAP exchange in Wireshark.

### NAS procedures (UE side, 5GMM / 5GSM)

- Registration (Initial / Periodic / Emergency)
- Deregistration (UE-initiated and network-initiated; switch-off & re-registration)
- PDU Session Establishment / Release (and modification)
- Service Request
- Identity / Authentication / Security Mode Control
- Configuration Update
- Paging
- Idle-mode ↔ Connected-mode transitions
- Handover: Xn handover (Path Switch Request) and N2 handover

### NGAP procedures (gNB ↔ AMF)

- NG Setup
- Initial UE Message
- Uplink NAS Transport / Downlink NAS Transport
- Initial Context Setup (Request/Response)
- PDU Session Resource Setup / Release
- UE Context Release (Command/Complete)
- Paging
- Path Switch Request (Xn HO) and Handover Request/Command/Notify (N2 HO)
- AMF Configuration Update

### NAS protocol timers (3GPP TS 24.501)

Each timer can be individually enabled/disabled in config `defaultUe.timers`: T3510, T3511, T3502, T3512, T3516, T3519, T3520, T3580, T3582

### SUCI / Subscriber privacy

> **Status:** Currently StormSim uses the **Null-Scheme** (unconcealed SUPI/IMSI).
> **Profile A (X25519)** and **Profile B (P-256)** ECIES concealing/deconcealing are
> **not yet implemented** - see [Future Work](#future-work).

### Radio-link impairment (RLink)

Inject realistic radio-link conditions on the UE ↔ gNB path:
- **Delay** (`rlink.delay_ms`): random 0–N ms added per NAS message.
- **Loss** (`rlink.loss_ratio`): probability of dropping a NAS message (0.0–1.0).

### Client / OAM & metrics

A remote OAM server (HTTP) plus an interactive `client` let you **inspect and trigger events at runtime** while a scenario is running, and collect per-transition **delay statistics** (mean, std-dev, P1–P99 percentiles), UE state histograms, and handover-delay metrics - see [docs/how-to-use-oam-client.md](docs/how-to-use-oam-client.md).

---

## Future Work

- **SUCI concealing/deconcealing** - implement **Profile A (X25519)** and **Profile B (P-256)** ECIES (currently Null-Scheme only).
- **FUZZ mode** vs **Replay mode**:
    - FUZZ: randomized fault injection at the **UE state/event** level and at the **NAS IE** level (mutate information elements) to probe Core robustness.
    - Replay: record a signalling trace and deterministically replay it.
- **Conditional Handover (CHO)** and **timer handling during handover** (e.g. preserving/restarting 5GMM timers across the HO).

---

## Requirements


| Component        | Requirement                                                                      |
| ---------------- | -------------------------------------------------------------------------------- |
| **OS**           | Ubuntu **20.04 – 22.04**                                                         |
| **Linux kernel** | `5.4.* ≤ kernel ≤ 6.1.*` (required by `gtp5g`; StormSim borrows it from free5GC) |
| **Go**           | **1.25+**                                                                        |
| **SCTP**         | `sudo apt install make lksctp-tools`                                             |
| **Build tools**  | `make`, `gcc` (kernel headers for `gtp5g`)                                       |

  
> The kernel upper bound (`6.1.*`) is a hard constraint of the `gtp5g` kernel module.
> Newer kernels (6.2+) are not supported until `gtp5g` catches up.

Make sure the `gtp5g` kernel module is built and loaded before running scenarios that establish PDU sessions.

---

## Simple way to run

Pull the published image from Docker Hub:
```bash
docker pull lvdund/stormsim:latest
```

Before running, make sure:
- the host is Linux
- the `gtp5g` kernel module is already built and loaded on the host

Run the emulator:
```bash
docker run --rm -it \
  --privileged \
  --network host \
  -v "$(pwd)/config:/app/config:ro" \
  lvdund/stormsim:latest
```

This starts the emulator with `/app/config/config.yml` inside the container, so your host
directory should contain `config/config.yml`.

Run the client in interactive mode:
```bash
docker run --rm -it \
  --network host \
  --entrypoint client \
  lvdund/stormsim:latest
```

Run the client with a single command:
```bash
docker run --rm -it \
  --network host \
  --entrypoint client \
  lvdund/stormsim:latest list-ue
```

More client commands and examples are documented in [`docs/how-to-use-oam-client.md`](docs/how-to-use-oam-client.md).

---

## Build & Compile

StormSim produces two binaries via `make`:

```bash
# Build both
make # = make build = make emulator client

# Or build individually
make emulator # → bin/emulator
make client # → bin/client

make clean # remove bin/
```

| Binary         | Purpose                                                                                                          |
| -------------- | ---------------------------------------------------------------------------------------------------------------- |
| `bin/emulator` | The simulator itself: loads a config, spawns UEs/gNBs, runs scenarios, and serves the OAM API.                   |
| `bin/client`   | Command-line & interactive client that talks to the running emulator's OAM server (`localhost:4000` by default). |

---

## Running the Emulator

```bash
# With a config file
sudo ./bin/emulator -c config/config.yml

# With config + PCAP capture of the N2/N1 signalling
sudo ./bin/emulator -c config/config.yml --pcap capture.pcap

# Built-in configuration help
./bin/emulator --config-help
```

> **`sudo` is required** because StormSim binds SCTP/SCTP sockets, attaches a live PCAP handle to a host interface, and manages `gtp5g` N3 tunnels.

The emulator will:
1. Load & validate the config.
2. Connect the gNB(s) to the AMF (`NG Setup`).
3. Spawn the configured UE groups and enqueue their events.
4. Serve the OAM API on `remote.ip:remote.port` (default `0.0.0.0:4000`).

> 📖 Configuration is covered in **[docs/how-to-config.md](docs/how-to-config.md)**; using the live client is covered in **[docs/how-to-use-oam-client.md](docs/how-to-use-oam-client.md)**.

---

## Contributor

- [lvdund](https://github.com/lvdund)
- [Phùng Kiều Hà](mailto:ha.phungthikieu@hust.edu.vn)  
- [Thái Quang Tùng](https://github.com/reogac)

---

## Citation

```latex
@software{stormsim,
  author       = {VD},
  title        = {{StormSim}: A Scalable {UE} and {gNodeB} Emulator for Testing, Validating, and Benchmarking {5G Core} Networks},
  year         = {2025},
  publisher    = {GitHub},
  howpublished = {\url{https://github.com/lvdund/StormSIM}},
  note         = {Accessed: 2026-07-01}
}
```

---

## License

Licensed under the **Apache License, Version 2.0**. See [LICENSE](LICENSE).

StormSIM uses `gtp5g` of free5gc for enabling gtp tunnel and learns design from UERANSIM vs PacketRusher.
