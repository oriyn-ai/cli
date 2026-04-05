mod auth;
mod commands;
mod telemetry;

use std::sync::Arc;

use anyhow::Result;
use clap::{Parser, Subcommand};

/// Oriyn CLI — query customer behavioral intelligence from the command line
#[derive(Parser)]
#[command(name = "oriyn", version, about)]
struct Cli {
    /// Base URL for the Oriyn API
    #[arg(long, default_value = "https://api.oriyn.ai", global = true)]
    api_base: String,

    /// Base URL for the Oriyn web app
    #[arg(long, default_value = "https://app.oriyn.ai", global = true)]
    web_base: String,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Authenticate with Oriyn via browser login
    Login,
    /// Remove stored credentials
    Logout,
    /// Show the currently authenticated user
    Whoami,
    /// Query customer behavioral intelligence
    Query {
        /// The natural-language prompt to send
        prompt: String,
    },
    /// Run hypothesis experiments against product personas
    Experiment {
        #[command(subcommand)]
        command: ExperimentCommands,
    },
    /// Manage anonymous usage telemetry
    Telemetry {
        /// Disable telemetry
        #[arg(long)]
        disable: bool,
        /// Enable telemetry
        #[arg(long)]
        enable: bool,
        /// Show current telemetry status
        #[arg(long)]
        status: bool,
    },
}

#[derive(Subcommand)]
enum ExperimentCommands {
    /// Run a new experiment
    Run {
        /// The product ID to run the experiment against
        #[arg(long)]
        product: String,
        /// The hypothesis to test
        #[arg(long)]
        hypothesis: String,
        /// Output raw JSON (for agent/programmatic consumption)
        #[arg(long)]
        json: bool,
    },
    /// List experiments for a product
    List {
        /// The product ID
        #[arg(long)]
        product: String,
        /// Output raw JSON
        #[arg(long)]
        json: bool,
    },
    /// Get a specific experiment's results
    Get {
        /// The product ID
        #[arg(long)]
        product: String,
        /// The experiment ID
        #[arg(long)]
        experiment: String,
        /// Output raw JSON
        #[arg(long)]
        json: bool,
    },
}

const SENTRY_DSN: &str = "https://7a9c0f680579c791f90ecee37a16375f@o4510953905651712.ingest.us.sentry.io/4511156841283584";

/// Redact "Bearer <token>" patterns from a string.
fn redact_tokens(s: &str) -> String {
    let mut result = s.to_string();
    while let Some(idx) = result.find("Bearer ") {
        let start = idx + 7;
        let end = result[start..]
            .find(|c: char| c.is_whitespace() || c == '"' || c == '\'')
            .map(|i| start + i)
            .unwrap_or(result.len());
        if start < end {
            result.replace_range(start..end, "[REDACTED]");
        } else {
            break;
        }
    }
    result
}

/// Scrub sensitive data from a Sentry event before it leaves the process.
fn scrub_event(
    mut event: sentry::protocol::Event<'static>,
) -> sentry::protocol::Event<'static> {
    for exception in &mut event.exception.values {
        if let Some(ref mut value) = exception.value {
            *value = redact_tokens(value);
        }
    }
    for breadcrumb in &mut event.breadcrumbs.values {
        if let Some(ref mut message) = breadcrumb.message {
            *message = redact_tokens(message);
        }
    }
    event.extra.retain(|k, _| {
        let lower = k.to_lowercase();
        !["token", "key", "password", "secret", "authorization", "credential"]
            .iter()
            .any(|s| lower.contains(s))
    });
    event
}

/// True for errors the API cannot see: keychain failures and deserialization failures.
fn is_infra_error(e: &anyhow::Error) -> bool {
    let msg = format!("{:#}", e);
    msg.contains("failed to access OS keychain")
        || msg.contains("failed to store credentials in OS keychain")
        || msg.contains("failed to parse stored credentials")
}

#[tokio::main]
async fn main() -> Result<()> {
    let _sentry = sentry::init((SENTRY_DSN, sentry::ClientOptions {
        release: sentry::release_name!(),
        send_default_pii: false,
        environment: Some(
            if cfg!(debug_assertions) { "development" } else { "production" }.into(),
        ),
        before_send: Some(Arc::new(|event| Some(scrub_event(event)))),
        ..Default::default()
    }));

    if let Some(user_id) = telemetry::get_user_id() {
        sentry::configure_scope(|scope| {
            scope.set_user(Some(sentry::User {
                id: Some(user_id),
                ..Default::default()
            }));
        });
    }

    let cli = Cli::parse();
    let t = telemetry::Telemetry::new().await;

    let (cmd_name, result) = match cli.command {
        Commands::Login => {
            let res = commands::login::run(&cli.web_base, &cli.api_base).await;
            t.capture(
                "cli_login",
                serde_json::json!({ "success": res.is_ok() }),
            )
            .await;
            ("login", res)
        }
        Commands::Logout => {
            let res = commands::logout::run();
            telemetry::clear_user_id();
            t.capture("cli_logout", serde_json::json!({})).await;
            ("logout", res)
        }
        Commands::Whoami => ("whoami", commands::whoami::run(&cli.api_base).await),
        Commands::Query { prompt } => {
            let prompt_len = prompt.len();
            let res = commands::query::run(&prompt, &cli.api_base).await;
            t.capture(
                "cli_query",
                serde_json::json!({ "prompt_length": prompt_len }),
            )
            .await;
            ("query", res)
        }
        Commands::Experiment { command } => match command {
            ExperimentCommands::Run {
                product,
                hypothesis,
                json,
            } => {
                let res =
                    commands::experiment::run(&product, &hypothesis, &cli.api_base, json).await;
                t.capture(
                    "cli_experiment_created",
                    serde_json::json!({ "product_id": product }),
                )
                .await;
                ("experiment run", res)
            }
            ExperimentCommands::List { product, json } => {
                let res = commands::experiment::list(&product, &cli.api_base, json).await;
                t.capture(
                    "cli_experiment_listed",
                    serde_json::json!({ "product_id": product }),
                )
                .await;
                ("experiment list", res)
            }
            ExperimentCommands::Get {
                product,
                experiment,
                json,
            } => {
                let res =
                    commands::experiment::get(&product, &experiment, &cli.api_base, json).await;
                t.capture(
                    "cli_experiment_viewed",
                    serde_json::json!({ "product_id": product, "experiment_id": experiment }),
                )
                .await;
                ("experiment get", res)
            }
        },
        Commands::Telemetry {
            disable,
            enable,
            status,
        } => {
            telemetry::manage(disable, enable, status);
            return Ok(());
        }
    };

    if let Err(ref e) = result {
        t.capture(
            "cli_error",
            serde_json::json!({ "command": cmd_name, "error_type": format!("{:#}", e) }),
        )
        .await;

        if is_infra_error(e) {
            sentry::capture_error(e.as_ref() as &dyn std::error::Error);
        }
    }

    if result.is_err() {
        drop(_sentry);
    }

    result
}
