mod auth;
mod commands;

use anyhow::Result;
use clap::{Parser, Subcommand};

/// Bridge CLI — query customer behavioral intelligence from the command line
#[derive(Parser)]
#[command(name = "bridge", version, about)]
struct Cli {
    /// Base URL for the Bridge API
    #[arg(long, default_value = "https://api.trybridge.dev", global = true)]
    api_base: String,

    /// Base URL for the Bridge web app
    #[arg(long, default_value = "https://app.trybridge.dev", global = true)]
    web_base: String,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Authenticate with Bridge via browser login
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
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Login { device } => commands::login::run(&cli.web_base, device).await,
        Commands::Logout => commands::logout::run(),
        Commands::Whoami => commands::whoami::run(&cli.api_base).await,
        Commands::Query { prompt } => commands::query::run(&prompt, &cli.api_base).await,
    }
}
