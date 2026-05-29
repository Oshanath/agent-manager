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

package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wso2/agent-manager/agent-manager-service/middleware/logger"
	"github.com/wso2/agent-manager/agent-manager-service/models"
	"github.com/wso2/agent-manager/agent-manager-service/repositories"
	"github.com/wso2/agent-manager/agent-manager-service/services"
	"github.com/wso2/agent-manager/agent-manager-service/utils"
)

// GatewayInternalController defines interface for gateway internal API HTTP handlers
type GatewayInternalController interface {
	GetLLMProvider(w http.ResponseWriter, r *http.Request)
	GetLLMProxy(w http.ResponseWriter, r *http.Request)
	GetMCPProxy(w http.ResponseWriter, r *http.Request)
	GetLLMProviderAPIKeys(w http.ResponseWriter, r *http.Request)
	GetLLMProxyAPIKeys(w http.ResponseWriter, r *http.Request)
	GetAPIKeys(w http.ResponseWriter, r *http.Request)
	GetSubscriptionPlans(w http.ResponseWriter, r *http.Request)
	PushGatewayManifest(w http.ResponseWriter, r *http.Request)
}

type gatewayInternalController struct {
	gatewayService         *services.PlatformGatewayService
	gatewayInternalService *services.GatewayInternalAPIService
	apiKeyRepo             repositories.APIKeyRepository
}

// NewGatewayInternalController creates a new gateway internal controller
func NewGatewayInternalController(
	gatewayService *services.PlatformGatewayService,
	gatewayInternalService *services.GatewayInternalAPIService,
	apiKeyRepo repositories.APIKeyRepository,
) GatewayInternalController {
	return &gatewayInternalController{
		gatewayService:         gatewayService,
		gatewayInternalService: gatewayInternalService,
		apiKeyRepo:             apiKeyRepo,
	}
}

// GetLLMProvider handles GET /api/internal/v1/llm-providers/:providerId
func (c *gatewayInternalController) GetLLMProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	// Extract client IP for logging
	clientIP := getClientIP(r)

	// Extract and validate API key from header
	apiKey := r.Header.Get("api-key")
	if apiKey == "" {
		log.Warn("Unauthorized access attempt - Missing API key", "ip", clientIP)
		http.Error(w, "API key is required. Provide 'api-key' header.", http.StatusUnauthorized)
		return
	}

	// Authenticate gateway using API key
	gateway, err := c.gatewayService.VerifyToken(apiKey)
	if err != nil {
		log.Warn("Authentication failed", "ip", clientIP, "error", err)
		http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
		return
	}

	orgName := gateway.OrganizationName
	gatewayID := gateway.UUID.String()
	providerID := r.PathValue("providerId")
	if providerID == "" {
		http.Error(w, "Provider ID is required", http.StatusBadRequest)
		return
	}

	provider, err := c.gatewayInternalService.GetActiveLLMProviderDeploymentByGateway(ctx, providerID, orgName, gatewayID)
	if err != nil {
		if errors.Is(err, utils.ErrDeploymentNotActive) {
			http.Error(w, "No active deployment found for this LLM provider on this gateway", http.StatusNotFound)
			return
		}
		if errors.Is(err, utils.ErrLLMProviderNotFound) {
			http.Error(w, "LLM provider not found", http.StatusNotFound)
			return
		}
		log.Error("Failed to get LLM provider", "error", err)
		http.Error(w, "Failed to get LLM provider", http.StatusInternalServerError)
		return
	}

	// Create ZIP file from LLM provider YAML file
	zipData, err := utils.CreateLLMProviderYamlZip(provider)
	if err != nil {
		log.Error("Failed to create ZIP file for LLM provider", "providerID", providerID, "error", err)
		http.Error(w, "Failed to create LLM provider package", http.StatusInternalServerError)
		return
	}

	// Set headers for ZIP file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"llm-provider-%s.zip\"", providerID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))

	// Return ZIP file
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(zipData); err != nil {
		log.Error("Failed to write ZIP response", "providerID", providerID, "error", err)
	}
}

