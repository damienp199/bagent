# Réorganisation des onglets projets — mode « ordre »

## Problème

Les onglets sont, dans l'ordre : `◷ Récents` (index 0), `★ Favoris` (index 1),
puis un onglet par projet (index 2+). L'ordre des projets vient de l'ordre des
lignes préfixées `>` dans `~/.config/bagent/workspaces`. Aujourd'hui rien ne
permet de réordonner les projets sans éditer le fichier à la main.

## Objectif

Permettre de réordonner les onglets projets entre eux, avec un minimum de
friction, sans jamais déplacer Récents/Favoris (qui restent en positions 0-1).

## Comportement

### Déclenchement
- Touche `o` quand le focus est en haut (`focusBar`) **et** sur un onglet de
  type projet (`KindProjet`).
- Sur Récents/Favoris, `o` ne fait rien (rien à réordonner).

### Dans le mode « ordre »
- `←` / `→` (et `h` / `l`) déplacent l'onglet courant d'une position parmi les
  projets uniquement :
  - borne gauche = juste après Favoris (premier projet) ;
  - borne droite = dernier onglet ;
  - au-delà : aucun effet (pas de wrap).
- L'onglet actif suit son propre déplacement (`pageIdx` se repositionne sur le
  projet déplacé après reconstruction des pages).
- `⏎` ou `Échap` terminent le mode. L'ordre est appliqué au fur et à mesure :
  pas d'annulation.
- `q` / `Ctrl+C` quittent l'application comme partout ailleurs.

### Indicateur visuel
- L'onglet en cours de déplacement est encadré de chevrons orange :
  `ALPHA ‹BETA› GAMMA`. Le fond orange de l'onglet actif est conservé.
- Footer dédié : `‹/› déplacer · ⏎/échap terminer · q quit`.

### Persistance
- Chaque déplacement réécrit l'ordre des lignes `>` dans le fichier
  `workspaces`, puis `reload()` reconstruit les pages. Récents/Favoris ne sont
  pas dans ce fichier ; seules les bornes d'affichage les gardent en tête.

## Découpage technique

Le déplacement raisonne dans **l'espace des onglets visibles**, pas dans
l'espace des lignes du fichier : un projet dont le dossier a disparu (`isDir`
faux) est invisible mais resterait présent dans le fichier. Échanger par
position de ligne ferait glisser un onglet derrière ces projets « morts » sans
effet visible (décalage backend/écran). On échange donc le projet courant avec
le projet **affiché** voisin, et on purge les entrées mortes au démarrage.

1. **`workspace.go`**
   - `swapEntries(lines []string, x, y string) []string` : fonction **pure** qui
     échange les positions des lignes `x` et `y`. No-op si l'une est absente.
   - `swapProjects(a, b string) bool` : wrapper I/O — échange les positions des
     projets `a` et `b` dans le fichier, réécrit si l'ordre change.
   - `pruneDeadFile(path string) bool` : retire d'un fichier de chemins les
     lignes pointant vers un dossier inexistant (préfixe `>` conservé).
   - `pruneDeadEntries() bool` : applique `pruneDeadFile` à tous les fichiers de
     chemins persistés (`workspaces`, `favorites`). Les récents sont filtrés
     dynamiquement, donc rien à purger côté persistance.

2. **`tui.go`**
   - Nouveau `modeReorder` dans l'enum `uiMode`.
   - `newModel` appelle `pruneDeadEntries()` avant `buildPages()`.
   - `updateBar` : `o` sur un `KindProjet` passe en `modeReorder`.
   - `updateReorder(msg)` : route `←/→/h/l` vers le déplacement, `⏎/échap`
     vers la sortie du mode, `q/ctrl+c` vers quit.
   - `moveCurrentProject(dir)` : détermine le projet visible voisin
     (`pages[pageIdx+dir]` si `KindProjet`, sinon no-op — borne à Favoris),
     appelle `swapProjects`, puis `reload()` + `gotoProjet(parent)` pour faire
     suivre `pageIdx`.
   - `Update` route vers `updateReorder` quand `m.mode == modeReorder`. Cela
     court-circuite l'interception globale de `←/→` dans `updateList` (qui
     navigue normalement entre onglets).

3. **`view.go`**
   - `renderTabs` : en `modeReorder`, encadre l'onglet actif de chevrons
     orange (`stArrow`).
   - `footerKeys` : en `modeReorder`, affiche le hint dédié.

## Tests

- `TestSwapEntries` : table de cas sur la fonction pure (échange adjacents,
  échange distants à travers une entrée morte, entrée absente). Sans I/O.
- `TestSwapProjectsFile` : exerce `swapProjects` de bout en bout en redirigeant
  `HOME` (`t.Setenv`), sans toucher au fichier de config réel.
- `TestPruneDeadEntries` : vérifie la purge des dossiers morts dans `workspaces`
  **et** `favorites`, idempotence (no-op si rien à nettoyer).
- `TestReorderSkipsDeadProjects` : régression — déplacer un onglet par-dessus un
  projet mort ne crée pas de pas fantôme (`pageIdx` et affichage restent
  synchronisés).
- Routage UI : `o` en `focusBar`/projet → `modeReorder` ; `échap` → `modeList`.
  Construits sur un `model` aux `pages` fixées en mémoire (pas d'I/O).
- Rendu : un `model` en `modeReorder` affiche les chevrons autour du bon onglet.

## Hors périmètre

- Déplacement par glisser ou par saut multi-positions.
- Réorganisation des items à l'intérieur d'une page.
- Ajustement du calcul de fenêtre glissante des onglets pour les 2 colonnes des
  chevrons (débordement transitoire négligeable).
