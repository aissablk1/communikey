# csend — lancer le nœud du bus en service

`csend serve` est le nœud réseau du bus (il dépose dans l'inbox local les messages
reçus). Pour qu'il tourne en permanence :

## macOS (launchd)

```sh
cp service/com.csend.serve.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.csend.serve.plist
# arrêter : launchctl unload ~/Library/LaunchAgents/com.csend.serve.plist
```

## Linux (systemd, service utilisateur)

```sh
mkdir -p ~/.config/systemd/user
cp service/csend-serve.service ~/.config/systemd/user/
systemctl --user enable --now csend-serve
# logs : journalctl --user -u csend-serve -f
```

## Notes
- Adapte le chemin du binaire (`which csend`) dans l'unité si besoin.
- En réseau (hors loopback), ajoute `--authz` + une `allowed.json` : le serveur
  n'accepte alors que des messages **E2E signés** par des expéditeurs autorisés (§38).
- Mono-machine, `csend serve` reste optionnel : la voie coopérative `inbox`/`recv`
  marche sans démon (fichiers).
