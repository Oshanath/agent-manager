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

import React, { useCallback, useMemo, useState } from "react";
import {
  PageLayout,
  SelectionDrawer,
  SelectionIndicator,
} from "@agent-management-platform/views";
import {
  Alert,
  Box,
  Button,
  CardContent,
  Form,
  ListingTable,
  Stack,
  Typography,
} from "@wso2/oxygen-ui";
import {
  AlertTriangle,
  Link,
  ServerCog,
} from "@wso2/oxygen-ui-icons-react";
import { generatePath, useNavigate, useParams } from "react-router-dom";
import {
  absoluteRouteMap,
  type MCPProxyListItem,
} from "@agent-management-platform/types";
import {
  useAddAgentMCPProxy,
  useListAgentMCPProxies,
  useListMCPProxies,
} from "@agent-management-platform/api-client";

function MCPServerDisplay({
  server,
  isSelected,
}: {
  server: MCPProxyListItem | null;
  isSelected: boolean;
}) {
  if (!server) return null;
  return (
    <Stack direction="row" spacing={2} flexGrow={1} alignItems="center">
      <SelectionIndicator selected={isSelected} />
      <Stack spacing={0.25} flexGrow={1}>
        <Typography variant="h6">{server.name}</Typography>
        {server.description && (
          <Typography variant="body2" color="text.secondary">
            {server.description}
          </Typography>
        )}
        <Stack direction="row" spacing={2}>
          {server.context && (
            <Typography variant="caption" color="text.secondary">
              Context: {server.context}
            </Typography>
          )}
          {server.version && (
            <Typography variant="caption" color="text.secondary">
              Version: {server.version}
            </Typography>
          )}
        </Stack>
      </Stack>
    </Stack>
  );
}

