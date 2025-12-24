pub mod server;
pub mod handlers;
pub mod messages;

use anyhow::Result;
use bytes::Bytes;

pub use messages::*;

pub trait NgapMessage {
    fn encode(&self) -> Result<Bytes>;
    fn decode(data: &[u8]) -> Result<Self> where Self: Sized;
}
