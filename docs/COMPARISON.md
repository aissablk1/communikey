# csend face aux autres outils multi-agents

Comparaison **honnête** : ce que csend fait mieux, et ce que les autres font mieux. Objectif :
t'aider à **choisir le bon outil**, pas te vendre csend à tout prix. (Faits vérifiés 2026-07-01 ;
csend est en **alpha** — voir [SECURITY.md](../SECURITY.md).)

## Deux catégories à ne pas confondre

- **Packs de capacités** (ClaudeKit, awesome-claude-skills, etc.) = *ce que ton agent sait faire*
  (skills, commandes, subagents intra-session). **csend n'est pas dans cette catégorie** et ne les
  remplace pas — ils sont complémentaires.
- **Bus de coordination** = *comment des sessions d'agents vivantes se parlent*. C'est ici que joue
  csend : **hcom**, MCP Agent Mail, murmur, Agent Teams (natif)… et csend.

## Le vrai comparatif (bus de coordination)

| | **csend** | hcom | Agent Teams (natif) | murmur / Agent Mail |
|---|:---:|:---:|:---:|:---:|
| Sessions vivantes qui se parlent | ✅ | ✅ | ✅ (même provider) | ✅ |
| Cross-provider | ◐ Claude ✅, Codex/Gemini calibrés | ✅ ~10 CLIs | ❌ | ✅ |
| Cross-machine | ✅ LAN (TLS hybride PQC) | ✅ MQTT | ❌ | ◐ / ❌ |
| Chiffrement des messages | ✅ **par destinataire** | ✅ (PSK partagé) | ❌ | ❌ |
| **Authentification d'expéditeur (signé)** | ✅ **Ed25519** | ❌ *« not authorization »* | ❌ | ❌ |
| Post-quantique | ✅ **ML-KEM-768** | ❌ | ❌ | ❌ |
| Recovery souveraine (Shamir/BIP-39) | ✅ | ❌ | ❌ | ❌ |
| Largeur providers / intégration turnkey | ◐ | ✅ | ✅ (natif) | ◐ |
| Maturité / adoption | ◐ alpha, neuf | ✅ établi | expérimental | jeune |
| Licence / coût | Apache-2.0, libre | MIT, libre | natif Claude | MIT/Apache, libre |

## Où chaque outil gagne (honnête)

- **hcom gagne** sur la **largeur** (10 CLIs), l'**intégration turnkey** (livraison par hooks) et la
  **maturité**. Si tu veux « brancher vite N CLIs entre eux » sans exigence de sécurité forte, hcom
  est excellent.
- **Agent Teams (natif) gagne** si tu restes **100 % Claude Code** et veux zéro install.
- **csend gagne** dès que la **confiance** compte : tu veux savoir **qui** a envoyé un message
  (signature, pas un mot de passe partagé), un **chiffrement par destinataire**, une **résistance
  post-quantique** et une **identité/recovery souveraines**. C'est le bus d'agents **auditable**.

## Choisis…

- **csend** si : environnement **sensible / régulé / multi-tenant / adversarial**, souveraineté,
  besoin de **provenance** et de **non-répudiation**, ou allergie au lock-in.
- **hcom** si : tu veux la **plus grande largeur de CLIs** et l'intégration la plus simple, sans
  exigence de sécurité cryptographique forte.
- **Agent Teams** si : tout ton monde est **Claude Code** et tu veux le natif.

> csend est jeune et l'assouplit : il est **en retard** sur la largeur et la maturité, **en avance**
> sur la confiance cryptographique. On préfère te le dire que te le cacher.
