// Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mcpproxy

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	amsvc "github.com/wso2/agent-manager/cli/pkg/clients/amsvc/gen"
	"github.com/wso2/agent-manager/cli/pkg/clierr"
	"github.com/wso2/agent-manager/cli/pkg/cmdutil"
	"github.com/wso2/agent-manager/cli/pkg/iostreams"
	"github.com/wso2/agent-manager/cli/pkg/render"
)

type TemplateOptions struct {
	IO           *iostreams.IOStreams
	Client       func(context.Context) (*amsvc.ClientWithResponses, error)
	ResolveScope func(*cobra.Command, bool, bool) (string, string, error)
	MakeScope    func(org, proj string) render.Scope

	Org   string
	Scope render.Scope

	URL             string
	AuthHeaderName  string
	AuthHeaderValue string
	OutputFile      string
	Force           bool
}

// TemplateResult is the JSON payload reported by `template`.
type TemplateResult struct {
	File      string `json:"file"`
	Name      string `json:"name"`
	Handle    string `json:"handle"`
	URL       string `json:"url"`
	Tools     int    `json:"tools"`
	Prompts   int    `json:"prompts"`
	Resources int    `json:"resources"`
}

func NewTemplateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &TemplateOptions{
		IO:           f.IOStreams,
		Client:       f.AgentManager,
		ResolveScope: f.ResolveOrgProject,
		MakeScope:    f.Scope,
	}
	cmd := &cobra.Command{
		Use:   "template <url>",
		Short: "Create an MCP proxy definition file from an MCP server",
		Long: "Connect to an MCP server URL, fetch its capabilities (tools, prompts, " +
			"resources) and server info, and write a <handle>.yaml definition file.\n\n" +
			"Review/edit the file, then create the proxy with:\n" +
			"  amctl mcp-proxy create <file>\n\n" +
			"If the server requires authentication, pass --auth-header-name. The " +
			"header value can be supplied with --auth-header-value, or through stdin " +
			"when the flag is omitted. The header value is used only to reach the " +
			"server and is never written to the file.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.URL = args[0]
			if err := validateTemplate(opts); err != nil {
				return render.Error(opts.IO, render.Scope{}, err)
			}
			org, _, err := opts.ResolveScope(cmd, true, false)
			scope := opts.MakeScope(org, "")
			if err != nil {
				return render.Error(opts.IO, scope, err)
			}
			opts.Org, opts.Scope = org, scope
			return runTemplate(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.AuthHeaderName, "auth-header-name", "", "Upstream auth header name (e.g. Authorization)")
	cmd.Flags().StringVar(&opts.AuthHeaderValue, "auth-header-value", "", "Upstream auth header value (read from stdin when --auth-header-name is set and this flag is omitted)")
	cmd.Flags().StringVarP(&opts.OutputFile, "output", "o", "", "Output file path (default <handle>.yaml)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite the output file if it already exists")
	return cmd
}

func validateTemplate(o *TemplateOptions) error {
	var v []string
	if strings.TrimSpace(o.URL) == "" {
		v = append(v, "url argument is required")
	}
	if strings.TrimSpace(o.AuthHeaderValue) != "" && strings.TrimSpace(o.AuthHeaderName) == "" {
		v = append(v, "--auth-header-name is required when --auth-header-value is set")
	}
	if len(v) == 0 {
		return nil
	}
	return cmdutil.FlagErrors(v)
}

