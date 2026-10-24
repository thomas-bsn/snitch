# Accès et configuration

## Installation et sessions locales

Le code d’installation apparaît dans le log JSON `one-time installation code` (champ `expires_in_minutes: 15`) au démarrage de `serve`. Il est utilisable une fois pendant 15 minutes ; ni la page ni l’API publique ne le renvoient. Aucun administrateur ni mot de passe par défaut n’est précréé. Une seule création réussit même si deux demandes arrivent ensemble. Après réussite, l’installation reste fermée après redémarrage ; utiliser ensuite « Se connecter » avec les identifiants personnels et « Se déconnecter » pour révoquer la session courante.

Si le navigateur ou le serveur s’interrompt **avant** la création, retrouver le code dans les logs du premier démarrage : un redémarrage conserve tout code encore valide et ne le réaffiche pas. Si ce code a été perdu ou a expiré, attendre son expiration puis redémarrer `serve` : un nouveau code est émis tant que l’installation n’est pas terminée. Le précédent est alors refusé. Une interruption **après** création ne rouvre jamais le wizard ; se reconnecter avec le compte créé. Aucun changement SQL manuel n’est nécessaire.

Les mots de passe utilisent Argon2id version 19 : mémoire 32 MiB, trois itérations, parallélisme 1, sel aléatoire de 16 octets et résultat de 32 octets. Les limites de saisie comptent les caractères Unicode côté serveur. Les jetons opaques de session contiennent 32 octets aléatoires ; seul leur SHA-256 est persisté. Une session expire après sept jours fixes, sans prolongation automatique. Déconnexion et expiration rendent le jeton inutilisable, y compris s’il est renvoyé directement à l’API.

Le cookie `snitch_session` utilise `Path=/`, `HttpOnly`, `SameSite=Lax` et une durée maximale de 604800 secondes. `Secure` est activé par défaut ; conserver HTTPS et `COOKIE_SECURE=true` en production. Chaque mutation doit transmettre `Origin` égal à `PUBLIC_URL`. Les mutations authentifiées exigent aussi `X-CSRF-Token`, obtenu avec `GET /api/auth/me`. Les fonctions privées exigent une session approuvée et les routes `/api/admin/…` le rôle administrateur ; masquer un écran n’est jamais la protection d’accès.

L’installation, la connexion et l’inscription partagent une limite par adresse réseau distante : rafale de dix demandes, puis une demande toutes les six secondes ; `429` annonce `Retry-After: 6`. Deux traitements de mots de passe au plus s’exécutent simultanément par processus ; un excès renvoie `429 busy`. Les en-têtes de proxy transmis par le client sont ignorés : derrière un proxy, les clients partageant son adresse partagent actuellement cette limite. Les refus de connexion sont identiques pour compte inexistant, mot de passe incorrect et compte en attente. Les secrets saisis ne sont ni journalisés ni renvoyés.

## Inscriptions et approbation

Après installation, « Créer un compte » permet de demander un accès local. L’approbation est requise par défaut : l’écran confirme la mise en attente et aucune session n’est ouverte. Le demandeur ne peut ni se connecter ni accéder aux fonctions privées tant qu’un administrateur n’a pas cliqué sur « Approuver » dans la liste des inscriptions en attente. Cette liste et son compteur proviennent de la base. Le nouveau compte garde le rôle utilisateur.

Dans « Accès et Discord », l’administrateur peut décocher l’approbation requise puis enregistrer les réglages. Ce choix s’applique aux **nouveaux** comptes locaux et Discord ; les demandes déjà en attente restent à approuver explicitement. Un compte créé sans approbation peut se connecter aussitôt avec ses identifiants. Réactiver ce réglage soumet les prochaines inscriptions à approbation sans modifier les comptes existants. Le réglage API `registration_approval` est un booléen obligatoire : une valeur absente ou nulle est refusée sans modifier la politique.

Les noms sont uniques sans distinction de casse. Les inscriptions utilisent les mêmes limites Unicode que l’installation ; un nom indisponible renvoie `409 username_unavailable`. Avant installation, l’inscription renvoie `409 setup_required` et ne crée jamais d’administrateur. Les routes de liste, d’approbation et de réglages sont réservées aux administrateurs ; leurs mutations exigent origine et CSRF. Une approbation répétée ou un identifiant inexistant renvoie `404 not_pending`.

## Discord facultatif et sauvegarde des secrets

Après la création de l’administrateur, « Accès et Discord » permet de continuer sans Discord et de conserver une installation entièrement locale. Seuls les administrateurs peuvent consulter et modifier ces réglages. Pour préparer la connexion :

