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

	"github.com/spf13/cobra"

	amsvc "github.com/wso2/agent-manager/cli/pkg/clients/amsvc/gen"
	"github.com/wso2/agent-manager/cli/pkg/clierr"
	"github.com/wso2/agent-manager/cli/pkg/cmdutil"
	"github.com/wso2/agent-manager/cli/pkg/iostreams"
	"github.com/wso2/agent-manager/cli/pkg/render"
)

type GetOptions struct {
	IO           *iostreams.IOStreams
	Client       func(context.Context) (*amsvc.ClientWithResponses, error)
	ResolveScope func(*cobra.Command, bool, bool) (string, string, error)
	MakeScope    func(org, proj string) render.Scope

	Org   string
	Scope render.Scope
	Proxy string
}

func NewGetCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &GetOptions{
		IO:           f.IOStreams,
		Client:       f.AgentManager,
		ResolveScope: f.ResolveOrgProject,
		MakeScope:    f.Scope,
	}
	cmd := &cobra.Command{
		Use:   "get <proxy>",
		Short: "Show details of an MCP proxy",
		Long:  "Show details of an MCP proxy by its UUID or handle.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, _, err := opts.ResolveScope(cmd, true, false)
			scope := opts.MakeScope(org, "")
			if err != nil {
				return render.Error(opts.IO, scope, err)
			}
			opts.Org, opts.Scope = org, scope
			opts.Proxy = args[0]
			return runGet(cmd.Context(), opts)
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return cmdutil.CompleteMCPProxies(cmd, f), cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

func runGet(ctx context.Context, o *GetOptions) error {
	if err := cmdutil.ValidatePathParam("proxy", o.Proxy); err != nil {
		return render.Error(o.IO, o.Scope, err)
	}
	client, err := o.Client(ctx)
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}
	resp, err := client.GetMCPProxyWithResponse(ctx, o.Org, o.Proxy)
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Transport, "%v", err))
	}
	if resp.JSON200 == nil {
		return render.Error(o.IO, o.Scope, cmdutil.ErrorFromServer(resp.HTTPResponse, cmdutil.FirstNonNil(resp.JSON400, resp.JSON401, resp.JSON404, resp.JSON500)))
	}

	if o.IO.JSON {
		return render.JSONSuccess(o.IO, o.Scope, resp.JSON200)
	}

	p := resp.JSON200
	w := o.IO.Out
	cs := o.IO.ColorScheme()
	fmt.Fprintf(w, "id:             %s\n", cs.Bold(p.Id))
	fmt.Fprintf(w, "name:           %s\n", p.Name)
	fmt.Fprintf(w, "version:        %s\n", p.Version)
	fmt.Fprintf(w, "description:    %s\n", deref(p.Description))
	fmt.Fprintf(w, "context:        %s\n", deref(p.Context))
	fmt.Fprintf(w, "spec version:   %s\n", deref(p.McpSpecVersion))
	if url := upstreamURL(p.Upstream); url != "" {
		fmt.Fprintf(w, "upstream:       %s\n", url)
	}
	fmt.Fprintf(w, "in catalog:     %t\n", p.InCatalog)
	fmt.Fprintf(w, "created by:     %s\n", deref(p.CreatedBy))
	if p.CreatedAt != nil {
		fmt.Fprintf(w, "created:        %s\n", cs.Gray(p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")))
	}
	return nil
}

// upstreamURL returns the main upstream endpoint URL if configured.
func upstreamURL(u amsvc.UpstreamConfig) string {
	if u.Main != nil && u.Main.Url != nil {
		return *u.Main.Url
	}
	return ""
}
