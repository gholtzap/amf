use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};

#[derive(Clone)]
pub struct SmfClient {
    client: Client,
}

impl SmfClient {
    pub async fn new() -> Result<Self> {
        Ok(Self {
            client: Client::new(),
        })
    }

    pub async fn create_sm_context(&self, request: SmContextCreateRequest) -> Result<SmContextCreateResponse> {
        Ok(SmContextCreateResponse {
            sm_context_id: String::new(),
            pdu_session_id: 0,
        })
    }

    pub async fn update_sm_context(&self, sm_context_id: &str, request: SmContextUpdateRequest) -> Result<()> {
        Ok(())
    }

    pub async fn release_sm_context(&self, sm_context_id: &str) -> Result<()> {
        Ok(())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmContextCreateRequest {
    pub supi: String,
    pub pdu_session_id: u8,
    pub dnn: String,
    pub s_nssai: Snssai,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Snssai {
    pub sst: u8,
    pub sd: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmContextCreateResponse {
    pub sm_context_id: String,
    pub pdu_session_id: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmContextUpdateRequest {
    pub n2_sm_info: Option<Vec<u8>>,
    pub n1_sm_message: Option<Vec<u8>>,
}
