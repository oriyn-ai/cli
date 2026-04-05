use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Deserialize, Serialize)]
struct ProductListItem {
    id: String,
    name: String,
    context_status: String,
}

#[derive(Deserialize, Serialize)]
struct ProductDetail {
    id: String,
    name: String,
    description: Option<String>,
    urls: Option<Vec<String>>,
    context: Option<serde_json::Value>,
    context_status: String,
    enrichment_status: String,
    created_at: String,
}

/// List all products for the authenticated user.
pub async fn list(api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products");

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

    let items: Vec<ProductListItem> = response
        .json()
        .await
        .context("failed to parse products list")?;

    if json {
        println!(
            "{}",
            serde_json::to_string(&items).context("failed to serialize")?
        );
        return Ok(());
    }

    if items.is_empty() {
        println!("No products found.");
        return Ok(());
    }

    println!("{:<38} {:<30} STATUS", "ID", "NAME");
    for item in &items {
        println!("{:<38} {:<30} {}", item.id, item.name, item.context_status);
    }
    Ok(())
}

/// Get details for a specific product.
pub async fn get(product_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}");

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

    let product: ProductDetail = response
        .json()
        .await
        .context("failed to parse product response")?;

    if json {
        println!(
            "{}",
            serde_json::to_string(&product).context("failed to serialize")?
        );
        return Ok(());
    }

    println!("ID:                {}", product.id);
    println!("Name:              {}", product.name);
    println!(
        "Description:       {}",
        product.description.as_deref().unwrap_or("-")
    );
    if let Some(urls) = &product.urls {
        if !urls.is_empty() {
            println!("URLs:              {}", urls.join(", "));
        }
    }
    println!("Context status:    {}", product.context_status);
    println!("Enrichment status: {}", product.enrichment_status);
    println!("Created:           {}", product.created_at);

    if let Some(context) = &product.context {
        println!();
        println!("Context:");
        println!(
            "{}",
            serde_json::to_string_pretty(context)
                .unwrap_or_else(|_| context.to_string())
        );
    }

    Ok(())
}
