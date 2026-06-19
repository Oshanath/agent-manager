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
	"regexp"
	"strings"
)

const (
	// defaultVersion is the proxy version written by `template` when the MCP
	// server does not report one.
	defaultVersion = "v1"
	// defaultAuthType is the upstream auth scheme used for a header-based
	// credential. The service authenticates using the header name + value; the
	// type is informational.
	defaultAuthType = "api-key"
)

// proxyFile is the on-disk representation written by `template` and consumed by
// `create`. It is intentionally CLI-local (not the generated API type) so the
// file stays small and human-editable.
//
// The upstream auth header *value* is never stored here: it is a secret and is
// supplied again at create time via --auth-header-value or stdin.
type proxyFile struct {
	Name           string            `yaml:"name"`
	Handle         string            `yaml:"handle"`
	Version        string            `yaml:"version"`
	Context        string            `yaml:"context,omitempty"`
	Description    string            `yaml:"description,omitempty"`
	McpSpecVersion string            `yaml:"mcpSpecVersion,omitempty"`
	Upstream       proxyFileUpstream `yaml:"upstream"`
	Capabilities   proxyFileCaps     `yaml:"capabilities,omitempty"`
	ServerInfo     map[string]any    `yaml:"serverInfo,omitempty"`
}

type proxyFileUpstream struct {
	URL string `yaml:"url"`
	// AuthHeaderName is the upstream auth header the proxy should send. When set,
	// `create` requires the matching value via --auth-header-value or stdin.
	AuthHeaderName string `yaml:"authHeaderName,omitempty"`
}

type proxyFileCaps struct {
	Tools     []map[string]any `yaml:"tools,omitempty"`
	Prompts   []map[string]any `yaml:"prompts,omitempty"`
	Resources []map[string]any `yaml:"resources,omitempty"`
}

// handleRegex mirrors the server-side handle rule: letters, digits, '-' and '_'.
var handleRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// nonHandleChars matches any run of characters not allowed in a handle.
var nonHandleChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// slugify converts an arbitrary MCP server name into a valid handle: lowercased,
// with each run of disallowed characters collapsed to a single '-' and leading/
// trailing '-' or '_' trimmed. Returns "" when nothing usable remains.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonHandleChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-_")
}

// normalize trims surrounding whitespace from every field that is forwarded to
// the API, so validation, auth resolution, and request building all observe the
// same values. Without this a user-edited definition could, for example, carry a
// whitespace-only authHeaderName that passes the "not set" check in one place
// but is still sent as auth in another.
func (pf *proxyFile) normalize() {
	pf.Name = strings.TrimSpace(pf.Name)
	pf.Handle = strings.TrimSpace(pf.Handle)
	pf.Version = strings.TrimSpace(pf.Version)
	pf.Context = strings.TrimSpace(pf.Context)
	pf.Description = strings.TrimSpace(pf.Description)
	pf.McpSpecVersion = strings.TrimSpace(pf.McpSpecVersion)
	pf.Upstream.URL = strings.TrimSpace(pf.Upstream.URL)
	pf.Upstream.AuthHeaderName = strings.TrimSpace(pf.Upstream.AuthHeaderName)
}
