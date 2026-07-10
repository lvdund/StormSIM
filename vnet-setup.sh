# =============================================================================
# Virtual Network Setup: 2 namespaced hosts + NAT to main PC
#
# Topology:
#   [ns-host1: 10.0.0.10/24] ──┐
#                              ├── [br-vnet: 10.0.0.1/24] ── NAT ── internet
#   [ns-host2: 10.0.0.11/24] ──┘
#
# Requirements: iproute2, iptables (run as root)
# =============================================================================
# How to run:
#
# sudo ./vnet-setup.sh up        # bring up
# sudo ./vnet-setup.sh down      # tear down
# sudo ./vnet-setup.sh restart   # reset
# sudo ./vnet-setup.sh status    # inspect state
# =============================================================================

set -euo pipefail

### CONFIG ###
BRIDGE="br-vnet"
BR_IP="10.0.0.1/24"
NS1="5g"
NS2="ue"
IP1="10.0.0.10/24"
IP2="10.0.0.11/24"
GW="10.0.0.1"

# Auto-detect main NIC (the one with a default route)
MAIN_IF=$(ip route show default | awk '/default/ {print $5}' | head -n1)

# ─────────────────────────────────────────────
teardown() {
    echo "[*] Tearing down existing vnet..."
    ip netns del "$NS1" 2>/dev/null || true
    ip netns del "$NS2" 2>/dev/null || true
    ip link del "$BRIDGE" 2>/dev/null || true
    ip link del "veth1" 2>/dev/null || true
    ip link del "veth2" 2>/dev/null || true
    # Remove NAT rule if it exists
    iptables -t nat -D POSTROUTING -s 10.0.0.0/24 -o "$MAIN_IF" -j MASQUERADE 2>/dev/null || true
    iptables -D FORWARD -i "$BRIDGE" -o "$MAIN_IF" -j ACCEPT 2>/dev/null || true
    iptables -D FORWARD -i "$MAIN_IF" -o "$BRIDGE" -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
    echo "[✓] Teardown complete."
}

setup() {
    echo "[*] Detected main interface: $MAIN_IF"

    # ── 1. Bridge ──────────────────────────────
    echo "[*] Creating bridge $BRIDGE..."
    ip link add name "$BRIDGE" type bridge
    ip link set "$BRIDGE" up
    ip addr add "$BR_IP" dev "$BRIDGE"

    # ── 2. Namespaces ──────────────────────────
    for ns in "$NS1" "$NS2"; do
        echo "[*] Creating network namespace: $ns"
        ip netns add "$ns"
        # Loopback inside namespace
        ip netns exec "$ns" ip link set lo up
    done

    # ── 3. veth pairs → bridge ─────────────────
    setup_veth() {
        local ns=$1 veth_host=$2 veth_ns=$3 ns_ip=$4
        echo "[*] Wiring $ns ($ns_ip) via $veth_host <-> $veth_ns..."
        ip link add "$veth_host" type veth peer name "$veth_ns"
        ip link set "$veth_host" master "$BRIDGE"
        ip link set "$veth_host" up
        ip link set "$veth_ns" netns "$ns"
        ip netns exec "$ns" ip link set "$veth_ns" up
        ip netns exec "$ns" ip addr add "$ns_ip" dev "$veth_ns"
        ip netns exec "$ns" ip route add default via "$GW"
    }

    setup_veth "$NS1" "veth1" "eth0" "$IP1"
    setup_veth "$NS2" "veth2" "eth0" "$IP2"

    # ── 4. IP forwarding ───────────────────────
    echo "[*] Enabling IP forwarding..."
    sysctl -qw net.ipv4.ip_forward=1

    # ── 5. NAT (masquerade) ────────────────────
    echo "[*] Adding NAT rules (MASQUERADE via $MAIN_IF)..."
    iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o "$MAIN_IF" -j MASQUERADE
    iptables -A FORWARD -i "$BRIDGE" -o "$MAIN_IF" -j ACCEPT
    iptables -A FORWARD -i "$MAIN_IF" -o "$BRIDGE" -m state --state RELATED,ESTABLISHED -j ACCEPT

    # ── 6. DNS inside namespaces ───────────────
    for ns in "$NS1" "$NS2"; do
        mkdir -p /etc/netns/"$ns"
        echo "nameserver 1.1.1.1" >/etc/netns/"$ns"/resolv.conf
    done

    echo ""
    echo "══════════════════════════════════════════"
    echo " Virtual network is UP"
    echo "══════════════════════════════════════════"
    echo "  Bridge : $BRIDGE  ($BR_IP)"
    echo "  $NS1  : 10.0.0.10  (veth1)"
    echo "  $NS2  : 10.0.0.11  (veth2)"
    echo "  NAT via: $MAIN_IF"
    echo ""
    echo " Usage examples:"
    echo "  sudo ip netns exec $NS1 ping 10.0.0.11       # host1 → host2"
    echo "  sudo ip netns exec $NS2 ping 10.0.0.10       # host2 → host1"
    echo "  sudo ip netns exec $NS1 ping 1.1.1.1         # host1 → internet (NAT)"
    echo "  sudo ip netns exec $NS1 bash                 # shell in host1"
    echo "══════════════════════════════════════════"
}

# ─────────────────────────────────────────────
case "${1:-up}" in
ue)
    sudo ip netns exec ue ./bin/emulator -c ./config/free5gc_1ue.yml
    ;;
client)
    sudo ip netns exec ue ./bin/client
    ;;
up)
    teardown # clean previous state first
    setup
    ;;
down)
    teardown
    ;;
restart)
    teardown
    setup
    ;;
status)
    echo "=== Namespaces ==="
    ip netns list
    echo "=== Bridge ==="
    ip addr show "$BRIDGE" 2>/dev/null || echo "(bridge not found)"
    echo "=== ns-host1 ==="
    ip netns exec "$NS1" ip addr 2>/dev/null || echo "(not running)"
    echo "=== ns-host2 ==="
    ip netns exec "$NS2" ip addr 2>/dev/null || echo "(not running)"
    echo "=== NAT rules ==="
    iptables -t nat -L POSTROUTING -n --line-numbers | grep 10.0.0 || echo "(no NAT rules)"
    ;;
*)
    echo "Usage: $0 [up|down|restart|status]"
    exit 1
    ;;
esac
