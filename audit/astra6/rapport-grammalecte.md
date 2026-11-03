# Faiblesses de couverture de Grammalecte et améliorations déterministes

Date de l’audit : 3 novembre 2026.

L’absence de corrections `est → et` dans la capture fournie est une limitation
reproductible de **Grammalecte 2.3.0 lui-même**. La distribution officielle en
Python et le portage Go actuel manquent tous deux les quatre premières erreurs
et détectent les deux dernières. La règle concernée reconnaît certaines
constructions grammaticales courtes ; elle ne reconnaît pas de manière générale
deux propositions séparées par un `est` erroné.

Le dépôt contient déjà les règles de **la dernière version stable publiée,
la 2.3.0**. Une mise à jour depuis une ancienne version ne résoudrait donc pas
ces exemples. Les améliorations peuvent rester entièrement déterministes :
étendre les constructions reconnues, préserver l’ambiguïté morphologique,
reconnaître la structure des propositions avec des bornes explicites et évaluer
chaque extension face à des contre-exemples valides.

Ce rapport couvre l’historique des versions, l’architecture des règles, les
échecs reproduits, les limites de l’intégration ainsi qu’une proposition de mise
en œuvre et d’évaluation. Il ne prétend pas fournir une évaluation linguistique
exhaustive de toutes les phrases françaises ou de toutes les combinaisons de
règles. Aucun fichier d’implémentation, de ressources générées, de configuration
ou de test n’a été modifié. Les
[données de l’audit](resultats-audit.json) jointes contiennent
l’intégralité des entrées et des diagnostics produits, y compris les cas
corrects servant de contrôle.

## Identité des versions et des ressources

