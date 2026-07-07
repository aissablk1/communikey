# Notes de développement — pièges rencontrés (§29)

Notes courtes sur les pièges d'outillage rencontrés en développant communikey, pour
ne pas les re-rencontrer. Faits vérifiés en session, pas de mémoire.

## Worktrees Superpowers — `worktree.baseRef` par défaut `fresh`

**Piège (rencontré le 2026-07-07)** : avec l'outillage de worktrees
(`superpowers:using-git-worktrees` / `EnterWorktree`), le mode par défaut **`fresh`**
crée la branche à partir d'**`origin/<branche>`** (la référence distante), **pas du
`HEAD` local**. Sur un dépôt dont le local est en avance sur `origin` (commits non
poussés), un worktree « fresh » repart donc **en arrière** et ne voit pas le travail
local récent.

**Contournement** : créer explicitement le worktree depuis la ref locale voulue —
```bash
git worktree add .worktrees/<nom> -b <branche> <ref-locale>   # ex. main, HEAD, ou un SHA
```
ou régler `worktree.baseRef` sur la ref locale attendue avant d'entrer le worktree.
Vérifier après création : `git -C .worktrees/<nom> log --oneline -1` doit pointer sur
le HEAD local attendu, pas sur un ancêtre distant.

**Règle** : après tout `worktree add`, **confirmer le SHA de base** avant de coder —
un worktree qui repart d'un ancêtre est la cause silencieuse de « mon code a disparu ».

## Package `main` unique — un fichier qui ne compile pas casse tout le paquet

communikey est un **seul paquet `main`** : `go test ./...` compile **tous** les
fichiers ensemble. Une erreur de compilation dans un fichier (même de test) fait
échouer la suite entière. Corollaire : écrire les tests par petits incréments et
lancer `go build ./...` tôt.

**Auteur** : Aïssa BELKOUSSA
