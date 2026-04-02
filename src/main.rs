mod auth;
mod commands;

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
    Login {
        /// Use device code flow (for headless/SSH environments)
        #[arg(long)]
        device: bool,
    },
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

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Login { device } => commands::login::run(&cli.web_base, device).await,
        Commands::Logout => commands::logout::run(),
        Commands::Whoami => commands::whoami::run(&cli.api_base).await,
        Commands::Query { prompt } => commands::query::run(&prompt, &cli.api_base).await,
        Commands::Experiment { command } => match command {
            ExperimentCommands::Run {
                product,
                hypothesis,
                json,
            } => commands::experiment::run(&product, &hypothesis, &cli.api_base, json).await,
            ExperimentCommands::List { product, json } => {
                commands::experiment::list(&product, &cli.api_base, json).await
            }
            ExperimentCommands::Get {
                product,
                experiment,
                json,
            } => {
                commands::experiment::get(&product, &experiment, &cli.api_base, json).await
            }
        },
    }
}
