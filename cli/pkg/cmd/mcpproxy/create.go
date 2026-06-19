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
	"bufio"
	"context"
	"fmt"
	"io"
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

type CreateOptions struct {
	IO           *iostreams.IOStreams
	Client       func(context.Context) (*amsvc.ClientWithResponses, error)
	ResolveScope func(*cobra.Command, bool, bool) (string, string, error)
	MakeScope    func(org, proj string) render.Scope

	Org   string
	Scope render.Scope

	File            string
	AuthHeaderValue string
	Gateways        []string
}

func NewCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &CreateOptions{
		IO:           f.IOStreams,
		Client:       f.AgentManager,
		ResolveScope: f.ResolveOrgProject,
		MakeScope:    f.Scope,
	}
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create an MCP proxy from a definition file",
		Long: "Create an MCP proxy from a definition file produced by " +
			"`amctl mcp-proxy template` (and optionally edited).\n\n" +
			"If the definition declares an upstream auth header, supply its value " +
			"with --auth-header-value or through stdin.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.File = args[0]
			org, _, err := opts.ResolveScope(cmd, true, false)
			scope := opts.MakeScope(org, "")
			if err != nil {
				return render.Error(opts.IO, scope, err)
			}
			opts.Org, opts.Scope = org, scope
			return runCreate(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.AuthHeaderValue, "auth-header-value", "", "Upstream auth header value (read from stdin when the definition declares upstream.authHeaderName and this flag is omitted)")
	cmd.Flags().StringSliceVar(&opts.Gateways, "gateways", nil, "Gateway IDs/handles to deploy the proxy to (repeatable)")
	return cmd
}

func runCreate(ctx context.Context, o *CreateOptions) error {
	raw, err := os.ReadFile(o.File)
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.InvalidFlag, "read %q: %v", o.File, err))
	}
	var pf proxyFile
	if err := yaml.Unmarshal(raw, &pf); err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.InvalidFlag, "parse %q: %v", o.File, err))
	}
	pf.normalize()

	if err := validateProxyFile(&pf); err != nil {
		return render.Error(o.IO, o.Scope, err)
	}
	authValue, err := o.resolveAuthValue(&pf)
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}

	req := buildCreateRequest(&pf, authValue, o.Gateways)

	client, err := o.Client(ctx)
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}
	resp, err := client.CreateMCPProxyWithResponse(ctx, o.Org, req)
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Transport, "%v", err))
	}
	if resp.JSON201 == nil {
		return render.Error(o.IO, o.Scope, cmdutil.ErrorFromServer(resp.HTTPResponse, cmdutil.FirstNonNil(resp.JSON400, resp.JSON401, resp.JSON409, resp.JSON500)))
	}

	if o.IO.JSON {
		return render.JSONSuccess(o.IO, o.Scope, resp.JSON201)
	}

	p := resp.JSON201
	cs := o.IO.StderrColorScheme()
	fmt.Fprintf(o.IO.ErrOut, "%s Created MCP proxy %s\n\n", cs.SuccessIcon(), cs.Bold(p.Id))
	fmt.Fprintf(o.IO.ErrOut, "  Name:    %s\n", p.Name)
	fmt.Fprintf(o.IO.ErrOut, "  Version: %s\n", p.Version)
	return nil
}

// resolveAuthValue determines the upstream auth header value when the
// definition declares a header name. It returns an error if a header name is set
// but no value is available, or a value is supplied with no header name to attach
// it to.
func (o *CreateOptions) resolveAuthValue(pf *proxyFile) (string, error) {
	hasHeader := strings.TrimSpace(pf.Upstream.AuthHeaderName) != ""
	if !hasHeader {
		if o.AuthHeaderValue != "" {
			return "", clierr.New(clierr.InvalidFlag, "an auth header value was provided but the definition has no upstream.authHeaderName")
		}
		return "", nil
	}
	value, err := resolveHeaderValue(o.IO, pf.Upstream.AuthHeaderName, o.AuthHeaderValue, "read auth header value from stdin")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", clierr.New(clierr.InvalidFlag, "this proxy requires an upstream auth value; pass --auth-header-value or pipe the value on stdin")
	}
	return value, nil
}

