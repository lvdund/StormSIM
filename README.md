# StormSim - 5G Mobile Network Simulator

<div align="center">
  <h3>A scalable UE and gNodeB emulator for benchmarking open-source 5G core networks</h3>
  <p>
    <strong>Supporting Free5GC, Open5GS, and other 3GPP-compliant 5G cores</strong>
  </p>
</div>

## Overview

StormSim is a comprehensive 5G network emulator designed for testing and benchmarking open-source 5G core networks. It provides scalable simulation of User Equipment (UE) and gNodeB behavior, supporting thousands of concurrent UEs with realistic traffic patterns and mobility scenarios.

### Key Features

- **Scalable UE Simulation**: Support for 1 to 10,000+ concurrent UEs
- **Full 5G Procedures**: Registration, PDU sessions, handovers (Xn/N2), Paging, Roaming, Service requests
- **Conformance Testing**: 3GPP-compliant signaling with fuzzy testing capabilities
- **Performance Benchmarking**: Built-in metrics collection and chaos injection by `Client`
- **Flexible Scenarios**: Configuration-driven test scenarios and UE behaviors
- **Real-time Control with Client**: REST API for dynamic UE/gNB control
- **Multi-Core Support**: Compatible with Free5GC, Open5GS, OAI

## Quick Start

### Prerequisites

- **Linux**: Ubuntu 20.04+ (kernel 5.4.x recommended)
- **Go**: Version 1.22.5+
- **SCTP**: Required kernel module

### Installation

1. **Install Go 1.22.5**:
```bash
wget https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -zxvf go1.22.5.linux-amd64.tar.gz
mkdir -p ~/go/{bin,pkg,src}
# The following assume that your shell is bash:
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin:$GOROOT/bin' >> ~/.bashrc
echo 'export GO111MODULE=auto' >> ~/.bashrc
source ~/.bashrc
```

2. **Install dependencies**:
```bash
sudo apt update
sudo apt install make lksctp-tools
```

3. **Build StormSim**:
```bash
git clone https://github.com/lvdund/StormSim.git stormsim
cd stormsim
make
```

### Basic Usage

```bash
# Run with custom configuration
sudo ./bin/emulator -c config/config.yml

# Replay recorded scenarios
sudo ./bin/emulator -r replay_file.log

# Show configuration help
sudo ./bin/emulator --config-help
```

## Configuration

StormSim is entirely configuration-driven through `config/config.yml`. Key sections include:

- **scenarios**: Define UE count, behaviors, and events
- **gnodeb**: Configure gNB parameters and multiple gNBs for handover
- **defaultUe**: Set UE authentication and profile parameters
- **amfif**: Specify AMF endpoints
- **remote**: Enable REST API for external control

Example minimal configuration:
```yaml
scenarios:
  - nUEs: 10
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
```

## Documentation

- **[Architecture Guide](docs/ARCH.md)** - System architecture, data flow, and state machines
- **[Configuration Guide](docs/CONFIGURATION.md)** - Detailed configuration options
- **[Testing Scenarios](docs/TESTING.md)** - Test cases and benchmarking
- **[Supported Features](docs/FEATURES.md)** - 5G procedures and capabilities
- **[Research Background](docs/RESEARCH.md)** - Academic context and objectives

## 5G Core Setup

### Free5GC
```bash
git clone --recursive -b v3.4.4 https://github.com/free5gc/free5gc.git
cd free5gc && make
# Configure and run according to Free5GC documentation
```

### Open5GS
Follow the [Open5GS installation guide](https://open5gs.org/open5gs/docs/guide/02-building-open5gs-from-sources/)

## Examples

### Single UE Registration Test
```bash
sudo ./bin/emulator -c config/single-ue.yml
```

### Load Testing (1000 UEs)
```bash
# Edit config.yml to set nUEs: 1000
sudo ./bin/emulator -c config/load-test.yml
```

### Multi Scenarios Testing
```bash
# Edit config.yml to set multi group in scenarios:
sudo ./bin/emulator -c config/multi-scenarios.yml
```

### Handover Testing
```bash
# Configure multiple gNBs and handover events
sudo ./bin/emulator -c config/handover.yml
```

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests for any improvements.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Authors

- [Vũ Dũng](https://github.com/lvdund)
- [Phùng Kiều Hà](mailto:ha.phungthikieu@hust.edu.vn)  
- [Thái Quang Tùng](https://github.com/reogac)

## Contact

For issues, questions, or contributions, please create a GitHub issue or contact: lvdund@gmail.com
