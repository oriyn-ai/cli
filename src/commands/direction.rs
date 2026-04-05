use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Deserialize, Serialize)]
struct RecommendationItem {
    title: String,
    rationale: String,
    priority: String,
}

#[derive(Deserialize, Serialize)]
struct DirectionItem {
    id: String,
    recommendations: Vec<RecommendationItem>,
    derived_from: serde_json::Value,
    generated_at: String,
}

#[derive(Deserialize, Serialize)]
struct EnrichmentResponse {
    enrichment_status: String,
    data: Vec<DirectionItem>,
}

/// View prescriptive direction for a product.
pub async fn run(product_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/direction");

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
        .context("failed to parse direction response")?;

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
        println!("No direction data found.");
        return Ok(());
    }

    for direction in &resp.data {
        println!("Recommendations:");
        for rec in &direction.recommendations {
            println!("  [{}] {}", rec.priority.to_uppercase(), rec.title);
            println!("    {}", rec.rationale);
        }
        if let Some(sources) = direction.derived_from.as_array() {
            let labels: Vec<&str> = sources.iter().filter_map(|s| s.as_str()).collect();
            if !labels.is_empty() {
                println!();
                println!("Derived from:");
                for label in &labels {
                    println!("  - {label}");
                }
            }
        }
        println!();
    }
    Ok(())
}
