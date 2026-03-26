mod commands;

use anyhow::Result;
use clap::{Parser, Subcommand};

/// Bridge CLI — query customer behavioral intelligence from the command line
#[derive(Parser)]
#[command(name = "bridge", version, about)]
struct Cli {
    /// Base URL for the Bridge API
    #[arg(long, default_value = "https://api.bridge.com", global = true)]
    api_base: String,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Authenticate with Bridge via OAuth
    Login,
    /// Query customer behavioral intelligence
    Query {
        /// The natural-language prompt to send
        prompt: String,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Login => commands::login::run().await,
        Commands::Query { prompt } => commands::query::run(&prompt, &cli.api_base).await,
    }
}
