//! Application configuration

use std::env;

#[derive(Debug, Clone)]
pub struct AppConfig {
    pub port: u16,
}

impl AppConfig {
    pub fn from_env() -> Self {
        dotenvy::dotenv().ok();
        
        Self {
            port: env::var("GATEWAY_PORT")
                .unwrap_or_else(|_| "8080".to_string())
                .parse()
                .expect("GATEWAY_PORT must be a number"),
        }
    }
}
