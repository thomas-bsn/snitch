# Accès et configuration

## Installation et sessions locales

Le code d’installation apparaît dans le log JSON au démarrage de `serve`, sur une ligne lisible dont le message donne l’adresse à ouvrir et le code (`one-time installation code: open <adresse> and enter <code> …`), avec aussi les champs `code`, `url` et `expires_in_minutes: 15`. L’adresse est `PUBLIC_URL` si elle est fournie, sinon `http://localhost:<port d’écoute>`. L’écran d’accueil de l’assistant indique la commande `docker compose logs app` mais n’affiche jamais le code ; `POST /api/setup/code` le vérifie dès sa saisie (200, ou `409 setup_closed_or_invalid_code` s’il est faux, expiré ou si l’installation est terminée) sans le consommer, avec la limite par adresse des entrées publiques. Il est utilisable une fois pendant 15 minutes ; ni la page ni l’API publique ne le renvoient. Aucun administrateur ni mot de passe par défaut n’est précréé. Une seule création réussit même si deux demandes arrivent ensemble. Après réussite, l’installation reste fermée après redémarrage ; utiliser ensuite « Se connecter » avec les identifiants personnels et « Se déconnecter » pour révoquer la session courante.

Si le navigateur ou le serveur s’interrompt **avant** la création, retrouver le code dans les logs du premier démarrage : un redémarrage conserve tout code encore valide et ne le réaffiche pas. Si ce code a été perdu ou a expiré, attendre son expiration puis redémarrer `serve` : un nouveau code est émis tant que l’installation n’est pas terminée. Le précédent est alors refusé. Une interruption **après** création ne rouvre jamais le wizard ; se reconnecter avec le compte créé. Aucun changement SQL manuel n’est nécessaire.

Les mots de passe utilisent Argon2id version 19 : mémoire 32 MiB, trois itérations, parallélisme 1, sel aléatoire de 16 octets et résultat de 32 octets. Les limites de saisie comptent les caractères Unicode côté serveur. Les jetons opaques de session contiennent 32 octets aléatoires ; seul leur SHA-256 est persisté. Une session expire après sept jours fixes, sans prolongation automatique. Déconnexion et expiration rendent le jeton inutilisable, y compris s’il est renvoyé directement à l’API.

Le cookie `snitch_session` utilise `Path=/`, `HttpOnly`, `SameSite=Lax` et une durée maximale de 604800 secondes. `Secure` est activé automatiquement quand la requête est en HTTPS (voir « Adresse publique » ci-dessous). Chaque mutation doit transmettre un `Origin` égal à l’adresse publique de la requête elle-même (schéma et hôte) ; une origine absente, `null` ou différente est refusée par `403 origin_rejected`. Les mutations authentifiées exigent aussi `X-CSRF-Token`, obtenu avec `GET /api/auth/me`. Les fonctions privées exigent une session approuvée et les routes `/api/admin/…` le rôle administrateur ; masquer un écran n’est jamais la protection d’accès.

L’installation, la connexion et l’inscription partagent une limite par adresse réseau distante : rafale de dix demandes, puis une demande toutes les six secondes ; `429` annonce `Retry-After: 6`. Deux traitements de mots de passe au plus s’exécutent simultanément par processus ; un excès renvoie `429 busy`. Les en-têtes de proxy transmis par le client sont ignorés : derrière un proxy, les clients partageant son adresse partagent actuellement cette limite. Les refus de connexion sont identiques pour compte inexistant, mot de passe incorrect et compte en attente. Les secrets saisis ne sont ni journalisés ni renvoyés.

## Adresse publique

Snitch démarre sans adresse publique renseignée : elle est déduite de chaque requête. Le schéma vient de la connexion (HTTPS si Snitch termine lui-même TLS) et l'hôte de l'en-tête `Host`. Cette adresse sert à la vérification d'origine des mutations, à l'adresse de retour Discord (affichée dans « Accès et Discord » et envoyée à Discord), à l'adresse inscrite dans le fichier de déploiement d'un Worker et au drapeau `Secure` du cookie de session. Un navigateur qui ouvre `https://snitch.exemple.org` ou `http://localhost:8080` voit donc l'adresse qu'il utilise.

**Derrière un mandataire inverse** qui termine HTTPS, Snitch ne voit que la requête HTTP interne. Il honore alors `X-Forwarded-Proto` et `X-Forwarded-Host` (première valeur de la liste), mais **uniquement si l'adresse TCP du pair** (jamais un en-tête) appartient à la liste des mandataires de confiance. Cette liste se règle avec `TRUSTED_PROXIES`, facultatif :

