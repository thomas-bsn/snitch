# Refonte du front : routeur, dépendances, internationalisation et transition

Statut : accepté pour la phase 6 bis (lot 1). Précise la section « Dépendances » et « Internationalisation » de la [spécification](../specs/phase-6bis.md) ; ne modifie ni le contrat HTTP ni les décisions métier des phases précédentes.

## Contexte

L'interface d'avant la phase 6 bis est un seul composant de plus de trois mille lignes, rendu par une feuille de style globale qui cible des éléments natifs (`button`, `input`, `h2`…). Le prototype de la direction « Atelier » a montré que cette feuille déforme tout composant neuf (boutons agrandis, avatar écrasé) : elle ne doit jamais être chargée avec la nouvelle interface. Il n'y a ni routeur (une ancre `#library`), ni adresse propre par écran, ni langue autre que le français, ni thème sombre.

Contraintes qui pèsent sur le choix :

- le serveur envoie une politique de sécurité stricte (`script-src 'self'; style-src 'self'`) : ni script ni style en ligne, pas d'élément `<style>` injecté ;
- l'application web est embarquée dans le binaire Go (`app/ui/dist`) et servie par lui, sans serveur de rendu ;
- chaque dépendance exécutée dans le navigateur entre dans les notices de distribution et doit porter son texte de licence ;
- l'ancienne interface doit rester utilisable pendant que les écrans sont reconstruits lot après lot, puis disparaître.

## Décision

