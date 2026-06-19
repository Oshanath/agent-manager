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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wso2/agent-manager/cli/pkg/iostreams"
)

func TestResolveHeaderValuePromptsAndReadsLineWhenInteractive(t *testing.T) {
	ios, in, _, errOut := iostreams.Test()
	ios.SetTerminal(true, true, true)
	in.WriteString("Bearer interactive-token\n")

	got, err := resolveHeaderValue(ios, "Authorization", "", "read auth header value from stdin")
	if err != nil {
		t.Fatalf("resolve header value: %v", err)
	}
	if got != "Bearer interactive-token" {
		t.Fatalf("value = %q, want %q", got, "Bearer interactive-token")
	}
	if prompt := errOut.String(); !strings.Contains(prompt, "Enter upstream auth header value for Authorization:") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestResolveHeaderValueTrimsFlagValue(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	got, err := resolveHeaderValue(ios, "Authorization", "  Bearer flag-token  \n", "read auth header value from stdin")
	if err != nil {
		t.Fatalf("resolve header value: %v", err)
	}
	if got != "Bearer flag-token" {
		t.Fatalf("value = %q, want %q", got, "Bearer flag-token")
	}
}

func TestRunCreateValidatesDefinitionBeforePromptingForAuth(t *testing.T) {
	ios, _, _, errOut := iostreams.Test()
	ios.SetTerminal(true, true, true)

	definition := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(definition, []byte(`handle: example
version: v1
upstream:
  url: https://api.example.com/mcp
  authHeaderName: Authorization
`), 0o600); err != nil {
		t.Fatalf("write definition: %v", err)
	}

	err := runCreate(context.Background(), &CreateOptions{
		IO:   ios,
		File: definition,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if strings.Contains(errOut.String(), "Enter upstream auth header value") {
		t.Fatalf("prompted before validation: %q", errOut.String())
	}
}

func TestResolveHeaderValueReadsPipeWithoutPrompt(t *testing.T) {
	ios, in, _, errOut := iostreams.Test()
	ios.SetTerminal(false, false, false)
	in.WriteString("Bearer piped-token\n")

	got, err := resolveHeaderValue(ios, "Authorization", "", "read auth header value from stdin")
	if err != nil {
		t.Fatalf("resolve header value: %v", err)
	}
	if got != "Bearer piped-token" {
		t.Fatalf("value = %q, want %q", got, "Bearer piped-token")
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected prompt for piped input: %q", errOut.String())
	}
}
