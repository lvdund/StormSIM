package oambackend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"stormsim/internal/common/logger"
	"time"

	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/oam"
	"github.com/urfave/cli/v3"
)

type UeApi interface {
	RemoteUeInfo() UeContextInfo
	RemoteUeStats() []string
	RemoteUeSessionInfo() []SessionInfo
	RemoteCreateSession(dnn string, slice models.Snssai) bool
	RemoteUeLogs(last int, level string) []logger.LogEntry
	RemoteUeDelayLogs(last int, protocol string) []NasDelayEntry
	RemoteUeDelayStats() DelayStats
}

type UeHandler struct {
	api         UeApi
	emuApi      Api
	nextContext *oam.HandlerContext
}

var UeCmds map[string]cli.Command = map[string]cli.Command{
	"info": {
		Name:                  "info",
		Usage:                 "Show ue info",
		Description:           "Show current UeContext Info",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			ue := h.api.RemoteUeInfo()
			if isAllFieldsEmpty(ue) {
				return fmt.Errorf("Cannot load ue context info")
			}
			headers := []string{"Key", "Value"}
			rows := [][]string{
				{"Name", ue.Name},
				{"Gnb", ue.Gnb},
				{"Hplmn", ue.Hplmn},
				{"Snssai", ue.Snssai},
				{"RanNgapId", fmt.Sprintf("%d", ue.RanNgapId)},
				{"AmfNgapId", fmt.Sprintf("%d", ue.AmfNgapId)},
				{"MMstate", ue.MMstate},
				{"ActiveSessions", fmt.Sprintf("%d", ue.ActiveSessions)},
			}

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "ue", "info")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Info written to %s\n", fullPath)
			} else {
				f.RenderTable("=== UE Context Info ===", headers, rows)
			}
			return nil
		},
	},
	"ssinfo": {
		Name:                  "ssinfo",
		Usage:                 "Show ue session info",
		Description:           "Show current Ue Sessions Info",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			ss := h.api.RemoteUeSessionInfo()
			if len(ss) == 0 {
				return fmt.Errorf("No session found")
			}
			headers := []string{"Id", "Dnn", "Snssai", "State", "Address"}
			rows := make([][]string, len(ss))
			for i, s := range ss {
				rows[i] = []string{
					fmt.Sprintf("%d", s.Id),
					s.Dnn,
					s.Snssai,
					s.SMstate,
					s.Address,
				}
			}

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "ue", "ssinfo")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Session info written to %s\n", fullPath)
			} else {
				f.RenderTable("=== UE Sessions Info ===", headers, rows)
			}
			return nil
		},
	},
	"stats": {
		Name:        "stats",
		Usage:       "Show stats of ue",
		Description: "Event/Message processing time, state of UE",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
			&cli.BoolFlag{
				Name:    "watch",
				Aliases: []string{"w"},
				Usage:   "Watch statistics in real-time",
			},
			&cli.StringFlag{
				Name:    "interval",
				Aliases: []string{"n"},
				Value:   "1s",
				Usage:   "Watch interval (e.g., 500ms, 2s)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			filePath := cmd.String("file")
			watch := cmd.Bool("watch")

			if watch && filePath != "" {
				return fmt.Errorf("cannot use --watch with --file")
			}

			interval, err := time.ParseDuration(cmd.String("interval"))
			if err != nil {
				return fmt.Errorf("invalid interval format: %w", err)
			}

			renderFunc := func() error {
				stats := h.api.RemoteUeStats()
				if len(stats) == 0 {
					return fmt.Errorf("No stats found")
				}
				headers := []string{"Stat"}
				rows := make([][]string, len(stats))
				for i, s := range stats {
					rows[i] = []string{s}
				}

				f := NewFormatter(cmd.Writer)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "ue", "stats")
					if err := f.WriteCSV(fullPath, headers, rows); err != nil {
						return err
					}
					fmt.Fprintf(cmd.Writer, "Stats written to %s\n", fullPath)
				} else {
					f.RenderTable("=== Event/Message processing time, state of UE ===", headers, rows)
				}
				return nil
			}

			if watch {
				return Watch(ctx, interval, cmd.Writer, renderFunc)
			}
			return renderFunc()
		},
	},
	"ps-create": {
		Name:                  "ps-create",
		Usage:                 "Establish a new Pdu session",
		Description:           "Estrablish a new Pdu session",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dnn",
				Usage:    "Data network name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "sd",
				Usage:    "Snssai' sd value",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "sst",
				Usage:    "Snssai' sst value",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			dnn := cmd.String("dnn")
			slice := models.Snssai{
				Sd:  cmd.String("sd"),
				Sst: cmd.Int("sst"),
			}
			if !h.psCreate(dnn, slice) {
				return fmt.Errorf("Fail to trigger session establishment")
			} else {
				fmt.Fprintf(cmd.Writer, "Session estalishment triggered\n")
			}
			return nil
		},
	},
	"logs": {
		Name:                  "logs",
		Usage:                 "Show logs for UE",
		Description:           "Display buffered logs for this UE",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "last",
				Usage: "Show last N log entries",
			},
			&cli.StringFlag{
				Name:  "level",
				Usage: "Filter by log level (INFO, WARN, ERROR, DEBUG)",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output in JSON format",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			last := cmd.Int("last")
			level := cmd.String("level")
			filePath := cmd.String("file")
			isJson := cmd.Bool("json")

			logs := h.api.RemoteUeLogs(last, level)
			if len(logs) == 0 {
				return fmt.Errorf("No logs found")
			}

			if isJson {
				data, _ := json.MarshalIndent(logs, "", "  ")
				output := string(data)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "ue", "logs")
					if err := os.WriteFile(fullPath, []byte(output), 0644); err != nil {
						return fmt.Errorf("failed to write to file: %w", err)
					}
					fmt.Fprintf(cmd.Writer, "Logs written to %s\n", fullPath)
				} else {
					fmt.Fprintf(cmd.Writer, "%s\n", output)
				}
				return nil
			}

			if filePath != "" {
				fullPath := ResolveFilename(filePath, "ue", "logs")
				headers := []string{"Timestamp", "Level", "State", "Message"}
				rows := make([][]string, len(logs))
				for i, entry := range logs {
					rows[i] = []string{
						entry.Timestamp.Format("15:04:05.000"),
						entry.Level,
						entry.State,
						entry.Message,
					}
				}
				f := NewFormatter(cmd.Writer)
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Logs written to %s\n", fullPath)
			} else {
				fmt.Fprintf(cmd.Writer, "=== Logs for UE (%d entries) ===\n", len(logs))
				for _, entry := range logs {
					fmt.Fprintf(cmd.Writer, "[%s] %s [%s] %s\n",
						entry.Timestamp.Format("15:04:05.000"),
						entry.Level, entry.State, entry.Message)
				}
			}
			return nil
		},
	},
	"delay-logs": {
		Name:                  "delay-logs",
		Usage:                 "Show NAS delay timer logs for UE",
		Description:           "Display NAS request-response delay measurements for this UE",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "last",
				Usage: "Show last N delay entries",
			},
			&cli.StringFlag{
				Name:  "protocol",
				Usage: "Filter by protocol (nas, ngap, rlink, all)",
				Value: "all",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			last := cmd.Int("last")
			protocol := cmd.String("protocol")
			filePath := cmd.String("file")

			logs := h.api.RemoteUeDelayLogs(last, protocol)
			if len(logs) == 0 {
				return fmt.Errorf("No delay logs found")
			}

			headers := []string{"Time", "Protocol", "Request", "Response", "Delay(ms)"}
			rows := make([][]string, len(logs))
			for i, entry := range logs {
				rows[i] = []string{
					entry.SendTime,
					entry.Protocol,
					entry.RequestType,
					entry.ResponseType,
					fmt.Sprintf("%.3f", entry.DelayMs),
				}
			}

			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "ue", "delay-logs")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Delay logs written to %s\n", fullPath)
			} else {
				f.RenderTable("=== NAS Delay Logs ===", headers, rows)
			}
			return nil
		},
	},
	"delay-stats": {
		Name:                  "delay-stats",
		Usage:                 "Show NAS delay statistics for UE",
		Description:           "Display aggregated NAS delay statistics (min, max, mean) for this UE",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
			&cli.BoolFlag{
				Name:    "watch",
				Aliases: []string{"w"},
				Usage:   "Watch statistics in real-time",
			},
			&cli.StringFlag{
				Name:    "interval",
				Aliases: []string{"n"},
				Value:   "1s",
				Usage:   "Watch interval (e.g., 500ms, 2s)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			filePath := cmd.String("file")
			watch := cmd.Bool("watch")

			if watch && filePath != "" {
				return fmt.Errorf("cannot use --watch with --file")
			}

			interval, err := time.ParseDuration(cmd.String("interval"))
			if err != nil {
				return fmt.Errorf("invalid interval format: %w", err)
			}

			renderFunc := func() error {
				stats := h.api.RemoteUeDelayStats()

				if len(stats.Procedures) == 0 {
					return fmt.Errorf("No procedure measurements recorded")
				}

				// User requested: ONLY procedure times, no aggregated stats
				// Format: procedure_name {time}
				headers := []string{"Procedure", "Time (ms)"}

				keys := make([]string, 0, len(stats.Procedures))
				for k := range stats.Procedures {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				rows := make([][]string, 0, len(keys))
				for _, k := range keys {
					p := stats.Procedures[k]
					// Show the Last duration as "time for procedure"
					rows = append(rows, []string{k, fmt.Sprintf("%.3f", p.Last)})
				}

				f := NewFormatter(cmd.Writer)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "ue", "delay-stats")
					if err := f.WriteCSV(fullPath, headers, rows); err != nil {
						return err
					}
					fmt.Fprintf(cmd.Writer, "Delay stats written to %s\n", fullPath)
				} else {
					f.RenderTable("=== NAS Delay Statistics ===", headers, rows)
				}
				return nil
			}

			if watch {
				return Watch(ctx, interval, cmd.Writer, renderFunc)
			}
			return renderFunc()
		},
	},
}

func init() {
	UeCmds["exit"] = cli.Command{
		Name:  "exit",
		Usage: "Return to main menu",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*UeHandler)
			// Reconstruct main context (emulator context)
			h.nextContext = oam.NewHandlerContext(STORMSIM_CTX_ID, &EmuHandler{api: h.emuApi}, EmuCmds, nil)
			return nil
		},
	}
}

func (h *UeHandler) psCreate(dnn string, slice models.Snssai) bool {
	return h.api.RemoteCreateSession(dnn, slice)
}

func (h *UeHandler) NextContext() *oam.HandlerContext {
	return h.nextContext
}