func runTemplate(ctx context.Context, o *TemplateOptions) error {
	o.AuthHeaderName = strings.TrimSpace(o.AuthHeaderName)
	authHeaderValue, err := resolveHeaderValue(o.IO, o.AuthHeaderName, o.AuthHeaderValue, "read auth header value from stdin")
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}
	if o.AuthHeaderName != "" && authHeaderValue == "" {
		return render.Error(o.IO, o.Scope, clierr.New(clierr.InvalidFlag, "this MCP server requires an upstream auth value; pass --auth-header-value or pipe the value on stdin"))
	}

	client, err := o.Client(ctx)
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}

	req := amsvc.FetchMCPServerInfoJSONRequestBody{Url: o.URL}
	if o.AuthHeaderName != "" {
		req.Auth = &amsvc.UpstreamAuth{
			Type:   amsvc.UpstreamAuthType(defaultAuthType),
			Header: &o.AuthHeaderName,
			Value:  &authHeaderValue,
		}
	}

	resp, err := client.FetchMCPServerInfoWithResponse(ctx, o.Org, req)
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Transport, "%v", err))
	}
	if resp.JSON200 == nil {
		return render.Error(o.IO, o.Scope, cmdutil.ErrorFromServer(resp.HTTPResponse, cmdutil.FirstNonNil(resp.JSON400, resp.JSON401, resp.JSON500)))
	}

	pf := buildProxyFile(o, resp.JSON200)

	filename := o.OutputFile
	if filename == "" {
		filename = pf.Handle + ".yaml"
	}
	if !o.Force {
		if _, statErr := os.Stat(filename); statErr == nil {
			return render.Error(o.IO, o.Scope, clierr.Newf(clierr.InvalidFlag, "file %q already exists; use --force to overwrite or --output to choose another path", filename))
		}
	}

	body, err := marshalProxyFile(pf, filename)
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Internal, "%v", err))
	}
	if err := os.WriteFile(filename, body, 0o600); err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Internal, "write %q: %v", filename, err))
	}

	result := TemplateResult{
		File:      filename,
		Name:      pf.Name,
		Handle:    pf.Handle,
		URL:       pf.Upstream.URL,
		Tools:     len(pf.Capabilities.Tools),
		Prompts:   len(pf.Capabilities.Prompts),
		Resources: len(pf.Capabilities.Resources),
	}
	if o.IO.JSON {
		return render.JSONSuccess(o.IO, o.Scope, result)
	}

	cs := o.IO.StderrColorScheme()
	w := o.IO.ErrOut
	fmt.Fprintf(w, "%s Created MCP proxy template %s\n\n", cs.SuccessIcon(), cs.Bold(pf.Name))
	fmt.Fprintf(w, "  Handle:    %s\n", pf.Handle)
	fmt.Fprintf(w, "  URL:       %s\n", pf.Upstream.URL)
	fmt.Fprintf(w, "  Tools:     %d\n", result.Tools)
	fmt.Fprintf(w, "  Prompts:   %d\n", result.Prompts)
	fmt.Fprintf(w, "  Resources: %d\n\n", result.Resources)
	fmt.Fprintf(w, "Wrote %s. Review it, then run:\n", cs.Bold(filename))
	quoted := shellQuote(filename)
	if pf.Upstream.AuthHeaderName != "" {
		fmt.Fprintf(w, "  amctl mcp-proxy create %s --auth-header-value <value>\n", quoted)
	} else {
		fmt.Fprintf(w, "  amctl mcp-proxy create %s\n", quoted)
	}
	return nil
}

// buildProxyFile maps the fetched server info into the on-disk proxy definition.
func buildProxyFile(o *TemplateOptions, info *amsvc.MCPServerInfoFetchResponse) proxyFile {
	name := serverName(info, o.URL)
	handle := slugify(name)
	if handle == "" {
		handle = "mcp-server"
	}

	pf := proxyFile{
		Name:    name,
		Handle:  handle,
		Version: defaultVersion,
		Context: "/" + handle,
		Upstream: proxyFileUpstream{
			URL:            o.URL,
			AuthHeaderName: o.AuthHeaderName,
		},
	}
	if info.Tools != nil {
		pf.Capabilities.Tools = *info.Tools
	}
	if info.Prompts != nil {
		pf.Capabilities.Prompts = *info.Prompts
	}
	if info.Resources != nil {
		pf.Capabilities.Resources = *info.Resources
	}
	if info.ServerInfo != nil {
		pf.ServerInfo = *info.ServerInfo
		if v, ok := (*info.ServerInfo)["version"].(string); ok && v != "" {
			pf.McpSpecVersion = v
		}
	}
	return pf
}

// serverName extracts a human-readable name from the MCP serverInfo, falling
// back to the URL host when the server does not report one.
func serverName(info *amsvc.MCPServerInfoFetchResponse, rawURL string) string {
	if info != nil && info.ServerInfo != nil {
		si := *info.ServerInfo
		for _, key := range []string{"name", "title"} {
			if v, ok := si[key].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return "mcp-server"
}

// marshalProxyFile renders the proxy definition with a guiding header comment.
func marshalProxyFile(pf proxyFile, filename string) ([]byte, error) {
	data, err := yaml.Marshal(pf)
	if err != nil {
		return nil, err
	}
	createHint := "amctl mcp-proxy create " + shellQuote(filename)
	if pf.Upstream.AuthHeaderName != "" {
		createHint += " --auth-header-value <value>"
	}
	header := "# MCP proxy definition generated by 'amctl mcp-proxy template'.\n" +
		"# Review and edit the fields below, then create the proxy with:\n" +
		"#   " + createHint + "\n" +
		"#\n" +
		"# 'name' is the display name; 'handle' is the unique id used by the proxy.\n" +
		"# The upstream auth header value is intentionally not stored here; pass it\n" +
		"# at create time with --auth-header-value or through stdin.\n\n"
	return append([]byte(header), data...), nil
}

// shellQuote returns path safe to paste into a shell command. Paths containing
// only safe characters are returned unchanged; anything else is wrapped in
// single quotes (with embedded single quotes escaped) so copy-paste survives
// spaces and shell metacharacters.
func shellQuote(path string) string {
	if path != "" && strings.IndexFunc(path, func(r rune) bool {
		return !(r == '-' || r == '_' || r == '.' || r == '/' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	}) == -1 {
		return path
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
