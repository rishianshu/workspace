//! gRPC client for communicating with Go Agent Service

use serde::{Deserialize, Serialize};
use std::time::Duration;

// Agent service client configuration
pub struct AgentServiceClient {
    endpoint: String,
    timeout: Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatRequest {
    pub query: String,
    pub conversation_id: String,
    #[serde(default)]
    pub context_entities: Vec<String>,
    pub session_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReasoningStep {
    pub step: i32,
    #[serde(rename = "type")]
    pub step_type: String,
    pub content: String,
    pub duration_ms: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Artifact {
    pub id: String,
    #[serde(rename = "type")]
    pub artifact_type: String,
    pub title: String,
    pub content: String,
    pub language: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatResponse {
    pub response: String,
    pub reasoning: Vec<ReasoningStep>,
    #[serde(default)]
    pub artifacts: Vec<Artifact>,
    #[serde(default)]
    pub citations: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionRequest {
    pub action_type: String,
    pub entity_id: String,
    pub entity_type: String,
    pub source: String,
    #[serde(default)]
    pub payload: serde_json::Value,
    pub conversation_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionResponse {
    pub success: bool,
    pub action_type: String,
    pub entity_id: String,
    pub message: String,
    pub timestamp: Option<String>,
}

impl AgentServiceClient {
    pub fn new(endpoint: &str) -> Self {
        Self {
            endpoint: endpoint.to_string(),
            timeout: Duration::from_secs(30),
        }
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Send a chat request to the Go Agent Service
    pub async fn chat(&self, request: ChatRequest) -> Result<ChatResponse, ClientError> {
        // For now, use HTTP/JSON until gRPC proto compilation is set up
        // This will be replaced with actual gRPC calls
        let client = reqwest::Client::new();
        
        let url = format!("{}/chat", self.endpoint);
        
        let response = client
            .post(&url)
            .json(&request)
            .timeout(self.timeout)
            .send()
            .await
            .map_err(|e| ClientError::ConnectionError(e.to_string()))?;

        if response.status().is_success() {
            response
                .json::<ChatResponse>()
                .await
                .map_err(|e| ClientError::ParseError(e.to_string()))
        } else {
            Err(ClientError::ServiceError(format!(
                "Service returned status: {}",
                response.status()
            )))
        }
    }

    /// Execute an action via the Go Agent Service
    pub async fn execute_action(&self, request: ActionRequest) -> Result<ActionResponse, ClientError> {
        let client = reqwest::Client::new();
        
        let url = format!("{}/action", self.endpoint);
        
        let response = client
            .post(&url)
            .json(&request)
            .timeout(self.timeout)
            .send()
            .await
            .map_err(|e| ClientError::ConnectionError(e.to_string()))?;

        if response.status().is_success() {
            response
                .json::<ActionResponse>()
                .await
                .map_err(|e| ClientError::ParseError(e.to_string()))
        } else {
            Err(ClientError::ServiceError(format!(
                "Service returned status: {}",
                response.status()
            )))
        }
    }
}

#[derive(Debug)]
pub enum ClientError {
    ConnectionError(String),
    ParseError(String),
    ServiceError(String),
    Timeout,
}

impl std::fmt::Display for ClientError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ClientError::ConnectionError(msg) => write!(f, "Connection error: {}", msg),
            ClientError::ParseError(msg) => write!(f, "Parse error: {}", msg),
            ClientError::ServiceError(msg) => write!(f, "Service error: {}", msg),
            ClientError::Timeout => write!(f, "Request timed out"),
        }
    }
}

impl std::error::Error for ClientError {}
