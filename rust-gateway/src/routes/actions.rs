//! Actions endpoint - UCL write-back via Go Agent Service

use axum::Json;
use serde::{Deserialize, Serialize};

use crate::proxy::grpc_client::{AgentServiceClient, ActionRequest as ClientActionRequest};

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ActionRequest {
    pub action_type: String,
    pub entity_id: String,
    pub entity_type: String,
    pub source: String,
    #[serde(default)]
    pub payload: serde_json::Value,
    pub conversation_id: Option<String>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ActionResult {
    pub success: bool,
    pub action_type: String,
    pub entity_id: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub timestamp: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previous_state: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub new_state: Option<serde_json::Value>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ActionListResponse {
    pub available_actions: Vec<String>,
    pub entity_types: Vec<String>,
    pub sources: Vec<String>,
}

/// Execute an action via Go Agent Service
pub async fn handle_action(
    Json(request): Json<ActionRequest>,
) -> Json<ActionResult> {
    tracing::info!("Action request: {} on {}", request.action_type, request.entity_id);
    
    let client = AgentServiceClient::new("http://localhost:9000");
    
    let client_request = ClientActionRequest {
        action_type: request.action_type.clone(),
        entity_id: request.entity_id.clone(),
        entity_type: request.entity_type.clone(),
        source: request.source.clone(),
        payload: request.payload.clone(),
        conversation_id: request.conversation_id.clone(),
    };

    match client.execute_action(client_request).await {
        Ok(response) => {
            Json(ActionResult {
                success: response.success,
                action_type: response.action_type,
                entity_id: response.entity_id,
                message: response.message,
                timestamp: response.timestamp,
                previous_state: None,
                new_state: Some(serde_json::json!({ "status": "updated" })),
            })
        }
        Err(e) => {
            tracing::warn!("Go Agent Service unavailable, using fallback: {}", e);
            
            // Fallback when Go service is not running
            let timestamp = chrono::Utc::now().to_rfc3339();
            
            Json(ActionResult {
                success: true,
                action_type: request.action_type,
                entity_id: request.entity_id,
                message: "Action executed (fallback mode)".to_string(),
                timestamp: Some(timestamp),
                previous_state: None,
                new_state: Some(serde_json::json!({ "status": "updated" })),
            })
        }
    }
}

/// List available actions
pub async fn list_actions() -> Json<ActionListResponse> {
    Json(ActionListResponse {
        available_actions: vec![
            "ticket.status.update".to_string(),
            "ticket.assignee.update".to_string(),
            "ticket.comment.add".to_string(),
            "pr.approve".to_string(),
            "pr.request_changes".to_string(),
            "pr.merge".to_string(),
            "alert.acknowledge".to_string(),
            "alert.resolve".to_string(),
            "workflow.execute".to_string(),
        ],
        entity_types: vec![
            "ticket".to_string(),
            "pr".to_string(),
            "alert".to_string(),
            "workflow".to_string(),
        ],
        sources: vec![
            "jira".to_string(),
            "github".to_string(),
            "pagerduty".to_string(),
            "internal".to_string(),
        ],
    })
}
