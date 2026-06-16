# bagent — consignes pour agents

Ce fichier s'adresse à un agent (LLM) à qui un utilisateur demande **d'installer bagent** en
pointant ce repo. Suis-le tel quel.

## Plateforme

macOS uniquement (Apple Silicon **et** Intel). Refuse sur Linux/Windows.

## Installer (méthode unique recommandée)

Exécute exactement cette commande — rien d'autre :

```sh
curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh
```

Le script télécharge le binaire précompilé depuis la dernière release GitHub, le (re)signe
ad-hoc, et l'installe dans `~/.local/bin` par renommage atomique. **Go n'est pas requis côté
client.** Le binaire vient des Releases, pas d'une compilation locale.

Si lancé depuis un clone du repo (présence de `go.mod`) **et** que Go est installé, le même
script compile depuis les sources au lieu de télécharger. C'est le mode dev.

## Pièges — à respecter impérativement

- **N'installe jamais le binaire par copie en place (`cp`).** Sur macOS, écraser un binaire signé
  déjà en cache déclenche `zsh: killed` (cache de signature AMFI du noyau). Installe toujours par
  **renommage atomique** : écris dans un fichier temporaire du dossier cible (`.bagent.new`), puis
  `mv -f` vers la destination finale. `install.sh` fait déjà ça — ne le contourne pas.
- **Ne notarise pas, ne demande pas de compte Apple.** Inutile : un binaire récupéré via `curl`
  ne reçoit pas l'attribut `com.apple.quarantine`, donc Gatekeeper ne bloque pas.
- **N'improvise pas** un `git clone` + `go build` + copie manuelle. Utilise `install.sh`.
- Vérifie que `~/.local/bin` est dans le `PATH` ; sinon, indique à l'utilisateur de l'ajouter à
  `~/.zshrc`.

## Vérifier l'installation

```sh
bagent --help
```
