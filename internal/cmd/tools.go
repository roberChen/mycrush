package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent/customtools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List discovered custom agentic tools",
	Long: `List all custom agentic tools discovered from the configured
custom_agent_tools_paths. Each tool is a directory containing a TOOL.md
file that defines a sub-agent the coder agent can invoke as a tool.`,
	Example: `# List all custom agentic tools
crush tools

# Show details for a specific tool
crush tools summarize_file`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := ResolveCwd(cmd)
		if err != nil {
			return err
		}

		dataDir, _ := cmd.Flags().GetString("data-dir")
		debug, _ := cmd.Flags().GetBool("debug")

		cfg, err := config.Init(cwd, dataDir, debug)
		if err != nil {
			return err
		}

		defs, states := customtools.Discover(cfg.Config().Options.CustomAgentToolsPaths)

		filter := strings.ToLower(strings.Join(args, " "))

		var matched []*customtools.Definition
		for _, d := range defs {
			if filter != "" && !strings.Contains(strings.ToLower(d.Name), filter) && !strings.Contains(strings.ToLower(d.Description), filter) {
				continue
			}
			matched = append(matched, d)
		}

		if !isatty.IsTerminal(os.Stdout.Fd()) {
			for _, d := range matched {
				fmt.Printf("%s\t%s\t%s\n", d.Name, d.FilePath, d.Description)
			}
			return nil
		}

		if len(matched) == 0 {
			if filter != "" {
				cmd.Println("No custom agentic tools matching", fmt.Sprintf("%q.", filter))
			} else {
				cmd.Println("No custom agentic tools discovered.")
			}
			cmd.Println()
			cmd.Println("Searched paths:")
			for _, p := range cfg.Config().Options.CustomAgentToolsPaths {
				cmd.Printf("  %s\n", p)
			}
			return nil
		}

		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(charmtone.Charple)
		dimStyle := lipgloss.NewStyle().Foreground(charmtone.Squid)
		pathStyle := lipgloss.NewStyle().Foreground(charmtone.Salt)

		for _, d := range matched {
			params := d.EffectiveParams()
			var paramNames []string
			for _, p := range params {
				if p.Required {
					paramNames = append(paramNames, p.Name+"*")
				} else {
					paramNames = append(paramNames, p.Name)
				}
			}

			cmd.Println(nameStyle.Render(d.Name) + dimStyle.Render("  ("+string(d.EffectiveModel())+" / "+string(d.EffectiveContextMode())+" / "+fmt.Sprintf("params: %s", strings.Join(paramNames, ", "))+")"))
			cmd.Printf("  %s\n", d.Description)
			cmd.Printf("  %s\n", pathStyle.Render(d.FilePath))
			cmd.Printf("  tools: %s\n", strings.Join(d.EffectiveAllowedTools(), ", "))
			if len(d.Skills) > 0 {
				cmd.Printf("  skills: %s\n", strings.Join(d.Skills, ", "))
			}
			cmd.Println()
		}

		// Report any parse errors.
		for _, st := range states {
			if st.State == customtools.StateError {
				cmd.Println(dimStyle.Render("! " + st.Path + ": " + st.Err.Error()))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
