package oambackend

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"stormsim/internal/common/stats"
	"time"

	"github.com/urfave/cli/v3"
)

func statsCommand() cli.Command {
	return cli.Command{
		Name:                  "stats",
		Usage:                 "Show procedure statistics",
		Description:           "Display current or historical statistics with various formats",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output in JSON format",
			},
			&cli.BoolFlag{
				Name:  "csv",
				Usage: "Output in CSV format",
			},
			&cli.BoolFlag{
				Name:  "history",
				Usage: "Show historical statistics",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "Show snapshots since duration (e.g., 5m, 1h)",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Output to file (use 'auto' for automatic filename)",
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
			useHistory := cmd.Bool("history") || cmd.String("since") != ""
			useJson := cmd.Bool("json")
			useCsv := cmd.Bool("csv")
			filePath := cmd.String("file")
			watch := cmd.Bool("watch")

			if watch && filePath != "" {
				return fmt.Errorf("cannot use --watch with --file")
			}

			interval, err := time.ParseDuration(cmd.String("interval"))
			if err != nil {
				return fmt.Errorf("invalid interval format: %w", err)
			}

			if filePath != "" {
				filePath = ResolveFilename(filePath, "stormsim", "stats")
			}

			renderFunc := func() error {
				var data interface{}
				var headers []string
				var rows [][]string

				if useHistory {
					var snapshots []stats.HistoricalSnapshot
					if sinceStr := cmd.String("since"); sinceStr != "" {
						duration, err := time.ParseDuration(sinceStr)
						if err != nil {
							return fmt.Errorf("invalid duration format: %w", err)
						}
						snapshots = stats.GlobalHistory.GetSnapshotsSince(time.Now().Add(-duration))
					} else {
						snapshots = stats.GlobalHistory.GetAllSnapshots()
					}
					data = snapshots

					headers = []string{"Timestamp", "Procedure", "Running", "Completed", "Failed"}
					for _, snap := range snapshots {
						ts := snap.Timestamp.Format(time.RFC3339)
						for proc, stat := range snap.Stats {
							rows = append(rows, []string{
								ts,
								string(proc),
								fmt.Sprintf("%d", stat.Running),
								fmt.Sprintf("%d", stat.Completed),
								fmt.Sprintf("%d", stat.Failed),
							})
						}
					}
				} else {
					snapshot := stats.GlobalStats.GetSnapshot()
					data = struct {
						Timestamp  time.Time                                  `json:"timestamp"`
						Procedures map[stats.ProcedureType]stats.StatSnapshot `json:"procedures"`
					}{
						Timestamp:  time.Now(),
						Procedures: snapshot,
					}

					headers = []string{"Procedure", "Running", "Completed", "Failed"}
					for proc, stat := range snapshot {
						rows = append(rows, []string{
							string(proc),
							fmt.Sprintf("%d", stat.Running),
							fmt.Sprintf("%d", stat.Completed),
							fmt.Sprintf("%d", stat.Failed),
						})
					}
				}

				if useJson {
					jsonData, err := json.MarshalIndent(data, "", "  ")
					if err != nil {
						return fmt.Errorf("failed to marshal JSON: %w", err)
					}
					if filePath != "" {
						if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Fprintf(cmd.Writer, "Stats written to %s\n", filePath)
					} else {
						fmt.Fprintf(cmd.Writer, "%s\n", string(jsonData))
					}
					return nil
				}

				formatter := NewFormatter(cmd.Writer)
				if filePath != "" {
					if err := formatter.WriteCSV(filePath, headers, rows); err != nil {
						return err
					}
					fmt.Fprintf(cmd.Writer, "Stats written to %s\n", filePath)
				} else if useCsv {
					w := csv.NewWriter(cmd.Writer)
					if err := w.Write(headers); err != nil {
						return err
					}
					if err := w.WriteAll(rows); err != nil {
						return err
					}
					w.Flush()
				} else {
					title := "=== Procedure Statistics ==="
					if useHistory {
						title = "=== Historical Procedure Statistics ==="
					}
					formatter.RenderTable(title, headers, rows)
				}

				return nil
			}

			if watch {
				return Watch(ctx, interval, cmd.Writer, renderFunc)
			}
			return renderFunc()
		},
	}
}
