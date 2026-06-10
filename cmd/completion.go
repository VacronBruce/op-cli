package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh]",
	Short:     "Generate shell completion script",
	ValidArgs: []string{"bash", "zsh"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Long: `Generate a shell completion script for op.

Bash — one-time:
  source <(op completion bash)

Bash — persistent (add to ~/.bashrc):
  echo 'source <(op completion bash)' >> ~/.bashrc

Zsh — one-time:
  source <(op completion zsh)

Zsh — persistent (add to ~/.zshrc):
  echo 'source <(op completion zsh)' >> ~/.zshrc

After adding to your shell profile, open a new terminal or run:
  exec $SHELL
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(completionCmd)
}
