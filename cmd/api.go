package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api <path>",
	Short: "Call the OpenProject API with the configured credentials",
	Long: `Call any OpenProject API v3 endpoint using the URL and key from ~/.oprc, so
nobody needs curl or the raw key.

The path may be "work_packages/1", "/work_packages/1", "/api/v3/work_packages/1",
or a full URL on the configured host (other hosts are refused). Query strings
pass through. The body goes to stdout, pretty-printed when it is JSON. A non-2xx
status prints the body to stderr, then "HTTP <code>", and exits 1.

Examples:
  op api statuses
  op api "work_packages?pageSize=3"
  op api work_packages/1 -X PATCH --input patch.json`,
	Args: cobra.ExactArgs(1),
	RunE: runAPI,
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringP("method", "X", "GET", "HTTP method")
	apiCmd.Flags().String("input", "", "JSON request body from a file, or - for stdin (not with GET)")
}

func runAPI(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	method, _ := cmd.Flags().GetString("method")
	method = strings.ToUpper(method)
	input, _ := cmd.Flags().GetString("input")

	var body []byte
	if input != "" {
		if method == "GET" {
			return fmt.Errorf("--input needs a method other than GET: add -X POST or -X PATCH")
		}
		var err error
		if input == "-" {
			body, err = io.ReadAll(cmd.InOrStdin())
		} else {
			body, err = os.ReadFile(input)
		}
		if err != nil {
			return fmt.Errorf("reading --input: %w", err)
		}
		if !json.Valid(body) {
			return fmt.Errorf("--input is not valid JSON")
		}
	}

	status, resp, err := client.Request(method, args[0], body)
	if err != nil {
		return err
	}

	out := os.Stdout
	if status < 200 || status > 299 {
		out = os.Stderr
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, resp, "", "  ") == nil {
		pretty.WriteByte('\n')
		resp = pretty.Bytes()
	}
	_, _ = out.Write(resp)

	if status < 200 || status > 299 {
		return fmt.Errorf("HTTP %d", status)
	}
	return nil
}
