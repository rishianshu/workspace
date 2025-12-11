//! Error types for the gateway

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;

#[derive(Debug)]
pub enum GatewayError {
    InternalError(String),
    BadRequest(String),
    ServiceUnavailable(String),
    Timeout,
}

impl IntoResponse for GatewayError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            GatewayError::InternalError(msg) => (StatusCode::INTERNAL_SERVER_ERROR, msg),
            GatewayError::BadRequest(msg) => (StatusCode::BAD_REQUEST, msg),
            GatewayError::ServiceUnavailable(msg) => (StatusCode::SERVICE_UNAVAILABLE, msg),
            GatewayError::Timeout => (StatusCode::GATEWAY_TIMEOUT, "Request timed out".to_string()),
        };

        let body = Json(json!({
            "error": message,
            "status": status.as_u16()
        }));

        (status, body).into_response()
    }
}
