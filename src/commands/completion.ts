import type { Command } from 'commander';
import { reportAndExit } from '../lib/handle-error.ts';

const BASH = `# oriyn bash completion
_oriyn_complete() {
  local cur="\${COMP_WORDS[COMP_CWORD]}"
  local cmds="auth link unlink products personas patterns experiments sync status config open upgrade completion"
  COMPREPLY=( $(compgen -W "$cmds" -- "$cur") )
}
complete -F _oriyn_complete oriyn`;

const ZSH = `#compdef oriyn
_oriyn() {
  local -a cmds
  cmds=(
    'auth:Manage CLI authentication'
    'link:Link this directory to a product'
    'unlink:Remove the project link'
    'products:List products'
    'personas:List or detail personas'
    'patterns:List mined patterns'
    'experiments:List or run experiments'
    'sync:Run synthesis + enrichment'
    'status:One-screen diagnostic'
    'config:Show or update CLI config'
    'open:Open the web app'
    'upgrade:Upgrade the CLI'
    'completion:Print shell completion'
  )
  _describe 'command' cmds
}
_oriyn`;

const FISH = `# oriyn fish completion
complete -c oriyn -n "__fish_use_subcommand" -a "auth link unlink products personas patterns experiments sync status config open upgrade completion"`;

export const registerCompletion = (program: Command): void => {
  program
    .command('completion <shell>')
    .description('Print shell completion script for bash, zsh, or fish')
    .action(async (shell: string) => {
      try {
        const map: Record<string, string> = { bash: BASH, zsh: ZSH, fish: FISH };
        const script = map[shell.toLowerCase()];
        if (!script) {
          throw new Error(`Unsupported shell '${shell}'. Use bash, zsh, or fish.`);
        }
        process.stdout.write(`${script}\n`);
      } catch (err) {
        reportAndExit(err);
      }
    });
};

export default registerCompletion;
