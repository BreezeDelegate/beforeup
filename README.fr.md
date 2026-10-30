<p align="center">
  <img src="assets/readme-banner.webp" alt="BeforeUp — contrôles Docker Compose avant déploiement" width="800" />
</p>

# BeforeUp

Contrôler un projet Docker Compose avant son démarrage ou son redéploiement : ports, montages, variables d’environnement et images locales.

Un fichier Compose peut être valide alors qu'un port est déjà utilisé, qu'un dossier indispensable manque ou que le compte courant n'a pas accès au moteur Docker. BeforeUp examine ces problèmes sans démarrer de conteneur. Il laisse Docker Compose interpréter la configuration.

**Linux uniquement.** La version 0.4.3 est la première version stable pour les hôtes Linux Docker Engine natifs pris en charge. Un diagnostic favorable ne garantit pas un déploiement réussi.

**Voir BeforeUp en action :** [démonstration terminal de 18 secondes](assets/demo-preflight.mp4) ([GIF](assets/demo-preflight.gif), [aperçu PNG](assets/demo-preflight.png)). Elle reprend les résultats réels obtenus sur un projet Compose Linux isolé.

## Installation

Prérequis : Linux, CLI Docker avec Compose v2 et `ss` (paquet `iproute2`). Pour Debian/Ubuntu amd64, télécharger `beforeup_0.4.3_amd64.deb` depuis la [release v0.4.3](https://github.com/BreezeDelegate/beforeup/releases/tag/v0.4.3), vérifier son SHA-256 avec `SHA256SUMS`, puis :

```sh
sha256sum -c SHA256SUMS --ignore-missing
sudo apt install ./beforeup_0.4.3_amd64.deb
beforeup check -f compose.yaml
```

Les archives exécutables Linux amd64/arm64 sont aussi disponibles dans les [releases](https://github.com/BreezeDelegate/beforeup/releases). Pour compiler depuis les sources, il faut Go 1.24 ou plus :

```sh
git clone https://github.com/BreezeDelegate/beforeup.git
cd beforeup
go build -o beforeup .
./beforeup check -f compose.yaml
```

Consulter le [guide d’installation](docs/install.md) et la [compatibilité vérifiée](docs/platform-support.md).

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

Le seuil disque vaut 2 Gio par défaut ; `--min-free-gib 0` désactive ce contrôle.

Le code de sortie est `0` sans blocage confirmé, `1` en présence d'un blocage, ou `2` si l'analyse n'a pas pu être menée. Avec `--strict`, un avertissement suffit à renvoyer `1`, sans le transformer artificiellement en blocage dans le rapport.

## Limites et confidentialité

BeforeUp n'exécute pas `docker compose up`, ne crée pas de répertoires et ne supprime rien. Il ne promet pas qu'une application sera en bonne santé après démarrage. L'outil n'affiche pas les chemins sources résolus des montages, les valeurs de configuration ni les erreurs brutes de Compose, car elles peuvent contenir des données sensibles.

Pour un moteur Docker distant ou une VM Docker Desktop, les contrôles locaux non fiables sont ignorés. Les tests Docker rootless, SELinux et IPv6 restent partiels : consulte la [matrice de compatibilité](docs/platform-support.md).

[Cas d’usage fréquents (anglais)](docs/common-issues.md) · [Documentation principale en anglais](README.md) · [Sécurité](SECURITY.md) · [Contribuer](CONTRIBUTING.md) · [Validation](docs/validation.md)

Licence MIT. [LICENSE](LICENSE). Projet de [BreezeDelegate](https://github.com/BreezeDelegate) ; activités professionnelles sur [vitrine.bar](https://vitrine.bar/).
