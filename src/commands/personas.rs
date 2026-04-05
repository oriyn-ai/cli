use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Deserialize, Serialize)]
struct PersonaItem {
    id: String,
    name: String,
    description: String,
    behavioral_traits: serde_json::Value,
    size_estimate: i32,
    generated_at: String,
}

#[derive(Deserialize, Serialize)]
struct EnrichmentResponse {
    enrichment_status: String,
    data: Vec<PersonaItem>,
}

/// View behavioral personas for a product.
pub async fn run(product_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/personas");

    let response = client
        .get(&url)
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

    let resp: EnrichmentResponse = response
        .json()
        .await
        .context("failed to parse personas response")?;

    if json {
        println!(
            "{}",
            serde_json::to_string(&resp).context("failed to serialize")?
        );
        return Ok(());
    }

    println!("Enrichment status: {}", resp.enrichment_status);
    println!();

    if resp.data.is_empty() {
        println!("No personas found.");
        return Ok(());
    }

    for persona in &resp.data {
        println!("{} (~{}% of users)", persona.name, persona.size_estimate);
        println!("  {}", persona.description);
        if let Some(traits) = persona.behavioral_traits.as_array() {
            for t in traits {
                if let Some(s) = t.as_str() {
                    println!("  - {s}");
                }
            }
        }
        println!();
    }
    Ok(())
}
