#!/usr/bin/env python3

import json
import matplotlib.pyplot as plt
import numpy as np

with open("open5gs.json", "r") as f:
    open5gs_data = json.load(f)

with open("free5gc.json", "r") as f:
    free5gc_data = json.load(f)

plt.rcParams["font.size"] = 20
plt.rcParams["axes.labelsize"] = 24
plt.rcParams["axes.titlesize"] = 24
plt.rcParams["xtick.labelsize"] = 20
plt.rcParams["ytick.labelsize"] = 20
plt.rcParams["legend.fontsize"] = 20
plt.rcParams["figure.titlesize"] = 26


def filter_pairs(pairs):
    filtered = []
    for pair in pairs:
        if pair["response"] == "Unknown":
            continue
        if (
            pair["mean_ms"] == 0.0
            and pair["std_dev_ms"] == 0.0
            and pair["min_ms"] == 0.0
            and pair["max_ms"] == 0.0
        ):
            continue
        filtered.append(pair)
    return filtered


def extract_nas_data(data, pair_name):
    delays = []
    means = []
    stds = []

    for group in data["group_nas_pair_statistics"]:
        delay = group["delay_ms"]
        filtered_pairs = filter_pairs(group["pairs"])

        for pair in filtered_pairs:
            pair_id = f"{pair['request']}->{pair['response']}"
            if pair_id == pair_name:
                delays.append(delay)
                means.append(pair["mean_ms"])
                stds.append(pair["std_dev_ms"])
                break

    return np.array(delays), np.array(means), np.array(stds)


def extract_procedure_data(data):
    delays = []
    means = []
    stds = []

    for group in data["group_procedure_statistics"]:
        delays.append(group["delay_ms"])
        means.append(group["mean_ms"])
        stds.append(group["std_dev_ms"])

    return np.array(delays), np.array(means), np.array(stds)


nas_pairs = [
    "AuthenticationResponse->SecurityModeCommand",
    "RegistrationRequest->AuthenticationRequest",
    "SecurityModeComplete->RegistrationAccept",
    "RegistrationComplete->ConfigurationUpdateCommand",
]

fig, axes = plt.subplots(2, 2, figsize=(20, 16))
axes = axes.flatten()

for idx, pair_name in enumerate(nas_pairs):
    ax = axes[idx]

    o5gs_delays, o5gs_means, o5gs_stds = extract_nas_data(open5gs_data, pair_name)
    f5gc_delays, f5gc_means, f5gc_stds = extract_nas_data(free5gc_data, pair_name)

    if len(o5gs_delays) > 0:
        ax.errorbar(
            o5gs_delays,
            o5gs_means,
            yerr=o5gs_stds,
            fmt="o",
            markersize=12,
            capsize=5,
            capthick=2,
            color="blue",
            ecolor="blue",
            linewidth=2,
            label="Open5GS",
        )

    if len(f5gc_delays) > 0:
        ax.errorbar(
            f5gc_delays,
            f5gc_means,
            yerr=f5gc_stds,
            fmt="s",
            markersize=12,
            capsize=5,
            capthick=2,
            color="red",
            ecolor="red",
            linewidth=2,
            label="Free5GC",
        )

    pair_label = pair_name.replace("->", " → ")
    ax.set_title(pair_label, fontsize=18)
    ax.set_xlabel("Delay (ms)", fontsize=18)
    ax.set_ylabel("Time (ms)", fontsize=18)
    ax.grid(True, alpha=0.3)
    ax.legend(loc="best", fontsize=16)

plt.tight_layout()
plt.savefig("nas_comparison.png", dpi=150, bbox_inches="tight")
print("Saved: nas_comparison.png")

fig, ax = plt.subplots(figsize=(12, 9))

o5gs_delays, o5gs_means, o5gs_stds = extract_procedure_data(open5gs_data)
f5gc_delays, f5gc_means, f5gc_stds = extract_procedure_data(free5gc_data)

ax.errorbar(
    o5gs_delays,
    o5gs_means,
    yerr=o5gs_stds,
    fmt="o",
    markersize=14,
    capsize=6,
    capthick=2.5,
    color="blue",
    ecolor="blue",
    linewidth=2.5,
    label="Open5GS",
)

ax.errorbar(
    f5gc_delays,
    f5gc_means,
    yerr=f5gc_stds,
    fmt="s",
    markersize=14,
    capsize=6,
    capthick=2.5,
    color="red",
    ecolor="red",
    linewidth=2.5,
    label="Free5GC",
)

ax.set_xlabel("Delay (ms)", fontsize=24)
ax.set_ylabel("Excuse Time (ms)", fontsize=24)
# ax.set_ylabel("Registration Time (ms)", fontsize=24)
# ax.set_title("Registration Procedure Comparison", fontsize=26)
ax.grid(True, alpha=0.3)
ax.legend(loc="upper right", fontsize=22)

plt.tight_layout()
plt.savefig("procedure_comparison.png", dpi=150, bbox_inches="tight")
print("Saved: procedure_comparison.png")

plt.show()