func resolveHeaderValue(ioStreams *iostreams.IOStreams, headerName, flagValue, readDescription string) (string, error) {
	if strings.TrimSpace(headerName) == "" || flagValue != "" {
		return strings.TrimSpace(flagValue), nil
	}
	if ioStreams.CanPrompt() {
		if _, err := fmt.Fprintf(ioStreams.ErrOut, "Enter upstream auth header value for %s: ", strings.TrimSpace(headerName)); err != nil {
			return "", clierr.Newf(clierr.InvalidFlag, "prompt for auth header value: %v", err)
		}
		line, err := bufio.NewReader(ioStreams.In).ReadString('\n')
		if err != nil && err != io.EOF {
			return "", clierr.Newf(clierr.InvalidFlag, "%s: %v", readDescription, err)
		}
		return strings.TrimSpace(line), nil
	}
	data, err := io.ReadAll(ioStreams.In)
	if err != nil {
		return "", clierr.Newf(clierr.InvalidFlag, "%s: %v", readDescription, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func validateProxyFile(pf *proxyFile) error {
	var v []string
	if strings.TrimSpace(pf.Name) == "" {
		v = append(v, "name is required in the definition file")
	}
	if strings.TrimSpace(pf.Handle) == "" {
		v = append(v, "handle is required in the definition file")
	} else if !handleRegex.MatchString(pf.Handle) {
		v = append(v, "handle must contain only letters, digits, '-' or '_'")
	}
	if strings.TrimSpace(pf.Version) == "" {
		v = append(v, "version is required in the definition file")
	}
	if strings.TrimSpace(pf.Upstream.URL) == "" {
		v = append(v, "upstream.url is required in the definition file")
	}
	if len(v) == 0 {
		return nil
	}
	return cmdutil.FlagErrors(v)
}

// buildCreateRequest maps the definition file (plus the resolved auth value and
// any --gateways override) into the create payload. It assumes pf has already
// been normalized (see proxyFile.normalize) so all fields are trimmed; a plain
// non-empty check is therefore sufficient and a field carries no surrounding
// whitespace into the request.
func buildCreateRequest(pf *proxyFile, authValue string, gateways []string) amsvc.CreateMCPProxyJSONRequestBody {
	main := &amsvc.UpstreamEndpoint{Url: &pf.Upstream.URL}
	if pf.Upstream.AuthHeaderName != "" {
		header := pf.Upstream.AuthHeaderName
		value := authValue
		main.Auth = &amsvc.UpstreamAuth{
			Type:   amsvc.UpstreamAuthType(defaultAuthType),
			Header: &header,
			Value:  &value,
		}
	}

	req := amsvc.CreateMCPProxyJSONRequestBody{
		Id:       pf.Handle,
		Name:     pf.Name,
		Version:  pf.Version,
		Upstream: amsvc.UpstreamConfig{Main: main},
	}
	if pf.Context != "" {
		req.Context = &pf.Context
	}
	if pf.Description != "" {
		req.Description = &pf.Description
	}
	if pf.McpSpecVersion != "" {
		req.McpSpecVersion = &pf.McpSpecVersion
	}

	caps := &amsvc.MCPProxyCapabilities{}
	hasCaps := false
	if len(pf.Capabilities.Tools) > 0 {
		caps.Tools = &pf.Capabilities.Tools
		hasCaps = true
	}
	if len(pf.Capabilities.Prompts) > 0 {
		caps.Prompts = &pf.Capabilities.Prompts
		hasCaps = true
	}
	if len(pf.Capabilities.Resources) > 0 {
		caps.Resources = &pf.Capabilities.Resources
		hasCaps = true
	}
	if hasCaps {
		req.Capabilities = caps
	}

	if len(gateways) > 0 {
		req.Gateways = &gateways
	}
	return req
}
