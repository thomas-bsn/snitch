# Précision mesurée et attribution automatique (phase 5, ticket 38)

Un administrateur voit la précision mesurée des références, reçoit une proposition d'activation lorsque ses critères sont atteints, puis active ou désactive l'attribution automatique. Tous les utilisateurs connectés voient l'état, aucun ne peut le modifier. Aucune valeur affichée n'est une garantie : les chiffres décrivent les échantillons validés disponibles, et le benchmark réel reste hors périmètre (phase 7). Les tests n'utilisent que des empreintes synthétiques.

## Mesure sur des clips distincts

La mesure rejoue le rapprochement de production (mêmes seuils, même décision, même règle d'ambiguïté) sur les Échantillons vocaux validés actifs, avec une règle de partage :

- Pour chaque Personne, ses clips (ceux qui portent un échantillon dont l'empreinte est contributive) sont triés ; la première moitié (arrondie au supérieur) fournit les **références**, les autres clips sont des **candidats de test**. Une Personne qui n'a qu'un clip ne fournit que des références.
- Un candidat dont le clip contient aussi une référence, de n'importe quelle Personne, est **écarté** (compté, jamais testé). Un échantillon est donc référence ou test, jamais les deux, et aucun test ne partage un clip avec une référence.
- Chaque test est comparé, par son empreinte comparable, aux Références construites à partir des seules références, avec la fonction et le mode de score du rapprochement (Référence cumulative par défaut, voir [rapprochement](voice-matching.md#score-et-référence-cumulative)), comme le ferait le rapprochement avec l'automatisation forcée : la mesure ne dépend donc pas de l'interrupteur. Un test sans empreinte comparable n'obtient aucun score. Seuls les tests qui seraient **attribués automatiquement** comptent pour la précision.

Chiffres exposés par `GET /api/admin/voice-accuracy` (administrateurs), calculés à la demande depuis la base :

| Chiffre | Définition |
| --- | --- |
| Confirmations | attributions automatiques simulées qui désignent la Personne validée par un humain |
| Précision globale | confirmations / attributions automatiques simulées (absente s'il n'y en a aucune) |
| Précision par Personne | idem pour les tests attribués à cette Personne (références, tests, prédictions, confirmations) |
| Couverture simulée | tests automatiques / tests |
| Couverture automatique (bibliothèque) | Locuteurs locaux non validés attribués automatiquement, ou qui le seraient dès l'activation (suggestion au moins égale au seuil automatique), sur l'ensemble des non validés |
| Vérification restante | suggestions plus faibles, ambigus et inconnus |

**Piste micro.** Son propriétaire est désigné par un administrateur, pas calculé par similarité, et elle ne change pas avec l'interrupteur (la désactivation ne touche que l'origine `matching`). Elle n'entre donc ni dans la couverture automatique ni dans le volume de vérification : elle est comptée à part (`microphone_assigned`).

## Critères et proposition

Deux critères configurables (`PUT /api/admin/voice-automation/criteria`) : la précision minimale (`precision_threshold`, défaut 0,95) et le nombre minimal de confirmations (`min_confirmations`, défaut 50). Ce sont des valeurs de départ conventionnelles, à régler après le benchmark ; ni l'une ni l'autre n'est une promesse. Quand les deux sont atteints et que l'automatisation est désactivée, le rapport porte `activation_proposed` et l'écran affiche la proposition ; rien ne change sans action de l'administrateur.

## Interrupteur

`PUT /api/admin/voice-automation {enabled, confirm_unmet_criteria?}` est le seul endroit qui modifie `auto_enabled` (administrateurs uniquement ; le contrôle d'accès est celui de toutes les routes admin).

- Activer sous les critères sans `confirm_unmet_criteria` : 409 `criteria_not_met`, rien ne change. Avec la confirmation, l'activation est possible et tracée comme forcée.
- Chaque changement effectif insère une ligne `voice_automation_events` (action, auteur instantané, critères atteints ou non, précision et confirmations du moment) ; répéter l'état courant ne crée rien.
- Après le changement, `rematchVoices` rapproche de nouveau toutes les empreintes déjà stockées : l'activation promeut les rapprochements forts en attributions automatiques, la **désactivation les ramène à des suggestions**. Les validations humaines, les références et la piste micro ne sont pas touchées ; aucune attribution automatique ne devient échantillon validé.
- Migration `00027_voice_automation.sql` : critères et dernier changement dans `voice_matching_settings`, historique dans `voice_automation_events`.

## Interface

Carte « Attribution automatique des voix » : état pour tous ; pour l'administrateur, chiffres, tableau par Personne, formulaire de critères, proposition, boutons d'activation (désactivé tant que les critères ne sont pas atteints, sauf case de confirmation explicite) ou de désactivation, historique.
