use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistrationRequest {
    pub registration_type: RegistrationType,
    pub ng_ksi: u8,
    pub mobile_identity: MobileIdentity,
    pub ue_security_capability: Option<UeSecurityCapability>,
    pub requested_nssai: Option<Vec<SNssai>>,
    pub last_visited_registered_tai: Option<Tai>,
    pub ue_network_capability: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum RegistrationType {
    Initial,
    MobilityUpdate,
    PeriodicUpdate,
    Emergency,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MobileIdentity {
    Suci(String),
    FiveGGuti(Guti),
    Imei(String),
    STmsi(u32),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Guti {
    pub plmn_id: PlmnId,
    pub amf_region_id: String,
    pub amf_set_id: String,
    pub amf_pointer: String,
    pub tmsi: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnId {
    pub mcc: String,
    pub mnc: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeSecurityCapability {
    pub nr_integrity_protection_algorithms: Vec<u8>,
    pub nr_encryption_algorithms: Vec<u8>,
    pub eutra_integrity_protection_algorithms: Option<Vec<u8>>,
    pub eutra_encryption_algorithms: Option<Vec<u8>>,
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
pub struct RegistrationAccept {
    pub registration_result: RegistrationResult,
    pub guti: Option<Guti>,
    pub allowed_nssai: Option<Vec<SNssai>>,
    pub tai_list: Option<Vec<Tai>>,
    pub t3512_value: Option<u32>,
    pub t3502_value: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistrationResult {
    pub registration_result_value: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistrationReject {
    pub cause: u8,
    pub t3502_value: Option<u32>,
    pub t3346_value: Option<u32>,
}
