import {
  List,
  ActionPanel,
  Action,
  Icon,
  getPreferenceValues,
  closeMainWindow,
  showToast,
  Toast,
} from "@raycast/api";
import { runAppleScript } from "@raycast/utils";
import { useEffect, useState } from "react";
import { execFile } from "child_process";
import { promisify } from "util";
import { homedir } from "os";

const pexecFile = promisify(execFile);

interface Workspace {
  name: string;
  path: string;
  group: string;
  fav: boolean;
}

interface Prefs {
  terminal: string;
  bagentPath: string;
}

function bagentBin(): string {
  const p = getPreferenceValues<Prefs>().bagentPath?.trim();
  return p && p.length > 0 ? p : `${homedir()}/.local/bin/bagent`;
}

// PATH enrichi de ~/.local/bin : les binaires (code…) n'y sont pas dans
// l'environnement non-interactif de Raycast.
function env() {
  return {
    ...process.env,
    PATH: `${homedir()}/.local/bin:${process.env.PATH ?? ""}`,
  };
}

async function loadWorkspaces(): Promise<Workspace[]> {
  const { stdout } = await pexecFile(bagentBin(), ["json"], { env: env() });
  return JSON.parse(stdout) as Workspace[];
}

// VSCode : ouverture directe via `open -a` (robuste, sans dépendre du CLI
// `code` ni du PATH transmis par Raycast).
async function openVSCode(path: string) {
  await pexecFile("open", ["-a", "Visual Studio Code", path]);
}

// Claude/Codex : nécessitent un terminal interactif. On ouvre le terminal
// choisi dans les préférences et on y lance `cd <path> && exec <tool>`.
async function openInTerminal(tool: "claude" | "codex", path: string) {
  const term = getPreferenceValues<Prefs>().terminal || "Terminal";
  const cmd = `cd ${shellQuote(path)} && exec ${tool}`;
  const script =
    term === "iTerm"
      ? `tell application "iTerm"
  activate
  create window with default profile
  tell current session of current window to write text ${appleScriptQuote(cmd)}
end tell`
      : `tell application "Terminal"
  activate
  do script ${appleScriptQuote(cmd)}
end tell`;
  await runAppleScript(script);
}

function shellQuote(s: string): string {
  return `'${s.replace(/'/g, `'\\''`)}'`;
}

function appleScriptQuote(s: string): string {
  return `"${s.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
}

async function run(action: () => Promise<void>) {
  try {
    await action();
    await closeMainWindow();
  } catch (e) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Échec de l'ouverture",
      message: String(e),
    });
  }
}

export default function Command() {
  const [items, setItems] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    loadWorkspaces()
      .then(setItems)
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false));
  }, []);

  if (error) {
    return (
      <List>
        <List.EmptyView
          icon={Icon.Warning}
          title="Le CLI bagent est introuvable"
          description="Installe bagent, puis relance cette commande. Si le binaire est ailleurs que ~/.local/bin, renseigne son chemin dans les préférences."
          actions={
            <ActionPanel>
              <Action.OpenInBrowser
                title="Voir l'installation de bagent"
                url="https://github.com/damienp199/bagent#installation"
              />
            </ActionPanel>
          }
        />
      </List>
    );
  }

  return (
    <List isLoading={loading} searchBarPlaceholder="Filtrer les workspaces…">
      {items.map((ws) => (
        <List.Item
          key={ws.path}
          icon={{ fileIcon: ws.path }}
          title={ws.name}
          subtitle={ws.group}
          keywords={[ws.group, ws.name]}
          accessories={ws.fav ? [{ icon: Icon.Star, tooltip: "Favori" }] : []}
          actions={
            <ActionPanel>
              <Action
                title="Ouvrir Dans Vscode"
                icon={Icon.Code}
                onAction={() => run(() => openVSCode(ws.path))}
              />
              <Action
                title="Ouvrir Dans Claude Code"
                icon={Icon.Terminal}
                shortcut={{ modifiers: ["cmd"], key: "return" }}
                onAction={() => run(() => openInTerminal("claude", ws.path))}
              />
              <Action
                title="Ouvrir Dans Codex"
                icon={Icon.Terminal}
                shortcut={{ modifiers: ["opt"], key: "return" }}
                onAction={() => run(() => openInTerminal("codex", ws.path))}
              />
              <Action.ShowInFinder
                path={ws.path}
                shortcut={{ modifiers: ["cmd"], key: "f" }}
              />
              <Action.CopyToClipboard
                title="Copier Le Chemin"
                content={ws.path}
                shortcut={{ modifiers: ["cmd"], key: "." }}
              />
            </ActionPanel>
          }
        />
      ))}
    </List>
  );
}
