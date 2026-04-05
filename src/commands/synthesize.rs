use anyhow::{Context, Result};
use serde::Deserialize;

use crate::auth;

#[derive(Deserialize)]
struct StatusResponse {
    status: String,
}

/// Trigger context synthesis for a product.
pub async fn run(product_id: &str, api_base: &str) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/context");

    let response = client
        .post(&url)
        .bearer_auth(&token)
        .send()
        .await
        .context("failed to reach the Oriyn API")?;

    let status = response.status();
    if !status.is_success() {
        let body = response
            .text()
            .await
            .unwrap_or_else(|_| "(no body)".to_string());
        anyhow::bail!("API returned {status}: {body}");
    }

    let resp: StatusResponse = response
        .json()
        .await
        .context("failed to parse response")?;

    println!("Context synthesis {}: {}", resp.status, product_id);
    Ok(())
}
