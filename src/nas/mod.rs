pub mod mm;
pub mod sm;
pub mod messages;
pub mod security;

use anyhow::Result;
use bytes::Bytes;

pub use messages::*;

pub trait NasMessage {
    fn encode(&self) -> Result<Bytes>;
    fn decode(data: &[u8]) -> Result<Self> where Self: Sized;
}