- absent : la boucle locale et les réseaux privés (127.0.0.0/8, ::1, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, fc00::/7, fe80::/10), c'est-à-dire un mandataire sur le même hôte ou sur un réseau de conteneurs. Un client venu d'Internet n'est jamais de confiance par défaut ;
- une liste d'adresses ou de plages (`172.18.0.0/16,203.0.113.7`) : restreint la confiance à ces pairs ;
- `none` : aucun en-tête de mandataire n'est honoré.

Le mandataire doit écraser (et non transmettre tels quels) les en-têtes `X-Forwarded-Proto` et `X-Forwarded-Host` reçus du client. Les valeurs mal formées (schéma autre que http/https, hôte contenant un chemin ou des caractères inattendus) sont ignorées. Une requête d'un pair non fiable voit ses en-têtes de mandataire ignorés, y compris pour la vérification d'origine : elle n'est jamais plus permissive qu'une requête directe. Le contrôle d'origine compare toujours l'en-tête `Origin` à l'adresse de la requête ; une page tierce ne peut pas poser les en-têtes `X-Forwarded-*` depuis un navigateur.

**Plusieurs mandataires (Caddy puis Traefik, notamment)** : un relais HTTP peut remplacer le schéma HTTPS par `http`. Derrière un pair de confiance uniquement, Snitch peut retrouver HTTPS si le navigateur envoie `Sec-Fetch-Site: same-origin` et une origine HTTPS avec exactement le même hôte et port que la requête. Pour une lecture GET/HEAD sans `Origin`, le `Referer` HTTPS du même hôte et port sert de repli. Seul le schéma peut être relevé vers HTTPS : aucun hôte n'est appris de ces en-têtes, aucune origine tierce n'est acceptée et aucune mutation sans `Origin` n'est autorisée. Les cookies, les liens Discord et les fichiers de Worker utilisent cette même déduction. Voir l'[ADR 0016](adr/0016-https-derriere-plusieurs-mandataires.md).

**Réglage facultatif** `PUBLIC_URL` : fixe l'adresse publique quand la déduction n'est pas fiable (mandataire qui ne transmet pas les en-têtes, navigateur sans métadonnées Fetch ou sans référent, accès par plusieurs noms). Il est prioritaire sur toute déduction et sur les en-têtes de mandataire. Les installations existantes qui le renseignent, avec ou sans `COOKIE_SECURE`, gardent exactement leur comportement. La limitation de débit reste fondée sur l'adresse du pair TCP ; elle n'utilise jamais les en-têtes de mandataire.

## Inscriptions et approbation

Après installation, « Créer un compte » permet de demander un accès local. L’approbation est requise par défaut : l’écran confirme la mise en attente et aucune session n’est ouverte. Le demandeur ne peut ni se connecter ni accéder aux fonctions privées tant qu’un administrateur n’a pas cliqué sur « Approuver » dans la liste des inscriptions en attente. Cette liste et son compteur proviennent de la base. Le nouveau compte garde le rôle utilisateur.

Dans « Accès et Discord », l’administrateur peut décocher l’approbation requise puis enregistrer les réglages. Ce choix s’applique aux **nouveaux** comptes locaux et Discord ; les demandes déjà en attente restent à approuver explicitement. Un compte créé sans approbation peut se connecter aussitôt avec ses identifiants. Réactiver ce réglage soumet les prochaines inscriptions à approbation sans modifier les comptes existants. Le réglage API `registration_approval` est un booléen obligatoire : une valeur absente ou nulle est refusée sans modifier la politique.

Les noms sont uniques sans distinction de casse. Les inscriptions utilisent les mêmes limites Unicode que l’installation ; un nom indisponible renvoie `409 username_unavailable`. Avant installation, l’inscription renvoie `409 setup_required` et ne crée jamais d’administrateur. Les routes de liste, d’approbation et de réglages sont réservées aux administrateurs ; leurs mutations exigent origine et CSRF. Une approbation répétée ou un identifiant inexistant renvoie `404 not_pending`.

## Discord facultatif et sauvegarde des secrets

Après la création de l’administrateur, « Accès et Discord » permet de continuer sans Discord et de conserver une installation entièrement locale. Seuls les administrateurs peuvent consulter et modifier ces réglages. Pour préparer la connexion :

