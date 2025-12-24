use anyhow::{Result, anyhow};
use std::net::SocketAddr;
use tracing::{info, warn, debug};

use crate::config::Config;
use crate::context::{RanContextManager, RanContext, RanState};
use crate::context::{SupportedTa, BroadcastPlmn};
use crate::context::{PlmnId as ContextPlmnId, SNssai as ContextSNssai};
use super::messages::*;
use super::codec::*;

pub async fn handle_ng_setup_request(
    request: NgSetupRequest,
    config: &Config,
    ran_context: &RanContextManager,
    addr: SocketAddr,
) -> Result<NgapPdu> {
    info!("Processing NG Setup Request from {}", addr);
    debug!("Request: {:?}", request);

    let ran_node_id = format!("{}_{}",
        request.global_ran_node_id.plmn_identity.mcc,
        request.global_ran_node_id.ran_node_id
    );

    if !validate_supported_tai_list(&request.supported_ta_list, config) {
        warn!("TAI validation failed for RAN node {}", ran_node_id);

        return Ok(NgapPdu::UnsuccessfulOutcome(UnsuccessfulOutcome {
            procedure_code: 21,
            criticality: 0,
            value: NgapMessageValue::NgSetupFailure(NgSetupFailure {
                cause: Cause {
                    cause_type: 1,
                    cause_value: 0,
                },
                time_to_wait: Some(10),
                critical_diagnostics: None,
            }),
        }));
    }

    let ran_ctx = RanContext {
        ran_id: ran_node_id.clone(),
        ran_name: ran_node_id.clone(),
        addr: addr.to_string(),
        state: RanState::Connected,
        supported_ta_list: request.supported_ta_list.iter().map(|ta| {
            SupportedTa {
                tac: ta.tac.clone(),
                broadcast_plmn_list: ta.broadcast_plmn_list.iter().map(|bp| {
                    BroadcastPlmn {
                        plmn_id: ContextPlmnId {
                            mcc: bp.plmn_identity.mcc.clone(),
                            mnc: bp.plmn_identity.mnc.clone(),
                        },
                        s_nssai_list: bp.tai_slice_support_list.iter().map(|ss| {
                            ContextSNssai {
                                sst: ss.s_nssai.sst,
                                sd: ss.s_nssai.sd.clone(),
                            }
                        }).collect(),
                    }
                }).collect(),
            }
        }).collect(),
        default_paging_drx: Some(request.default_paging_drx),
    };

    ran_context.update(ran_ctx.clone());
    info!("RAN node {} successfully registered", ran_node_id);

    let response = NgSetupResponse {
        amf_name: config.amf.amf_name.clone(),
        served_guami_list: config.amf.guami_list.iter().map(|g| {
            ServedGuami {
                plmn_identity: PlmnIdentity {
                    mcc: g.plmn_id.mcc.clone(),
                    mnc: g.plmn_id.mnc.clone(),
                },
                amf_region_id: g.amf_region_id.clone(),
                amf_set_id: g.amf_set_id.clone(),
                amf_pointer: g.amf_pointer.clone(),
            }
        }).collect(),
        relative_amf_capacity: config.amf.relative_capacity,
        plmn_support_list: config.amf.plmn_support_list.iter().map(|ps| {
            PlmnSupportItem {
                plmn_identity: PlmnIdentity {
                    mcc: ps.plmn_id.mcc.clone(),
                    mnc: ps.plmn_id.mnc.clone(),
                },
                slice_support_list: ps.s_nssai_list.iter().map(|s| {
                    SliceSupportItem {
                        s_nssai: SNssai {
                            sst: s.sst,
                            sd: s.sd.clone(),
                        }
                    }
                }).collect(),
            }
        }).collect(),
    };

    Ok(NgapPdu::SuccessfulOutcome(SuccessfulOutcome {
        procedure_code: 21,
        criticality: 0,
        value: NgapMessageValue::NgSetupResponse(response),
    }))
}

fn validate_supported_tai_list(tai_list: &[SupportedTaItem], config: &Config) -> bool {
    if tai_list.is_empty() {
        return false;
    }

    for ta in tai_list {
        let mut found = false;

        for plmn_support in &config.amf.plmn_support_list {
            for configured_tai in &plmn_support.tai_list {
                for broadcast_plmn in &ta.broadcast_plmn_list {
                    if configured_tai.tac == ta.tac &&
                       configured_tai.plmn_id.mcc == broadcast_plmn.plmn_identity.mcc &&
                       configured_tai.plmn_id.mnc == broadcast_plmn.plmn_identity.mnc {
                        found = true;
                        break;
                    }
                }
                if found {
                    break;
                }
            }
            if found {
                break;
            }
        }

        if !found {
            return false;
        }
    }

    true
}

pub async fn handle_initial_ue_message() -> Result<()> {
    Ok(())
}

pub async fn handle_uplink_nas_transport() -> Result<()> {
    Ok(())
}

pub async fn handle_initial_context_setup_response() -> Result<()> {
    Ok(())
}

pub async fn handle_pdu_session_resource_setup_response() -> Result<()> {
    Ok(())
}

pub async fn handle_ue_context_release_request() -> Result<()> {
    Ok(())
}
