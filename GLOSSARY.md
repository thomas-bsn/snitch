# Snitch

Snitch permet de retrouver les paroles et les voix présentes dans des clips de jeu.

## Langage

**Clip** : Une courte vidéo de jeu, accompagnée de sa provenance et des paroles qui y sont retrouvées.
_Avoid_ : Enregistrement (qui peut être plus long), fichier (qui désigne un emplacement).

**Analyse de clip** : Une exécution durable lancée par un administrateur, ou automatiquement à l'ajout d'une vidéo, qui enchaîne les étapes applicables d'un clip (préparation, transcription, puis les suivantes) et explique l'état de chacune. Une seule est active par clip ; les résultats valides sont réutilisés.
_Avoid_ : Traitement (trop vague), tâche (qui désigne un job).

**Plan audio** : Les pistes qu'une Analyse de clip doit examiner ou retenir, avec les choix explicites de l'utilisateur et les règles utilisées pour interpréter leur parole. Changer ces choix donne un autre Plan audio.
_Avoid_ : Piste choisie (qui ne couvre qu'une piste), Analyse de clip (qui désigne l'exécution entière).

**Diagnostic de piste** : L'observation, par un détecteur de parole versionné, de la présence de parole dans une piste audio réelle d'un Plan audio : parole détectée, aucune parole détectée, ambigu, échec ou non examiné, avec ses causes et intervalles. Une erreur, un modèle absent ou une faible certitude ne sont jamais une absence de parole ; le diagnostic n'exclut pas une piste à lui seul.
_Avoid_ : Détection de piste jeu (le titre ou l'ordre d'une piste ne dit rien de son contenu).

**Analyse complète automatique** : Le réglage qui fait démarrer une Analyse de clip complète, de la transcription jusqu'aux voix, dès qu'une vidéo est ajoutée. Il ne concerne que les nouveaux ajouts.

**Suppression d'analyse** : Le retrait volontaire des résultats d'étapes choisies d'un clip, avec tout ce qui en dépend. Le clip, ses fichiers et les Personnes restent ; les Références perdent les échantillons qui venaient de ce clip.
_Avoid_ : Suppression du clip, réinitialisation.

**Source** : Une origine de clips, comme un dossier surveillé ou un dépôt manuel.

**Racine vidéo** : Le dossier unique dans lequel Snitch range et lit les vidéos de la bibliothèque, et le seul où il peut écrire.
_Avoid_ : Source (qui désigne aussi les origines historiques en lecture seule).

**Vidéo non indexée** : Un fichier vidéo présent dans la Racine vidéo que Snitch n'a encore ni lu ni reconnu par son contenu. Ce n'est pas encore un clip ; l'ouvrir l'indexe, sans le transcrire.