// GetLLMProxy handles GET /api/internal/v1/llm-proxies/:proxyId
func (c *gatewayInternalController) GetLLMProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	// Extract client IP for logging
	clientIP := getClientIP(r)

	// Extract and validate API key from header
	apiKey := r.Header.Get("api-key")
	if apiKey == "" {
		log.Warn("Unauthorized access attempt - Missing API key", "ip", clientIP)
		http.Error(w, "API key is required. Provide 'api-key' header.", http.StatusUnauthorized)
		return
	}

	// Authenticate gateway using API key
	gateway, err := c.gatewayService.VerifyToken(apiKey)
	if err != nil {
		log.Warn("Authentication failed", "ip", clientIP, "error", err)
		http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
		return
	}

	orgName := gateway.OrganizationName
	gatewayID := gateway.UUID.String()
	proxyID := r.PathValue("proxyId")
	if proxyID == "" {
		http.Error(w, "Proxy ID is required", http.StatusBadRequest)
		return
	}

	proxy, err := c.gatewayInternalService.GetActiveLLMProxyDeploymentByGateway(ctx, proxyID, orgName, gatewayID)
	if err != nil {
		if errors.Is(err, utils.ErrDeploymentNotActive) {
			http.Error(w, "No active deployment found for this LLM proxy on this gateway", http.StatusNotFound)
			return
		}
		if errors.Is(err, utils.ErrLLMProxyNotFound) {
			http.Error(w, "LLM proxy not found", http.StatusNotFound)
			return
		}
		log.Error("Failed to get LLM proxy", "error", err)
		http.Error(w, "Failed to get LLM proxy", http.StatusInternalServerError)
		return
	}

	// Create ZIP file from LLM proxy YAML file
	zipData, err := utils.CreateLLMProxyYamlZip(proxy)
	if err != nil {
		log.Error("Failed to create ZIP file for LLM proxy", "proxyID", proxyID, "error", err)
		http.Error(w, "Failed to create LLM proxy package", http.StatusInternalServerError)
		return
	}

	// Set headers for ZIP file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"llm-proxy-%s.zip\"", proxyID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))

	// Return ZIP file
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(zipData); err != nil {
		log.Error("Failed to write ZIP response", "proxyID", proxyID, "error", err)
	}
}

