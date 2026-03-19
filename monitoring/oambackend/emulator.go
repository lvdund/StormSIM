package oambackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/reogac/utils/oam"
	"github.com/urfave/cli/v3"
)

type EmuHandler struct {
	api         Api
	nextContext *oam.HandlerContext //next context
}

var EmuCmds map[string]cli.Command = map[string]cli.Command{
	"list-ue": {
		Name:                  "list-ue",
		Usage:                 "List Ue contexts",
		Description:           "List all current UeContexts",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "level",
				Usage: "Get log by level (info, warn, error)",
			},
			&cli.StringFlag{
				Name:  "state",
				Usage: "Filter UEs by MM state (Registered, RegisteredInitiated, Deregistered, DeregistrationInitiated, AuthenticationInitiated)",
			},
			&cli.StringFlag{
				Name:  "not-state",
				Usage: "Filter UEs NOT in MM state (Registered, RegisteredInitiated, Deregistered, DeregistrationInitiated, AuthenticationInitiated)",
			},
			&cli.IntFlag{
				Name:  "last",
				Usage: "Show last N failed UEs (0 = show all)",
				Value: 0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*EmuHandler)
			level := cmd.String("level")
			state := cmd.String("state")
			notState := cmd.String("not-state")
			last := cmd.Int("last")
			if ues := h.api.RemoteGetListUes(level, state, notState, last); len(ues) == 0 {
				return fmt.Errorf("No UE contexts found")
			} else {
				fmt.Fprintf(cmd.Writer, "msin:")
				for _, msin := range ues {
					fmt.Fprintf(cmd.Writer, " - %s\n", msin)
				}
			}
			return nil
		},
	},
	"select-ue": {
		Name:                  "select-ue",
		Usage:                 "Move into an Ue context",
		Description:           "Move into an Ue context",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "msin",
				Usage:    "Emulator generated msin",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*EmuHandler)
			msin := cmd.String("msin")
			if srvCtx, err := h.buildUeContext(msin); err != nil {
				return err
			} else {
				h.nextContext = srvCtx
			}
			return nil
		},
	},

	"list-gnb": {
		Name:                  "list-gnb",
		Usage:                 "List all GnBs",
		Description:           "Show information from all emulated GnBs",
		EnableShellCompletion: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*EmuHandler)
			if gnbs := h.api.RemoteGetListGnbs(); len(gnbs) == 0 {
				return fmt.Errorf("No GnB found")
			} else {
				info, _ := json.MarshalIndent(gnbs, "", "  ")
				fmt.Fprintf(cmd.Writer, "%s\n", string(info))
			}
			return nil
		},
	},
	"select-gnb": {
		Name:                  "select-gnb",
		Usage:                 "Move into a gnb context",
		Description:           "Move into a gnb context",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "gnbId",
				Usage:    "Emulator generated gnbId",
				Required: true,
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*EmuHandler)
			gnbId := cmd.String("gnbId")
			if srvCtx, err := h.buildGnbContext(gnbId); err != nil {
				return err
			} else {
				h.nextContext = srvCtx
			}
			return nil
		},
	},
	"stats": statsCommand(),
	"group-delay-stats": {
		Name:                  "group-delay-stats",
		Usage:                 "Show aggregated NAS delay statistics across all UEs",
		Description:           "Display mean NAS request-response delay computed across all UEs",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "procedure",
				Usage: "Show procedure-level statistics",
			},
			&cli.BoolFlag{
				Name:  "nas",
				Usage: "Show NAS pair statistics",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*EmuHandler)
			stats := h.api.RemoteGetAllUeDelayStats()

			showProc := cmd.Bool("procedure")
			showNas := cmd.Bool("nas")

			if stats.UeCount == 0 {
				return fmt.Errorf("No delay measurements recorded for any UE")
			}

			f := NewFormatter(cmd.Writer)

			if showProc {
				if len(stats.Procedures) == 0 {
					return fmt.Errorf("No procedure statistics available")
				}

				keys := make([]string, 0, len(stats.Procedures))
				for k := range stats.Procedures {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				// User wants simpler output: "procedure {mean time}"
				// We can use RenderTable but with no header or simple header?
				// Example:
				// registration       123.456

				// Using RenderTable with headers
				headers := []string{"Procedure", "Min(ms)", "Max(ms)", "Mean(ms)", "Std Dev(ms)"}
				rows := make([][]string, 0, len(keys))
				for _, k := range keys {
					p := stats.Procedures[k]
					rows = append(rows, []string{
						k,
						fmt.Sprintf("%.3f", p.Min),
						fmt.Sprintf("%.3f", p.Max),
						fmt.Sprintf("%.3f", p.Mean),
						fmt.Sprintf("%.3f", p.StdDev),
					})
				}
				f.RenderTable("=== Group Procedure Statistics ===", headers, rows)
				return nil
			} else if showNas {
				if len(stats.NasPairs) == 0 {
					return fmt.Errorf("No NAS pair statistics available")
				}

				// Sort by request type for consistency
				sort.Slice(stats.NasPairs, func(i, j int) bool {
					if stats.NasPairs[i].Request == stats.NasPairs[j].Request {
						return stats.NasPairs[i].Response < stats.NasPairs[j].Response
					}
					return stats.NasPairs[i].Request < stats.NasPairs[j].Request
				})

				headers := []string{"Request", "Response", "Min(ms)", "Max(ms)", "Mean(ms)", "Std Dev(ms)"}
				rows := make([][]string, 0, len(stats.NasPairs))
				for _, p := range stats.NasPairs {
					rows = append(rows, []string{
						p.Request,
						p.Response,
						fmt.Sprintf("%.3f", p.Min),
						fmt.Sprintf("%.3f", p.Max),
						fmt.Sprintf("%.3f", p.Mean),
						fmt.Sprintf("%.3f", p.StdDev),
					})
				}
				f.RenderTable("=== Group NAS Pair Statistics ===", headers, rows)
				return nil
			}

			fmt.Fprintf(cmd.Writer, "=== Group NAS Delay Statistics ===\n")
			fmt.Fprintf(cmd.Writer, "UEs with measurements: %d\n", stats.UeCount)
			fmt.Fprintf(cmd.Writer, "Total measurements:    %d\n", stats.Count)
			fmt.Fprintf(cmd.Writer, "Mean delay:            %.3f ms\n", stats.Mean)
			return nil
		},
	},
}

