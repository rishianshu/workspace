//! Chat endpoint - proxies to Go Agent Service

use axum::Json;
use serde::{Deserialize, Serialize};

use crate::proxy::grpc_client::{AgentServiceClient, ChatRequest as ClientChatRequest};

#[derive(Debug, Deserialize)]
pub struct ChatRequest {
    pub query: String,
    pub conversation_id: String,
    #[serde(default)]
    pub context_entities: Vec<String>,
    pub session_id: Option<String>,
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
    
    // Try to forward to Go Agent Service
    let client = AgentServiceClient::new("http://localhost:9000");
    
    let client_request = ClientChatRequest {
        query: request.query.clone(),
        conversation_id: request.conversation_id.clone(),
        context_entities: request.context_entities.clone(),
        session_id: request.session_id.clone(),
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
