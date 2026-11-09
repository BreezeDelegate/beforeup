<p align="center">
  <img src="assets/readme-banner.webp" alt="BeforeUp : contrôles Docker Compose avant déploiement" width="800" />
</p>

# BeforeUp

<p align="center">
  <a href="https://github.com/BreezeDelegate/beforeup/releases"><img alt="GitHub release file downloads" src="https://img.shields.io/github/downloads/BreezeDelegate/beforeup/total?label=release%20file%20downloads" /></a>
  <a href="https://github.com/BreezeDelegate/beforeup/releases/latest"><img alt="Latest stable release" src="https://img.shields.io/github/v/release/BreezeDelegate/beforeup?label=stable%20release" /></a>
  <img alt="MIT license" src="https://img.shields.io/github/license/BreezeDelegate/beforeup" />
</p>

*Le total des téléchargements compte les fichiers de release demandés, y compris les préversions, sommes de contrôle et vérifications du mainteneur. Il ne représente ni des utilisateurs uniques ni des installations.*

**Voyez ce que votre prochain déploiement Docker Compose modifierait avant de remplacer les conteneurs actifs.**

`docker compose config` peut valider un fichier sans le comparer aux conteneurs déjà en fonctionnement. BeforeUp ajoute deux contrôles en lecture seule sur un **hôte Linux avec Docker Engine natif** :

- **`beforeup compare -f compose.yaml`** compare la configuration proposée avec les conteneurs actifs du même projet Compose. Il signale certaines modifications des ports publiés, des montages de données, des variables d’environnement déclarées et des références d’images. Il peut aussi comparer les identifiants d’images locales quand le tag reste identique.
- **`beforeup check -f compose.yaml`** examine des conditions de l’hôte que la validation Compose ne suffit pas à établir : ports occupés, sources de montages ou dossiers de construction absents, accès au moteur Docker et espace disque disponible.

```sh
beforeup compare -f compose.yaml
beforeup check -f compose.yaml
```

BeforeUp n’exécute pas `up`, `down`, `pull` ou `build` et ne modifie aucun conteneur. Un rapport sans anomalie **ne garantit pas** un déploiement réussi ni sans risque. La comparaison est partielle et nécessite l’accès aux conteneurs actifs concernés.

## Démonstration : Compose valide le fichier, mais un port est occupé

![Démonstration animée de 18 secondes : la configuration Docker Compose est valide, puis BeforeUp signale un port occupé et une source de montage absente](assets/demo-preflight.gif)

*Reconstitution animée de résultats observés sur un projet Compose Linux isolé. Elle illustre `check`, pas `compare`. Un port occupé constitue un avertissement, pas la preuve d’un échec du déploiement.* [Voir le MP4](assets/demo-preflight.mp4) | [Aperçu statique](assets/demo-preflight.png)

## Installation

