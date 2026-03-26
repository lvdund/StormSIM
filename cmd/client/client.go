package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/urfave/cli/v3"

	"stormsim/monitoring/oambackend"
)

// Request/Response types matching server
type CmdRequest struct {
	ContextId string   `json:"ContextId"`
	Name      string   `json:"Name"`
	Args      []string `json:"Args"`
}

type ContextInfo struct {
	Id       string        `json:"Id"`
	Prompt   string        `json:"Prompt"`
	Commands []CommandInfo `json:"Commands"`
}

type CommandInfo struct {
	Name        string     `json:"name"`
	Usage       string     `json:"usage"`
	Description string     `json:"description"`
	Flags       []FlagInfo `json:"flags"`
}

type FlagInfo struct {
	Name  string `json:"name"`
	Usage string `json:"usage"`
}

type CmdResponse struct {
	Message string       `json:"Message"`
	Error   string       `json:"Error"`
	Context *ContextInfo `json:"Context"`
}

func main() {
	// Single command mode (e.g. ./client stats -w)
	if len(os.Args) > 1 {
		handleSingleCommand(os.Args[1:])
		return
	}

	runInteractive()
}

func runInteractive() {
	shell := ishell.New()

	ctxId := "stormsim"
	var availableCommands []CommandInfo = convertCommands(oambackend.EmuCmds)

	shell.SetPrompt("stormsim> ")

	// Handle 'exit' explicitly
	shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			if ctxId == "stormsim" {
				shell.Stop()
			} else {
				// Send "exit" command to server
				executeCommand(c, &ctxId, []string{"exit"}, shell, &availableCommands)
			}
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "quit",
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			if ctxId == "stormsim" {
				shell.Stop()
			} else {
				// Send "exit" command to server
				executeCommand(c, &ctxId, []string{"exit"}, shell, &availableCommands)
			}
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "Clear screen",
		Func: func(c *ishell.Context) {
			c.ClearScreen()
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "List available commands",
		Func: func(c *ishell.Context) {
			if len(availableCommands) == 0 {
				c.Println("No commands available. Try running a command to fetch context.")
				return
			}
			c.Println("Available Commands:")
			for _, cmd := range availableCommands {
				c.Printf("  %-15s %s\n", cmd.Name, cmd.Description)
				if cmd.Usage != "" {
					c.Printf("                  Usage: %s\n", cmd.Usage)
				}
			}
		},
	})

	// Catch-all for dynamic commands
	shell.NotFound(func(c *ishell.Context) {
		if len(c.Args) == 0 {
			return
		}

		// In NotFound, c.Args contains the full command line split by space
		executeCommand(c, &ctxId, c.Args, shell, &availableCommands)
	})

	shell.Run()
}

func handleSingleCommand(args []string) {
	// Simple wrapper to run one command and exit
	ctxId := "stormsim"
	var availableCommands []CommandInfo = convertCommands(oambackend.EmuCmds)

	// Create a dummy context/printer
	// We can't use ishell context easily here without starting shell.
	// So we implement logic directly.
	executeCommandLogic(nil, &ctxId, args, nil, &availableCommands)
}

// Wrapper to handle both ishell and standard output
func executeCommand(c *ishell.Context, ctxId *string, args []string, shell *ishell.Shell, commands *[]CommandInfo) {
	executeCommandLogic(c, ctxId, args, shell, commands)
}

func executeCommandLogic(c *ishell.Context, ctxId *string, args []string, shell *ishell.Shell, commands *[]CommandInfo) {
	watch := false
	interval := 1 * time.Second

	cleanArgs := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--watch" || arg == "-w" {
			watch = true
			continue
		}
		if arg == "--interval" || arg == "-n" {
			if i+1 < len(args) {
				if d, err := time.ParseDuration(args[i+1]); err == nil {
					interval = d
					i++ // skip value
					continue
				}
			}
		}
		cleanArgs = append(cleanArgs, arg)
	}

	if len(cleanArgs) == 0 {
		return
	}

	cmdName := cleanArgs[0]
	cmdArgs := cleanArgs[1:]

	if watch {
		// Setup signal handling for the loop
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		defer signal.Stop(sigChan)

	loop:
		for {
			// Clear screen (Move to home)
			if c != nil {
				c.Print("\033[H")
			} else {
				fmt.Print("\033[H")
			}

			err := sendRequest(c, ctxId, cmdName, cmdArgs, shell, commands)

			// Clear rest of screen
			if c != nil {
				c.Print("\033[J")
			} else {
				fmt.Print("\033[J")
			}

			if err != nil {
				printError(c, err)
				break
			}

			select {
			case <-sigChan:
				if c != nil {
					c.ClearScreen()
				} else {
					fmt.Print("\033[H\033[2J")
				}
				break loop
			case <-time.After(interval):
			}
		}
	} else {
		err := sendRequest(c, ctxId, cmdName, cmdArgs, shell, commands)
		if err != nil {
			printError(c, err)
		}
	}
}

func sendRequest(c *ishell.Context, ctxId *string, name string, args []string, shell *ishell.Shell, commands *[]CommandInfo) error {
	req := CmdRequest{
		ContextId: *ctxId,
		Name:      name,
		Args:      args,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Assuming default port 4000
	resp, err := http.Post("http://localhost:4000/cmd", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("connection refused")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse response
	// The response might be text if 404 or 500, but usually JSON
	var cmdResp CmdResponse
	if err := json.Unmarshal(body, &cmdResp); err != nil {
		// If not JSON, print raw body (could be panic or 404 text)
		println(c, string(body))
		return nil
	}

	if cmdResp.Error != "" {
		println(c, "Error: "+cmdResp.Error)
	} else {
		if cmdResp.Message != "" {
			print(c, cmdResp.Message)
		}
	}

	// Update context if present
	if cmdResp.Context != nil {
		*ctxId = cmdResp.Context.Id
		if shell != nil {
			shell.SetPrompt(cmdResp.Context.Prompt)
		}
		if commands != nil && len(cmdResp.Context.Commands) > 0 {
			*commands = cmdResp.Context.Commands
		}
	}

	return nil
}

func print(c *ishell.Context, msg string) {
	if c != nil {
		c.Print(msg)
	} else {
		fmt.Print(msg)
	}
}

func println(c *ishell.Context, msg string) {
	if c != nil {
		c.Println(msg)
	} else {
		fmt.Println(msg)
	}
}

func printError(c *ishell.Context, err error) {
	if c != nil {
		c.Println("Error:", err)
	} else {
		fmt.Println("Error:", err)
	}
}

func convertCommands(cmds map[string]cli.Command) []CommandInfo {
	var result []CommandInfo
	for _, cmd := range cmds {
		info := CommandInfo{
			Name:        cmd.Name,
			Usage:       cmd.Usage,
			Description: cmd.Description,
		}
		result = append(result, info)
	}
	// Sort by name for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
