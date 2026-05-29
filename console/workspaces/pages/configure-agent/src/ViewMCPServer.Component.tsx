/**
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import { useMemo } from "react";
import { PageLayout } from "@agent-management-platform/views";
import {
  Alert,
  Chip,
  Form,
  Skeleton,
  Stack,
  Typography,
} from "@wso2/oxygen-ui";
import { AlertTriangle } from "@wso2/oxygen-ui-icons-react";
import { useListAgentMCPProxies } from "@agent-management-platform/api-client";
import {
  absoluteRouteMap,
  type AgentMCPProxyListItem,
  type EnvironmentVariableConfig,
  type EnvProviderConfigMappings,
} from "@agent-management-platform/types";
import { generatePath, Navigate, useNavigate, useParams } from "react-router-dom";
import { EnvironmentVariablesReference } from "./Configure/subComponents/EnvironmentVariablesReference";

export const ViewMCPServerComponent = () => {
  const { orgId, projectId, agentId, proxyId } = useParams<{
    orgId: string;
    projectId: string;
    agentId: string;
    proxyId: string;
  }>();
  const decodedServerId = useMemo(() => decodeRouteParam(proxyId), [proxyId]);
  const navigate = useNavigate();

  const { data, isLoading, isError, error } = useListAgentMCPProxies({
    orgName: orgId,
    projName: projectId,
    agentName: agentId,
  });

  const mappings = useMemo(() => data?.list ?? [], [data]);
  const server = useMemo(
    () => mappings.find((item) => item.id === decodedServerId),
    [mappings, decodedServerId],
  );

  const backHref =
    orgId && projectId && agentId
      ? generatePath(
          absoluteRouteMap.children.org.children.projects.children.agents
            .children.configure.path,
          { orgId, projectId, agentId },
        )
      : "#";

  const serverDetails = server as
    | (AgentMCPProxyListItem & {
        environmentVariables?: EnvironmentVariableConfig[];
        envMappings?: Record<string, EnvProviderConfigMappings>;
      })
    | undefined;
  const envVarRows = serverDetails?.environmentVariables ?? [];
  const envMappings = Object.entries(serverDetails?.envMappings ?? {});

  if (isLoading) {
    return (
      <PageLayout
        title="MCP Server Configuration"
        backHref={backHref}
        disableIcon
        backLabel="Back to Configure"
      >
        <Stack spacing={2}>
          <Skeleton variant="rounded" height={56} />
          <Skeleton variant="rounded" height={180} />
          <Skeleton variant="rounded" height={240} />
        </Stack>
      </PageLayout>
    );
  }

  if (isError) {
    return (
      <PageLayout
        title="MCP Server Configuration"
        backHref={backHref}
        disableIcon
        backLabel="Back to Configure"
      >
        <Alert severity="error" icon={<AlertTriangle size={18} />}>
          {error instanceof Error
            ? error.message
            : "Failed to load MCP server configuration."}
        </Alert>
      </PageLayout>
    );
  }

  if (!server) {
    return <Navigate to={backHref} replace />;
  }

  // Links to the organization-level MCP proxy this agent config points at.
  const mcpProxyHref =
    orgId && server.proxyId
      ? generatePath(
          absoluteRouteMap.children.org.children.mcpProxies.children.view.path,
          { orgId, proxyId: server.proxyId },
        )
      : undefined;

  return (
    <PageLayout
      title={server.name}
      backHref={backHref}
      disableIcon
      backLabel="Back to Configuration Listing"
    >
      {server.description && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          {server.description}
        </Typography>
      )}

      <Stack spacing={3}>
        {envVarRows.length > 0 && (
          <EnvironmentVariablesReference
            description="The following environment variables are injected into the agent deployment so it can reach this MCP server. The agent reads the MCP server URL from these variables."
            rows={envVarRows.map((envVar) => ({
              key: envVar.key,
              name: envVar.name,
              description: describeMCPEnvVar(envVar.key),
            }))}
          />
        )}

        <Form.Section>
          <Form.Header>MCP Server</Form.Header>
          <Stack spacing={2.5}>
            <Form.CardButton
              onClick={() => mcpProxyHref && navigate(mcpProxyHref)}
              disabled={!mcpProxyHref}
              aria-label={`View MCP proxy ${server.proxyName ?? server.name} in the organization`}
            >
              <Form.CardContent>
                <Stack spacing={0.5} flexGrow={1}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="h6">
                      {server.proxyName ?? server.name}
                    </Typography>
                    {server.version && (
                      <Chip label={server.version} size="small" variant="outlined" />
                    )}
                  </Stack>

                  <Typography variant="caption" color="text.secondary">
                    Context:{" "}
                    <Typography
                      component="span"
                      variant="body2"
                      color={server.context ? "text.primary" : "text.disabled"}
                    >
                      {server.context ?? "Not configured"}
                    </Typography>
                  </Typography>

                  {envMappings.length > 0 ? (
                    envMappings.map(([envName, mapping]) => (
                      <Typography
                        key={envName}
                        variant="caption"
                        color="text.secondary"
                      >
                        {`Environment URL (${envName})`}:{" "}
                        <Typography
                          component="span"
                          variant="body2"
                          color={
                            mapping.configuration?.url
                              ? "text.primary"
                              : "text.disabled"
                          }
                          sx={{ wordBreak: "break-all" }}
                        >
                          {mapping.configuration?.url ?? "Not configured"}
                        </Typography>
                      </Typography>
                    ))
                  ) : (
                    <Typography variant="caption" color="text.secondary">
                      Environment URL:{" "}
                      <Typography
                        component="span"
                        variant="body2"
                        color={server.proxyUrl ? "text.primary" : "text.disabled"}
                        sx={{ wordBreak: "break-all" }}
                      >
                        {server.proxyUrl ?? "Not configured"}
                      </Typography>
                    </Typography>
                  )}
                </Stack>
              </Form.CardContent>
            </Form.CardButton>
          </Stack>
        </Form.Section>
      </Stack>
    </PageLayout>
  );
};

function decodeRouteParam(value?: string) {
  if (!value) return "";
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function describeMCPEnvVar(key: string): string {
  if (/url/i.test(key)) return "Base URL of the MCP server endpoint";
  // Humanize the key: add spaces before capitals and capitalize the first letter.
  return key.replace(/([A-Z])/g, " $1").replace(/^./, (str) => str.toUpperCase());
}

export default ViewMCPServerComponent;
