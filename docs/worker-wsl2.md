# Worker sous WSL2 (P1, ticket #125)

Le lanceur d'installation détecte WSL2 (`uname -r` ou `/proc/version` contenant `WSL2`) et ajoute des diagnostics. Linux natif (P0) n'exécute aucun d'eux. WSL1 est refusé (`unsupported_platform`). Aucun nouveau code de cause : les échecs réutilisent les codes stables de l'ADR 0021 et le terminal nomme le diagnostic (`Failed preflight <nom>`), jamais l'événement envoyé à Snitch.

| Diagnostic | Étape | Code de cause | Vérifie | Reprise affichée |
|---|---|---|---|---|
| `wsl-storage` | prerequisites | `volume_not_writable` | l'état du lanceur n'est pas sous `/mnt/<lettre>/` (disque Windows) | `cd ~`, supprimer le dossier, relancer depuis le dossier personnel Linux |
| `wsl-docker` | prerequisites | `docker_unavailable` | `docker info` répond depuis WSL2 | démarrer Docker Desktop et activer l'intégration WSL, ou démarrer dockerd |
| `snitch-reachability` | image | `network_unreachable` | Snitch répond depuis le runtime Docker (conteneur de l'image du Worker) | adresse résolue par Docker, pare-feu Windows, VPN |
| `transcription-reachability` | image | `network_unreachable` | le point d'accès de transcription répond depuis Docker | remplacer `localhost` par `host.docker.internal` ou l'adresse de la machine |
| `docker-cuda` | gpu | `gpu_unavailable` | `libcuda.so.1` chargeable dans un conteneur `--gpus all` | pilote NVIDIA Windows à jour, pas de pilote Linux dans WSL, `wsl --shutdown` |

Le volume du modèle reste un volume Docker nommé (système de fichiers Linux de WSL2), donc compatible UID 10001. Une réponse HTTP, même 4xx/5xx, prouve la joignabilité. Aucun modèle vocal n'est téléchargé à l'installation (ADR 0038). Sans GPU visible, la voix est indisponible et la transcription continue (comportement P0).

## Preuves automatisées

`cd app && go test -tags integration ./internal/httpapi -run TestInstaller` : faux `docker`, `uname`, `curl`, `nvidia-smi` et faux `/proc/version` (`SNITCH_PROC_VERSION`), préfixe de montage simulé (`SNITCH_WSL_MOUNT_PREFIX`), données synthétiques. Couvre détection (noyau, proc), WSL1, chaque diagnostic, l'indice `localhost`, le stockage `/mnt/c` et l'absence de diagnostic WSL sur Linux natif.

## Procédure de validation de bout en bout (non exécutée)

Non exécutée dans ce ticket : aucun hôte Windows/WSL2 n'était disponible. À jouer sur une machine Windows 11 avec WSL2, une distribution Ubuntu, Docker (Desktop avec intégration WSL, ou Engine) et, pour la voix, un GPU NVIDIA avec pilote Windows compatible WSL. Utiliser une instance Snitch d'essai et des tâches synthétiques.

1. Dans Snitch (Admin > Workers), générer la commande d'installation (HTTPS, adresse joignable depuis Docker).
2. Dans le terminal WSL2, depuis `~`, coller la commande. Attendu : « WSL2 detected », phases réussies, Worker connecté dans l'interface.
3. Nominal : un job de transcription synthétique est traité. Aucun modèle vocal n'est téléchargé ni testé par l'installation, même avec GPU (aucune question, aucun jeton) ; la voix devient prête quand un modèle est installé par ordre depuis Snitch (Admin > Voix).
4. Stockage : `cd /mnt/c/Users/<vous>` avec `HOME` pointant ici (`HOME=$PWD bash …`) doit échouer avec `wsl-storage`; le message de reprise est suivi.
5. Docker : arrêter Docker Desktop, relancer : `wsl-docker` / `docker_unavailable`; redémarrer, relancer : reprise idempotente.
6. Réseau : configurer un point d'accès `http://localhost:…` : `transcription-reachability` / `network_unreachable`; corriger avec `host.docker.internal`, relancer.
7. CUDA : sur une machine sans pilote Windows compatible, `docker-cuda` / `gpu_unavailable`.
8. Consigner résultats, version de Windows, WSL (`wsl --version`), Docker et pilote dans le bilan de #125.