1. Créer votre propre application dans le [Discord Developer Portal](https://discord.com/developers/applications). Aucun identifiant du propriétaire de ce dépôt n’est nécessaire.
2. Dans les réglages OAuth2 de cette application, récupérer le **Client ID** et le **Client Secret**, puis ajouter l’URL de retour affichée par Snitch aux redirections autorisées : `<adresse publique>/api/auth/discord/callback`, où l’adresse est celle par laquelle vous ouvrez Snitch (affichée telle quelle dans « Accès et Discord »).
3. Enregistrer ces deux valeurs dans Snitch. Le premier enregistrement exige un secret ; un identifiant seul ou des champs constitués uniquement d’espaces sont refusés. Le statut configuré confirme la présence des deux valeurs, sans vérifier leur validité auprès de Discord.
4. Pour une modification ultérieure, laisser le secret vide conserve sa valeur. Un nouveau secret le remplace. « Désactiver Discord » (ou enregistrer un identifiant vide) efface les deux valeurs ; les comptes locaux restent disponibles. Réactiver demandera de saisir à nouveau le secret.

Ces étapes suivent la [documentation OAuth2 officielle de Discord](https://docs.discord.com/developers/topics/oauth2). Enregistrer les réglages ne déclenche aucun appel fournisseur. La connexion ou la liaison ouvre ensuite Discord dans le navigateur ; le serveur échange le code avec Discord puis lit l’identité publique via la seule portée `identify`, sans accès aux messages ni à l’adresse e-mail. Aucun jeton fournisseur n’est renvoyé au frontend ni conservé en base.

Le secret n’est jamais renvoyé au navigateur après saisie, ni écrit dans les logs. L’interface n’en indique que la présence. Les valeurs sont chiffrées en PostgreSQL par AES-256-GCM avec nonce aléatoire et nom du réglage authentifié ; toute altération ou mauvaise clé empêche leur lecture. La clé maître est générée par l’application au premier démarrage et conservée dans le volume `secrets`, hors de la base ; le Super-administrateur (provisoirement tout Administrateur) l’exporte depuis l’administration pour la conserver **séparément** de PostgreSQL. Une restauration complète demande la base **et la même clé** ; une nouvelle clé ne déchiffre pas les anciens secrets. Une clé absente ou différente n’empêche pas le démarrage : les réglages illisibles sont traités comme non configurés (Discord est alors désactivé, les comptes locaux fonctionnent), l’administration les liste (`unreadable_secrets` dans la réponse des réglages) et les ressaisir suffit. Voir [déploiement](deployment.md#clé-maître) et [sauvegarde](backups.md). Le test complet de restauration reste prévu en phase 7.

## Connexion et liaison Discord

« Continuer avec Discord » annonce la redirection externe. Une nouvelle identité suit la politique d’inscription courante : par défaut, retour avec un message d’attente sans session autorisée ; après approbation, la même identité retrouve le même compte. Un pseudo fournisseur identique à un compte local n’associe jamais les deux comptes. Une collision avec le nom technique `discord_<identité>` crée un nom distinct.

Depuis un compte connecté, « Lier Discord » associe explicitement l’identité à ce compte. Le retour exige exactement la session qui a commencé la liaison : déconnexion, expiration ou nouvelle connexion imposent de recommencer. Une identité ne peut appartenir qu’à un compte ; une liaison concurrente ou un remplacement d’identité est refusé sans écraser l’association existante. Une liaison réussie conserve la session courante.

Le state et le témoin navigateur contiennent chacun 32 octets aléatoires et sont hachés en base ; ils expirent après dix minutes. Le state est consommé atomiquement une seule fois avant l’échange fournisseur. Un autre navigateur, un retour expiré ou rejoué est refusé. Le refus de consentement, les erreurs fournisseur et les pannes internes du retour OAuth ramènent le navigateur vers un message français permettant de réessayer ou de se connecter localement. Les clients API reçoivent des erreurs JSON (`400 invalid_oauth_state`, `403 link_session_required`, `409 discord_already_linked`, `502 discord_unavailable`, `503 unavailable`) ; le refus de consentement retourne `303 /?notice=discord_denied`. Les appels Discord ont leur propre limite de dix demandes puis une toutes les six secondes ; ils n’occupent pas les deux traitements de mots de passe locaux. Chaque échange sortant est borné par un délai de dix secondes.

La migration `00002_oauth_session.sql` invalide les anciens parcours OAuth en cours pour imposer la liaison à la session d’origine. Exécuter `migrate` avant de relancer la nouvelle API ; recommencer toute redirection engagée avant la mise à jour.

## Configuration et santé

| Variable | Usage |
| --- | --- |
| `DATABASE_URL` | URL PostgreSQL requise ; conserver les identifiants hors du dépôt. |
| `MASTER_KEY` | Facultative (32 octets aléatoires en base64). Sans valeur, `serve` génère la clé et la conserve dans `SECRETS_DIR` ; une valeur fournie est prioritaire et n’y est pas recopiée. Mal formée, elle est refusée. |
| `SECRETS_DIR` | Dossier du volume de secrets, `/data/secrets` par défaut ; `serve` y écrit `master_key` (mode 0600) quand `MASTER_KEY` est absente et refuse de démarrer s’il n’est pas inscriptible. |
| `PUBLIC_URL` | Facultative. Origine HTTP(S) publique sans chemin ni identifiants qui fixe l'adresse publique ; absente, elle est déduite de chaque requête. Prioritaire si fournie. |
| `COOKIE_SECURE` | Facultative. Absente, le cookie est sécurisé quand l'adresse publique est HTTPS. `true` ou `false` décide seul (`true` exige un `PUBLIC_URL` HTTPS s'il est fourni). |
| `TRUSTED_PROXIES` | Facultative. Adresses IP ou plages CIDR (séparées par des virgules) dont les en-têtes de mandataire sont honorés ; `none` n'en honore aucun. Par défaut : boucle locale et réseaux privés. |
| `LISTEN_ADDR` | Adresse et port d'écoute, `:8080` par défaut. `healthcheck` utilise la même variable. |
| `EMBEDDINGS_URL` | URL HTTP(S) facultative du point d'accès compatible embeddings ; pas d'identifiants ni query string. |
| `EMBEDDINGS_MODEL` | Modèle déjà disponible chez le fournisseur ; aucun téléchargement implicite. Vide : pas d'appel au fournisseur. |

Quand ce fournisseur est configuré, les nouvelles transcriptions sont ajoutées à une file durable. La page d’administration expose l’état de l’index, le rattrapage explicite des transcriptions existantes et la reconstruction vers un autre modèle. Les recherches hybrides combinent le classement textuel et vectoriel ; si le fournisseur est indisponible, elles signalent le repli plein texte.

L’exemple [.env.example](../.env.example) prépare les variables du déploiement local Compose ; aucune image GHCR n’est encore publiée. Pour ce démarrage local, les embeddings sont facultatifs et peuvent rester absents. Aucun chemin vers la bibliothèque de l'hôte n'est nécessaire.

- `GET /api/health/live` : processus vivant, HTTP 200, `{"status":"ok"}`.
- `GET /api/health/ready` : schéma accessible, HTTP 200 ; `degraded` si embeddings absents ou injoignables, `ok` si leur sonde réussit. Une base inaccessible donne HTTP 503 avec `database_unavailable`.
- `GET /api/setup` : état d'installation, Discord activé et phases reportées ; aucun secret d'installation. Il n'y a plus de nom d'instance : « Snitch » est le nom unique, et `DISPLAY_NAME` est ignorée si une installation existante la fournit encore.
- `GET /api/admin/setup-progress` (Administrateurs) : progression de l'installation guidée. Chaque étape facultative (`discord`, `semantic_search`, `worker`, `import`) est `done` si elle est réellement configurée, `skipped` si la personne l'a traversée sans la configurer, `todo` sinon ; `dismissed` indique que le bandeau « Finir la configuration » est fermé. `PUT` (`step`, `state` `done` ou `skipped`) note le passage ; `POST /api/admin/setup-progress/dismiss` ferme le bandeau pour de bon. Une installation existante à la mise à jour n'a pas de bandeau.
- `./bin/snitch healthcheck` : sortie 0 si l'API est prête, y compris en mode dégradé.

Les logs de l'application sont du JSON avec erreurs génériques, sans URL PostgreSQL, clé maître ni corps de requête. Le pool est limité à six connexions. SIGTERM laisse jusqu'à dix secondes aux requêtes en cours avant la fermeture du pool.


## Photos de profil Discord

Une photo de profil vient de l'une de ces sources, choisie dans Compte › Profil : **Discord** (synchronisée), **Ma photo** (importée) ou les initiales. Un compte créé par Discord affiche sa photo Discord ; un compte local qui lie Discord l'adopte s'il n'a pas de photo personnelle. Une photo importée est toujours prioritaire : la synchronisation ne la remplace jamais, et « Discord » permet d'y revenir sans perdre l'autre photo. Délier Discord arrête la synchronisation et supprime la photo Discord téléchargée ; la photo personnelle reste.

La synchronisation demande un **bot Discord** (Administration › Inscriptions › « Discord Bot — Profile Pictures »). Créez un bot dans le portail développeur de Discord (aucune permission ni intention privilégiée n'est nécessaire : il ne lit que les profils publics), copiez son jeton, puis « Connect Bot ». Le jeton est vérifié auprès de Discord avant d'être enregistré chiffré ; il n'est jamais réaffiché. Seul le Super-administrateur le configure, le teste (« Test Connection ») ou le supprime (« Disconnect », qui garde les photos). Les profils sont actualisés toutes les 24 heures, même hors connexion, juste après une création ou une liaison, à la demande de l'utilisateur (« Refresh Discord Picture », au plus une fois toutes les 30 secondes) et par « Sync Now » pour tous les comptes. Voir l'[ADR 0039](adr/0039-synchronisation-des-avatars-discord.md).
