# Snitch

Snitch permet de retrouver les paroles et les voix présentes dans des clips de jeu.

## Langage

**Clip** : Une courte vidéo de jeu, accompagnée de sa provenance et des paroles qui y sont retrouvées.
_Avoid_ : Enregistrement (qui peut être plus long), fichier (qui désigne un emplacement).

**Analyse de clip** : Une exécution durable lancée par un administrateur, qui enchaîne les étapes applicables d'un clip (préparation, transcription, puis les suivantes) et explique l'état de chacune. Une seule est active par clip ; les résultats valides sont réutilisés.
_Avoid_ : Traitement (trop vague), tâche (qui désigne un job).

**Source** : Une origine de clips, comme un dossier surveillé ou un dépôt manuel.

**Locuteur local** : Une voix distinguée au sein d'un clip, sans identité nécessairement connue.
_Avoid_ : Personne.

**Personne** : Une identité humaine à laquelle des voix de plusieurs clips peuvent être attribuées.
_Avoid_ : Locuteur local, compte.

**Échantillon vocal validé** : Un extrait dont l'attribution à une personne a été confirmée par un utilisateur.

**Attribution** : Le lien entre une voix observée et une personne, avec son niveau de confiance et son état de validation.

**Niveau de décision** : Le statut d'une attribution : validée par un humain, automatique, suggestion, ambiguë ou inconnue. Il indique ce que l'utilisateur peut tenir pour acquis et ce qu'il doit vérifier.
_Avoid_ : Score, confiance (qui mesure la proximité des voix, pas la décision).

**Attribution automatique** : Une attribution déduite par Snitch, distincte d'une validation humaine. Elle n'est jamais un échantillon vocal validé et ne sert pas de référence. Seul l'administrateur active ou désactive ce mode.

**Référence** : L'ensemble des échantillons vocaux validés d'une Personne, avec lesquels les autres voix sont comparées.

**Voix ambiguë** : Une voix proche de plusieurs Personnes sans qu'aucune ne se détache ; elle est présentée avec ses candidates et jamais attribuée d'elle-même.

**Voix inconnue** : Une voix sans validation humaine ni attribution automatique ; une suggestion ou une voix ambiguë reste inconnue tant qu'un utilisateur n'a pas tranché.

**Cas de vérification** : Une voix inconnue ou ambiguë qu'un utilisateur doit trancher, identifiée par son Locuteur local. Il disparaît de la file dès qu'une Personne lui est attribuée par validation humaine ; « passer » un cas ne le tranche pas.

**Groupe de voix inconnues** : Un ensemble de voix inconnues proches proposé pour être nommé en une seule fois comme une Personne. Il n'est jamais confirmé automatiquement, et une voix isolée forme un groupe à elle seule.

**Piste micro** : La piste audio d'une source qui ne contient que la voix de son propriétaire ; elle lui est attribuée directement, sans diarisation.

**Profil vocal** : Les échantillons validés, empreintes et attributions d'une Personne. Son titulaire peut le supprimer ; ses voix redeviennent alors inconnues et ne sont plus rapprochées d'elle.

**Transcription d'origine** : Le texte d'un clip tel que produit par la transcription automatique. Il n'est jamais modifié et sert de référence pour mesurer la qualité.

**Configuration de transcription** : Les réglages définis dans Snitch pour produire les transcriptions de la bibliothèque : serveur, modèle et langue. Elle peut exister avant tout Worker.

**Adresse de transcription propre au Worker** : Une adresse de serveur de transcription choisie pour un Worker lorsque l'adresse commune n'est pas accessible depuis sa machine. Elle ne change ni le modèle ni la langue de la bibliothèque.

**Configuration de transcription active** : La version des réglages qu'un Worker a testée et utilise pour les nouveaux travaux. Une version demandée peut être différente tant que sa validation n'a pas réussi.

**Invitation d'installation** : Un accès à usage unique et de courte durée, généré par un administrateur, qui permet à une machine de s'installer comme Worker. Une fois utilisée ou expirée, il faut en générer une nouvelle.
_Avoid_ : Jeton du Worker (qui désigne l'accès durable du Worker installé).

**Voix prête** : État d'un Worker dont les modèles vocaux actifs sont présents et dont une inférence de test a réussi. Des modèles simplement téléchargés ne suffisent pas.
_Avoid_ : Modèles prêts, voix installée.

**Catalogue de modèles vocaux** : Les modèles de diarisation et d'empreinte que Snitch sait exécuter. Seul un modèle du catalogue peut devenir candidat ou actif ; un dépôt quelconque n'en fait pas partie.

**Modèle vocal actif** : Le modèle de diarisation, ou d'empreinte, que Snitch utilise pour tous les nouveaux travaux vocaux, quel que soit le Worker. Un Worker qui ne l'a pas prêt ne traite pas ces travaux.
_Avoid_ : Modèle du Worker, modèle installé.

**Modèle vocal candidat** : Un modèle du catalogue choisi par l'administrateur pour être préparé sur les Workers et essayé avant d'être activé. Il ne produit aucun résultat de la bibliothèque tant qu'il n'est pas actif.

**Essai de diarisation** : Une diarisation d'un clip avec un modèle candidat ou un nombre de locuteurs imposé, présentée à côté du résultat courant. Elle ne change ni les locuteurs locaux, ni les attributions, ni les corrections du clip.

**Transcription désactivée** : Choix explicite de ne pas produire de transcription pour la bibliothèque. Il se distingue d'un réglage encore incomplet ou non testé.

**Correction** : Le texte, ou le locuteur, qu'un utilisateur donne à un segment à la place de ce que Snitch avait compris. Elle recouvre la transcription d'origine sans la remplacer ; chaque changement garde ancienne valeur, nouvelle valeur, auteur et date, et peut être restauré. Une correction n'est ni une validation vocale ni une attribution.
_Avoid_ : Modification de la transcription, édition.

**Segment corrigé** : Un segment dont le texte ou le locuteur affiché est une correction humaine. Sa recherche et son affichage utilisent le texte corrigé ; après une retranscription réussie, ses corrections deviennent obsolètes et ne sont pas reprises.

**Super-administrateur** : Le propriétaire de l'installation : le compte créé par le wizard d'installation. Il est unique, ne peut être ni rétrogradé, ni supprimé, ni fermer son compte. Seul lui exporte la clé maître, configure Discord, règle l'approbation des inscriptions et gère les comptes d'Administrateurs.

**Administrateur** : Un utilisateur autorisé à exploiter l'application (Workers, dossiers, file de tâches, transcription, voix) et à approuver les inscriptions. Il n'a pas accès aux réglages réservés au Super-administrateur.

**Inscription en attente** : Une demande d'accès qui nécessite l'approbation d'un administrateur avant connexion.

**Bibliothèque** : L'ensemble des clips connus de Snitch, y compris ceux dont l'original est introuvable.

**Cause d'échec** : La raison, exprimée dans un vocabulaire fermé et stable, pour laquelle une étape de traitement n'a pas abouti. Elle est accompagnée de l'étape, de l'heure et de l'action de reprise conseillée. Seul un Administrateur la voit ; les autres utilisateurs voient seulement l'état du clip.
_Avoid_ : Message d'erreur (texte libre, non stable), log.
