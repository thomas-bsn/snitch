# Déclaration de la machine d’un Worker

Statut : accepté pour la phase 6 bis.

## Contexte

Un administrateur doit reconnaître les machines qui exécutent les Workers. Le Worker fonctionne dans un conteneur : son nom et son système ne décrivent pas nécessairement l’hôte. Les informations envoyées ne doivent pas contenir de secret et un Worker ancien doit rester compatible.

## Décision

À chaque pouls, le Worker peut déclarer un nom, le système du conteneur, le processeur, la mémoire accessible, le GPU, sa mémoire et son pilote, ainsi que sa propre version. Ces informations sont indicatives, facultatives et réservées aux administrateurs. Le serveur refuse les valeurs hors bornes ou ressemblant à des secrets. L’adresse est celle de la connexion observée par Snitch, sans port ; derrière un mandataire, elle peut désigner ce dernier.

`SNITCH_WORKER_MACHINE_NAME` permet de nommer explicitement l’hôte ; à défaut, le nom du conteneur est utilisé. Le système est présenté comme celui du conteneur, jamais comme une détection fiable du système de l’hôte. Les valeurs indisponibles sont indiquées comme non déclarées. Les anciennes déclarations sans ces champs restent valides.

## Alternatives écartées

Déduire le système de l’hôte depuis le conteneur ou lui donner un accès étendu à l’hôte serait trompeur ou ajouterait des privilèges injustifiés. Rendre les champs obligatoires casserait les Workers déjà déployés.

## Conséquences

Le GPU, le processeur et la mémoire sont détectés au premier pouls puis conservés en mémoire ; redémarrer le Worker après un changement matériel. Le GPU et le pilote ne sont visibles que si `nvidia-smi` est disponible dans le conteneur. La mémoire annoncée tient compte de la limite cgroup v2 lorsqu’elle est disponible. Les valeurs ne constituent ni un inventaire matériel garanti ni une mesure de performance.
