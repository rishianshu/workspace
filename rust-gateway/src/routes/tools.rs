//! Tools API routes for tool discovery and execution - proxies to go-agent-service

use axum::{
    extract::{Path, Query},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::env;

// ========================
// Types
// ========================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolDefinition {
    pub name: String,
    pub description: String,
    pub actions: Vec<ActionDefinition>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub endpoint_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub template_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionDefinition {
    pub name: String,
    pub description: String,
    #[serde(rename = "inputSchema", skip_serializing_if = "Option::is_none")]
    pub input_schema: Option<String>,
    #[serde(rename = "outputSchema", skip_serializing_if = "Option::is_none")]
    pub output_schema: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Project {
    pub id: String,
    pub slug: String,
    pub display_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Endpoint {
    pub id: String,
    pub nucleus_endpoint_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
    pub template_id: String,
    pub display_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_system: Option<String>,
    pub capabilities: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecuteToolRequest {
    pub name: String,
    pub action: String,
    pub params: serde_json::Value,
    #[serde(rename = "endpointId", skip_serializing_if = "Option::is_none")]
    pub endpoint_id: Option<String>,
    #[serde(rename = "keyToken", skip_serializing_if = "Option::is_none")]
    pub key_token: Option<String>,
    #[serde(rename = "userId", skip_serializing_if = "Option::is_none")]
    pub user_id: Option<String>,
    #[serde(rename = "projectId", skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolResult {
    pub success: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrainSearchRequest {
    pub query: String,
    #[serde(skip_serializing_if = "Option::is_none", rename = "projectId")]
    pub project_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ProjectQuery {
    #[serde(rename = "projectId")]
    pub project_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ToolQuery {
    #[serde(rename = "userId")]
    pub user_id: Option<String>,
    #[serde(rename = "projectId")]
    pub project_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AppQuery {
    #[serde(rename = "userId")]
    pub user_id: Option<String>,
    #[serde(rename = "projectId")]
    pub project_id: Option<String>,
    #[serde(rename = "templateId")]
    pub template_id: Option<String>,
    #[serde(rename = "instanceKey")]
    pub instance_key: Option<String>,
    #[serde(rename = "id")]
    pub id: Option<String>,
}

// ========================
// Helper
// ========================

fn get_agent_url() -> String {
    env::var("AGENT_SERVICE_URL").unwrap_or_else(|_| "http://localhost:9001".into())
}

async fn get_client() -> Client {
    Client::new()
}

// ========================
// Handlers
// ========================

/// GET /api/tools - List all available tools (proxies to go-agent)
pub async fn list_tools(Query(query): Query<ToolQuery>) -> impl IntoResponse {
    let mut url = format!("{}/tools", get_agent_url());
    let mut params: Vec<String> = Vec::new();
    if let Some(user_id) = query.user_id {
        params.push(format!("userId={}", user_id));
    }
    if let Some(project_id) = query.project_id {
        params.push(format!("projectId={}", project_id));
    }
    if !params.is_empty() {
        url = format!("{}?{}", url, params.join("&"));
    }
    
    match get_client().await.get(&url).send().await {
        Ok(resp) => {
            match resp.json::<Vec<ToolDefinition>>().await {
                Ok(tools) => Json(tools).into_response(),
                Err(e) => {
                    tracing::error!("Failed to parse tools response: {}", e);
                    (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
                }
            }
        }
        Err(e) => {
            tracing::error!("Failed to call go-agent /tools: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// POST /api/tools/execute - Execute a tool (proxies to go-agent)
pub async fn execute_tool(Json(req): Json<ExecuteToolRequest>) -> impl IntoResponse {
    tracing::info!("Executing tool: {} action: {}", req.name, req.action);
    
    let url = format!("{}/tools/execute", get_agent_url());
    
    match get_client().await.post(&url).json(&req).send().await {
        Ok(resp) => {
            match resp.json::<ToolResult>().await {
                Ok(result) => Json(result).into_response(),
                Err(e) => {
                    tracing::error!("Failed to parse execute response: {}", e);
                    (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
                }
            }
        }
        Err(e) => {
            tracing::error!("Failed to call go-agent /tools/execute: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// POST /api/apps/instances - Upsert app instance
pub async fn upsert_app_instance(Json(payload): Json<serde_json::Value>) -> impl IntoResponse {
    let url = format!("{}/apps/instances", get_agent_url());
    match get_client().await.post(&url).json(&payload).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse app instance response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/instances: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/apps/instances - Get app instance by id or templateId+instanceKey
pub async fn get_app_instance(Query(query): Query<AppQuery>) -> impl IntoResponse {
    let mut url = format!("{}/apps/instances", get_agent_url());
    let mut params: Vec<String> = Vec::new();
    if let Some(id) = query.id {
        params.push(format!("id={}", id));
    }
    if let Some(template_id) = query.template_id {
        params.push(format!("templateId={}", template_id));
    }
    if let Some(instance_key) = query.instance_key {
        params.push(format!("instanceKey={}", instance_key));
    }
    if !params.is_empty() {
        url = format!("{}?{}", url, params.join("&"));
    }
    match get_client().await.get(&url).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse app instance response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/instances: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// POST /api/apps/users - Upsert user app
pub async fn upsert_user_app(Json(payload): Json<serde_json::Value>) -> impl IntoResponse {
    let url = format!("{}/apps/users", get_agent_url());
    match get_client().await.post(&url).json(&payload).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse user app response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/users: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/apps/users - List user apps
pub async fn list_user_apps(Query(query): Query<AppQuery>) -> impl IntoResponse {
    let mut url = format!("{}/apps/users", get_agent_url());
    let mut params: Vec<String> = Vec::new();
    if let Some(user_id) = query.user_id {
        params.push(format!("userId={}", user_id));
    }
    if !params.is_empty() {
        url = format!("{}?{}", url, params.join("&"));
    }
    match get_client().await.get(&url).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse user apps response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/users: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// POST /api/apps/projects - Upsert project app
pub async fn upsert_project_app(Json(payload): Json<serde_json::Value>) -> impl IntoResponse {
    let url = format!("{}/apps/projects", get_agent_url());
    match get_client().await.post(&url).json(&payload).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse project app response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/projects: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/apps/projects - List project apps
pub async fn list_project_apps(Query(query): Query<AppQuery>) -> impl IntoResponse {
    let mut url = format!("{}/apps/projects", get_agent_url());
    let mut params: Vec<String> = Vec::new();
    if let Some(project_id) = query.project_id {
        params.push(format!("projectId={}", project_id));
    }
    if let Some(user_id) = query.user_id {
        params.push(format!("userId={}", user_id));
    }
    if !params.is_empty() {
        url = format!("{}?{}", url, params.join("&"));
    }
    match get_client().await.get(&url).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse project apps response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /apps/projects: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/projects - List projects (proxies to go-agent)
pub async fn list_projects() -> impl IntoResponse {
    let url = format!("{}/projects", get_agent_url());
    
    match get_client().await.get(&url).send().await {
        Ok(resp) => {
            match resp.json::<serde_json::Value>().await {
                Ok(projects) => Json(projects).into_response(),
                Err(e) => {
                    tracing::error!("Failed to parse projects response: {}", e);
                    (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
                }
            }
        }
        Err(e) => {
            tracing::error!("Failed to call go-agent /projects: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/projects/:id - Get project by ID
pub async fn get_project(Path(id): Path<String>) -> impl IntoResponse {
    let url = format!("{}/projects/{}", get_agent_url(), id);

    match get_client().await.get(&url).send().await {
        Ok(resp) => {
            if resp.status() == StatusCode::NOT_FOUND {
                return (StatusCode::NOT_FOUND, "Project not found").into_response();
            }
            match resp.json::<serde_json::Value>().await {
                Ok(project) => Json(project).into_response(),
                Err(e) => {
                    tracing::error!("Failed to parse project response: {}", e);
                    (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
                }
            }
        }
        Err(e) => {
            tracing::error!("Failed to call go-agent /projects/:id: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// GET /api/endpoints - List endpoints
pub async fn list_endpoints(Query(query): Query<ProjectQuery>) -> impl IntoResponse {
    let project_id = match query.project_id {
        Some(value) => value,
        None => {
            return (StatusCode::BAD_REQUEST, "projectId is required").into_response();
        }
    };

    let url = format!("{}/endpoints?projectId={}", get_agent_url(), project_id);

    match get_client().await.get(&url).send().await {
        Ok(resp) => match resp.json::<serde_json::Value>().await {
            Ok(result) => Json(result).into_response(),
            Err(e) => {
                tracing::error!("Failed to parse endpoints response: {}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
            }
        },
        Err(e) => {
            tracing::error!("Failed to call go-agent /endpoints: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}

/// POST /api/brain/search - Brain search (proxies to go-agent)
pub async fn brain_search(Json(req): Json<BrainSearchRequest>) -> impl IntoResponse {
    tracing::info!("Brain search: {} project: {:?}", req.query, req.project_id);
    
    let url = format!("{}/brain/search", get_agent_url());
    
    match get_client().await.post(&url).json(&req).send().await {
        Ok(resp) => {
            match resp.json::<serde_json::Value>().await {
                Ok(result) => Json(result).into_response(),
                Err(e) => {
                    tracing::error!("Failed to parse brain search response: {}", e);
                    (StatusCode::INTERNAL_SERVER_ERROR, "Failed to parse response").into_response()
                }
            }
        }
        Err(e) => {
            tracing::error!("Failed to call go-agent /brain/search: {}", e);
            (StatusCode::SERVICE_UNAVAILABLE, "Agent service unavailable").into_response()
        }
    }
}
