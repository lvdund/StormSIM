import psutil
import time
from datetime import datetime
from pathlib import Path

def main():
    # Ensure logs directory exists
    log_dir = Path("logs")
    log_dir.mkdir(exist_ok=True)

    # Create log filename with date & time
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    logfile = log_dir / f"data-{timestamp}.log"

    with open(logfile, "w") as f:
        try:
            while True:
                # Current time
                now = datetime.now().strftime("%H:%M:%S")

                # CPU usage per core
                cores = psutil.cpu_percent(interval=1, percpu=True)

                # RAM usage (%)
                ram = psutil.virtual_memory().percent

                # Build log line
                line = f"{now} - {', '.join(f'{c:.1f}' for c in cores)}, {ram:.1f}"

                # Print to terminal
                print(line)

                # Write to file
                f.write(line + "\n")
                f.flush()
        except KeyboardInterrupt:
            print(f"\nStopped logging. Data saved to {logfile}")

if __name__ == "__main__":
    main()