Le [site officiel](https://www.grammalecte.net/) indique que la version actuelle
est la 2.3.0, datée du 15 décembre 2025.
L’[historique officiel des versions pour Firefox](https://addons.mozilla.org/fr/firefox/addon/grammalecte-fr/versions/)
présente également la 2.3.0 comme dernière version. Il s’agit de la version
décrite dans l’[annonce d’Olivier](https://linuxfr.org/users/or/journaux/grammalecte-v2-3),
publiée le 15 décembre 2025. Ces pages ont été consultées à la date de l’audit.

| Composant | Identité vérifiée |
|---|---|
| Sources de référence | `08511c222029b3be20bf547563315d10fb48d44b`, commit du 15 décembre 2025 à 17:58:20 UTC |
| Version des sources | `gc_lang/fr/config.ini:9` déclare `2.3.0` ; `gc_lang/fr/changelog.md:3` mentionne la version 2.3 |
| Sources Go | `652e147fe2be913400e64efa0fa8d81fd19a6d68` |
| Provenance des règles embarquées | [graphs_provenance.json](../../rules/data/graphs_provenance.json) indique le commit de référence |
| Sources embarquées | Après décompression, `rules/data/fr.grx.gz` est identique octet par octet au fichier `rules.grx` de la révision fixée |
| SHA256 des sources des règles | `08e8aaa48e43b2fc9cd860ba0916a4d9aed3aa49e4feffe7d60a09354374e77c` |
| Archive Python officielle | [Grammalecte-fr-v2.3.0.zip](https://www.grammalecte.net/zip/Grammalecte-fr-v2.3.0.zip), 6 389 026 octets |
| SHA256 de l’archive téléchargée | `aaa4219704857778038ecc1db18f6a907994c0125b8e762dcecc69130280684f` |

La somme de contrôle de l’archive identifie le fichier téléchargé ; ce n’est pas
une somme de contrôle signée indépendamment par le projet amont. Le moteur
Python contenu dans cette archive déclare la version 2.3.0.

Les numéros de version ne suffisent pas à eux seuls. La comparaison avec
l’archive officielle a également établi les correspondances suivantes :

- Les 10 graphes de jetons correspondent aux données de référence Python figées,
  après normalisation des clés entières JSON.
- Les 5 331 actions des graphes correspondent après résolution des indices
  d’URL du moteur officiel en chaînes de caractères. Les 2 607 arbres
  d’expressions des fonctions générées pour les graphes correspondent également.
- Les 181 déclencheurs par expression régulière correspondent en ordre,
  identifiant, ligne source et motif.
- Les 14 160 lignes de tests grammaticaux officielles correspondent aux lignes
  de tests des sources après mise en forme.
- Les dictionnaires `fr-allvars` embarqué et officiel ont des données centrales
  identiques. Les différences concernent les métadonnées (`sDate`,
  `sDescription`, `sFileName`) et l’ordre de `l2grams` ; ces ensembles sont égaux.

Le HEAD distant et la branche `master` du fork GitHub configuré pointent
toujours vers le commit de référence fixé. La chronologie du dépôt Fossil
canonique à l’adresse `http://grammalecte.net:8080/timeline` a renvoyé une erreur
HTTP 503. Par conséquent, l’identité de la **dernière version stable publiée**
est vérifiée ; cet audit ne certifie pas l’état le plus récent des développements
amont non publiés.

Ces vérifications identifient les sources et les ressources du dépôt. Une
capture d’écran ne permet pas d’identifier l’exécutable chargé par un éditeur.
Cette question distincte du déploiement est abordée plus loin.

## Comment le correcteur produit un diagnostic

Grammalecte utilise déjà des règles déterministes. Sa force réside dans un vaste
ensemble de connaissances linguistiques explicites, avec des données lexicales
et morphologiques réutilisables. Sa limite tient à la couverture finie de ces
règles et à leurs interactions.

| Étape | Fonction | Conséquence sur la couverture |
|---|---|---|
| Traitement du paragraphe | Des règles à base d’expressions régulières examinent la typographie, la ponctuation et d’autres motifs du paragraphe ; certaines réécrivent le texte de travail | L’entrée peut être transformée avant l’analyse des phrases |
| Segmentation en phrases | Des heuristiques fondées sur la ponctuation et les majuscules délimitent les phrases | Une limite inattendue peut empêcher la reconnaissance d’une construction |
| Découpage en jetons et recherche lexicale | Les jetons reçoivent les lemmes possibles et les descriptions morphologiques des dictionnaires | Les mots inconnus ou les analyses ambiguës influencent les conditions suivantes |
| Graphes de jetons ordonnés | Reconnaissent des formes littérales, des lemmes, des morphologies, des étiquettes et des expressions conditionnelles | Seules les constructions représentées peuvent déclencher des actions |
| Désambiguïsation et simplification | Les règles sélectionnent des analyses, étiquettent des segments, fusionnent ou suppriment des jetons et réécrivent le texte de travail | Les règles suivantes travaillent sur une représentation transformée, pas uniquement sur la phrase visible |
| Résolution des diagnostics | Les priorités et les immunités déterminent les erreurs conservées | Une règle reconnue ne produit pas nécessairement un diagnostic visible |
| Intégration dans l’application cliente | Convertit les plages du texte source, filtre les règles, limite les diagnostics et affiche les suggestions | La sortie du moteur et l’affichage dans l’éditeur sont des étapes distinctes |

Ce dispositif est plus élaboré qu’une liste de remplacements bruts par
expressions régulières. Toutefois, le moteur ne construit pas d’arbre général
de dépendances syntaxiques du français auquel chaque phrase serait confrontée.
Une phrase qui ne déclenche aucune règle n’a **aucune erreur détectée** ; cela
ne constitue pas une preuve de sa correction grammaticale.

La vérification orthographique de Graphspell et la vérification grammaticale
sont également distinctes. `et` et `est` sont tous deux des mots valides du
dictionnaire. De meilleures suggestions orthographiques, un dictionnaire
personnel plus vaste ou la reconnaissance du nom d’une personne ne suffisent
pas à déterminer lequel doit figurer entre deux propositions.

Le projet officiel indique privilégier la réduction des fausses alertes et
l’absence de signalement lorsque les indices sont insuffisants. Cette politique
prudente explique un compromis général entre précision et rappel ; elle ne rend
pas pour autant chaque construction manquée inévitable.
[Présentation du projet](https://www.grammalecte.net/)

### Inventaire des règles

Le fichier `gc_lang/fr/rules.grx` de la révision fixée contient 43 926 lignes,
39 déclarations d’options et 14 160 lignes `TEST:`. Le compilateur natif recense
181 déclencheurs par expression régulière, 237 actions associées aux expressions
régulières, 1 353 groupes sources de graphes et 10 graphes de jetons. L’expansion
produit 4 396 046 chemins et 38 348 nœuds de graphes. Ces unités sont différentes :
des millions de chemins développés ne signifient pas des millions de règles
linguistiques conçues indépendamment.
[Documentation du compilateur](../../rules/README.md)

| Type d’action de graphe | Nombre |
|---|---:|
| Diagnostic | 3 781 |
| Réécriture | 583 |
| Désambiguïsation morphologique | 336 |
| Étiquetage | 206 |
| Condition préalable | 188 |
| Immunité | 155 |
| Analyse de plusieurs jetons | 82 |
| Total | 5 331 |

Les principales catégories de diagnostics des graphes sont les confusions
(`conf`, 1 397), l’accord du groupe nominal (`gn`, 667), les participes passés
(`ppas`, 381), la conjugaison (`conj`, 235), les traits d’union (`tu`, 187),
le mode verbal (`vmode`, 155), la reconnaissance optique de caractères
(`ocr`, 143), les locutions (`loc`, 129) et les infinitifs (`infi`, 128).
D’autres catégories couvrent l’impératif, l’interrogation, la typographie,
les majuscules, les pléonasmes, les dates et des vérifications connexes.
Ces nombres dénombrent des actions de diagnostic ; ils ne mesurent ni la
couverture ni la fréquence des erreurs.

## Reproduction de la capture d’écran

L’exécutable Go a été reconstruit à partir des sources actuelles dans un
répertoire temporaire. Chaque phrase a été vérifiée avec ce binaire et avec
le paquet officiel Python 2.3.0 téléchargé indépendamment, en utilisant le pays
`FR`, le contexte `Python`, les options par défaut et aucune règle ignorée.

| Phrase de la capture | Python 2.3.0 | Portage Go |
|---|---|---|
| Je m’appelle benoît est je suis simiesque. | Aucun diagnostic grammatical | Identique |
| Je m’appelle Alfred est je suis simiesque. | Aucun diagnostic grammatical | Identique |
| Je mange Benoit est je suis simiesque. | Aucun diagnostic grammatical | Identique |
| Je hais Benoit est je suis simiesque. | Aucun diagnostic grammatical | Identique |
| Je suis benoît est je suis simiesque. | `est → et`, plage `[15,18)` | Identique |
| Je suis fatigué est je vais dormir. | `est → et`, plage `[16,19)` | Identique |

Les positions comptent les points de code Unicode à partir de zéro, avec une
borne de fin exclue. Les deux détections utilisent
`gv2__conf_et_est2__b7_a1_1`, dont l’identifiant de ligne source est `#35304`.
Il s’agit des cas C001, C003, C005, C007, C009 et C011 dans les données de
l’audit. Les six phrases correspondantes avec `et` ne produisent aucun
diagnostic grammatical avec les réglages par défaut.

Joindre les six phrases par des espaces conserve les deux mêmes détections.
Avec l’en-tête `Et, est :`, le correcteur demande en outre une espace insécable
avant les deux-points. L’entrée de la CLI délimitée par des sauts de ligne et
l’entrée Python correctement segmentée produisent les mêmes résultats.
Cela exclut la jonction des lignes comme cause des quatre omissions dans
cette reproduction.

### La construction précisément manquante

La principale famille `est → et` est `__conf_et_est2__`, dans le graphe
`verbes2`, à l’emplacement `gc_lang/fr/rules.grx:35270–35354`. Elle contient
12 actions de diagnostic visant des constructions précises. Deux d’entre elles
expliquent la capture :

| Action | Condition simplifiée | Limite de couverture |
|---|---|---|
| `b6_a1_1`, motif à la ligne 35300 | Début de phrase, pronom sujet, verbe conjugué, `est`, pronom sujet, verbe conjugué | Aucun complément conservé entre le premier verbe et `est` |
| `b7_a1_1`, motif à la ligne 35304 | Début de phrase, pronom sujet, une forme d’`être`, adjectif ou participe passé, `est` | Nécessite l’analyse adjectivale ou participiale ; n’exige pas de proposition suivante |

Le prédicat morphologique réel de b6 accepte une étiquette de personne verbale
`:[123][sp]` pour chacun des deux mots prédicatifs. Celui de b7 accepte `:[QA]`,
qui correspond à une morphologie de participe passé ou d’adjectif pour le mot
précédent. Les listes de sujets et les ancrages de début sont explicites dans
les sources des règles.

Dans `Je mange Benoit est je suis simiesque`, le mot `Benoit` est un complément
supplémentaire. La phrase ne correspond pas à la construction courte b6.
Dans `Je m’appelle Alfred est je suis simiesque`, `Alfred` crée le même obstacle.

Une trace de débogage du moteur officiel confirme que cette dernière phrase
est réécrite, avant `verbes2`, en une séquence de travail équivalente à :

```text
Je   appelle Alfred est je suis simiesque
```

Le clitique `m’` est supprimé par `gv1__purge_préverbes1__` à la ligne source
34903 ; le complément reste présent. Attribuer l’échec à la seule apostrophe
ou au seul clitique réfléchi serait donc une mauvaise identification de sa cause.

### Pourquoi benoît se comporte différemment d’Alfred

Les analyses du dictionnaire expliquent le traitement apparemment incohérent
des noms :

| Forme | Analyse pertinente du dictionnaire |
|---|---|
| `benoît` | Adjectif, masculin singulier |
| `Benoît` | Prénom, ainsi que l’analyse adjectivale de la forme en minuscule |
| `benoit`, `Benoit` | Adjectif dans la variante orthographique réformée |
| `Alfred` | Prénom masculin |
| `Alice` | Prénom féminin |
| `Marie` | Prénom, ainsi que des formes fléchies de `marier` |
| `Xylophrax` | Aucune analyse dans le dictionnaire par défaut |
| `fatigué` | Participe passé avec morphologie adjectivale |
| `est` | À la fois le nom du point cardinal et le verbe `être` |

La condition b7 est satisfaite pour `benoît` grâce à la lecture adjectivale.
Elle ne démontre pas que le correcteur comprend que l’auteur se présente par
son nom. Remplacer la phrase par `Je suis Alfred est je vais dormir` fait
disparaître l’avertissement. Il en va de même avec `Alice`, `Marie` et le nom
inconnu utilisé dans cet essai.

La fonction auxiliaire amont `g_morph`, dans
`gc_core/py/lang_core/gc_functions.py:193–215`, utilise la morphologie du jeton
après désambiguïsation lorsqu’elle est disponible, et consulte sinon le
dictionnaire. Elle renvoie faux pour un ensemble d’analyses vide. En règle
générale, un test positif est satisfait dès qu’une analyse correspond ; un motif
négatif peut rejeter le jeton dès qu’une lecture interdite existe. De petits
changements lexicaux peuvent donc modifier le résultat d’une règle, même si
la structure de phrase voulue est identique.

### Autres règles de cette famille

Les constructions courtes b6/b7 ne sont pas les seules règles `et/est`.
Les sources contiennent également :

- `__conf_et_est1__`, ligne 14135 : des constructions telles que `entre`
  suivi de deux noms séparés par `est`.
- `__conf_est_et__`, ligne 23403 : des vérifications dans le sens inverse,
  `et → est`, portant sur des sujets courts et des prédicats adjectivaux ou
  participiaux, avec des exceptions explicites pour les coordinations valides.
- `__conf_où_est__`, ligne 23439 : des motifs restreints `où et ...`, qui
  préservent les coordinations valides telles que `où et quand`.
- `__conf_ce_qui_verbe__`, ligne 10409, et des vérifications générales de
  confusion entre pronom et verbe qui peuvent également proposer `est`.
- L’action b8 `avoir + participe`, ligne 35308. Sa liste explicite de sujets
  contient `je`, mais pas `j’`. L’erreur de l’essai
  `J’ai mangé est je vais dormir` n’est pas détectée ; la trace de débogage
  conserve le jeton initial `J’` jusqu’au graphe `verbes2` inclus.

Ainsi, une règle générale pronom/verbe détecte `Il et fatigué` et propose
`ait`, `est` et `hait`, alors que `Cette maison et à vendre` n’est pas détecté.
Détecter une erreur, inclure la bonne suggestion et classer une correction
pertinente unique sont des capacités distinctes.

## Résultats du corpus de diagnostic élargi

Les données jointes contiennent **180 cas** : 75 cas d’erreur annotés et
105 cas de contrôle. Le corpus a été conçu pour explorer ces constructions ;
il n’a pas été échantillonné dans des textes réellement écrits. Certaines
structures reviennent volontairement dans plusieurs groupes. Les nombres
ci-dessous décrivent uniquement ce corpus ; ils n’estiment pas la qualité
globale de la correction du français.

Une cible n’est comptée comme détectée que si un diagnostic couvre le jeton
annoté et inclut le remplacement attendu. Un avertissement sans rapport ne
compte pas comme une réussite. La présence du remplacement ne signifie ni
qu’il est classé en premier ni que toutes les autres suggestions sont correctes.

| Groupe d’erreurs | Cas annotés | Corrections cibles trouvées par défaut | Avec toutes les options activées |
|---|---:|---:|---:|
| Capture exacte | 6 | 2 | 2 |
| Variantes de noms et de graphies | 14 | 3 | 3 |
| Compléments, clitiques, adverbes et variantes de temps verbaux | 21 | 12 | 12 |
| Autres structures de propositions | 12 | 6 | 6 |
| Confusion inverse `et → est` | 10 | 9 | 9 |
| Sélection de cas d’accord | 6 | 5 | 5 |
| Sélection d’autres homophones | 6 | 5 | 5 |
| Total | 75 | 42 | 42 |

Les **180 sorties complètes et ordonnées de diagnostics correspondent entre
Python et Go** dans chacune des trois configurations : réglages par défaut,
toutes les options déclarées activées, et réglages par défaut avec `conf`
désactivé. Cela représente 540 comparaisons concordantes de couples
cas/configuration. Les options Python ont été définies globalement avant
chaque mode, puis rétablies, afin que les appels à la fonction auxiliaire
`option()` utilisent la même configuration effective que la CLI Go.
L’exécution avec toutes les options inclut des métadonnées de diagnostic et
des catégories spécialisées ; il s’agit d’une expérience de sensibilité,
pas d’un profil recommandé pour l’écriture.

Activer davantage de catégories n’a permis de retrouver aucune cible manquée
dans ce corpus. L’option de détection des confusions est déjà activée par
défaut. La désactiver supprime les deux détections `et/est` existantes de la
capture.

Les 75 phrases corrigées correspondantes ne produisent aucun diagnostic
grammatical par défaut. Aucun des 30 cas de contrôle supplémentaires conçus
pour mettre les règles à l’épreuve ne reçoit de diagnostic `et/est`.
Cinq reçoivent des avertissements typographiques français justifiés, pour des
espaces ordinaires avant un point d’interrogation ou à l’intérieur de guillemets.
Ils ne doivent pas être comptés comme des faux positifs grammaticaux.

### Ce qui change le résultat

| Entrée | Résultat observé |
|---|---|
| Je mange est je vais dormir. | `est → et` |
| Je mange une pomme est je vais dormir. | Non détecté |
| Je mange avec Alfred est je vais dormir. | Non détecté |
| Je mange tranquillement est je vais dormir. | `est → et` |
| Je ne mange pas est je vais dormir. | `est → et` |
| Je le mange lentement est je vais dormir. | `est → et` |
| Je suis vraiment très fatigué est je vais dormir. | `est → et` |
| Je suis un homme est je vais dormir. | Non détecté |
| Je suis professeur est je vais dormir. | Non détecté |
| Je suis à Paris est je vais dormir. | Non détecté |
| Je pense que je mange est je vais dormir. | Non détecté |
| Il dit que je suis fatigué est je vais dormir. | Non détecté |
| Alfred mange est Alice dort. | Non détecté |

Les adverbes, la négation et les clitiques sont souvent traités par des
simplifications antérieures. Il serait incorrect d’en déduire que chaque mot
supplémentaire empêche la règle de s’appliquer. Les distinctions pertinentes
concernent les compléments conservés, les classes de sujets explicites, les
limites de propositions et les conditions morphologiques.

### Autres échecs au-delà de la confusion entre et et est

Le petit échantillon exploratoire montre que le problème architectural ne se
limite pas à une paire d’homophones :

- `Les chats mange ici` et `Les chats de mon voisin mange ici` reçoivent tous
  deux la suggestion `mangent`. L’ajout d’une proposition relative produit
  `Les chats de mon voisin qui vit à Paris mange ici`, sans aucun diagnostic.
  Ces observations établissent une lacune structurelle précise, pas une
  incapacité générale à traiter les relatives : d’autres cas avec des relatives
  fonctionnent effectivement.
- `Je ne sais pas ou il est` ne reçoit aucun diagnostic, alors que la phrase
  correspondante avec `où` est valide.
- `Les maisons que je vois sont grand` reçoit l’utile `grand → grandes`,
  mais aussi l’inapproprié `sont → son`, issu de
  `g2__conf_son_sont__b4_a1_1`.
- `Ils son partis` reçoit l’utile `son → sont`, mais aussi l’inapproprié
  `partis → parti`, issu de `g3__gn_mon_ton_son_1m__b1_a4_1`.

Ces deux derniers exemples montrent des analyses locales concurrentes et de
mauvaises corrections en cascade dans des phrases déjà erronées. Ce ne sont
pas des faux positifs sur le jeu de contrôle correct. Ils montrent en revanche
pourquoi appliquer automatiquement toutes les corrections proposées serait
risqué. Revérifier une correction acceptée par l’utilisateur est utile, mais
une diminution du nombre de diagnostics ne prouve pas à elle seule qu’une
correction est linguistiquement juste.

## Faiblesses générales et limites opérationnelles

### Couverture des constructions et ambiguïté

Les motifs explicites couvrent de nombreuses erreurs courantes, mais ne se
combinent pas pour analyser une syntaxe sans restriction. Une règle qui fonctionne
avec un sujet et un verbe simples peut ne pas couvrir les compléments d’objet,
les attributs, les propositions relatives, les propositions enchâssées ou les
prédicats coordonnés. L’accumulation de variantes de phrases indépendantes alourdit
la maintenance et crée des interactions difficiles à voir entre les passes.

L’incertitude lexicale est tout aussi importante. Les noms propres peuvent être
inconnus ou homographes de mots courants. Une analyse comme adjectif en minuscules
peut permettre un avertissement, alors qu’une analyse comme nom propre reconnu
ne le permet pas. Ajouter un nom propre à un dictionnaire orthographique ne
fournit pas la construction manquante à l’échelle de la proposition. Inversement,
traiter systématiquement tous les mots commençant par une majuscule ou tous les
mots inconnus comme des noms propres créerait d’autres erreurs.

Le sens inhabituel de `Je mange Benoit` n’est pas le problème grammatical testé.
La syntaxe française peut exprimer un énoncé invraisemblable ou dérangeant.
Un correcteur grammatical déterministe ne doit pas assimiler une surprise
sémantique à une violation grammaticale. L’intention générale, la vérité
factuelle, l’interprétation du discours et la plausibilité sémantique d’énoncés
quelconques restent hors de la couverture démontrée par cet audit.

### Ordre d’exécution et suppression des diagnostics

Les passes fondées sur des expressions régulières et les graphes s’exécutent dans
un ordre défini. Les réécritures, les analyses morphologiques sélectionnées, les
étiquettes et les immunités peuvent modifier les correspondances ultérieures.
Le moteur Go ne conserve également qu’un seul diagnostic par position de début ;
un diagnostic ultérieur ne le remplace que si sa priorité est supérieure.
Les immunités des graphes peuvent empêcher ou supprimer des diagnostics.
Voir [engine.go](../../grammar/engine.go), en particulier les lignes 445–460, et
[graph_engine.go](../../grammar/graph_engine.go), lignes 350–399.

Il faut distinguer l’absence de construction correspondante, une condition non
satisfaite, un diagnostic supprimé et un diagnostic filtré par l’application
consommatrice. Les principales omissions de la capture sont reproduites au
niveau du moteur ; la seule existence de mécanismes de suppression ne suffit
pas à établir qu’ils en sont la cause.

### Omission silencieuse des phrases longues

Les deux implémentations sautent l’analyse grammaticale de la phrase lorsque la
longueur de la phrase segmentée est inférieure ou égale à 2, ou supérieure ou
égale à **3 000 points de code Unicode**. Les règles de paragraphe continuent de
s’exécuter. Ce contrôle se trouve dans le fichier amont
`gc_core/py/lang_core/gc_engine.py:281` et dans
[grammar/engine.go:254](../../grammar/engine.go).

Cette limite a été reproduite avec les constructions exactes suivantes :

```python
text2999 = "Dans une " + "très " * 592 + "grande salle, les chats mange."
text3000 = "Dans une " + "très " * 592 + "immense salle, les chats mange."
```

À 2 999 points de code, les deux moteurs proposent `mangent` pour `[2993,2998)`
avec la règle `gv2__conj_les_nom__b1_a3_1`. À 3 000, les deux ne renvoient aucun
diagnostic. Les versions courtes avec `grande` ou `immense` détectent l’erreur.
Ce test synthétique de frontière isole la limite ; il ne mesure pas les
performances sur de la prose ordinaire.

Ne renvoyer aucune erreur grammaticale pour une phrase qui n’a pas été analysée
constitue une faiblesse importante de visibilité sur le fonctionnement du moteur.
Toute modification future de cette limite devrait préserver une exécution bornée
et signaler une analyse incomplète, plutôt que de l’assimiler à un texte sans
faute.

### Intégration et déploiement

Le code actuel du serveur de langage active la grammaire et l’orthographe, utilise
le pays `FR` et le contexte `Server`, et fixe par défaut un maximum de
200 diagnostics. `conf` est activé dans les contextes Python et Server.
Aucun mécanisme de suppression propre à `et/est` n’a été trouvé.

Une application déployée peut néanmoins modifier le résultat en désactivant des
catégories, en ignorant ou suspendant des identifiants de règles, ou par la
projection du Markdown, la conversion des plages, la déduplication ou la
troncature. Les zones de code Markdown et de HTML sont volontairement exclues.
Les lignes de prose adjacentes non vides peuvent être réunies avant l’analyse.

Le serveur examiné applique aussi une limite de document de 2 Mio par défaut et
un délai maximal d’analyse de 30 secondes. Les documents trop volumineux et les
échecs d’analyse peuvent entraîner la publication d’une liste vide de diagnostics,
accompagnée d’un message dans le journal. Ce sont d’autres limites de visibilité,
pas des explications établies pour les six phrases courtes.

La dépendance de module de l’application consommatrice désigne la révision Go
`652e147fe2be` ; l’adaptateur de l’éditeur peut charger un exécutable explicitement
configuré ou trouvé dans PATH. Compiler vers un autre chemin ne remplace pas
l’exécutable en cours d’utilisation. Cet audit n’a ni modifié ni redémarré
l’éditeur et ne certifie pas le binaire qu’il a chargé. La reproduction
indépendante suffit à établir la lacune des règles sans inspecter ce déploiement.

Références du code d’intégration, relativement au dépôt parent :
`postillonne/internal/checker/checker.go:34,90,119,206`,
`postillonne/internal/prose/prose.go:45`,
`postillonne/internal/prose/markdown.go:25`,
`postillonne/internal/server/server.go:38,490,516,568`, et
`postillonne/zed/src/lib.rs:41`. Elles décrivent l’arbre de travail examiné à la
date de l’audit ; elles ne constituent pas des affirmations sur une version
distribuée.

### Ce que prouvent les tests existants

Les 14 160 cas du code source amont comprennent 5 735 lignes avec des plages
erronées marquées et 8 425 sans ; 5 724 comportent des suggestions attendues.
Le groupe principal `__conf_et_est2__` comprend 18 exemples marqués et un exemple
non marqué. Le groupe dédié à la confusion inverse comprend trois exemples
marqués et huit non marqués. Ces décomptes locaux révèlent une diversité limitée,
pas la couverture totale de toutes les autres règles susceptibles de diagnostiquer
la même confusion.

Le test Go [parity_test.go](../../grammar/parity_test.go) compare les objets de diagnostic
complets et leur ordre à la référence Python figée. Six listes de suggestions,
dont la dépendance à la graine de hachage Python est documentée, sont comparées
comme des multiensembles ; l’ordre reste strict pour les autres résultats.
Les tests actuels de parité et d’analyse détaillée ont de nouveau réussi pendant
cet audit, en 32,745 secondes :

```sh
go test ./grammar -run 'TestFullPythonDiagnosticParity|TestFullInformation' -count=1
```

L’oracle des données de référence d’origine utilise la logique du moteur Python,
ainsi que des graphes et une génération de dictionnaire vérifiés indépendamment ;
sa provenance est décrite dans [ENGINE.md](../../grammar/testdata/ENGINE.md).
La nouvelle comparaison de 180 cas utilise en complément le moteur et le
dictionnaire de la distribution officielle.

Aucune des deux suites ne constitue un banc d’évaluation de la précision ou du
rappel linguistiques fondé sur un échantillonnage indépendant. Reproduire
fidèlement une omission amont est une réussite pour la compatibilité du portage
et un échec de l’aide apportée à la personne qui écrit. Ces objectifs nécessitent
des mesures distinctes. Les comptes rendus existants de compilation complète,
de détection des accès concurrents incorrects et de vérification multiplateforme
dans [VALIDATION.md](../../VALIDATION.md) restent des comptes rendus historiques ;
cet audit documentaire a réexécuté les vérifications grammaticales ciblées,
pas toutes les suites de validation.

## Améliorer la couverture sans inférence probabiliste

Le problème immédiat ne nécessite pas de modèle de langage. La distinction utile
oppose les motifs limités à des chaînes de caractères aux contraintes
structurelles explicites. Un reconnaisseur borné fondé sur les catégories
lexicales et les composants des propositions peut être entièrement déterministe,
auditable et reproductible.

Les propositions ci-dessous n’ont été ni implémentées ni évaluées. Leur bénéfice
attendu découle des constructions manquantes observées ; leur précision doit
être établie par des tests avant leur activation.

### Préserver une référence de compatibilité

Conserver un profil identifiable compatible avec l’amont et un profil
d’amélioration versionné séparément. Toute différence de comportement
intentionnelle doit être accompagnée d’une justification documentée, d’un exemple
et d’un contre-exemple valide. Cela permet de distinguer une amélioration des
règles amont d’un défaut de portage et de maintenir des comparaisons pertinentes
au fil des évolutions de l’amont.

Utiliser une seule source de référence pour les règles supplémentaires et leurs
formes compilées. Ne pas modifier les graphes générés à la main. Préserver
l’attribution et la provenance des ressources, et exécuter les vérifications de
compatibilité amont indépendamment des assertions portant sur les améliorations.

### Étendre d’abord les constructions à forte certitude

Le plus petit incrément utile consiste à ajouter quelques familles structurelles
explicites, plutôt qu’à rechercher des correspondances sans restriction dans un
texte arbitraire :

| Famille | Exemple positif | Éléments requis |
|---|---|---|
| Prédicat de dénomination | Je m’appelle Alfred est je suis fatigué. | Une construction de dénomination reconnue à gauche et un ensemble sujet/prédicat complet à droite |
| Verbe avec un nom propre en complément d’objet | Je hais Benoit est je vais partir. | Un prédicat à verbe conjugué avec une position de complément d’objet admissible ; aucune frontière de proposition intermédiaire |
| Copule avec identité ou attribut nominal | Je suis Alfred est je vais dormir. | Une construction copulative reconnue, et non la seule appartenance à la catégorie des adjectifs |
| Complément d’objet nominal | Je mange une pomme est je vais dormir. | Un groupe nominal borné qui termine le premier prédicat |
| Sujet élidé et temps composé | J’ai mangé est je vais dormir. | Une prise en charge explicite de `j’` et de la structure auxiliaire/participe |

Chaque règle doit exiger des éléments attestant la proposition qui suit.
L’action b7 actuelle ne le fait pas ; élargir simplement sa condition sur le mot
précédent conserverait cette faiblesse.

Les noms propres inconnus ne devraient être admis que dans une position
contrainte de dénomination ou de complément d’objet, avec une analyse explicite
comme nom propre inconnu, limitée à ce candidat. Ils ne devraient pas être
acceptés globalement comme des noms quelconques, et l’analyse grammaticale doit
rester distincte de la décision d’émettre ou non un avertissement orthographique.

### Introduire une reconnaissance bornée et réutilisable des propositions

À mesure que la couverture s’étend, réutiliser un petit ensemble de composants
structurels : sujets, prédicats à verbe conjugué, groupes de clitiques, négation,
groupes adverbiaux, groupes nominaux, compléments prépositionnels et attributs
associés à une copule. Ces composants peuvent être implémentés par des transitions
à états finis ou par une table déterministe bornée d’analyses candidates.
Un analyseur syntaxique statistique généraliste n’est pas nécessaire pour ces
premières constructions.

Pour une correction candidate `est → et`, une procédure de décision prudente
consisterait à :

1. Repérer un `est` candidat dans la prose ordinaire, en conservant les plages
   dans le texte source et les annotations de ponctuation et de protection.
2. Reconnaître une proposition à gauche possédant son propre sujet et un prédicat
   à verbe conjugué accordé, avec uniquement des compléments explicitement admis.
   Commencer par les constructions en début de phrase ; traiter les propositions
   enchâssées comme une extension distincte, couverte par des tests.
3. Reconnaître une proposition à droite avec un pronom sujet explicite et un
   prédicat à verbe conjugué accordé, en autorisant des clitiques, des négations
   et des adverbes spécifiés. Étendre ensuite cette reconnaissance aux sujets
   constitués d’un groupe nominal.
4. Rejeter les candidats qui traversent des citations, des parenthèses, des
   frontières de propositions ou des zones protégées dont le reconnaisseur ne
   peut pas rendre compte.
5. Comparer les analyses admises avec `est` et avec `et`, en tenant compte du nom
   `est` et des véritables emplois comme copule ou auxiliaire. Ne suggérer `et`
   que dans une construction prise en charge, avec des éléments positifs solides
   en faveur d’une coordination.
6. S’abstenir lorsque plusieurs analyses sont en concurrence ou que la structure
   requise est incomplète ; sinon, produire une explication liée à la construction
   reconnue.

L’échec de l’analyse du texte d’origine avec une grammaire *limitée* ne prouve pas
qu’il est erroné. La condition positive autorisant la construction et les
contre-exemples valides sont essentiels. Éviter une règle équivalente à « deux
verbes sont présents, donc remplacer `est` » : les propositions relatives,
les propositions enchâssées et les citations métalinguistiques la mettraient
facilement en défaut.

Pour `et → est`, exiger un sujet singulier approprié et un prédicat avec copule
ou auxiliaire, tout en préservant les coordinations nominales valides.
`Paul et Marie`, `Le rouge et le noir`, les fragments, les titres, les ellipses
et les éléments de liste ne doivent pas être forcés dans des analyses de phrases
copulatives complètes.

### Conserver l’ambiguïté et le contexte d’origine

Conserver les lectures morphologiques candidates jusqu’à ce qu’une construction
fournisse assez d’éléments pour les écarter. Un nom propre possédant aussi une
lecture adjectivale ne doit pas déterminer accidentellement si la coordination
sera vérifiée. Des informations lexicales fixes sur la valence des verbes peuvent
aider à repérer les compléments admissibles, mais ne doivent pas servir de
prétexte pour rejeter toute construction inconnue.

Les règles tardives doivent conserver l’accès au texte d’origine ou à des
annotations fiables. Par exemple, les règles amont marquent les citations avec
`eg1mot` et `egxmot` vers la ligne 1728 du code source, puis suppriment les
caractères de citation du flux de travail vers la ligne 1749. Ne consulter que
les jetons des dernières passes pourrait transformer des mots cités en une fausse frontière
de proposition ou en un faux candidat à la coordination.

Le jeu de cas valides conçus pour mettre les règles en défaut devrait préserver
au minimum :

```text
Le fait est que je suis fatigué.
Il est, je crois, déjà parti.
Le livre que je lis est passionnant.
Ce que je suis est difficile à expliquer.
Benoît est et restera mon ami.
Le vent d’est souffle et je ferme la fenêtre.
La phrase est « je suis fatigué ».
Paul et Marie sont ici.
Il mange et dort ici.
Et si je venais demain ?
```

Les exemples cités utilisent des espaces ordinaires pour faciliter la lecture ;
les avertissements typographiques doivent être évalués séparément de la
correction grammaticale proposée.

### Rendre explicables l’absence et la suppression des diagnostics

Ajouter une trace de diagnostic adaptée au développement : règle candidate,
phase, plage d’origine, jetons de travail, analyses morphologiques possibles
et sélectionnées, résultat de la condition, réécriture, conflit de priorité,
immunité et filtrage par l’application consommatrice. Les traces de débogage
Python existantes aident à examiner le cas présent ; des explications structurées
faciliteraient une analyse systématique des régressions.

Indiquer le caractère complet ou incomplet de l’analyse séparément des constats
grammaticaux. Une limite de longueur, un dépassement de délai, un motif non pris
en charge ou une annulation d’analyse doivent avoir un statut explicite.
Joindre aux informations de débogage la révision réelle du moteur et des règles,
les empreintes des ressources, les options actives, les règles ignorées et
l’identité de l’exécutable.

### Préserver le déterminisme comme propriété testée

Le déterminisme signifie qu’une même entrée et une même configuration déclarée
produisent les mêmes diagnostics dans le même ordre. Il ne signifie pas que
chaque diagnostic est correct.

- Fixer les versions des règles, du dictionnaire, de la normalisation et des
  données Unicode.
- Définir explicitement l’ordre des règles, le départage des égalités, l’ordre
  des suggestions et la déduplication ; ne pas utiliser l’ordre de parcours de
  conteneurs non ordonnés comme mécanisme de décision.
- Utiliser des structures bornées et des limites de travail déterministes.
  Une annulation fondée sur le temps écoulé peut varier avec la charge de la
  machine : la présenter comme une exécution incomplète, et non comme un résultat
  vide obtenu avec succès.
- Exclure les modèles externes, les recherches réseau et le comportement adaptatif
  de la décision de correction. Des ressources lexicales fixes et des contraintes
  écrites à la main suffisent pour le périmètre proposé.
- Vérifier la répétabilité des résultats entre plusieurs démarrages de processus
  et plusieurs ordonnancements concurrents, en préservant les positions Unicode
  et l’identité des diagnostics.

Le portage Go rend déjà déterministes plusieurs résultats amont issus d’ensembles.
Sa gestion explicite des fonctionnalités d’expressions régulières Python non
prises en charge et des annulations doit être conservée. Il s’agit d’une
implémentation de compatibilité pour la grammaire fixée à une révision donnée,
pas d’une promesse de prendre en charge toute expression régulière Python
possible. Voir [PORTING.md](../../PORTING.md).

## Évaluation et ordre d’implémentation

Le premier corpus d’acceptation devrait comprendre les six erreurs exactes de
la capture, leurs versions corrigées, toutes les variantes de noms propres et
les cas plus larges du fichier de résultats. Pour chaque nouvelle construction,
ajouter des cas positifs et des cas valides difficiles *avant* de modifier les
règles. Tester à la fois `est → et` et `et → est`.

Répartir les futures données d’évaluation par famille de constructions et par
source documentaire, et pas seulement de façon aléatoire entre des substitutions
presque identiques. Conserver un jeu d’évaluation réservé, laissé intact,
contenant de la prose française ordinaire annotée indépendamment. Les mutations
synthétiques sont utiles pour la couverture structurelle, mais ne doivent pas
être confondues avec des erreurs de rédaction survenues naturellement.
Les cas contestés ou ambigus nécessitent des étiquettes distinctes.

| Mesure | Question à laquelle elle répond |
|---|---|
| Rappel sur les plages cibles, par famille d’erreurs | Quelles erreurs visées sont détectées ? |
| Précision des corrections candidates annotées | À quelle fréquence la plage signalée est-elle réellement erronée ? |
| Fausses alertes grammaticales pour 1 000 phrases correctes | À quelle fréquence le correcteur interrompt-il une prose valide ? |
| Rang de la correction attendue et alternatives inappropriées | La correction proposée est-elle réellement utile ? |
| Corrections erronées supplémentaires dans les phrases fautives | Les analyses locales produisent-elles de nouvelles erreurs en cascade ? |
| Taux d’abstention et d’analyse incomplète | Quels textes n’ont pas pu être vérifiés de façon concluante ? |
| Latence, mémoire et limite de travail déterministe | La couverture accrue reste-t-elle exploitable en pratique ? |
| Égalité des résultats entre exécutions répétées et concurrentes | Les résultats sont-ils reproductibles ? |

Indiquer la taille des échantillons et l’incertitude. L’absence d’alertes de
confusion sur les 105 cas témoins actuels ne permet pas d’établir un taux de faux
positifs en production. Ne pas publier le résultat ciblé de 42/75 comme le rappel
global de Grammalecte.

| Priorité | Livrable d’une implémentation ultérieure | Éléments de preuve pour l’acceptation |
|---|---|---|
| 1 | Figer le corpus de cet audit et un jeu réservé distinct de cas valides | Une référence reproductible et des annotations vérifiées humainement |
| 2 | Ajouter les constructions de dénomination, d’identité, de complément d’objet et de sujet élidé | Les six erreurs de la capture sont détectées ; leurs versions corrigées sont préservées ; aucun nouveau faux positif ciblé sur les cas témoins fixes |
| 3 | Ajouter les raisons de décision dans les traces et un indicateur du caractère complet de l’analyse | Les omissions connues et les mécanismes de suppression peuvent être distingués |
| 4 | Factoriser les composants réutilisables de propositions et les étendre aux structures enchâssées | Les gains sont présentés par famille de constructions, avec une précision mesurée sur le jeu réservé et une durée d’exécution bornée |
| 5 | Étendre l’approche aux autres homophones et aux accords | Des gains de qualité indépendants, sans changement de compatibilité inexpliqué |

Maintenir les tests de compatibilité stricte au vert pour le profil de référence.
Exécuter les nouvelles attentes comportementales pour le profil d’amélioration
et examiner chaque diagnostic modifié. Ne pas appliquer automatiquement une
suggestion au seul motif qu’elle provient d’une règle déterministe ou qu’une
nouvelle vérification produit moins d’avertissements.

## Données et reproduction

Le fichier [resultats-audit.json](resultats-audit.json) contient :

- Les 180 entrées, leurs groupes, les plages ciblées et les remplacements attendus.
- Les diagnostics complets et ordonnés de Python et de Go dans les trois
  configurations, avec les identifiants de règle, les messages, les couleurs,
  les positions et les suggestions.
- Les analyses morphologiques des noms et homophones concernés, les révisions
  des sources, les versions des environnements d’exécution, les options par
  défaut et l’empreinte de l’archive téléchargée.
- Les résultats de la comparaison avec la distribution officielle et les
  empreintes des fichiers pour les graphes, les actions, les expressions,
  les déclencheurs regex, le corpus de tests et le dictionnaire.
- Treize expériences supplémentaires sur le texte complet de la capture et
  le seuil de 2 999/3 000 caractères, avec leurs entrées et sorties intégrales.

L’environnement utilisait Python 3.9.6 et Go 1.26.1 sur darwin/arm64. La
compilation a été réalisée sans modifier les fichiers du dépôt, depuis la
racine du module Go :

```sh
go build -trimpath -o /tmp/gogrammalecte-audit ./cmd/gogrammalecte
printf '%s' 'Je m’appelle Alfred est je suis simiesque.' | /tmp/gogrammalecte-audit check
printf '%s' 'Je suis fatigué est je vais dormir.' | /tmp/gogrammalecte-audit check
printf '%s' 'Je suis benoît est je suis simiesque.' | /tmp/gogrammalecte-audit check --disable conf
```

Les résultats attendus sont respectivement : aucun diagnostic, la suggestion
`et` de la règle b7, puis aucun diagnostic. `check --full` permet d’afficher
l’analyse des phrases et des jetons.

Pour rejouer chaque cas après extraction de l’archive officielle dans un
répertoire temporaire, définir `GRAMMALECTE_RELEASE` avec le chemin de ce
répertoire, puis exécuter l’extrait suivant depuis la racine du module Go.
Cet extrait n’écrit aucun fichier source ni fichier de test :

```python
import json, os, subprocess, sys
sys.dont_write_bytecode = True
sys.path.insert(0, os.environ["GRAMMALECTE_RELEASE"])
from grammalecte.fr import gc_engine as gc

data = json.load(open("audit/astra6/resultats-audit.json"))
gc.load("Python")
defaults = dict(gc.getOptions())
modes = [
    ("default", defaults, []),
    ("all_options", {k: True for k in defaults},
     ["--enable", ",".join(sorted(defaults))]),
    ("conf_disabled", dict(defaults, conf=False), ["--disable", "conf"]),
]
for case in data["cases"]:
    for label, settings, flags in modes:
        gc.setOptions(settings)
        expected = list(gc.parse(case["input"], "FR"))
        result = subprocess.run(
            ["/tmp/gogrammalecte-audit", "check"] + flags,
            input=case["input"], text=True, capture_output=True, check=True,
        )
        actual = json.loads(result.stdout)
        assert expected == case["python_" + label], (case["id"], label, "reference")
        assert actual == expected, (case["id"], label, "port")
gc.setOptions(defaults)
print("540 case/configuration comparisons match")
```

Le corpus principal traite un paragraphe par appel. La CLI Go sépare elle-même
les lignes et rétablit les positions relatives au document. En Python,
`gc.parse` est une API de paragraphe : lui transmettre directement un document
entier contenant des sauts de ligne n’est pas équivalent. Dans les données
supplémentaires, `official_python_direct` conserve cet appel non équivalent pour
expliciter la différence ; `official_python_paragraphs` est la sortie comparable.
Les expériences sur le document complet comparent la règle, la plage, la
catégorie, le message et les suggestions après segmentation en paragraphes.
Les extraits de contexte des diagnostics peuvent refléter des limites de
paragraphe différentes.

Les emplacements des règles amont cités dans ce rapport renvoient à
`gc_lang/fr/rules.grx`, à la révision de référence fixée. Ils sont également
consultables dans les
[sources amont à cette révision](https://github.com/benoit-pereira-da-silva/grammalecte/blob/08511c222029b3be20bf547563315d10fb48d44b/gc_lang/fr/rules.grx).
Grammalecte et ses règles d’origine sont l’œuvre d’Olivier R. Les mentions
d’attribution et les licences amont existantes restent applicables.
