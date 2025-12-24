use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeContext {
    pub amf_ue_ngap_id: u64,
    pub ran_ue_ngap_id: Option<u64>,
    pub supi: Option<String>,
    pub suci: Option<String>,
    pub guti: Option<Guti>,
    pub pei: Option<String>,
    pub state: UeState,
    pub registration_type: Option<RegistrationType>,
    pub security_context: Option<SecurityContext>,
    pub kamf: Option<Vec<u8>>,
    pub kseaf: Option<Vec<u8>>,
    pub nas_uplink_count: u32,
    pub nas_downlink_count: u32,
    pub tai: Option<Tai>,
    pub ecgi: Option<String>,
    pub ran_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum UeState {
    Deregistered,
    Registered,
    Connected,
    Idle,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum RegistrationType {
    Initial,
    MobilityUpdate,
    PeriodicUpdate,
    Emergency,
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
pub struct Tai {
    pub plmn_id: PlmnId,
    pub tac: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityContext {
    pub ksi: u8,
    pub abba: Vec<u8>,
    pub k_nas_int: Vec<u8>,
    pub k_nas_enc: Vec<u8>,
    pub integrity_algorithm: IntegrityAlgorithm,
    pub ciphering_algorithm: CipheringAlgorithm,
    pub ue_security_capability: UeSecurityCapability,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum IntegrityAlgorithm {
    NIA0,
    NIA1,
    NIA2,
    NIA3,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum CipheringAlgorithm {
    NEA0,
    NEA1,
    NEA2,
    NEA3,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeSecurityCapability {
    pub nr_integrity_protection_algorithms: Vec<IntegrityAlgorithm>,
    pub nr_encryption_algorithms: Vec<CipheringAlgorithm>,
}

#[derive(Clone)]
pub struct UeContextManager {
    contexts: Arc<DashMap<u64, UeContext>>,
    supi_to_amf_ue_id: Arc<DashMap<String, u64>>,
    next_amf_ue_ngap_id: Arc<parking_lot::Mutex<u64>>,
}

impl UeContextManager {
    pub fn new() -> Self {
        Self {
            contexts: Arc::new(DashMap::new()),
            supi_to_amf_ue_id: Arc::new(DashMap::new()),
            next_amf_ue_ngap_id: Arc::new(parking_lot::Mutex::new(1)),
        }
    }

    pub fn allocate_amf_ue_ngap_id(&self) -> u64 {
        let mut id = self.next_amf_ue_ngap_id.lock();
        let current = *id;
        *id += 1;
        current
    }

    pub fn create_ue_context(&self, amf_ue_ngap_id: u64) -> UeContext {
        let context = UeContext {
            amf_ue_ngap_id,
            ran_ue_ngap_id: None,
            supi: None,
            suci: None,
            guti: None,
            pei: None,
            state: UeState::Deregistered,
            registration_type: None,
            security_context: None,
            kamf: None,
            kseaf: None,
            nas_uplink_count: 0,
            nas_downlink_count: 0,
            tai: None,
            ecgi: None,
            ran_id: None,
        };
        self.contexts.insert(amf_ue_ngap_id, context.clone());
        context
    }

    pub fn get(&self, amf_ue_ngap_id: u64) -> Option<UeContext> {
        self.contexts.get(&amf_ue_ngap_id).map(|r| r.clone())
    }

    pub fn get_by_supi(&self, supi: &str) -> Option<UeContext> {
        self.supi_to_amf_ue_id
            .get(supi)
            .and_then(|id| self.contexts.get(&*id).map(|r| r.clone()))
    }

    pub fn update(&self, context: UeContext) {
        if let Some(ref supi) = context.supi {
            self.supi_to_amf_ue_id.insert(supi.clone(), context.amf_ue_ngap_id);
        }
        self.contexts.insert(context.amf_ue_ngap_id, context);
    }

    pub fn remove(&self, amf_ue_ngap_id: u64) -> Option<UeContext> {
        self.contexts.remove(&amf_ue_ngap_id).map(|(_, context)| {
            if let Some(ref supi) = context.supi {
                self.supi_to_amf_ue_id.remove(supi);
            }
            context
        })
    }
}

impl Default for UeContextManager {
    fn default() -> Self {
        Self::new()
    }
}
