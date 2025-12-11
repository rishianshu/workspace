//! Application configuration

use std::env;

#[derive(Debug, Clone)]
pub struct AppConfig {
    pub port: u16,
    pub agent_service_url: String,
    pub nucleus_url: String,
}

impl AppConfig {
    pub fn from_env() -> Self {
        dotenvy::dotenv().ok();
        
        Self {
            port: env::var("GATEWAY_PORT")
                .unwrap_or_else(|_| "8080".to_string())
                .parse()
                .expect("GATEWAY_PORT must be a number"),
            agent_service_url: env::var("AGENT_SERVICE_URL")
                .unwrap_or_else(|_| "http://localhost:9000".to_string()),
            nucleus_url: env::var("NUCLEUS_URL")
                .unwrap_or_else(|_| "http://localhost:4000".to_string()),
        }
    }
}
