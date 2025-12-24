use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use std::sync::Arc;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RanContext {
    pub ran_id: String,
    pub ran_name: String,
    pub addr: SocketAddr,
    pub state: RanState,
    pub supported_ta_list: Vec<SupportedTa>,
    pub default_paging_drx: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum RanState {
    Disconnected,
    Connected,
    Active,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SupportedTa {
    pub tac: String,
    pub broadcast_plmn_list: Vec<BroadcastPlmn>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BroadcastPlmn {
    pub plmn_id: PlmnId,
    pub s_nssai_list: Vec<SNssai>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlmnId {
    pub mcc: String,
    pub mnc: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SNssai {
    pub sst: u8,
    pub sd: Option<String>,
}

#[derive(Clone)]
pub struct RanContextManager {
    contexts: Arc<DashMap<String, RanContext>>,
    addr_to_ran_id: Arc<DashMap<SocketAddr, String>>,
}

impl RanContextManager {
    pub fn new() -> Self {
        Self {
            contexts: Arc::new(DashMap::new()),
            addr_to_ran_id: Arc::new(DashMap::new()),
        }
    }

    pub fn create_ran_context(&self, ran_id: String, addr: SocketAddr) -> RanContext {
        let context = RanContext {
            ran_id: ran_id.clone(),
            ran_name: String::new(),
            addr,
            state: RanState::Disconnected,
            supported_ta_list: Vec::new(),
            default_paging_drx: None,
        };
        self.contexts.insert(ran_id.clone(), context.clone());
        self.addr_to_ran_id.insert(addr, ran_id);
        context
    }

    pub fn get(&self, ran_id: &str) -> Option<RanContext> {
        self.contexts.get(ran_id).map(|r| r.clone())
    }

    pub fn get_by_addr(&self, addr: &SocketAddr) -> Option<RanContext> {
        self.addr_to_ran_id
            .get(addr)
            .and_then(|id| self.contexts.get(&*id).map(|r| r.clone()))
    }

    pub fn update(&self, context: RanContext) {
        self.addr_to_ran_id.insert(context.addr, context.ran_id.clone());
        self.contexts.insert(context.ran_id.clone(), context);
    }

    pub fn remove(&self, ran_id: &str) -> Option<RanContext> {
        self.contexts.remove(ran_id).map(|(_, context)| {
            self.addr_to_ran_id.remove(&context.addr);
            context
        })
    }
}

impl Default for RanContextManager {
    fn default() -> Self {
        Self::new()
    }
}
