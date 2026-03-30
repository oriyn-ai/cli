use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Serialize)]
struct QueryRequest<'a> {
    prompt: &'a str,
}

#[derive(Deserialize)]
struct QueryResponse {
    answer: String,
}

/// Send a query to the Bridge API and print the response.
pub async fn run(prompt: &str, api_base: &str) -> Result<()> {
    let token = auth::get_api_key()?;

    let client = reqwest::Client::new();
    let url = format!("{api_base}/v1/query");

    let response = client
        .post(&url)
        .bearer_auth(&token)
        .json(&QueryRequest { prompt })
        .send()
        .await
        .context("failed to reach the Bridge API")?;

    let status = response.status();
    if !status.is_success() {
        let body = response
            .text()
            .await
            .unwrap_or_else(|_| "(no body)".to_string());
        anyhow::bail!("API returned {status}: {body}");
    }

    let body: QueryResponse = response
        .json()
        .await
        .context("failed to parse API response")?;

    println!("{}", body.answer);
    Ok(())
}