// GetMCPProxy handles GET /api/internal/v1/mcp-proxies/:proxyId
func (c *gatewayInternalController) GetMCPProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)
	clientIP := getClientIP(r)

	apiKey := r.Header.Get("api-key")
	if apiKey == "" {
		log.Warn("Unauthorized access attempt - Missing API key", "ip", clientIP)
		http.Error(w, "API key is required. Provide 'api-key' header.", http.StatusUnauthorized)
		return
	}

	gateway, err := c.gatewayService.VerifyToken(apiKey)
	if err != nil {
		log.Warn("Authentication failed", "ip", clientIP, "error", err)
		http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
		return
	}

	orgName := gateway.OrganizationName
	gatewayID := gateway.UUID.String()
	proxyID := r.PathValue("proxyId")
	if proxyID == "" {
		http.Error(w, "Proxy ID is required", http.StatusBadRequest)
		return
	}

	proxy, err := c.gatewayInternalService.GetActiveMCPProxyDeploymentByGateway(ctx, proxyID, orgName, gatewayID)
	if err != nil {
		if errors.Is(err, utils.ErrDeploymentNotActive) {
			http.Error(w, "No active deployment found for this MCP proxy on this gateway", http.StatusNotFound)
			return
		}
		if errors.Is(err, utils.ErrMCPProxyNotFound) {
			http.Error(w, "MCP proxy not found", http.StatusNotFound)
			return
		}
		log.Error("Failed to get MCP proxy", "error", err)
		http.Error(w, "Failed to get MCP proxy", http.StatusInternalServerError)
		return
	}

	zipData, err := utils.CreateMCPProxyYamlZip(proxy)
	if err != nil {
		log.Error("Failed to create ZIP file for MCP proxy", "proxyID", proxyID, "error", err)
		http.Error(w, "Failed to create MCP proxy package", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"mcp-proxy-%s.zip\"", proxyID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(zipData); err != nil {
		log.Error("Failed to write ZIP response", "proxyID", proxyID, "error", err)
	}
}

// GetLLMProviderAPIKeys handles GET /api/internal/v1/llm-providers/api-keys
func (c *gatewayInternalController) GetLLMProviderAPIKeys(w http.ResponseWriter, r *http.Request) {
	c.getAPIKeysByKind(w, r, models.KindLLMProvider)
}

// GetLLMProxyAPIKeys handles GET /api/internal/v1/llm-proxies/api-keys
func (c *gatewayInternalController) GetLLMProxyAPIKeys(w http.ResponseWriter, r *http.Request) {
	c.getAPIKeysByKind(w, r, models.KindLLMProxy)
}

// GetAPIKeys handles GET /api/internal/v1/apis/api-keys
func (c *gatewayInternalController) GetAPIKeys(w http.ResponseWriter, r *http.Request) {
	c.getAPIKeysByKind(w, r, models.KindAgent)
}

// controlPlaneAPIKeyResponse matches the structure expected by the gateway controller's bulk-sync
type controlPlaneAPIKeyResponse struct {
	ETag         string            `json:"etag"`
	UUID         string            `json:"uuid"`
	Name         string            `json:"name"`
	MaskedAPIKey string            `json:"maskedApiKey"`
	APIKeyHashes map[string]string `json:"apiKeyHashes"`
	ArtifactUUID string            `json:"artifactUuid"`
	Status       string            `json:"status"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
	ExpiresAt    *string           `json:"expiresAt,omitempty"`
	Source       string            `json:"source"`
}

func (c *gatewayInternalController) getAPIKeysByKind(w http.ResponseWriter, r *http.Request, kind string) {
	log := logger.GetLogger(r.Context())

	apiKey := r.Header.Get("api-key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusUnauthorized)
		return
	}

	gateway, err := c.gatewayService.VerifyToken(apiKey)
	if err != nil {
		http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
		return
	}

	keys, err := c.apiKeyRepo.ListByArtifactKind(gateway.OrganizationName, kind)
	if err != nil {
		log.Error("Failed to list API keys", "kind", kind, "error", err)
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	result := make([]controlPlaneAPIKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp := controlPlaneAPIKeyResponse{
			UUID:         k.UUID.String(),
			Name:         k.Name,
			MaskedAPIKey: k.MaskedAPIKey,
			APIKeyHashes: map[string]string{"sha256": k.APIKeyHash},
			ArtifactUUID: k.ArtifactUUID.String(),
			Status:       k.Status,
			CreatedAt:    k.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:    k.UpdatedAt.UTC().Format(time.RFC3339),
			Source:       "external",
		}
		if k.ExpiresAt != nil {
			exp := k.ExpiresAt.UTC().Format(time.RFC3339)
			resp.ExpiresAt = &exp
		}
		result = append(result, resp)
	}

	utils.WriteSuccessResponse(w, http.StatusOK, result)
}

// GetSubscriptionPlans handles GET /api/internal/v1/subscription-plans
// Returns subscription plans for the authenticated gateway's organization.
func (c *gatewayInternalController) GetSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Subscription plans not implemented", http.StatusNotImplemented)
}

// PushGatewayManifest handles POST /api/internal/v1/gateways/{gatewayId}/manifest
// Receives the gateway's installed policy manifest.
func (c *gatewayInternalController) PushGatewayManifest(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	gatewayID := r.PathValue("gatewayId")
	if gatewayID == "" {
		http.Error(w, "Gateway ID is required", http.StatusBadRequest)
		return
	}

	apiKey := r.Header.Get("api-key")
	if apiKey == "" {
		log.Warn("Unauthorized gateway manifest push - Missing API key", "gatewayId", gatewayID)
		http.Error(w, "API key is required. Provide 'api-key' header.", http.StatusUnauthorized)
		return
	}
	gateway, err := c.gatewayService.VerifyToken(apiKey)
	if err != nil {
		log.Warn("Gateway manifest authentication failed", "gatewayId", gatewayID, "error", err)
		http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
		return
	}
	if gateway.UUID.String() != gatewayID {
		log.Warn("Gateway manifest token does not match gateway", "gatewayId", gatewayID, "tokenGatewayId", gateway.UUID)
		http.Error(w, "Gateway token does not match gateway ID", http.StatusForbidden)
		return
	}

	var manifest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
		log.Error("Failed to decode gateway manifest", "gatewayId", gatewayID, "error", err)
		http.Error(w, "Invalid gateway manifest", http.StatusBadRequest)
		return
	}
	if err := c.gatewayService.SaveGatewayPolicyManifest(gatewayID, manifest); err != nil {
		if errors.Is(err, utils.ErrGatewayNotFound) {
			http.Error(w, "Gateway not found", http.StatusNotFound)
			return
		}
		log.Error("Failed to store gateway manifest", "gatewayId", gatewayID, "error", err)
		http.Error(w, "Failed to store gateway manifest", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
