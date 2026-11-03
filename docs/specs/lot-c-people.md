# Lot C — Onglet Personnes

Fait partie des évolutions des lots A à D (voir [ROADMAP](../../ROADMAP.md)). Référence visuelle : variante D de [`prototype/fiche-video`](https://github.com/thomas-bsn/snitch/tree/prototype/fiche-video) (`/prototype/people?variant=D`, écran de traitement : `&traiter=40`).

## Problème

La vérification se fait cas par cas, sans vue par clip ; rien n'indique quelles vidéos restent à traiter. L'annuaire est une liste compacte, et « Groupes » ne dit pas ce qu'on y fait.

## Solution

Trois onglets : **Annuaire**, **À traiter**, **Voix inconnues**, plus un écran de traitement dédié à un clip.

## Comportements

- **Annuaire** (par défaut) : grille de cartes (avatar, nom, vidéos, temps de parole, qualité de la Référence), recherche, nouvelle Personne. La fiche d'une Personne s'ouvre en pleine page : compte lié, chiffres, alerte de Référence faible, vidéos où on l'entend, Fusionner, Séparer, Supprimer (comportements existants).
- **À traiter** : une ligne par clip ayant au moins un Cas de vérification ou une voix d'un Groupe non nommé, avec les nombres à confirmer, ambiguës, inconnues et les groupes concernés ; filtres « À confirmer », « Inconnues / ambiguës », « Avec un groupe » ; compteur sur l'onglet et dans la barre latérale. Calculé à la lecture, en respectant la visibilité des clips et les dispositions des voix.
- **Écran de traitement d'un clip** (« Traiter ») :
  - Forme d'onde de toute la conversation, avec la voix en cours mise en évidence ;
  - file des voix à gauche ;
  - carte « Qui parle ici ? » : extraits avec leur texte et leur bouton d'écoute, suggestion avec sa proximité et Oui / Non, candidats, autres Personnes, nouvelle Personne, Random, Fausse détection (Voix ignorée), Passer ;
  - raccourcis clavier : Espace, O, N, R, I ;
  - passage automatique à la voix suivante ; à la fin, « Vidéo suivante » ; lien vers la fiche complète.

  Il réutilise le composant de décision de la fiche.
- **Voix inconnues** : liste des groupes (vidéos où on les entend, ressemblance éventuelle) ; panneau de nommage (écouter chaque voix, « ce n'est pas la même », nommer dans N vidéos, Random, Ignorer pour tout le groupe), avec le comportement existant (aucun groupe confirmé automatiquement, refus si le groupe a changé).
- **Adresses** : l'adresse de vérification de l'[ADR 0025](../adr/0025-adresses-de-verification-et-de-groupes.md) (`/people/verification?case=`) ouvre l'écran de traitement du clip de ce cas. `/people/groups?voice=` ouvre le nommage de ce groupe. Les notifications existantes restent valides.

## Critères d'acceptation

- À traiter liste exactement les clips visibles ayant une voix à trancher ; une voix ignorée ou Random n'y compte pas (test API).
- Traiter toutes les voix d'un clip le retire de la liste (parcours navigateur).
- Les anciennes adresses de vérification et de groupes mènent au bon écran.
- Accessibilité (raccourcis non bloquants dans les champs), thèmes, mobile.

## Dépendances

Fiche vidéo ticket 3 (dispositions) et ticket 4 (composant de décision) ; lot B ticket 3 (pics).

## Tickets

1. **Onglets Annuaire, À traiter, Voix inconnues et adresses** : structure et redirections. Aucun blocage.
2. **Annuaire en grille et fiche de Personne pleine page**. Bloqué par 1.
3. **À traiter : liste par clip et filtres** (API agrégée). Bloqué par 1, fiche vidéo ticket 3.
4. **Écran de traitement d'un clip**. Bloqué par 3, fiche vidéo ticket 4.
5. **Voix inconnues : liste et panneau de nommage**. Bloqué par 1, fiche vidéo ticket 3.