Prérequis : Linux, CLI Docker avec Compose v2 et `ss` (paquet `iproute2`). Pour Debian/Ubuntu amd64, télécharger `beforeup_0.4.4_amd64.deb` depuis la [release v0.4.4](https://github.com/BreezeDelegate/beforeup/releases/tag/v0.4.4), vérifier son SHA-256 avec `SHA256SUMS`, puis :

```sh
sha256sum -c SHA256SUMS --ignore-missing
sudo apt install ./beforeup_0.4.4_amd64.deb
beforeup check -f compose.yaml
```

Les archives exécutables Linux amd64/arm64 sont aussi disponibles dans les [releases](https://github.com/BreezeDelegate/beforeup/releases). Pour compiler depuis les sources, il faut Go 1.24 ou plus :

```sh
git clone https://github.com/BreezeDelegate/beforeup.git
cd beforeup
go build -o beforeup .
./beforeup check -f compose.yaml
```

Consulter le [guide d’installation](docs/install.md) et la [compatibilité vérifiée](docs/platform-support.md). Une [préversion expérimentale pour Windows avec Docker Desktop](docs/windows-preview.md) a été testée avec les conteneurs Linux. La version Linux stable reste v0.4.4.

```sh
beforeup check
beforeup check -f compose.yaml --format json
beforeup check -f compose.yaml --profile debug --env-file staging.env
beforeup compare -f compose.yaml
beforeup compare -f compose.yaml --strict --format json
```

Les options `-f`, `--profile` et `--env-file` peuvent être répétées.

## Comparer avec une installation en cours

`beforeup compare -f compose.yaml` examine les conteneurs **encore actifs** du même projet. Il signale les ports ajoutés ou retirés, les changements de montages, ainsi que les différences de valeurs d’environnement et de références d’images par rapport aux conteneurs actifs, sans exposer les valeurs sensibles.

La comparaison reste partielle : elle vérifie les références d'images et les variables déclarées, mais ne prédit pas le comportement de l'application. Si les conteneurs ne sont pas accessibles, elle ne prétend pas disposer d'une référence. Lorsqu'un service comporte plusieurs répliques, BeforeUp le signale et continue d'analyser les autres services. Rien n'est redémarré ou modifié.

## Ce qui est vérifié

- La configuration est-elle interprétable par Docker Compose ?
- Le compte utilisé peut-il communiquer avec Docker Engine ?
- Un port publié TCP ou UDP est-il déjà occupé ou déclaré deux fois par des services distincts ?
- Un chemin utilisé comme volume local est-il absent ?
- Un répertoire local utilisé comme contexte de construction (`build.context`) est-il absent ?
- Le stockage de Docker est-il sous un seuil d'espace libre ?

Les ports déjà publiés par le même service sont reconnus. Un autre projet Docker utilisant le port demandé est signalé séparément. Une collision potentielle entre IPv4 et IPv6 reste un avertissement : le comportement dépend du réseau hôte. Cette situation ne bloque pas forcément le démarrage : il faut d'abord vérifier à quel service appartient le port. De même, certains montages autorisent Docker à créer automatiquement un dossier absent. Ces cas sont signalés comme des avertissements.

Avec Docker rootless, la présence d'un dossier ne garantit pas l'accès depuis le conteneur. Les ports inférieurs à 1024 sont signalés prudemment, car des réglages système peuvent les autoriser. Docker Desktop Linux utilise une VM : les contrôles de sockets, dossiers et disque de l'hôte local sont alors ignorés.

Trois avertissements prudents existent uniquement pour les versions **Docker Compose 5.5.0, 5.5.1, 5.6.0 et 5.6.1** vérifiées avec Engine 29.8.1 sur Debian 13 : `RESTART_POLICY_OVERRIDDEN` (une politique de redémarrage déclarée peut être silencieusement ignorée), `RESTART_POLICY_INVALID` (une valeur positive de `max_attempts` sans `condition: on-failure` peut faire échouer la création) et `PRE_START_REPLICA_UNSUPPORTED` (`pre_start.per_replica: true` est accepté à la validation puis refusé au démarrage). Ce sont des avertissements, pas des blocages certifiés. Les autres versions de Compose ne sont pas classées sans tests indépendants, et l'absence d'avertissement ne prouve pas leur compatibilité. Retirer `per_replica` change la sémantique d'initialisation.

Le seuil disque vaut 2 Gio par défaut ; `--min-free-gib 0` désactive ce contrôle.

Le code de sortie est `0` sans blocage confirmé, `1` en présence d'un blocage, ou `2` si l'analyse n'a pas pu être menée. Avec `--strict`, un avertissement suffit à renvoyer `1`, sans le transformer artificiellement en blocage dans le rapport.

## Limites et confidentialité

BeforeUp n'exécute pas `docker compose up`, ne crée pas de répertoires et ne supprime rien. Il ne promet pas qu'une application sera en bonne santé après démarrage. L'outil n'affiche pas les chemins sources résolus des montages, les valeurs de configuration ni les erreurs brutes de Compose, car elles peuvent contenir des données sensibles.

Pour un moteur Docker distant ou une VM Docker Desktop, les contrôles locaux non fiables sont ignorés. Les tests Docker rootless, SELinux et IPv6 restent partiels : consulte la [matrice de compatibilité](docs/platform-support.md).

[Exemple CI/CD sur l'hôte Linux (anglais)](docs/ci.md) · [Cas d’usage fréquents (anglais)](docs/common-issues.md) · [Documentation principale en anglais](README.md) · [Sécurité](SECURITY.md) · [Contribuer](CONTRIBUTING.md) · [Validation](docs/validation.md)

Licence MIT. [LICENSE](LICENSE). Projet de [BreezeDelegate](https://github.com/BreezeDelegate) ; activités professionnelles sur [vitrine.bar](https://vitrine.bar/).
