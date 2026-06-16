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

1. **`workspace.go`**
   - `reorderEntries(lines []string, target string, dir int) []string` :
     fonction **pure** qui échange l'entrée `target` (ligne complète, ex.
     `">"+parent`) avec sa voisine `>` dans le sens `dir` (±1). No-op si la
     cible est introuvable, au bord, ou si la voisine n'est pas une entrée `>`.
     C'est le cœur testable de la logique (comme `remapPath`).
   - `moveProject(parent string, dir int) bool` : wrapper I/O — charge les
     lignes, applique `reorderEntries`, réécrit le fichier si l'ordre a changé.
     Renvoie `true` si un déplacement a eu lieu.

2. **`tui.go`**
   - Nouveau `modeReorder` dans l'enum `uiMode`.
   - `updateBar` : `o` sur un `KindProjet` passe en `modeReorder`.
   - `updateReorder(msg)` : route `←/→/h/l` vers le déplacement, `⏎/échap`
     vers la sortie du mode, `q/ctrl+c` vers quit.
   - `moveCurrentProject(dir)` : appelle `moveProject`, puis `reload()` +
     `gotoProjet(parent)` pour faire suivre `pageIdx`.
   - `Update` route vers `updateReorder` quand `m.mode == modeReorder`. Cela
     court-circuite l'interception globale de `←/→` dans `updateList` (qui
     navigue normalement entre onglets).

3. **`view.go`**
   - `renderTabs` : en `modeReorder`, encadre l'onglet actif de chevrons
     orange (`stArrow`).
   - `footerKeys` : en `modeReorder`, affiche le hint dédié.

## Tests

- `reorder_test.go` — `TestReorderEntries` : table de cas sur la fonction pure
  (déplacement gauche/droite, bornes, cible absente). Déterministe, sans I/O.
- Routage UI : `o` en `focusBar`/projet → `modeReorder` ; `échap` → `modeList`.
  Construits sur un `model` aux `pages` fixées en mémoire (pas d'I/O).
- Rendu : un `model` en `modeReorder` affiche les chevrons autour du bon onglet.

`moveProject` est un wrapper I/O trivial sur le fichier global (comme
`addEntry`/`removeEntry`, non testés) ; sa logique réelle est couverte par
`reorderEntries`.

## Hors périmètre

- Déplacement par glisser ou par saut multi-positions.
- Réorganisation des items à l'intérieur d'une page.
- Ajustement du calcul de fenêtre glissante des onglets pour les 2 colonnes des
  chevrons (débordement transitoire négligeable).