**Indexation de la Racine** : L'action explicite d'un administrateur qui donne leur fiche clip, en une fois, à toutes les vidéos de la Racine vidéo qui n'en ont pas, avec ou sans leur analyse complète. Elle ne retraite jamais un clip existant, peut être mise en pause, reprise ou annulée, et continue sans que l'on reste sur la page.
_Avoid_ : Scan (qui désigne la lecture du disque), import (qui désigne l'envoi de fichiers).

**Scan de toute la Racine** : L'action explicite d'un administrateur qui met en file, en une fois et sans limite de nombre, toutes les vidéos éligibles de la Racine vidéo : les vidéos non indexées sont indexées, les clips jamais analysés sont analysés selon les réglages de traitement, un clip déjà analysé n'est pas réanalysé. Elle laisse toujours de côté les Dossiers privés, quel que soit le rôle, ainsi que les éléments masqués, les dossiers système et les clips privés. Elle continue en arrière-plan et reprend après un redémarrage.
_Avoid_ : Indexation de la Racine (qui ne traite que les vidéos sans clip et peut viser un Dossier privé pour le Super-administrateur), réanalyse.

**Demande de création de dossier** : La demande d'un utilisateur de créer un dossier dans la Racine vidéo. Elle attend l'approbation d'un Administrateur ; le dossier d'un Administrateur est créé sans demande.

**Dossier privé** : Un dossier de premier niveau d'une Source que seul le Super-administrateur peut parcourir. Les autres utilisateurs voient son nom avec un cadenas, pas ses clips, sauf ceux qu'ils possèdent ou qui proviennent aussi d'un emplacement visible. Il est distinct de la visibilité d'un clip.
_Avoid_ : Source privée, clip privé.

**Logo de dossier** : L'image importée par un administrateur pour remplacer la jaquette automatique d'un dossier de la Racine vidéo. Jusqu'à quatre images distinctes sont conservées ; sélectionner une ancienne version ou revenir à la jaquette automatique ne crée ni ne supprime d'entrée d'historique. Modifier le logo d'un Dossier privé reste réservé au Super-administrateur.
_Avoid_ : Jaquette (qui désigne l'image automatique), avatar de dossier.

**Propriétés d'une entrée** : Les chiffres et dates d'un dossier ou d'une vidéo de la Bibliothèque : taille sur le disque, vidéos, durée, propriétaires et dates d'ajout. Pour un dossier, elles portent sur tout son sous-arbre et seulement sur les clips que l'appelant peut lire.
_Avoid_ : Métadonnées (réservé à la capture), détails du clip.

**Locuteur local** : Une voix distinguée au sein d'un clip, sans identité nécessairement connue.
_Avoid_ : Personne.

**Personne** : Une identité humaine à laquelle des voix de plusieurs clips peuvent être attribuées.
_Avoid_ : Locuteur local, compte.

**Échantillon vocal validé** : Un extrait dont l'attribution à une personne a été confirmée par un utilisateur.

**Attribution** : Le lien entre une voix observée et une personne, avec son niveau de confiance et son état de validation.

**Niveau de décision** : Le statut d'une attribution : validée par un humain, automatique, suggestion, ambiguë ou inconnue. Il indique ce que l'utilisateur peut tenir pour acquis et ce qu'il doit vérifier.
_Avoid_ : Score, confiance (qui mesure la proximité des voix, pas la décision).

**Attribution automatique** : Une attribution déduite par Snitch, distincte d'une validation humaine. Elle n'est jamais un échantillon vocal validé et ne sert pas de référence. Seul l'administrateur active ou désactive ce mode.

**Référence** : L'ensemble des échantillons vocaux validés d'une Personne, avec lesquels les autres voix sont comparées. Ses échantillons se cumulent : plusieurs extraits brefs valent ensemble ce qu'aucun ne vaut seul (ADR 0064).
_Avoid_ : Profil cumulatif, centroïde.

**Chevauchement** : Un moment d'une piste où plusieurs Locuteurs locaux parlent en même temps.
_Avoid_ : Overlap, voix simultanées (qui désigne aussi deux pistes différentes).

**Parole nette** : La parole d'un Locuteur local hors Chevauchement. C'est elle qui sert à calculer son empreinte et à mesurer ce qu'un échantillon apporte à une Référence.
_Avoid_ : Parole propre, temps de parole (qui compte aussi les Chevauchements).

**Éligibilité d'une empreinte** : Ce que Snitch permet à une empreinte, d'après sa Parole nette et une règle versionnée que l'administrateur règle. Une empreinte *comparable* est rapprochée seule d'une Référence ; une empreinte *contributive* peut entrer dans une Référence une fois validée par un humain, avec moins de Parole nette. Une empreinte chevauchée n'est jamais ni l'une ni l'autre. Changer la règle requalifie les empreintes sans les recalculer (ADR 0064).
_Avoid_ : Empreinte utilisable, qualité minimale.

**Source séparée** : Une voix isolée par un modèle à partir d'un Chevauchement. Elle sert à retrouver des paroles, jamais à calculer une empreinte ni à enrichir une Référence. Un extrait d'écoute découpé dans une piste n'en est pas une : il contient toutes les voix du moment.
_Avoid_ : Piste séparée (qui désigne une vraie piste audio du clip), extrait.

**Voix ambiguë** : Une voix proche de plusieurs Personnes sans qu'aucune ne se détache ; elle est présentée avec ses candidates et jamais attribuée d'elle-même.

**Voix inconnue** : Une voix sans validation humaine ni attribution automatique ; une suggestion ou une voix ambiguë reste inconnue tant qu'un utilisateur n'a pas tranché. Une voix ignorée ou une voix Random n'est pas une voix inconnue.

**Part d'un segment** : Des mots consécutifs d'un segment dits par un même Locuteur local. Un segment où plusieurs voix se répondent se lit en plusieurs parts ; il n'est pas redécoupé, et une Correction s'applique toujours au segment entier (ADR 0065).
_Avoid_ : Sous-segment, tour (qui désigne la parole détectée par la diarisation, pas des mots).

**Couverture d'un Locuteur local** : Ce que la transcription retient d'un Locuteur local détecté : des paroles lui sont attribuées, il parle sous des paroles attribuées à d'autres Locuteurs locaux, ou aucune parole ne recouvre ses tours. Elle suit les parts : une voix à qui des mots d'un segment reviennent n'y parle pas sous les paroles des autres. Un Locuteur local détecté sans parole attribuée reste visible.

**Provenance d'une voix** : La piste audio réelle sur laquelle un Locuteur local a été détecté. Deux pistes ne partagent jamais un Locuteur local, et fusionner des textes ne fusionne ni voix, ni Personnes, ni Échantillons vocaux validés (ADR 0052).

**Voix ignorée** : Un Locuteur local déclaré par un utilisateur comme une fausse détection. Il ne compte pas : aucun locuteur n'est affiché pour ses paroles, et il n'est ni vérifié, ni groupé, ni rapproché d'une Personne. La décision est réversible.
_Avoid_ : Bruit, voix supprimée.

**Voix Random** : Un Locuteur local reconnu comme une vraie voix qu'on choisit de ne pas identifier. Ses paroles sont attribuées à « Random » ; il n'est ni vérifié, ni groupé, ni rapproché d'une Personne. La décision est réversible.
_Avoid_ : Anonyme, inconnu (qui attend encore une décision).

**Refus d'identification** : La décision qu'une voix n'est pas une Personne donnée. La voix redevient sans identité et cette Personne ne lui est plus proposée. Il ne supprime ni la voix ni ses paroles.
_Avoid_ : Suppression de la voix, désattribution.

**Cas de vérification** : Une voix inconnue ou ambiguë qu'un utilisateur doit trancher, identifiée par son Locuteur local. Il disparaît de la file dès qu'une Personne lui est attribuée par validation humaine, ou que la voix est déclarée ignorée ou Random ; « passer » un cas ne le tranche pas.

**À traiter** : Les clips qui ont encore au moins une voix à trancher : un Cas de vérification ou une voix d'un Groupe de voix inconnues non nommé. On les traite clip par clip.
_Avoid_ : Vérification (qui désigne un seul cas), file.

**Groupe de voix inconnues** : Un ensemble de voix inconnues proches proposé pour être nommé en une seule fois comme une Personne. Il n'est jamais confirmé automatiquement, et une voix isolée forme un groupe à elle seule. L'interface les présente sous le nom « Voix inconnues ».
_Avoid_ : Groupe (seul), cluster.

**Forme d'onde** : Le tracé de l'audio d'un clip, coloré selon la voix qui parle à chaque instant. Elle sert à repérer qui parle quand.
_Avoid_ : Spectre, spectrogramme (qui montre les fréquences).

**Piste micro** : La piste audio d'une source qui ne contient que la voix de son propriétaire ; elle lui est attribuée directement, sans diarisation.

**Profil vocal** : Les échantillons validés, empreintes et attributions d'une Personne. Son titulaire peut le supprimer ; ses voix redeviennent alors inconnues et ne sont plus rapprochées d'elle.

**Transcription d'origine** : Le texte d'un clip tel que produit par la transcription automatique. Il n'est jamais modifié et sert de référence pour mesurer la qualité.

**Révision de transcription** : Un ensemble identifié de paroles d'un clip, conservant la ou les pistes d'origine de chaque segment et mot. La révision publiée est celle que l'on consulte, corrige et recherche ; une révision candidate attend encore sa publication.
_Avoid_ : Dernière transcription (qui ne précise pas laquelle est publiée).

**Configuration de transcription** : Les réglages définis dans Snitch pour produire les transcriptions de la bibliothèque : serveur, modèle et langue. Elle peut exister avant tout Worker.

**Adresse de transcription propre au Worker** : Une adresse de serveur de transcription choisie pour un Worker lorsque l'adresse commune n'est pas accessible depuis sa machine. Elle ne change ni le modèle ni la langue de la bibliothèque.

**Configuration de transcription active** : La version des réglages qu'un Worker a testée et utilise pour les nouveaux travaux. Une version demandée peut être différente tant que sa validation n'a pas réussi.

**Invitation d'installation** : Un accès à usage unique et de courte durée, généré par un administrateur, qui permet à une machine de s'installer comme Worker. Une fois utilisée ou expirée, il faut en générer une nouvelle.
_Avoid_ : Jeton du Worker (qui désigne l'accès durable du Worker installé).

**Voix prête** : État d'un Worker dont les modèles vocaux actifs sont présents et dont une inférence de test a réussi. Des modèles simplement téléchargés ne suffisent pas.
_Avoid_ : Modèles prêts, voix installée.

**Journal du Worker** : Les lignes que le Worker écrit depuis le démarrage de son processus (sa session), envoyées à Snitch pour qu'un administrateur les lise sans accéder à la machine. Snitch n'en garde que les 5000 dernières par Worker ; le journal complet reste celui du conteneur.
_Avoid_ : Logs d'un job (qui désigne le journal d'une tâche précise).

**Pause d'un Worker** : Un Worker en pause ne prend plus de nouveau job ; ceux en cours se terminent, et il reste joignable (présence, ordres d'engines, journal). Ce n'est pas un arrêt du conteneur : arrêter ou redémarrer la machine passe par le redémarrage géré.
_Avoid_ : Arrêt, stop, hors ligne.

**Token Hugging Face central** : Le jeton d'accès aux modèles vocaux saisi une fois dans Snitch par le Super-administrateur et distribué aux Workers autorisés. Un jeton déposé directement sur un Worker reste le sien et n'est jamais remplacé en silence.
_Avoid_ : Jeton du Worker (accès du Worker à Snitch), clé maître.

**Catalogue de modèles vocaux** : Les références de modèles locaux (diarisation, empreinte, séparation) que Snitch sait exécuter, éditées par le Super-administrateur. Seul un modèle du catalogue dont un adaptateur approuvé sait réellement le charger peut être installé ; un dépôt quelconque, ou un modèle sans adaptateur (`unsupported`), n'est jamais exécuté. Le moteur n'est pas un choix de l'utilisateur. Un modèle de séparation y déclare ses limites : le nombre de voix qu'il sépare et sa fréquence d'échantillonnage. Il peut être prêt sur un Worker sans qu'aucun travail de la bibliothèque ne l'utilise. ADR 0038, ADR 0066.

**Famille d'engine** : Un ensemble de modèles que Snitch sait exécuter avec une même bibliothèque d'exécution, pour les tâches qu'il couvre. Tout modèle du catalogue rattaché à la famille y est chargé sans code propre à ce modèle ; les familles connues sont celles de la table de Snitch. Un modèle sans famille est refusé comme « famille non supportée », cause distincte d'un engine absent. ADR 0056.

**Engine à la demande** : Un engine désigné par une Famille d'engine et une version, installé sur un Worker après confirmation de l'Administrateur, sans signature ni import. Plusieurs versions d'une même famille coexistent sur un Worker. Sa résolution n'est pas reproductible d'une installation à l'autre, et sa confiance repose sur l'opérateur seul ; le parcours signé reste disponible sous « Avancé ». ADR 0056.

**Runtime approuvé** : Le code d'exécution d'un moteur vocal (par exemple TitaNet ou ECAPA-TDNN) que Snitch autorise, sous la forme d'une version signée par un éditeur de confiance. Un modèle du catalogue décrit des poids ; seul un Runtime approuvé permet de les exécuter. ADR 0055. Reste valable pour le parcours signé, désormais sous « Avancé » ; le parcours par défaut est l'Engine à la demande (ADR 0056).

**Release d'engine** : Une version précise et immuable d'un Runtime approuvé. Une Release révoquée ne peut plus être installée ; ce qui est déjà installé continue de servir. Reste valable pour le parcours signé, sous « Avancé ».

**Variante de modèle épinglée** : Un modèle de famille d'empreinte que le Worker accepte parce que ses fichiers correspondent, octet pour octet, à une révision connue de l'adaptateur. Une autre révision ou architecture est refusée avec une raison, jamais devinée ; la configuration d'un modèle (par exemple le YAML de SpeechBrain) est lue comme une donnée et n'est jamais exécutée. ADR 0055. Reste valable pour le parcours signé ; le parcours par défaut charge tout modèle d'une Famille d'engine (ADR 0056).

**Jeu de données de benchmark** : Ensemble de voix déjà validées par un humain, réparties en références et tests, sur lequel on mesure plusieurs modèles. La Personne de chaque voix est figée à la création. Il ne modifie jamais la bibliothèque. ADR 0056.

**Run de benchmark** : Un essai de plusieurs modèles sur un jeu de données, avec ses tâches, sa progression et ses résultats par modèle. Un seul run est actif à la fois. Il ne change ni le modèle actif ni les résultats de production. ADR 0056.

**Inventaire du Worker** : Ce qu'un Worker détient réellement de chaque Release d'engine : non installée, demandée, en téléchargement, en installation, en vérification, prête ou en échec, avec la cause et l'historique. Il ne change que par un ordre explicite de l'Administrateur ou par le constat du Worker ; une reconnexion n'installe jamais rien.

**Liaison de runtime** : La Release d'engine avec laquelle un modèle prêt s'exécute sur un Worker. Une mise à jour installe la nouvelle Release à côté de l'ancienne, réutilise les poids déjà validés et ne publie la liaison qu'après une inférence de test réussie ; entre-temps le modèle sert avec l'ancienne. Un retour arrière rebascule explicitement vers une Release déjà installée et approuvée, jamais vers une Release révoquée.

**Empreinte vocale réelle** : Une empreinte produite par un modèle d'empreinte du catalogue exécuté pour de bon sur un Worker (et non par un double de test). Elle porte le modèle, sa version, ses dimensions, un vecteur normalisé calculé sur la Parole nette, la durée de parole utilisée, une qualité et ses mesures (Chevauchement, écrêtage, niveau, cohérence) ; elle n'est jamais comparée à celle d'une autre famille, version ou dimension. Une preuve sur la machine de l'opérateur (matériel, mémoire, durée) reste distincte des tests automatiques.

**Enveloppe de release** : Le fichier signé qu'un éditeur de confiance remet au Super-administrateur pour importer une Release d'engine : l'identité de la clé, le manifeste et sa signature. La clé privée de l'éditeur reste chez lui ; Snitch ne la voit jamais. Reste valable pour le parcours signé, sous « Avancé », réservé au propriétaire.

**Matériel du Worker** : Ce qu'un Worker déclare de sa machine (Python, plateforme, CUDA, RAM, VRAM, disque libre) et si sa configuration autorise le repli CPU. Une Release d'engine est comparée à ce matériel avant tout ordre ; un Worker qui n'a rien déclaré n'est pas bloqué et vérifie lui-même avant d'installer.

**Repli CPU** : L'exécution d'une Release CUDA sur le processeur. Jamais automatique : elle exige l'autorisation du contrat de la Release et celle de la configuration du Worker, et n'installe rien. Un runtime CPU est une Release approuvée distincte, installée volontairement.

**Annulation et nettoyage d'une installation** : Sortir d'une installation d'engine en échec ou en cours sans perdre ce qui fonctionne. L'annulation est coopérative : le Worker s'arrête à son prochain point sûr et le confirme. Le nettoyage est une demande explicite qui ne supprime que les fichiers incomplets de l'installation concernée, jamais un modèle ni un runtime validé. Un échec de préparation ne touche pas la Release déjà prête du même engine ; remplacer revient à ordonner une autre Release compatible.

**Retrait d'une Release** : La désinstallation explicite d'une Release d'engine sur un Worker. Elle est refusée tant qu'un modèle du Worker lui est lié (avec ses jobs engagés et ses transitions d'empreintes), n'est livrée au Worker qu'une fois les références libérées, n'interrompt aucun job et ne supprime que la version désignée, jamais un modèle.

**Worker éligible à un modèle** : Un Worker qui détient ce modèle prêt à la révision du catalogue et, si son moteur n'est pas embarqué, la Release d'engine liée prête sur ce même Worker. Des poids sans runtime prêt n'ouvrent aucun travail, sans téléchargement implicite ni interruption des travaux déjà engagés.

**Modèle vocal actif** : Le modèle de diarisation, d'empreinte ou de séparation que Snitch utilise pour tous les nouveaux travaux vocaux de ce type, quel que soit le Worker. Un Worker qui ne l'a pas prêt ne traite pas ces travaux.
_Avoid_ : Modèle du Worker, modèle installé.

**Modèle vocal candidat** : Un modèle du catalogue en cours de préparation sur des Workers, avant de devenir actif. Il ne produit aucun résultat de la bibliothèque tant qu'il n'est pas actif. Depuis l'ADR 0038, il n'est plus unique : chaque Worker prépare le sien par ordre de Snitch. _Historique_ : la fonction d'essai de diarisation sur la fiche d'un clip (ADR 0022, migrations 00057 et 00058) a été retirée à la demande de l'utilisateur ; la table des essais est supprimée par la migration 00090.

**Modèle désiré d'un Worker** : Le modèle qu'un administrateur a ordonné à un Worker d'installer pour un type. Il peut différer du modèle actif pendant un remplacement ; un modèle prêt mais non actif n'attribue aucun travail. Snitch le possède, le Worker n'en connaît que l'état réel.
_Avoid_ : Modèle choisi par le Worker.

**Ordre de modèle** : Une instruction enregistrée dans Snitch pour un Worker et un type (installer, retirer), avec son auteur, son résultat et une clé qui évite l'exécution en double. Le Worker n'agit sur aucun modèle sans ordre.

**Changement du modèle d'empreinte** : Le passage explicite à un autre modèle d'empreinte. Les empreintes de deux modèles ne se comparent jamais ; Snitch montre d'abord le nombre de références à recalculer, recalcule celles des Personnes avec le nouveau modèle sans toucher à la production, puis bascule d'un bloc. L'ancien modèle reste en place jusqu'à la fin de cette migration essentielle (ADR 0044).

**Transcription désactivée** : Choix explicite de ne pas produire de transcription pour la bibliothèque. Il se distingue d'un réglage encore incomplet ou non testé.

**Correction** : Le texte, ou le locuteur, qu'un utilisateur donne à un segment à la place de ce que Snitch avait compris. Elle recouvre la transcription d'origine sans la remplacer ; chaque changement garde ancienne valeur, nouvelle valeur, auteur et date, et peut être restauré. Une correction n'est ni une validation vocale ni une attribution.
_Avoid_ : Modification de la transcription, édition.

**Segment corrigé** : Un segment dont le texte ou le locuteur affiché est une correction humaine. Sa recherche et son affichage utilisent le texte corrigé ; après une retranscription réussie, ses corrections deviennent obsolètes et ne sont pas reprises.

**Super-administrateur** : Le propriétaire de l'installation : le compte créé par le wizard d'installation. Il est unique, ne peut être ni rétrogradé, ni supprimé, ni fermer son compte. Seul lui exporte la clé maître, configure Discord, règle l'approbation des inscriptions et gère les comptes d'Administrateurs.

**Administrateur** : Un utilisateur autorisé à exploiter l'application (Workers, dossiers, file de tâches, transcription, voix) et à approuver les inscriptions. Il n'a pas accès aux réglages réservés au Super-administrateur.

**Inscription en attente** : Une demande d'accès qui nécessite l'approbation d'un administrateur avant connexion.

**Demande d'accès** : L'Inscription en attente vue par les administrateurs : elle leur est notifiée dès sa création, puis approuvée ou refusée. Refuser retire le compte demandé.

**Webhook de notifications** : Le salon Discord où Snitch relaie un résumé expurgé de certaines notifications. Il ne donne aucun accès à Snitch ; seul le Super-administrateur le configure.

**Bibliothèque** : L'ensemble des clips connus de Snitch, y compris ceux dont l'original est introuvable.

**Cause d'échec** : La raison, exprimée dans un vocabulaire fermé et stable, pour laquelle une étape de traitement n'a pas abouti. Elle est accompagnée de l'étape, de l'heure et de l'action de reprise conseillée. Seul un Administrateur la voit ; les autres utilisateurs voient seulement l'état du clip.
_Avoid_ : Message d'erreur (texte libre, non stable), log.