1. Créer votre propre application dans le [Discord Developer Portal](https://discord.com/developers/applications). Aucun identifiant du propriétaire de ce dépôt n’est nécessaire.
2. Dans les réglages OAuth2 de cette application, récupérer le **Client ID** et le **Client Secret**, puis ajouter l’URL de retour affichée par Snitch aux redirections autorisées : `PUBLIC_URL/api/auth/discord/callback`, avec l’origine publique exacte de votre installation.
3. Enregistrer ces deux valeurs dans Snitch. Le premier enregistrement exige un secret ; un identifiant seul ou des champs constitués uniquement d’espaces sont refusés. Le statut configuré confirme la présence des deux valeurs, sans vérifier leur validité auprès de Discord.
4. Pour une modification ultérieure, laisser le secret vide conserve sa valeur. Un nouveau secret le remplace. « Désactiver Discord » (ou enregistrer un identifiant vide) efface les deux valeurs ; les comptes locaux restent disponibles. Réactiver demandera de saisir à nouveau le secret.

Ces étapes suivent la [documentation OAuth2 officielle de Discord](https://docs.discord.com/developers/topics/oauth2). Enregistrer les réglages ne déclenche aucun appel fournisseur. La connexion ou la liaison ouvre ensuite Discord dans le navigateur ; le serveur échange le code avec Discord puis lit l’identité publique via la seule portée `identify`, sans accès aux messages ni à l’adresse e-mail. Aucun jeton fournisseur n’est renvoyé au frontend ni conservé en base.

Le secret n’est jamais renvoyé au navigateur après saisie, ni écrit dans les logs. L’interface n’en indique que la présence. Les valeurs sont chiffrées en PostgreSQL par AES-256-GCM avec nonce aléatoire et nom du réglage authentifié ; toute altération ou mauvaise clé empêche leur lecture. Conserver `MASTER_KEY` entre redémarrages et la sauvegarder **séparément** de PostgreSQL dans un emplacement sûr. Une restauration utilisable demande la base **et la même clé** ; une nouvelle clé ne déchiffre pas les anciens secrets. Le test complet de restauration reste prévu en phase 7.

## Connexion et liaison Discord

« Continuer avec Discord » annonce la redirection externe. Une nouvelle identité suit la politique d’inscription courante : par défaut, retour avec un message d’attente sans session autorisée ; après approbation, la même identité retrouve le même compte. Un pseudo fournisseur identique à un compte local n’associe jamais les deux comptes. Une collision avec le nom technique `discord_<identité>` crée un nom distinct.

Depuis un compte connecté, « Lier Discord » associe explicitement l’identité à ce compte. Le retour exige exactement la session qui a commencé la liaison : déconnexion, expiration ou nouvelle connexion imposent de recommencer. Une identité ne peut appartenir qu’à un compte ; une liaison concurrente ou un remplacement d’identité est refusé sans écraser l’association existante. Une liaison réussie conserve la session courante.

Le state et le témoin navigateur contiennent chacun 32 octets aléatoires et sont hachés en base ; ils expirent après dix minutes. Le state est consommé atomiquement une seule fois avant l’échange fournisseur. Un autre navigateur, un retour expiré ou rejoué est refusé. Le refus de consentement, les erreurs fournisseur et les pannes internes du retour OAuth ramènent le navigateur vers un message français permettant de réessayer ou de se connecter localement. Les clients API reçoivent des erreurs JSON (`400 invalid_oauth_state`, `403 link_session_required`, `409 discord_already_linked`, `502 discord_unavailable`, `503 unavailable`) ; le refus de consentement retourne `303 /?notice=discord_denied`. Les appels Discord ont leur propre limite de dix demandes puis une toutes les six secondes ; ils n’occupent pas les deux traitements de mots de passe locaux. Chaque échange sortant est borné par un délai de dix secondes.

La migration `00002_oauth_session.sql` invalide les anciens parcours OAuth en cours pour imposer la liaison à la session d’origine. Exécuter `migrate` avant de relancer la nouvelle API ; recommencer toute redirection engagée avant la mise à jour.

## Configuration et santé

| Variable | Usage |
| --- | --- |
| `DATABASE_URL` | URL PostgreSQL requise ; conserver les identifiants hors du dépôt. |
| `MASTER_KEY` | 32 octets aléatoires en base64, requis pour `serve` ; sauvegarder séparément de la base et conserver la même clé entre redémarrages. |
| `PUBLIC_URL` | Origine HTTP(S) publique sans chemin ni identifiants ; `http://localhost:8080` par défaut. |
| `COOKIE_SECURE` | `true` par défaut, exige HTTPS ; `false` uniquement pour le développement HTTP. |
| `LISTEN_ADDR` | Adresse et port d'écoute, `:8080` par défaut. `healthcheck` utilise la même variable. |
| `DISPLAY_NAME` | Nom affiché dans le navigateur, `Snitch` par défaut. |
| `EMBEDDINGS_URL` | URL HTTP(S) facultative du point d'accès compatible embeddings ; pas d'identifiants ni query string. |
| `EMBEDDINGS_MODEL` | Modèle déjà disponible chez le fournisseur ; aucun téléchargement implicite. Vide : pas d'appel au fournisseur. |

L’exemple [.env.example](../.env.example) prépare les variables du déploiement local Compose ; aucune image GHCR n’est encore publiée. Pour ce démarrage local, les embeddings sont facultatifs et peuvent rester absents. Aucun chemin vers la bibliothèque de l'hôte n'est nécessaire.

- `GET /api/health/live` : processus vivant, HTTP 200, `{"status":"ok"}`.
- `GET /api/health/ready` : schéma accessible, HTTP 200 ; `degraded` si embeddings absents ou injoignables, `ok` si leur sonde réussit. Une base inaccessible donne HTTP 503 avec `database_unavailable`.
- `GET /api/setup` : état d'installation, nom d'affichage et phases reportées ; aucun secret d'installation.
- `./bin/snitch healthcheck` : sortie 0 si l'API est prête, y compris en mode dégradé.

Les logs de l'application sont du JSON avec erreurs génériques, sans URL PostgreSQL, clé maître ni corps de requête. Le pool est limité à six connexions. SIGTERM laisse jusqu'à dix secondes aux requêtes en cours avant la fermeture du pool.

