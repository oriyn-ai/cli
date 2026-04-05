use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Deserialize, Serialize)]
struct PatternItem {
    id: String,
    title: String,
    description: String,
    frequency: String,
    significance: String,
    raw_sequence: serde_json::Value,
    generated_at: String,
}

#[derive(Deserialize, Serialize)]
struct EnrichmentResponse {
    enrichment_status: String,
    data: Vec<PatternItem>,
}

/// View behavioral patterns for a product.
pub async fn run(product_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/patterns");

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
        .context("failed to parse patterns response")?;

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
        println!("No patterns found.");
        return Ok(());
    }

    for pattern in &resp.data {
        println!("{}", pattern.title);
        println!("  {}", pattern.description);
        println!("  Frequency:    {}", pattern.frequency);
        println!("  Significance: {}", pattern.significance);
        if let Some(steps) = pattern.raw_sequence.as_array() {
            let labels: Vec<&str> = steps.iter().filter_map(|s| s.as_str()).collect();
            if !labels.is_empty() {
                println!("  Sequence:     {}", labels.join(" -> "));
            }
        }
        println!();
    }
    Ok(())
}
