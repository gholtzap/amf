use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};

use crate::config::AusfConfig;

#[derive(Clone)]
pub struct AusfClient {
    client: Client,
    ausf_uri: Option<String>,
}

impl AusfClient {
    pub async fn new(config: &AusfConfig) -> Result<Self> {
        Ok(Self {
            client: Client::new(),
            ausf_uri: config.uri.clone(),
        })
    }

    pub async fn authenticate(&self, request: AuthenticationRequest) -> Result<AuthenticationResponse> {
        Ok(AuthenticationResponse {
            auth_result: AuthResult::Success,
            kseaf: vec![],
            supi: String::new(),
        })
    }

    pub async fn confirm_auth(&self, request: AuthConfirmRequest) -> Result<AuthConfirmResponse> {
        Ok(AuthConfirmResponse {
            auth_result: AuthResult::Success,
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationRequest {
    pub suci: String,
    pub serving_network_name: String,
    pub resynchronization_info: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationResponse {
    pub auth_result: AuthResult,
    pub kseaf: Vec<u8>,
    pub supi: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthConfirmRequest {
    pub res_star: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthConfirmResponse {
    pub auth_result: AuthResult,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AuthResult {
    Success,
    Failure,
    Ongoing,
}
