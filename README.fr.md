# BeforeUp

Vérifier qu'un projet Docker Compose a des chances de démarrer sur la machine, avant de le lancer.

Un fichier Compose peut être valide alors qu'un port est déjà utilisé, qu'un dossier indispensable manque ou que le compte courant n'a pas accès au moteur Docker. BeforeUp examine ces problèmes sans démarrer de conteneur. Il laisse Docker Compose interpréter la configuration.

**Linux uniquement pour le moment.** Le projet est encore jeune, et un diagnostic favorable ne garantit pas un déploiement réussi.

## Installation

Il faut le CLI Docker avec Compose v2, Linux, `ss` (paquet `iproute2`) et Go 1.24+ pour compiler depuis les sources.

```sh
git clone https://github.com/BreezeDelegate/beforeup.git
cd beforeup
go build -o beforeup .
./beforeup check -f compose.yaml
```

Des exécutables et leurs empreintes SHA-256 sont disponibles dans les [releases](https://github.com/BreezeDelegate/beforeup/releases).

```sh
beforeup check
beforeup check -f compose.yaml --format json
beforeup check -f compose.yaml --profile debug --env-file staging.env
beforeup compare -f compose.yaml
```

Les options `-f`, `--profile` et `--env-file` peuvent être répétées.

## Comparer avec une installation en cours

`beforeup compare -f compose.yaml` examine les conteneurs **encore actifs** du même projet. Il signale les ports ajoutés ou retirés, les changements de source des montages et les montages qui disparaîtraient de la configuration proposée.

La comparaison reste partielle : elle n'analyse ni les changements d'image, ni les variables d'environnement, ni le comportement de l'application. Si les conteneurs ne sont pas accessibles, elle ne prétend pas disposer d'une référence. Rien n'est redémarré ou modifié.

## Ce qui est vérifié

- La configuration est-elle interprétable par Docker Compose ?
- Le compte utilisé peut-il communiquer avec Docker Engine ?
- Un port publié TCP ou UDP est-il déjà occupé ou déclaré deux fois par des services distincts ?
- Un chemin utilisé comme volume local est-il absent ?
- Le stockage de Docker est-il sous un seuil d'espace libre ?

Les ports déjà publiés par le même service sont reconnus. Un autre projet Docker utilisant le port demandé est signalé séparément. Cette situation ne bloque pas forcément le démarrage : il faut d'abord vérifier à quel service appartient le port. De même, certains montages autorisent Docker à créer automatiquement un dossier absent. Ces cas sont signalés comme des avertissements.

Le seuil disque vaut 2 Gio par défaut ; `--min-free-gib 0` désactive ce contrôle.

Le code de sortie est `0` sans blocage confirmé, `1` en présence d'un blocage, ou `2` si l'analyse n'a pas pu être menée.

## Limites et confidentialité

BeforeUp n'exécute pas `docker compose up`, ne crée pas de répertoires et ne supprime rien. Il ne promet pas qu'une application sera en bonne santé après démarrage. L'outil n'affiche pas les chemins sources résolus des montages, les valeurs de configuration ni les erreurs brutes de Compose, car elles peuvent contenir des données sensibles.

Pour un moteur Docker distant, les vérifications propres à la machine locale sont ignorées : exécute BeforeUp sur la machine cible.

[Documentation principale en anglais](README.md) · [Sécurité](SECURITY.md) · [Contribuer](CONTRIBUTING.md) · [Validation](docs/validation.md)

Licence MIT. [LICENSE](LICENSE). Projet de [BreezeDelegate](https://github.com/BreezeDelegate) ; activités professionnelles sur [vitrine.bar](https://vitrine.bar/).