**Deux points d'entrée, une seule application servie.** Vite construit deux pages : la nouvelle interface (`index.html`, adresse racine) et l'ancienne (`legacy/index.html`, servie sous `/legacy/`). Les deux ne partagent aucun module qui porte du CSS : la feuille de style globale, déplacée avec le code de l'ancienne interface, n'est chargée que par la page `/legacy/`. Le code neuf vit sous `web/src/app/` et n'importe jamais l'ancien. À la fin de la phase, supprimer `legacy/`, `web/src/legacy/`, `web/src/voice/` et `web/src/segments/` (réécrits) suffit. Le serveur renvoie la coque de la nouvelle interface pour toute adresse inconnue (qui n'est ni `/api/` ni un fichier embarqué) et la coque de l'ancienne pour toute adresse sous `/legacy/`.

**Routeur : `react-router` en mode déclaratif (`BrowserRouter`).** Adresses propres (`/library`, `/search?q=…`, `/admin/…`), historique du navigateur, redirection vers l'écran demandé après la connexion, paramètres d'adresse pour les filtres et pour le moment lu d'un clip. Pas de routage par hash : il empêcherait le serveur de distinguer les écrans et casserait les liens partagés.

**Style : Tailwind CSS v4** (plugin Vite, outil de construction seulement). Les jetons de design Atelier sont des variables CSS sémantiques (`--canvas`, `--side`, `--ink`, `--mute`, `--accent`…), redéfinies pour le thème sombre sous `:root[data-theme="dark"]`, et exposées comme couleurs Tailwind : un composant écrit `bg-side text-mute` et suit le thème sans variante `dark:`. Le thème est un attribut `data-theme` sur `<html>`, posé avant le premier affichage par un petit fichier script servi à part (`preferences-init.js`, car les scripts en ligne sont interdits) pour éviter tout flash. Police Inter variable, auto-hébergée par `@fontsource-variable/inter` (aucune requête vers un tiers).

**Primitives accessibles : Base UI (`@base-ui/react`).** Menus, fenêtres modales, panneaux, onglets, bascules : focus piégé, Échap, navigation au clavier, rôles et états ARIA, sans style. Le choix s'est fait sur un essai réel sous la politique de sécurité du serveur : la fenêtre modale de Radix UI injecte un élément `<style>` pour verrouiller le défilement, ce que `style-src 'self'` refuse (erreur de politique dans la console, verrouillage perdu) ; celle de Base UI applique ses styles par l'API du navigateur, sans aucune violation. Le menu du profil est écrit avec Base UI, et un parcours navigateur échoue si la console signale une violation de politique.

**Icônes : `lucide-react`** (ISC), importées une à une (arborescence élaguée), décoratives par défaut (`aria-hidden`) ; tout bouton qui n'a qu'une icône porte un libellé.

**Internationalisation : catalogues maison, sans bibliothèque.** `en.ts` est la référence, `fr.ts` doit avoir exactement les mêmes clés (le compilateur le vérifie), l'accès se fait par chemin typé (`t("menu.signOut")`) et la clé inconnue ne compile pas. Anglais par défaut, aucune détection de la langue du navigateur. La langue d'un utilisateur connecté est stockée dans son compte (colonne `users.language`, **nulle tant qu'aucun choix n'a été fait**, lue et écrite par `GET`/`PUT /api/account/preferences`, renvoyée aussi dans la session) ; celle d'un visiteur non connecté l'est dans le navigateur. Le compte gagne sur tous les appareils ; un compte sans choix adopte celui que le navigateur mémorise (choisi sur la page de connexion, par exemple). Le thème, lui, reste propre à l'appareil.

**Erreurs du serveur traduites à partir de leurs codes.** Le serveur renvoie déjà des codes stables (`{"error":"…"}`). Un inventaire a été tiré des gestionnaires HTTP (123 codes, dont ceux des retours Discord) ; `errors.en.ts` et `errors.fr.ts` donnent le texte de chacun, et un code inconnu retombe sur un message générique qui affiche le code. `scripts/check_error_codes.py`, lancé par la CI, échoue si un code du serveur n'a pas ses deux textes.

**Un seul endroit pour « est administrateur ».** `isAdministrator` (front, `roles.ts`) et `isAdministrator` (Go, `account.go`) traitent l'Administrateur et le futur Super-administrateur de la même façon ; tous les anciens contrôles `role == "admin"` y passent. Le ticket des rôles n'aura que ces deux fonctions à étendre.

**Serveur injoignable.** Toute requête sans réponse exploitable (échec réseau, page de passerelle qui n'est pas du JSON de l'API) marque le serveur injoignable ; la coque affiche alors un message et un bouton « Réessayer » qui recharge l'installation et la session. Une réponse portant un code d'erreur n'est pas une panne : elle est traduite là où elle arrive.

**Transition.** Les retours du serveur vers l'adresse racine (Discord) atterrissent dans la nouvelle interface, qui affiche leur message. Tant que l'installation guidée n'existe pas dans la nouvelle interface (lot 2), une instance sans administrateur renvoie vers `/legacy/`. Les parcours navigateur des phases précédentes tournent sur `/legacy/` jusqu'à ce que chaque écran soit reconstruit ; les nouveaux parcours (connexion, langue, thème, adresses, accessibilité) tournent sur la nouvelle interface.

## Alternatives écartées

- **Radix UI** (ou shadcn/ui, qui le recopie) : même qualité d'accessibilité, mais le verrou de défilement de ses fenêtres modales viole la politique de sécurité. Le contourner imposait soit d'affaiblir la politique (`'unsafe-inline'`), soit de renoncer au verrou.
- **Affaiblir la politique de sécurité** (`style-src 'unsafe-inline'` ou nonce) : un gain de commodité contre un durcissement qui protège l'interface d'administration.
- **React Aria Components** : complet mais plus lourd et plus coûteux à habiller que Base UI pour le même besoin.
- **`<dialog>` natif seul** : suffisant pour une fenêtre, insuffisant pour menus, onglets et panneaux ; on aurait réécrit des primitives.
- **i18next, react-i18next, FormatJS** : fonctions (pluriels ICU, chargement dynamique, détection) dont rien ne justifie le poids ni la surface ; des catalogues typés couvrent le besoin et attrapent les clés manquantes à la compilation. À réévaluer si les pluriels deviennent nombreux.
- **TanStack Router / Query, SWR** : le routage déclaratif suffit ; une bibliothèque de requêtes serait une décision à prendre avec les écrans qui en ont besoin (lot 3), pas par avance.
- **CSS à la main ou CSS Modules** (comme le prototype) : retombe dans les règles globales que la refonte supprime, sans les jetons partagés ni les variantes responsive.
- **Phosphor, Heroicons** : équivalents ; lucide a le plus large catalogue et une licence permissive.
- **Colonne `users.language` non nulle avec `'en'` par défaut** : un compte existant, ou un visiteur ayant choisi le français avant de se connecter, retomberait en anglais à la connexion.
- **Détecter la langue du navigateur** : écarté par la spécification (anglais par défaut, choix explicite).
- **Remplacer l'ancienne interface d'un coup** : le lot 1 resterait inutilisable tant que tous les écrans ne sont pas reconstruits.

## Conséquences

- Les dépendances sont épinglées une à une (versions exactes dans `web/package.json`, fichier de verrouillage commité). Elles entrent dans l'image : `@base-ui/*` et Floating UI, `react-router`, `lucide-react` (ISC), `@fontsource-variable/inter` (SIL OFL 1.1, police livrée avec son texte de licence). Tailwind, son plugin Vite et `@axe-core/playwright` sont des outils de construction et de test. `deploy/NOTICES.md` est à jour ; la construction de l'image échoue toujours si une dépendance n'a pas de texte de licence (vérifié pour les 22 paquets d'exécution).
- Le paquet de la nouvelle interface pèse environ 230 Ko (70 Ko compressés), plus les sous-ensembles de la police qui ne sont chargés que si une page en a besoin.
- Les navigateurs ne voient que des fichiers ; le serveur renvoie la coque pour toute adresse inconnue : une faute de frappe dans une adresse affiche l'écran « Page introuvable » de l'application, pas une erreur du serveur.
- Les parcours navigateur n'utilisent la nouvelle interface qu'avec une installation faite par l'API (le code d'installation reste lu dans les journaux), les sessions étant ouvertes par l'API : le plafond de dix connexions en rafale du serveur est respecté.
- Un contrôle d'accessibilité automatique (axe, règles WCAG 2.2 AA) tourne sur chaque écran de la coque, en thème clair et sombre et à 390 pixels.
- Chaque lot suivant ajoute ses textes aux deux catalogues ; ajouter une langue consiste à ajouter un fichier du même type et son code à `LANGUAGES` et à la contrainte de la base.
- Tant que l'ancienne interface existe, deux jeux de composants cohabitent : c'est voulu et borné par la fin de la phase. Elle n'est plus modifiée sauf pour garder les parcours existants verts.
