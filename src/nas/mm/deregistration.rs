use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeregistrationRequest {
    pub deregistration_type: DeregistrationType,
    pub ng_ksi: Option<u8>,
    pub mobile_identity: Option<MobileIdentity>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeregistrationType {
    pub switch_off: bool,
    pub re_registration_required: bool,
    pub access_type: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MobileIdentity {
    Guti(String),
    Suci(String),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeregistrationAccept {}
