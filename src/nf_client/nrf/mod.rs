use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};

use crate::config::NrfConfig;

#[derive(Clone)]
pub struct NrfClient {
    client: Client,
    nrf_uri: String,
}

impl NrfClient {
    pub async fn new(config: &NrfConfig) -> Result<Self> {
        Ok(Self {
            client: Client::new(),
            nrf_uri: config.uri.clone(),
        })
    }

    pub async fn register(&self) -> Result<()> {
        Ok(())
    }

    pub async fn deregister(&self) -> Result<()> {
        Ok(())
    }

    pub async fn heartbeat(&self) -> Result<()> {
        Ok(())
    }

    pub async fn discover_nf(&self, nf_type: NfType) -> Result<Vec<NfProfile>> {
        Ok(Vec::new())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NfType {
    Amf,
    Smf,
    Ausf,
    Udm,
    Pcf,
    Nrf,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NfProfile {
    pub nf_instance_id: String,
    pub nf_type: NfType,
    pub nf_status: NfStatus,
    pub ipv4_addresses: Option<Vec<String>>,
    pub fqdn: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NfStatus {
    Registered,
    Suspended,
}
