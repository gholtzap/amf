use anyhow::Result;
use mongodb::{Client, Database as MongoDatabase};
use serde::{Deserialize, Serialize};

use crate::config::DatabaseConfig;
use crate::context::{UeContext, RanContext};

#[derive(Clone)]
pub struct Database {
    client: Client,
    db: MongoDatabase,
}

impl Database {
    pub async fn new(config: &DatabaseConfig) -> Result<Self> {
        let client = Client::with_uri_str(&config.uri).await?;
        let db = client.database(&config.database_name);

        Ok(Self { client, db })
    }

    pub async fn save_ue_context(&self, context: &UeContext) -> Result<()> {
        let collection = self.db.collection::<UeContext>("ue_contexts");
        let filter = mongodb::bson::doc! { "amf_ue_ngap_id": context.amf_ue_ngap_id as i64 };
        let options = mongodb::options::ReplaceOptions::builder()
            .upsert(true)
            .build();
        collection.replace_one(filter, context).with_options(options).await?;
        Ok(())
    }

    pub async fn load_ue_contexts(&self) -> Result<Vec<UeContext>> {
        let collection = self.db.collection::<UeContext>("ue_contexts");
        let mut cursor = collection.find(mongodb::bson::doc! {}).await?;
        let mut contexts = Vec::new();

        use futures::stream::StreamExt;
        while let Some(result) = cursor.next().await {
            contexts.push(result?);
        }

        Ok(contexts)
    }

    pub async fn save_ran_context(&self, context: &RanContext) -> Result<()> {
        let collection = self.db.collection::<RanContext>("ran_contexts");
        let filter = mongodb::bson::doc! { "ran_id": &context.ran_id };
        let options = mongodb::options::ReplaceOptions::builder()
            .upsert(true)
            .build();
        collection.replace_one(filter, context).with_options(options).await?;
        Ok(())
    }

    pub async fn load_ran_contexts(&self) -> Result<Vec<RanContext>> {
        let collection = self.db.collection::<RanContext>("ran_contexts");
        let mut cursor = collection.find(mongodb::bson::doc! {}).await?;
        let mut contexts = Vec::new();

        use futures::stream::StreamExt;
        while let Some(result) = cursor.next().await {
            contexts.push(result?);
        }

        Ok(contexts)
    }
}
