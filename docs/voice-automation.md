# Précision mesurée et attribution automatique (phase 5, ticket 38)

Un administrateur voit la précision mesurée des références, reçoit une proposition d'activation lorsque ses critères sont atteints, puis active ou désactive l'attribution automatique. Tous les utilisateurs connectés voient l'état, aucun ne peut le modifier. Aucune valeur affichée n'est une garantie : les chiffres décrivent les échantillons validés disponibles, et le benchmark réel reste hors périmètre (phase 7). Les tests n'utilisent que des empreintes synthétiques.

## Mesure sur des clips distincts

La mesure rejoue le rapprochement de production (mêmes seuils, même décision, même règle d'ambiguïté) sur les Échantillons vocaux validés actifs, avec une règle de partage :

- Pour chaque Personne, ses clips (ceux qui portent un échantillon dont l'empreinte est contributive) sont triés ; la première moitié (arrondie au supérieur) fournit les **références**, les autres clips sont des **candidats de test**. Une Personne qui n'a qu'un clip ne fournit que des références.
- Un candidat dont le clip contient aussi une référence, de n'importe quelle Personne, est **écarté** (compté, jamais testé). Un échantillon est donc référence ou test, jamais les deux, et aucun test ne partage un clip avec une référence.
- Chaque test est comparé, par son empreinte comparable, aux Références construites à partir des seules références, avec la fonction et le mode de score du rapprochement (Référence cumulative par défaut, voir [rapprochement](voice-matching.md#score-et-référence-cumulative)), comme le ferait le rapprochement avec l'automatisation forcée : la mesure ne dépend donc pas de l'interrupteur. Chaque test est décidé avec les [seuils de son groupe d'empreintes](voice-matching.md#seuils-par-groupe-dempreintes) ; un groupe sans seuils appliqués n'attribue rien automatiquement, comme en production. Un test sans empreinte comparable n'obtient aucun score. Seuls les tests qui seraient **attribués automatiquement** comptent pour la précision.

Chiffres exposés par `GET /api/admin/voice-accuracy` (administrateurs), calculés à la demande depuis la base :

| Chiffre | Définition |
| --- | --- |
| Confirmations | attributions automatiques simulées qui désignent la Personne validée par un humain |
| Précision globale | confirmations / attributions automatiques simulées (absente s'il n'y en a aucune) |
| Précision par Personne | idem pour les tests attribués à cette Personne (références, tests, prédictions, confirmations) |
| Couverture simulée | tests automatiques / tests |
| Couverture automatique (bibliothèque) | Locuteurs locaux non validés attribués automatiquement, ou qui le seraient dès l'activation (suggestion au moins égale au seuil automatique appliqué à son groupe d'empreintes ; jamais pour un groupe sans seuils), sur l'ensemble des non validés |
| Vérification restante | suggestions plus faibles, ambigus et inconnus |

**Piste micro.** Son propriétaire est désigné par un administrateur, pas calculé par similarité, et elle ne change pas avec l'interrupteur (la désactivation ne touche que l'origine `matching`). Elle n'entre donc ni dans la couverture automatique ni dans le volume de vérification : elle est comptée à part (`microphone_assigned`).

## Calibration des seuils

Le rapport porte aussi `calibration` (ADR 0064, #393) : la mesure ci-dessus, rejouée sur le [groupe d'empreintes actif](voice-matching.md#seuils-par-groupe-dempreintes) seulement, **pour les deux modes de score côte à côte** (Référence cumulative et meilleur échantillon), avec la même règle de partage : aucun test ne partage un clip avec une référence. Fonction pure `calibrate` (`voice_calibration.go`), qui appelle la fonction de score et la décision de production.

| Chiffre | Définition |
| --- | --- |
| Distribution | scores de chaque test contre la Référence de sa Personne (« même Personne ») et contre celles des autres (« autre Personne »), en vingt classes de 0,05 (un score négatif compte dans la première) |
| Balayage | pour un seuil automatique de 0,05 à 1 par pas de 0,05 (seuil de suggestion abaissé à ce seuil s'il le dépasse, marge du groupe) : attributions automatiques, erreurs, voix laissées à vérifier, inconnus reconnus à tort |
| Inconnus simulés | chaque test rejoué **sans la Référence de sa Personne**, comme une voix que personne n'a validée : une attribution automatique est une fausse reconnaissance. Mesuré sans annotation supplémentaire ; donné en nombre (`unknown_automatic`) et en taux sur les inconnus simulés rejoués, un par test (`unknown_automatic_rate`) |
| Seuils du mode (`thresholds`) | seuils appliqués au groupe pour ce mode ; absent sans seuils appliqués pour ce mode |
| Aux seuils appliqués (`current`) | les mêmes chiffres aux seuils du groupe pour ce mode ; absent sans seuils appliqués pour ce mode |
| Proposition | seuil automatique : le plus bas, au centième, à partir duquel (et au-dessus) la précision des attributions automatiques reste au moins égale au critère `precision_threshold` et **aucun** inconnu simulé n'est reconnu ; seuil de suggestion : le 5ᵉ centile des scores « même Personne », arrondi au centième inférieur et au plus égal au seuil automatique ; marge : couvre 95 % des confusions mesurées (tests dont la Personne est devancée par une autre), arrondie au centième supérieur, entre 0,01 et 0,2, inchangée sans confusion |

**Effectifs trop faibles.** En dessous de 20 tests ou de 3 Personnes ayant une Référence et au moins un test (`min_tests`, `min_persons`), le rapport le dit (`withheld = too_few`) au lieu de proposer ; `no_safe_threshold` si aucun seuil ne tient le critère sans fausse reconnaissance.

**Rien n'est appliqué automatiquement.** La proposition s'applique par `PUT /api/admin/voice-matching` avec `source = proposal` et le `score_mode` mesuré, action explicite et tracée de l'administrateur ; le serveur recalcule la proposition du groupe et du mode et refuse (`409 voice_threshold_proposal_stale`) des valeurs qui n'en sont pas exactement, pour qu'une saisie ne soit jamais tracée comme issue de la mesure ; l'écran propose ce bouton pour chaque mode, les seuils d'un mode non actif ne prenant effet que lorsqu'il devient actif. Le rapport montre aussi les seuils appliqués du groupe pour le mode actif (`thresholds`) et les vingt dernières versions du groupe, tous modes confondus.

**Un score n'est pas une probabilité** : c'est une similarité cosinus entre −1 et 1 dont l'échelle dépend du modèle d'empreinte. L'écran l'affiche comme un nombre, jamais en pourcentage.

## Critères et proposition

Deux critères configurables (`PUT /api/admin/voice-automation/criteria`) : la précision minimale (`precision_threshold`, défaut 0,95) et le nombre minimal de confirmations (`min_confirmations`, défaut 50). Ce sont des valeurs de départ conventionnelles, à régler après le benchmark ; ni l'une ni l'autre n'est une promesse. Quand les deux sont atteints et que l'automatisation est désactivée, le rapport porte `activation_proposed` et l'écran affiche la proposition ; rien ne change sans action de l'administrateur.

## Interrupteur

`PUT /api/admin/voice-automation {enabled, confirm_unmet_criteria?}` est le seul endroit qui modifie `auto_enabled` (administrateurs uniquement ; le contrôle d'accès est celui de toutes les routes admin).

- Activer sous les critères sans `confirm_unmet_criteria` : 409 `criteria_not_met`, rien ne change. Avec la confirmation, l'activation est possible et tracée comme forcée.
- Chaque changement effectif insère une ligne `voice_automation_events` (action, auteur instantané, critères atteints ou non, précision et confirmations du moment) ; répéter l'état courant ne crée rien.
- Après le changement, `rematchVoices` rapproche de nouveau toutes les empreintes déjà stockées : l'activation promeut les rapprochements forts en attributions automatiques, la **désactivation les ramène à des suggestions**. Les validations humaines, les références et la piste micro ne sont pas touchées ; aucune attribution automatique ne devient échantillon validé.
- Migration `00027_voice_automation.sql` : critères et dernier changement dans `voice_matching_settings`, historique dans `voice_automation_events`.

## Interface

Carte « Attribution automatique des voix » : état pour tous ; pour l'administrateur, chiffres, tableau par Personne, formulaire de critères, proposition, boutons d'activation (désactivé tant que les critères ne sont pas atteints, sauf case de confirmation explicite) ou de désactivation, historique. Les réglages avancés montrent la calibration : les deux modes de score côte à côte (chiffres, proposition ou raison de son absence, distribution et balayage repliés) et, pour le mode actif, « Appliquer cette proposition au modèle ». L'en-tête signale l'attribution automatique suspendue quand le groupe actif n'a pas de seuils.