func (b *EmuHandler) NextContext() *oam.HandlerContext {
	return b.nextContext
}

// build context for Ue
func (b *EmuHandler) buildUeContext(msin string) (ctx *oam.HandlerContext, err error) {
	//Find UE first
	if api := b.api.RemoteGetUeApi(msin); api == nil {
		err = fmt.Errorf("UE with id=%s not found", msin)
	} else {
		//then build context
		ctxId := fmt.Sprintf("%s:%s", UE_CTX_PREFIX, msin)
		ctx = oam.NewHandlerContext(ctxId, &UeHandler{
			api:    api,
			emuApi: b.api,
		}, UeCmds, nil)
	}
	return
}

// build context for Gnb
func (b *EmuHandler) buildGnbContext(gnbId string) (ctx *oam.HandlerContext, err error) {
	//Find Gnb first
	if api := b.api.RemoteGetGnbApi(gnbId); api == nil {
		err = fmt.Errorf("Gnb with id=%s  not found", gnbId)
	} else {
		//then build context
		ctxId := fmt.Sprintf("%s:%s", GNB_CTX_PREFIX, gnbId)
		ctx = oam.NewHandlerContext(ctxId, &GnbHandler{
			api:    api,
			emuApi: b.api,
		}, GnbCmds, nil)

	}
	return
}
