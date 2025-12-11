//! Rust Edge Gateway for Workspace Agent
//! 
//! High-performance HTTP/WebSocket gateway that proxies requests to the Go Agent Service.

mod routes;
mod proxy;
mod middleware;
mod config;
mod error;

use axum::{
    routing::{get, post},
    Router,
};
use std::net::SocketAddr;
use tower_http::{
    cors::{Any, CorsLayer},
    trace::TraceLayer,
};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::try_from_default_env()
            .unwrap_or_else(|_| "rust_gateway=debug,tower_http=debug".into()))
        .with(tracing_subscriber::fmt::layer())
        .init();

    // Load configuration
    let config = config::AppConfig::from_env();
    
    tracing::info!("Starting Rust Gateway on port {}", config.port);

    // Build CORS layer
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // Build router
    let app = Router::new()
        // Health check
        .route("/health", get(routes::health::health_check))
        // Agent API
        .route("/api/agent/chat", post(routes::chat::handle_chat))
        .route("/ws/agent/stream", get(routes::stream::handle_stream))
        // Actions API
        .route("/api/actions", post(routes::actions::handle_action))
        .route("/api/actions", get(routes::actions::list_actions))
        // Layers
        .layer(cors)
        .layer(TraceLayer::new_for_http());

    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], config.port));
    tracing::info!("Gateway listening on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
