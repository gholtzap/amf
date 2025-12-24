use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NgSetupRequest {
    pub global_ran_node_id: GlobalRanNodeId,
    pub supported_ta_list: Vec<SupportedTaItem>,
    pub default_paging_drx: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalRanNodeId {
    pub plmn_identity: PlmnIdentity,
    pub ran_node_id: String,
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
