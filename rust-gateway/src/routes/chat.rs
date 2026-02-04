//! Chat endpoint - proxies to Go Agent Service

use axum::Json;
use serde::{Deserialize, Serialize};
use std::env;

use crate::proxy::grpc_client::{AgentServiceClient, ChatRequest as ClientChatRequest};

#[derive(Debug, Deserialize)]
pub struct ChatRequest {
    pub query: String,
    pub conversation_id: String,
    #[serde(default)]
    pub context_entities: Vec<String>,
    pub session_id: Option<String>,
    #[serde(rename = "userId")]
    pub user_id: Option<String>,
    #[serde(rename = "projectId")]
    pub project_id: Option<String>,
    // LLM provider selection
    pub provider: Option<String>,
    pub model: Option<String>,
    // Conversation history for multi-turn context
    #[serde(default)]
    pub history: Vec<HistoryMessage>,
    // Attached files with content
    #[serde(default, rename = "attachedFiles")]
    pub attached_files: Vec<AttachedFile>,
}

#[derive(Debug, Deserialize)]
pub struct HistoryMessage {
    pub role: String,
    pub content: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct AttachedFile {
    pub name: String,
    #[serde(rename = "type")]
    pub file_type: String,
    pub content: String,
}

#[derive(Debug, Serialize)]
pub struct ReasoningStep {
    pub step: i32,
    #[serde(rename = "type")]
    pub step_type: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration_ms: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct Artifact {
    pub id: String,
    #[serde(rename = "type")]
    pub artifact_type: String,
    pub title: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub language: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct ChatResponse {
    pub response: String,
    pub reasoning: Vec<ReasoningStep>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifacts: Option<Vec<Artifact>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub citations: Option<Vec<String>>,
}

/// Handle chat request - forwards to Go Agent Service
pub async fn handle_chat(
    Json(request): Json<ChatRequest>,
) -> Json<ChatResponse> {
    tracing::info!("Chat request: {:?}", request.query);
    
    // Try to forward to Go Agent Service (HTTP)
    let agent_url = env::var("AGENT_SERVICE_URL").unwrap_or_else(|_| "http://localhost:9001".to_string());
    let client = AgentServiceClient::new(&agent_url);
    
    let client_request = ClientChatRequest {
        query: request.query.clone(),
        conversation_id: request.conversation_id.clone(),
        context_entities: request.context_entities.clone(),
        session_id: request.session_id.clone(),
        user_id: request.user_id.clone(),
        project_id: request.project_id.clone(),
        provider: request.provider.clone(),
        model: request.model.clone(),
        history: request.history.iter().map(|h| crate::proxy::grpc_client::HistoryMessage {
            role: h.role.clone(),
            content: h.content.clone(),
        }).collect(),
        attached_files: request.attached_files.iter().map(|f| crate::proxy::grpc_client::AttachedFile {
            name: f.name.clone(),
            file_type: f.file_type.clone(),
            content: f.content.clone(),
        }).collect(),
    };

    match client.chat(client_request).await {
        Ok(response) => {
            tracing::info!("Got response from Go Agent Service");
            
            let reasoning: Vec<ReasoningStep> = response.reasoning
                .into_iter()
                .map(|r| ReasoningStep {
                    step: r.step,
                    step_type: r.step_type,
                    content: r.content,
                    duration_ms: r.duration_ms,
                })
                .collect();

            let artifacts: Option<Vec<Artifact>> = if response.artifacts.is_empty() {
                None
            } else {
                Some(response.artifacts
                    .into_iter()
                    .map(|a| Artifact {
                        id: a.id,
                        artifact_type: a.artifact_type,
                        title: a.title,
                        content: a.content,
                        language: a.language,
                    })
                    .collect())
            };

            Json(ChatResponse {
                response: response.response,
                reasoning,
                artifacts,
                citations: if response.citations.is_empty() { None } else { Some(response.citations) },
            })
        }
        Err(e) => {
            tracing::warn!("Go Agent Service unavailable, using fallback: {}", e);
            
            // Fallback response when Go service is not running
            Json(ChatResponse {
                response: format!("Processing query: {}", request.query),
                reasoning: vec![
                    ReasoningStep {
                        step: 1,
                        step_type: "analysis".to_string(),
                        content: "Analyzing your request...".to_string(),
                        duration_ms: Some(100),
                    },
                ],
                artifacts: None,
                citations: None,
            })
        }
    }
}