export const AddMCPServerComponent: React.FC = () => {
  const { orgId, projectId, agentId } = useParams<{
    orgId: string;
    projectId: string;
    agentId: string;
  }>();
  const navigate = useNavigate();

  const [selectedServerId, setSelectedServerId] = useState<string | null>(null);
  const [serverDrawerOpen, setServerDrawerOpen] = useState(false);

  const backHref =
    orgId && projectId && agentId
      ? generatePath(
          absoluteRouteMap.children.org.children.projects.children.agents
            .children.configure.path,
          { orgId, projectId, agentId },
        )
      : "#";

  const { data: proxiesData, isLoading: isLoadingProxies } = useListMCPProxies(
    { orgName: orgId },
  );

  const { data: boundData, isLoading: isLoadingBound } = useListAgentMCPProxies({
    orgName: orgId,
    projName: projectId,
    agentName: agentId,
  });

  // Proxy handles already mapped to this agent; these are excluded from the
  // selectable list so an MCP server can only be mapped once per agent.
  const mappedProxyIds = useMemo(
    () =>
      new Set(
        (boundData?.list ?? [])
          .map((m) => m.proxyId)
          .filter((id): id is string => !!id),
      ),
    [boundData],
  );

  const allServers = useMemo(() => proxiesData?.list ?? [], [proxiesData]);

  // Only MCP servers that haven't been mapped to this agent yet.
  const servers = useMemo(
    () => allServers.filter((s) => !(s.id && mappedProxyIds.has(s.id))),
    [allServers, mappedProxyIds],
  );

  const isLoadingServers = isLoadingProxies || isLoadingBound;

  const selectedServer = useMemo(
    () => servers.find((s) => s.id === selectedServerId) ?? null,
    [servers, selectedServerId],
  );

  const { mutate: addProxy, isPending, isError, error, reset } = useAddAgentMCPProxy();

  const isFormValid = !!selectedServerId;

  const handleSave = useCallback(() => {
    if (!isFormValid || !orgId || !projectId || !agentId || !selectedServerId) return;

    addProxy(
      {
        params: { orgName: orgId, projName: projectId, agentName: agentId },
        body: {
          proxyId: selectedServerId,
        },
      },
      {
        onSuccess: () => {
          navigate(backHref);
        },
      },
    );
  }, [
    isFormValid,
    orgId,
    projectId,
    agentId,
    selectedServerId,
    addProxy,
    navigate,
    backHref,
  ]);

  return (
    <PageLayout
      title="Add MCP Server"
      backHref={backHref}
      disableIcon
      backLabel="Back to Configure"
    >
      <Stack spacing={3}>
        {isError ? (
          <Alert severity="error" icon={<AlertTriangle size={18} />} onClose={reset}>
            {String(error instanceof Error ? error.message : "Failed to add MCP server. Please try again.")}
          </Alert>
        ) : null}

        <Form.Section>
          <Form.Header>MCP Server</Form.Header>
          <Form.Section>
            <Form.Subheader>Select MCP Server</Form.Subheader>
            {selectedServer ? (
              <Form.CardButton
                onClick={() => setServerDrawerOpen(true)}
                selected
                aria-label={`Selected: ${selectedServer.name}. Click to change.`}
              >
                <Form.CardContent>
                  <MCPServerDisplay server={selectedServer} isSelected />
                </Form.CardContent>
              </Form.CardButton>
            ) : (
              <Box>
                {!isLoadingServers && servers.length === 0 ? (
                  <ListingTable.Container>
                    <ListingTable.EmptyState
                      illustration={<ServerCog size={64} />}
                      title={
                        allServers.length === 0
                          ? "No MCP servers available"
                          : "All MCP servers added"
                      }
                      description={
                        allServers.length === 0
                          ? "No MCP servers found. Add MCP servers from the organization MCP Proxies page first."
                          : "Every MCP server in this organization is already mapped to this agent."
                      }
                      action={
                        allServers.length === 0 && orgId ? (
                          <Button
                            variant="contained"
                            size="small"
                            startIcon={<Link size={16} />}
                            onClick={() =>
                              navigate(
                                generatePath(
                                  absoluteRouteMap.children.org.children.mcpProxies
                                    .children.add.path,
                                  { orgId },
                                ),
                              )
                            }
                          >
                            Add MCP Server
                          </Button>
                        ) : undefined
                      }
                    />
                  </ListingTable.Container>
                ) : (
                  <CardContent>
                    <Button
                      variant="outlined"
                      onClick={() => setServerDrawerOpen(true)}
                      disabled={isLoadingServers || servers.length === 0}
                      startIcon={<Link size={16} />}
                    >
                      Select an MCP Server
                    </Button>
                  </CardContent>
                )}
              </Box>
            )}
          </Form.Section>

          <SelectionDrawer
            open={serverDrawerOpen}
            onClose={() => setServerDrawerOpen(false)}
            icon={<ServerCog size={24} />}
            title="Select MCP Server"
            description="Choose an MCP server to attach to this agent."
            searchPlaceholder="Search MCP servers"
            items={servers}
            isLoading={isLoadingServers}
            getItemKey={(server) => server.id ?? ""}
            isItemSelected={(server) => selectedServerId === server.id}
            matchesSearch={(server, query) =>
              (server.name ?? "").toLowerCase().includes(query) ||
              (server.description ?? "").toLowerCase().includes(query) ||
              (server.context ?? "").toLowerCase().includes(query)
            }
            onSelect={(server) => setSelectedServerId(server.id ?? null)}
            renderItem={(server, isSelected) => (
              <MCPServerDisplay server={server} isSelected={isSelected} />
            )}
            getItemAriaLabel={(server, isSelected) =>
              `${server.name}. ${isSelected ? "Selected" : "Click to select"}`
            }
            emptyState={{
              title: "No MCP servers available",
              description: "No MCP servers are available in the organization.",
            }}
            searchEmptyState={{
              title: "No MCP servers match your search",
              description: "Try a different keyword or clear the search filter.",
            }}
          />
        </Form.Section>

        <Box sx={{ display: "flex", gap: 1 }}>
          <Button variant="outlined" onClick={() => navigate(backHref)}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={!isFormValid || isPending}
          >
            {isPending ? "Saving…" : "Save"}
          </Button>
        </Box>
      </Stack>
    </PageLayout>
  );
};

export default AddMCPServerComponent;
