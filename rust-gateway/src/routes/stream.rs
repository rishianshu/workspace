//! WebSocket streaming endpoint for real-time chat responses

use axum::{
    extract::ws::{Message, WebSocket, WebSocketUpgrade},
    response::Response,
};
use futures::{sink::SinkExt, stream::StreamExt};
use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize)]
pub struct StreamRequest {
    pub query: String,
    pub conversation_id: String,
    #[serde(default)]
    pub context_entities: Vec<String>,
    pub session_id: Option<String>,
    pub provider: Option<String>,
    pub model: Option<String>,
    #[serde(default)]
    pub history: Vec<HistoryMessage>,
}

#[derive(Debug, Serialize)]
pub struct StreamChunk {
    #[serde(rename = "type")]
    pub chunk_type: String, // "delta", "reasoning", "artifact", "done"
    #[serde(skip_serializing_if = "Option::is_none")]
    pub delta: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reasoning: Option<ReasoningUpdate>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifact: Option<ArtifactChunk>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct HistoryMessage {
    pub role: String,
    pub content: String,
}

#[derive(Debug, Serialize)]
pub struct ReasoningUpdate {
    pub step: i32,
    pub step_type: String,
    pub content: String,
}

#[derive(Debug, Serialize)]
pub struct ArtifactChunk {
    pub id: String,
    pub title: String,
    pub content_delta: String,
}

/// Handle WebSocket upgrade for streaming
pub async fn handle_stream(ws: WebSocketUpgrade) -> Response {
    ws.on_upgrade(handle_socket)
}

async fn handle_socket(mut socket: WebSocket) {
    tracing::info!("WebSocket streaming connection established");
    
    while let Some(msg) = socket.recv().await {
        match msg {
            Ok(Message::Text(text)) => {
                tracing::debug!("Stream request: {}", text);
                
                // Parse request
                let request: StreamRequest = match serde_json::from_str(&text) {
                    Ok(r) => r,
                    Err(e) => {
                        let _ = socket.send(Message::Text(format!(r#"{{"type":"error","message":"{}"}}"#, e))).await;
                        continue;
                    }
                };
                
                // Forward to Go Agent Service and stream response
                match call_agent_stream(&request).await {
                    Ok(response) => {
                        // Stream thinking steps first
                        for (i, step) in ["Analyzing query", "Retrieving context", "Synthesizing response"].iter().enumerate() {
                            let chunk = StreamChunk {
                                chunk_type: "reasoning".to_string(),
                                delta: None,
                                reasoning: Some(ReasoningUpdate {
                                    step: (i + 1) as i32,
                                    step_type: if i == 0 { "analysis" } else if i == 1 { "retrieval" } else { "synthesis" }.to_string(),
                                    content: step.to_string(),
                                }),
                                artifact: None,
                            };
                            if socket.send(Message::Text(serde_json::to_string(&chunk).unwrap())).await.is_err() {
                                break;
                            }
                            tokio::time::sleep(tokio::time::Duration::from_millis(200)).await;
                        }
                        
                        // Stream response text in chunks (simulate token-by-token)
                        let words: Vec<&str> = response.split_whitespace().collect();
                        for (i, word) in words.iter().enumerate() {
                            let chunk = StreamChunk {
                                chunk_type: "delta".to_string(),
                                delta: Some(if i == 0 { word.to_string() } else { format!(" {}", word) }),
                                reasoning: None,
                                artifact: None,
                            };
                            if socket.send(Message::Text(serde_json::to_string(&chunk).unwrap())).await.is_err() {
                                break;
                            }
                            // Variable delay for natural feeling
                            tokio::time::sleep(tokio::time::Duration::from_millis(30 + (i % 50) as u64)).await;
                        }
                        
                        // Send done signal
                        let done = StreamChunk {
                            chunk_type: "done".to_string(),
                            delta: None,
                            reasoning: None,
                            artifact: None,
                        };
                        let _ = socket.send(Message::Text(serde_json::to_string(&done).unwrap())).await;
                    }
                    Err(e) => {
                        let _ = socket.send(Message::Text(format!(r#"{{"type":"error","message":"{}"}}"#, e))).await;
                    }
                }
            }
            Ok(Message::Close(_)) => {
                tracing::info!("WebSocket closed by client");
                break;
            }
            Err(e) => {
                tracing::error!("WebSocket error: {}", e);
                break;
            }
            _ => {}
        }
    }
    
    tracing::info!("WebSocket streaming connection closed");
}

/// Call Go Agent Service and get response (non-streaming for now, then we chunk it)
async fn call_agent_stream(request: &StreamRequest) -> Result<String, String> {
    let client = reqwest::Client::new();
    let agent_url = std::env::var("AGENT_SERVICE_URL").unwrap_or_else(|_| "http://localhost:9001".to_string());
    
    let response = client
        .post(format!("{}/chat", agent_url))
        .json(&serde_json::json!({
            "query": request.query,
            "conversation_id": request.conversation_id,
            "context_entities": request.context_entities,
            "session_id": request.session_id,
            "provider": request.provider,
            "model": request.model,
            "history": request.history,
        }))
        .send()
        .await
        .map_err(|e| format!("Failed to reach agent service: {}", e))?;
    
    if !response.status().is_success() {
        return Err(format!("Agent service error: {}", response.status()));
    }
    
    let body: serde_json::Value = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse response: {}", e))?;
    
    Ok(body["response"].as_str().unwrap_or("I understand. How can I help?").to_string())
}
