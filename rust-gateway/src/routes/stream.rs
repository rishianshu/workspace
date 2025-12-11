//! WebSocket streaming endpoint

use axum::{
    extract::ws::{Message, WebSocket, WebSocketUpgrade},
    response::Response,
};
use futures::{sink::SinkExt, stream::StreamExt};

/// Handle WebSocket upgrade for streaming
pub async fn handle_stream(ws: WebSocketUpgrade) -> Response {
    ws.on_upgrade(handle_socket)
}

async fn handle_socket(mut socket: WebSocket) {
    tracing::info!("WebSocket connection established");
    
    while let Some(msg) = socket.recv().await {
        match msg {
            Ok(Message::Text(text)) => {
                tracing::debug!("Received: {}", text);
                
                // TODO: Forward to Go Agent Service and stream response
                // For now, echo back with a streaming simulation
                
                // Simulate streaming response
                let response_parts = vec![
                    "Processing",
                    " your",
                    " request",
                    "...",
                ];
                
                for part in response_parts {
                    if socket.send(Message::Text(part.to_string())).await.is_err() {
                        break;
                    }
                    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                }
                
                // Send completion signal
                let _ = socket.send(Message::Text("[DONE]".to_string())).await;
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
    
    tracing::info!("WebSocket connection closed");
}
