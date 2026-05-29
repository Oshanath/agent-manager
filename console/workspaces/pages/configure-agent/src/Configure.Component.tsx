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

import React, { useMemo } from "react";
import { Divider } from "@wso2/oxygen-ui";
import { generatePath, useParams } from "react-router-dom";
import { PageLayout } from "@agent-management-platform/views";
import { useListAgentModelConfigs } from "@agent-management-platform/api-client";
import { absoluteRouteMap } from "@agent-management-platform/types";
import {
  AgentConfigTableSection,
  type AgentConfigTableLabels,
} from "./Configure/subComponents/AgentConfigTableSection";

const configureRoutes =
  absoluteRouteMap.children.org.children.projects.children.agents.children
    .configure.children;

const llmLabels: AgentConfigTableLabels = {
  title: "LLM Providers",
  searchPlaceholder: "Search by name, description, or type...",
  addButtonLabel: "Add LLM Provider",
  emptyTitle: "No LLM Service providers configured",
  emptyDescription:
    "Add an LLM service provider at the organization level, then attach it to this agent using Add Service Provider above.",
  errorTitle: "Failed to load model configs",
  errorFallback: "Failed to load model configs. Please try again.",
  searchEmptyTitle: "No model configs match your search criteria",
  searchEmptyDescription: "Try adjusting your search keywords.",
  removeTitle: "Remove Model Config",
  removeTooltip: "Remove config",
  removeConfirmation: () =>
    "Are you sure you want to remove this LLM provider configuration from the agent?",
  removeAriaLabel: (config) => `Remove provider ${config.name || config.uuid}`,
};

const mcpLabels: AgentConfigTableLabels = {
  title: "MCP Servers",
  searchPlaceholder: "Search by name or description...",
  addButtonLabel: "Add MCP Server",
  emptyTitle: "No MCP Servers selected",
  emptyDescription: "Add MCP Servers that this agent can use.",
  errorTitle: "Failed to load MCP servers",
  errorFallback: "Failed to load MCP servers. Please try again.",
  searchEmptyTitle: "No MCP Servers match your search criteria",
  searchEmptyDescription: "Try adjusting your search keywords.",
  removeTitle: "Remove MCP Server",
  removeTooltip: "Remove MCP server",
  removeConfirmation: (config) =>
    `Are you sure you want to remove "${config.name}" from this agent?`,
  removeAriaLabel: (config) => `Remove ${config.name}`,
};

export const ConfigureComponent: React.FC = () => {
  const { orgId, projectId, agentId } = useParams<{
    orgId: string;
    projectId: string;
    agentId: string;
  }>();

  // A single call fetches every model config for the agent; the two tables
  // below render slices of it filtered by type.
  const { data, isLoading, error } = useListAgentModelConfigs(
    { orgName: orgId, projName: projectId, agentName: agentId },
    { limit: 1000, offset: 0 },
  );

  const configs = useMemo(() => data?.configs ?? [], [data]);
  const llmConfigs = useMemo(
    () => configs.filter((c) => c.type !== "mcp"),
    [configs],
  );
  const mcpConfigs = useMemo(
    () => configs.filter((c) => c.type === "mcp"),
    [configs],
  );

  const hasParams = Boolean(orgId && projectId && agentId);

  const llmAddPath = hasParams
    ? generatePath(configureRoutes.llmProviders.children.add.path, {
        orgId,
        projectId,
        agentId,
      })
    : "#";
  const mcpAddPath = hasParams
    ? generatePath(configureRoutes.mcpProxies.children.add.path, {
        orgId,
        projectId,
        agentId,
      })
    : "#";

  const getLlmViewPath = (configId: string) =>
    hasParams
      ? generatePath(configureRoutes.llmProviders.children.view.path, {
          orgId,
          projectId,
          agentId,
          configId,
        })
      : "#";
  const getMcpViewPath = (configId: string) =>
    hasParams
      ? generatePath(configureRoutes.mcpProxies.children.view.path, {
          orgId,
          projectId,
          agentId,
          proxyId: encodeURIComponent(configId),
        })
      : "#";

  return (
    <PageLayout title="Configure Agent" disableIcon>
      <AgentConfigTableSection
        configs={llmConfigs}
        isLoading={isLoading}
        error={error}
        labels={llmLabels}
        addPath={llmAddPath}
        getViewPath={getLlmViewPath}
      />
      <Divider sx={{ my: 3 }} />
      <AgentConfigTableSection
        configs={mcpConfigs}
        isLoading={isLoading}
        error={error}
        labels={mcpLabels}
        addPath={mcpAddPath}
        getViewPath={getMcpViewPath}
      />
    </PageLayout>
  );
};

export default ConfigureComponent;
