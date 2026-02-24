# GMX Documentation

Documentation complète pour le compilateur GMX.

## Structure

```
docs/
├── index.md                      # Landing page
├── guide/                        # Documentation utilisateur
│   ├── getting-started.md        # Installation et premier projet
│   ├── components.md             # Structure des fichiers .gmx
│   ├── models.md                 # Modèles et annotations
│   ├── script.md                 # GMX Script language
│   ├── templates.md              # Templates HTMX
│   ├── services.md               # Configuration services
│   └── security.md               # Sécurité et CSRF
└── contributing/                 # Documentation contributeur
    ├── architecture.md           # Architecture du compilateur
    ├── ast.md                    # Types AST
    ├── lexer-parser.md           # Lexer et parser
    ├── generator.md              # Génération de code
    ├── script-transpiler.md      # Transpileur GMX → Go
    └── testing.md                # Stratégie de test
```

## Développement Local

### Prérequis

```bash
pip install mkdocs-material
```

### Serveur de Développement

```bash
mkdocs serve
```

Ouvrir http://127.0.0.1:8000

### Build

```bash
mkdocs build
```

Génère le site dans `site/`.

## Déploiement

### GitHub Pages

```bash
mkdocs gh-deploy
```

### Netlify / Vercel

Build command: `mkdocs build`
Publish directory: `site`

## Statistiques

- **14 fichiers** markdown
- **~5900 lignes** de documentation
- **7 pages** dans le guide utilisateur
- **6 pages** dans le guide contributeur

## Contribution

Pour contribuer à la documentation :

1. Modifier les fichiers `.md` appropriés
2. Tester localement avec `mkdocs serve`
3. Créer une pull request

### Style Guide

- Utiliser des titres clairs et concis
- Inclure des exemples de code avec coloration syntaxique
- Utiliser des admonitions (`!!!note`, `!!!warning`, `!!!tip`) pour les remarques importantes
- Maintenir la cohérence avec le style existant
- Documenter UNIQUEMENT ce qui est implémenté (pas de features futures sans avertissement)

## Licence

Même licence que le projet GMX principal.
