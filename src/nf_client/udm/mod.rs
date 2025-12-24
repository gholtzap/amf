use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};

use crate::config::UdmConfig;

#[derive(Clone)]
pub struct UdmClient {
    client: Client,
    udm_uri: Option<String>,
}

impl UdmClient {
    pub async fn new(config: &UdmConfig) -> Result<Self> {
        Ok(Self {
            client: Client::new(),
            udm_uri: config.uri.clone(),
        })
    }

    pub async fn get_am_data(&self, supi: &str) -> Result<AccessAndMobilitySubscriptionData> {
        Ok(AccessAndMobilitySubscriptionData {
            gpsis: None,
            subscribed_ue_ambr: None,
            nssai: None,
        })
    }

    pub async fn register_amf(&self, request: AmfRegistrationRequest) -> Result<()> {
        Ok(())
    }

    pub async fn deregister_amf(&self, supi: &str) -> Result<()> {
        Ok(())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessAndMobilitySubscriptionData {
    pub gpsis: Option<Vec<String>>,
    pub subscribed_ue_ambr: Option<Ambr>,
    pub nssai: Option<Nssai>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Ambr {
    pub uplink: String,
    pub downlink: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nssai {
    pub default_single_nssais: Vec<Snssai>,
    pub single_nssais: Option<Vec<Snssai>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Snssai {
    pub sst: u8,
    pub sd: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmfRegistrationRequest {
    pub supi: String,
    pub amf_instance_id: String,
}
