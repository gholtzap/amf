use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NgSetupRequest {
    pub global_ran_node_id: GlobalRanNodeId,
    pub supported_ta_list: Vec<SupportedTaItem>,
    pub default_paging_drx: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum GlobalRanNodeId {
    GNB(GlobalGnbId),
    NgENB(GlobalNgEnbId),
    N3IWF(GlobalN3iwfId),
    TNGF(GlobalTngfId),
    TWIF(GlobalTwifId),
    WAGF(GlobalWagfId),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalGnbId {
    pub plmn_identity: PlmnIdentity,
    pub gnb_id: GnbId,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum GnbId {
    GnbId { value: String, bit_length: u8 },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalNgEnbId {
    pub plmn_identity: PlmnIdentity,
    pub ng_enb_id: NgEnbId,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum NgEnbId {
    MacroNgEnbId(String),
    ShortMacroNgEnbId(String),
    LongMacroNgEnbId(String),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalN3iwfId {
    pub plmn_identity: PlmnIdentity,
    pub n3iwf_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalTngfId {
    pub plmn_identity: PlmnIdentity,
    pub tngf_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalTwifId {
    pub plmn_identity: PlmnIdentity,
    pub twif_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalWagfId {
    pub plmn_identity: PlmnIdentity,
    pub wagf_id: String,
}

impl GlobalRanNodeId {
    pub fn plmn_identity(&self) -> &PlmnIdentity {
        match self {
            GlobalRanNodeId::GNB(id) => &id.plmn_identity,
            GlobalRanNodeId::NgENB(id) => &id.plmn_identity,
            GlobalRanNodeId::N3IWF(id) => &id.plmn_identity,
            GlobalRanNodeId::TNGF(id) => &id.plmn_identity,
            GlobalRanNodeId::TWIF(id) => &id.plmn_identity,
            GlobalRanNodeId::WAGF(id) => &id.plmn_identity,
        }
    }

    pub fn ran_node_id(&self) -> String {
        match self {
            GlobalRanNodeId::GNB(id) => match &id.gnb_id {
                GnbId::GnbId { value, .. } => value.clone(),
            },
            GlobalRanNodeId::NgENB(id) => match &id.ng_enb_id {
                NgEnbId::MacroNgEnbId(v) => v.clone(),
                NgEnbId::ShortMacroNgEnbId(v) => v.clone(),
                NgEnbId::LongMacroNgEnbId(v) => v.clone(),
            },
            GlobalRanNodeId::N3IWF(id) => id.n3iwf_id.clone(),
            GlobalRanNodeId::TNGF(id) => id.tngf_id.clone(),
            GlobalRanNodeId::TWIF(id) => id.twif_id.clone(),
            GlobalRanNodeId::WAGF(id) => id.wagf_id.clone(),
        }
    }

    pub fn node_type(&self) -> &'static str {
        match self {
            GlobalRanNodeId::GNB(_) => "gNB",
            GlobalRanNodeId::NgENB(_) => "ng-eNB",
            GlobalRanNodeId::N3IWF(_) => "N3IWF",
            GlobalRanNodeId::TNGF(_) => "TNGF",
            GlobalRanNodeId::TWIF(_) => "TWIF",
            GlobalRanNodeId::WAGF(_) => "W-AGF",
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnIdentity {
    pub mcc: String,
    pub mnc: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SupportedTaItem {
    pub tac: String,
    pub broadcast_plmn_list: Vec<BroadcastPlmnItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BroadcastPlmnItem {
    pub plmn_identity: PlmnIdentity,
    pub tai_slice_support_list: Vec<SliceSupportItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SliceSupportItem {
    pub s_nssai: SNssai,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SNssai {
    pub sst: u8,
    pub sd: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NgSetupResponse {
    pub amf_name: String,
    pub served_guami_list: Vec<ServedGuami>,
    pub relative_amf_capacity: u8,
    pub plmn_support_list: Vec<PlmnSupportItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServedGuami {
    pub plmn_identity: PlmnIdentity,
    pub amf_region_id: String,
    pub amf_set_id: String,
    pub amf_pointer: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnSupportItem {
    pub plmn_identity: PlmnIdentity,
    pub slice_support_list: Vec<SliceSupportItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NgSetupFailure {
    pub cause: Cause,
    pub time_to_wait: Option<u8>,
    pub critical_diagnostics: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Cause {
    pub cause_type: u8,
    pub cause_value: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InitialUeMessage {
    pub ran_ue_ngap_id: u64,
    pub nas_pdu: Vec<u8>,
    pub user_location_info: UserLocationInfo,
    pub rrc_establishment_cause: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserLocationInfo {
    pub nr_cgi: Option<NrCgi>,
    pub tai: Tai,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NrCgi {
    pub plmn_identity: PlmnIdentity,
    pub nr_cell_identity: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tai {
    pub plmn_identity: PlmnIdentity,
    pub tac: String,
}
