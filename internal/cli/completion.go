package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/spf13/cobra"
)

// CompletionRegistry handles dynamic flag completions
type CompletionRegistry struct {
	deps *Dependencies
}

// NewCompletionRegistry creates a new completion registry
func NewCompletionRegistry(deps *Dependencies) *CompletionRegistry {
	return &CompletionRegistry{deps: deps}
}

// ProjectCompletion returns a completion function for project names
func (r *CompletionRegistry) ProjectCompletion() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		projects, err := r.deps.ProjectService.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, p := range projects {
			if strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(toComplete)) {
				completions = append(completions, p.Name)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// AreaCompletion returns a completion function for area names
func (r *CompletionRegistry) AreaCompletion() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		areas, err := r.deps.AreaService.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, a := range areas {
			if strings.HasPrefix(strings.ToLower(a.Name), strings.ToLower(toComplete)) {
				completions = append(completions, a.Name)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// RegisterProjectFlag registers project completion on a command's --project flag
func (r *CompletionRegistry) RegisterProjectFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("project", r.ProjectCompletion())
}

// RegisterAreaFlag registers area completion on a command's --area flag
func (r *CompletionRegistry) RegisterAreaFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("area", r.AreaCompletion())
}

// SortCompletion returns a completion function for sort fields
func (r *CompletionRegistry) SortCompletion() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		fields := task.ValidSortFields()
		var completions []string

		// Handle multi-field completion (after comma)
		lastComma := strings.LastIndex(toComplete, ",")
		prefix := ""
		search := toComplete
		if lastComma >= 0 {
			prefix = toComplete[:lastComma+1]
			search = toComplete[lastComma+1:]
		}

		for _, f := range fields {
			if strings.HasPrefix(f, strings.ToLower(search)) {
				completions = append(completions, prefix+f)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// RegisterSortFlag registers sort completion on a command's --sort flag
func (r *CompletionRegistry) RegisterSortFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("sort", r.SortCompletion())
}

// RegisterAll registers project, area, and sort completion on a command
func (r *CompletionRegistry) RegisterAll(cmd *cobra.Command) {
	r.RegisterProjectFlag(cmd)
	r.RegisterAreaFlag(cmd)
	r.RegisterSortFlag(cmd)
}

// NewCompletionCmd creates the completion command for generating shell scripts
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for tt.

Bash:
  $ source <(tt completion bash)

  # To load completions for each session, add to ~/.bashrc:
  $ echo 'source <(tt completion bash)' >> ~/.bashrc

Zsh:
  # If shell completion is not already enabled, enable it:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  $ source <(tt completion zsh)

  # To load completions for each session, add to ~/.zshrc:
  $ echo 'source <(tt completion zsh)' >> ~/.zshrc

Fish:
  $ tt completion fish | source

  # To load completions for each session:
  $ tt completion fish > ~/.config/fish/completions/tt.fish
`,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
