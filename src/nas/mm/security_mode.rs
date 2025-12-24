use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityModeCommand {
    pub selected_nas_security_algorithms: NasSecurityAlgorithms,
    pub ng_ksi: u8,
    pub replayed_ue_security_capabilities: UeSecurityCapabilities,
    pub imeisv_request: Option<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NasSecurityAlgorithms {
    pub type_of_integrity_protection_algorithm: u8,
    pub type_of_ciphering_algorithm: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeSecurityCapabilities {
    pub nr_integrity_protection_algorithms: Vec<u8>,
    pub nr_encryption_algorithms: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityModeComplete {
    pub imeisv: Option<String>,
    pub nas_message_container: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityModeReject {
    pub cause: u8,
}
