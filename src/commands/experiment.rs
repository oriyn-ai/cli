use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

use crate::auth;

#[derive(Serialize)]
struct CreateExperimentRequest<'a> {
    hypothesis: &'a str,
}

#[derive(Deserialize)]
struct CreateExperimentResponse {
    experiment_id: String,
}

#[derive(Deserialize, Serialize)]
struct ExperimentResponse {
    id: String,
    hypothesis: String,
    status: String,
    created_by_email: String,
    summary: Option<ExperimentSummary>,
}

#[derive(Deserialize, Serialize)]
struct ExperimentSummary {
    verdict: String,
    confidence: f32,
    summary: String,
    persona_breakdown: Vec<PersonaBreakdownItem>,
}

#[derive(Deserialize, Serialize)]
struct PersonaBreakdownItem {
    persona: String,
    response: String,
    reasoning: String,
}

#[derive(Deserialize, Serialize)]
struct ExperimentListItem {
    id: String,
    hypothesis: String,
    status: String,
    verdict: Option<String>,
    created_by_email: String,
    created_at: String,
}

/// List experiments for a product.
pub async fn list(product_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/experiments");

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

    let items: Vec<ExperimentListItem> = response
        .json()
        .await
        .context("failed to parse experiments list")?;

    if json {
        println!(
            "{}",
            serde_json::to_string(&items).context("failed to serialize")?
        );
        return Ok(());
    }

    if items.is_empty() {
        println!("No experiments found.");
        return Ok(());
    }

    println!(
        "{:<38} {:<12} {:<10} {:<30} HYPOTHESIS",
        "ID", "STATUS", "VERDICT", "RUN BY"
    );
    for item in &items {
        let verdict = item.verdict.as_deref().unwrap_or("-");
        let hypothesis_truncated: String = item.hypothesis.chars().take(50).collect();
        println!(
            "{:<38} {:<12} {:<10} {:<30} {}",
            item.id, item.status, verdict, item.created_by_email, hypothesis_truncated
        );
    }
    Ok(())
}

/// Get a specific experiment's results.
pub async fn get(product_id: &str, experiment_id: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();
    let url = format!("{api_base}/products/{product_id}/experiments/{experiment_id}");

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

    let experiment: ExperimentResponse = response
        .json()
        .await
        .context("failed to parse experiment response")?;

    if json {
        println!(
            "{}",
            serde_json::to_string(&experiment).context("failed to serialize")?
        );
    } else {
        println!("Hypothesis: {}", experiment.hypothesis);
        println!("Status:     {}", experiment.status);
        println!("Run by:     {}", experiment.created_by_email);
        println!();
        print_results(&experiment);
    }
    Ok(())
}

/// Run a hypothesis experiment against product personas.
pub async fn run(product_id: &str, hypothesis: &str, api_base: &str, json: bool) -> Result<()> {
    let token = auth::get_valid_access_token().await?;
    let client = reqwest::Client::new();

    // Create experiment
    let create_url = format!("{api_base}/products/{product_id}/experiments");

    let response = client
        .post(&create_url)
        .bearer_auth(&token)
        .json(&CreateExperimentRequest { hypothesis })
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

    let created: CreateExperimentResponse = response
        .json()
        .await
        .context("failed to parse create experiment response")?;

    if !json {
        println!("Experiment started ({})", created.experiment_id);
        println!("Polling for results...");
    }

    // Poll for results
    let poll_url = format!(
        "{api_base}/products/{product_id}/experiments/{}",
        created.experiment_id
    );

    loop {
        tokio::time::sleep(std::time::Duration::from_secs(2)).await;

        let response = client
            .get(&poll_url)
            .bearer_auth(&token)
            .send()
            .await
            .context("failed to poll experiment status")?;

        let status = response.status();
        if !status.is_success() {
            let body = response
                .text()
                .await
                .unwrap_or_else(|_| "(no body)".to_string());
            anyhow::bail!("API returned {status}: {body}");
        }

        let experiment: ExperimentResponse = response
            .json()
            .await
            .context("failed to parse experiment response")?;

        match experiment.status.as_str() {
            "processing" => {
                if !json {
                    print!(".");
                }
                continue;
            }
            "failed" => {
                anyhow::bail!("Experiment failed");
            }
            "complete" => {
                if json {
                    println!(
                        "{}",
                        serde_json::to_string(&experiment)
                            .context("failed to serialize experiment response")?
                    );
                } else {
                    println!();
                    print_results(&experiment);
                }
                return Ok(());
            }
            other => {
                anyhow::bail!("Unexpected experiment status: {other}");
            }
        }
    }
}

fn print_results(experiment: &ExperimentResponse) {
    let summary = match &experiment.summary {
        Some(s) => s,
        None => {
            println!("Experiment complete but no summary available.");
            return;
        }
    };

    let confidence_pct = (summary.confidence * 100.0).round() as u32;

    println!("Verdict:    {}", summary.verdict);
    println!("Confidence: {}%", confidence_pct);
    println!();
    println!("Summary:");
    println!("{}", summary.summary);
    println!();
    println!("Persona Breakdown:");
    for item in &summary.persona_breakdown {
        println!("  {} ({}): {}", item.persona, item.response, item.reasoning);
    }
}
