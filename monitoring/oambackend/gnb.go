package oambackend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"stormsim/internal/common/logger"
	"strings"
	"time"

	"github.com/reogac/utils/oam"
	"github.com/urfave/cli/v3"
)

type GnbApi interface {
	RemoteGnbInfo() GnbInfo
	RemoteListAmf() []AmfInfo
	RemoteCountUes() int
	RemoteListUeCtxs() []string
	RemoteReleaseUe(msin string) bool
	RemoteGnbLogs(last int, level string) []logger.LogEntry
	RemoteGnbDelayLogs(last int) []NasDelayEntry
	RemoteGnbDelayStats() DelayStats
}

type GnbHandler struct {
	api         GnbApi
	emuApi      Api
	nextContext *oam.HandlerContext
}

var GnbCmds map[string]cli.Command = map[string]cli.Command{
	"info": {
		Name:                  "info",
		Usage:                 "Show gnb info",
		Description:           "Show current GnbContext",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			gnb := h.api.RemoteGnbInfo()
			if isAllFieldsEmpty(gnb) {
				return fmt.Errorf("Cannot load gnb context info")
			}
			headers := []string{"Key", "Value"}
			rows := [][]string{
				{"Name", gnb.Name},
				{"Plmn", gnb.Plmn},
				{"Snssai", gnb.Snssai},
				{"NgapAddr", gnb.NgapAddr},
				{"GtpAddr", gnb.GtpAddr},
			}

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "gnb", "info")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Info written to %s\n", fullPath)
			} else {
				f.RenderTable("=== GNB Context Info ===", headers, rows)
			}
			return nil
		},
	},
	"list-amf": {
		Name:                  "list-amf",
		Usage:                 "List Amf contexts",
		Description:           "List all current Amf of a the Gnb",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			amfs := h.api.RemoteListAmf()
			if len(amfs) == 0 {
				return fmt.Errorf("No Amf found")
			}
			headers := []string{"Name", "Address", "State", "PLMNs", "Slices"}
			rows := make([][]string, len(amfs))
			for i, amf := range amfs {
				rows[i] = []string{
					amf.Name,
					fmt.Sprintf("%s:%d", amf.Address.Ip, amf.Address.Port),
					amf.State,
					strings.Join(amf.PlmnSupport, ", "),
					strings.Join(amf.SliceSupport, ", "),
				}
			}

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "gnb", "list-amf")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "AMF list written to %s\n", fullPath)
			} else {
				f.RenderTable("=== AMF Contexts ===", headers, rows)
			}
			return nil
		},
	},

	"count-ue": {
		Name:                  "count-ue",
		Usage:                 "Count Ue contexts",
		Description:           "Count all current UeContexts of a the Gnb",
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
			h := ctx.Value("handler").(*GnbHandler)
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
				count := h.countUe()
				headers := []string{"Key", "Value"}
				rows := [][]string{
					{"Number of UE in gnb", fmt.Sprintf("%d", count)},
				}

				f := NewFormatter(cmd.Writer)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "gnb", "count-ue")
					if err := f.WriteCSV(fullPath, headers, rows); err != nil {
						return err
					}
					fmt.Fprintf(cmd.Writer, "UE count written to %s\n", fullPath)
				} else {
					f.RenderTable("=== UE Count ===", headers, rows)
				}
				return nil
			}

			if watch {
				return Watch(ctx, interval, cmd.Writer, renderFunc)
			}
			return renderFunc()
		},
	},
	"list-ue": {
		Name:                  "list-ue",
		Usage:                 "List Ue contexts",
		Description:           "List all current UeContexts a the Gnb",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			ues := h.listUeContexts()
			if len(ues) == 0 {
				return fmt.Errorf("No UE contexts found")
			}
			headers := []string{"MSIN"}
			rows := make([][]string, len(ues))
			for i, msin := range ues {
				rows[i] = []string{msin}
			}

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "gnb", "list-ue")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "UE list written to %s\n", fullPath)
			} else {
				f.RenderTable("=== UE Contexts ===", headers, rows)
			}
			return nil
		},
	},
	"release-ue": {
		Name:                  "release-ue",
		Usage:                 "Release connection to UE",
		Description:           "Release connetion to UE",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "msin",
				Usage:    "Ue identity at gnB",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			msin := cmd.String("msin")
			if !h.releaseUe(msin) {
				return fmt.Errorf("Fail to trigger ue release")
			} else {
				fmt.Fprintf(cmd.Writer, "Ue release triggered\n")
			}
			return nil
		},
	},
	"logs": {
		Name:                  "logs",
		Usage:                 "Show logs for gNB",
		Description:           "Display buffered logs for this gNB",
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
			h := ctx.Value("handler").(*GnbHandler)
			last := cmd.Int("last")
			level := cmd.String("level")
			filePath := cmd.String("file")
			isJson := cmd.Bool("json")

			logs := h.api.RemoteGnbLogs(last, level)
			if len(logs) == 0 {
				return fmt.Errorf("No logs found")
			}

			if isJson {
				data, _ := json.MarshalIndent(logs, "", "  ")
				output := string(data)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "gnb", "logs")
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
				fullPath := ResolveFilename(filePath, "gnb", "logs")
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
				fmt.Fprintf(cmd.Writer, "=== Logs for gNB (%d entries) ===\n", len(logs))
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
		Usage:                 "Show NGAP delay timer logs for gNB",
		Description:           "Display NGAP request-response delay measurements for this gNB",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "last",
				Usage: "Show last N delay entries",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file instead of stdout",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			last := cmd.Int("last")

			logs := h.api.RemoteGnbDelayLogs(last)
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

			filePath := cmd.String("file")
			f := NewFormatter(cmd.Writer)
			if filePath != "" {
				fullPath := ResolveFilename(filePath, "gnb", "delay-logs")
				if err := f.WriteCSV(fullPath, headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(cmd.Writer, "Delay logs written to %s\n", fullPath)
			} else {
				f.RenderTable("=== NGAP Delay Logs ===", headers, rows)
			}
			return nil
		},
	},
	"delay-stats": {
		Name:                  "delay-stats",
		Usage:                 "Show NGAP delay statistics for gNB",
		Description:           "Display aggregated NGAP delay statistics (min, max, mean) for this gNB",
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
			h := ctx.Value("handler").(*GnbHandler)
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
				stats := h.api.RemoteGnbDelayStats()

				if stats.Count == 0 {
					return fmt.Errorf("No delay measurements recorded")
				}

				headers := []string{"Key", "Value"}
				rows := [][]string{
					{"Measurements", fmt.Sprintf("%d", stats.Count)},
					{"Min (ms)", fmt.Sprintf("%.3f", stats.Min)},
					{"Max (ms)", fmt.Sprintf("%.3f", stats.Max)},
					{"Mean (ms)", fmt.Sprintf("%.3f", stats.Mean)},
					{"Std Dev (ms)", fmt.Sprintf("%.3f", stats.StdDev)},
				}

				f := NewFormatter(cmd.Writer)
				if filePath != "" {
					fullPath := ResolveFilename(filePath, "gnb", "delay-stats")
					if err := f.WriteCSV(fullPath, headers, rows); err != nil {
						return err
					}
					fmt.Fprintf(cmd.Writer, "Delay stats written to %s\n", fullPath)
				} else {
					f.RenderTable("=== NGAP Delay Statistics ===", headers, rows)
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
	GnbCmds["exit"] = cli.Command{
		Name:  "exit",
		Usage: "Return to main menu",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GnbHandler)
			// Reconstruct main context (emulator context)
			h.nextContext = oam.NewHandlerContext(STORMSIM_CTX_ID, &EmuHandler{api: h.emuApi}, EmuCmds, nil)
			return nil
		},
	}
}

func (h GnbHandler) countUe() int {
	return h.api.RemoteCountUes()
}
func (h GnbHandler) releaseUe(msin string) bool {
	return h.api.RemoteReleaseUe(msin)
}

func (h GnbHandler) listUeContexts() []string {
	return h.api.RemoteListUeCtxs()
}

func (h *GnbHandler) NextContext() *oam.HandlerContext {
	return h.nextContext
}

func isAllFieldsEmpty(s any) bool {
	val := reflect.ValueOf(s)
	for i := 0; i < val.NumField(); i++ {
		if str, ok := val.Field(i).Interface().(string); ok {
			if str != "" {
				return false
			}
		}
	}
	return true
}
