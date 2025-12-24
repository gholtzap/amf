use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub amf: AmfConfig,
    pub sbi: SbiConfig,
    pub ngap: NgapConfig,
    pub database: DatabaseConfig,
    pub nrf: NrfConfig,
    pub ausf: AusfConfig,
    pub udm: UdmConfig,
    pub security: SecurityConfig,
    pub timers: TimersConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmfConfig {
    pub amf_name: String,
    pub region_id: String,
    pub set_id: String,
    pub pointer: String,
    pub guami_list: Vec<Guami>,
    pub plmn_support_list: Vec<PlmnSupport>,
    pub relative_capacity: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Guami {
    pub plmn_id: PlmnId,
    pub amf_region_id: String,
    pub amf_set_id: String,
    pub amf_pointer: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnId {
    pub mcc: String,
    pub mnc: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnSupport {
    pub plmn_id: PlmnId,
    pub s_nssai_list: Vec<SNssai>,
    pub tai_list: Vec<Tai>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SNssai {
    pub sst: u8,
    pub sd: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tai {
    pub plmn_id: PlmnId,
    pub tac: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SbiConfig {
    pub bind_addr: String,
    pub scheme: String,
    pub registered_ip_addr: String,
    pub port: u16,
    pub api_root: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NgapConfig {
    pub bind_addr: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub uri: String,
    pub database_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NrfConfig {
    pub uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AusfConfig {
    pub uri: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UdmConfig {
    pub uri: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityConfig {
    pub integrity_order: Vec<String>,
    pub ciphering_order: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimersConfig {
    pub t3502: u32,
    pub t3510: u32,
    pub t3511: u32,
    pub t3512: u32,
    pub t3513: u32,
    pub t3516: u32,
    pub t3517: u32,
    pub t3519: u32,
    pub t3520: u32,
    pub t3521: u32,
    pub t3522: u32,
    pub t3525: u32,
    pub t3540: u32,
    pub t3550: u32,
    pub t3555: u32,
    pub t3560: u32,
    pub t3565: u32,
    pub t3570: u32,
}

pub async fn load_config(path: impl AsRef<Path>) -> Result<Config> {
    let content = tokio::fs::read_to_string(path).await?;
    let config: Config = serde_json::from_str(&content)?;
    Ok(config)
}
