# Spécification Technique : GMX (Go-HTMX eXtension)

**Version :** 1.0

**Rôle :** Framework Web Full-stack Typé

**Philosophie :** "L'efficacité de Go, la simplicité de HTMX, l'expressivité de TypeScript."

## 1. Vision de l'Architecture

GMX est un framework "Transpiler-first". Il compile un langage déclaratif unique (`.gmx`) vers du code Go pur, performant et conforme aux standards **SOLID** et **12-Factor App**. Il utilise **HTMX** pour la réactivité sans JavaScript client personnalisé.

### Composants Clés :

* **Compiler (gmx-c) :** Analyse le DSL, gère la réflexion pour la base de données et génère le code Go.
* **Runtime :** Un serveur Go optimisé utilisant `Fiber` ou `Echo`, gérant l'injection de dépendances et la persistence.
* **Engine HTMX :** Orchestration automatique des échanges d'attributs `hx-*` basés sur les types définis.

---

## 2. Définition du Langage (DSL)

### A. Modèles et Persistence (Syntaxe Prisma-like)

Le typage est fort. Le compilateur génère automatiquement les structures Go et les migrations SQL (via GORM ou Ent).

```typescript
model User {
  id: uuid @pk
  email: string @unique
  posts: Post[]
}

model Post {
  id: uuid @pk
  title: string
  author: User @relation(references: [id])
}

```

### B. Logique Business (TypeScript-Go Hybrid)

Syntaxe simplifiée pour la logique, compilée en fonctions Go thread-safe.

```typescript
// Injection de service externe (12-factor)
service EmailService @env("SMTP_URL")

func register(email: string, pass: string) (bool, error) {
    if email == "" {
        return false, error("Email requis")
    }
    // Appel à une lib Go native (ex: crypto)
    let hashed = crypto.GenerateFromPassword(pass) 
    return db.User.Create(email, hashed)
}

```

### C. UI & Réactivité (HTML + HTMX + Tailwind)

Les fichiers `.gmx` intègrent des blocs de rendu où HTMX est le moteur de transition.

```html
<section id="feed">
  <form hx-post="/post" hx-target="#feed" hx-swap="prepend">
    <input type="text" name="title" class="p-2 border-blue-500" />
    <button type="submit">Publier</button>
  </form>
  
  { posts.map(p => <div class="card">{p.title}</div>) }
</section>

```

---

## 3. Roadmap d'implémentation (Jira Style)

### Epic 1 : Core Compiler & Parser (Lexer/Parser)

* **GMX-101 : Implémentation du Lexer GMX**
* *Task :* Tokenisation des mots-clés (`model`, `func`, `service`, `hx-*`).
* *Validation :* Test unitaire validant un flux de tokens sur un fichier `.gmx` complexe.


* **GMX-102 : Parser d'AST (Abstract Syntax Tree)**
* *Task :* Construction de l'arbre syntaxique pour les modèles et les fonctions.
* *Validation :* Validation que `User @relation` produit un nœud de relation correct.


* **GMX-103 : Code Generator (Go Target)**
* *Task :* Transpilation de l'AST en fichiers `.go` valides.
* *Validation :* `go build` doit passer sur le code généré.



### Epic 2 : Persistence & Data Engine

* **GMX-201 : Réflexion & Mapping DB**
* *Task :* Mapper les `model` GMX vers des structs Go avec Tags GORM.
* *Validation :* Création automatique d'une table SQLite à partir d'un modèle.


* **GMX-202 : Service & Env Injection**
* *Task :* Gestion de la directive `@env` pour injecter les variables système.
* *Validation :* Test d'intégration vérifiant que `EmailService` récupère la bonne string d'env.



### Epic 3 : Bridge HTMX & Frontend

* **GMX-301 : Intégration HTMX Automatisée**
* *Task :* Génération des routes backend correspondant aux attributs `hx-post/get`.
* *Validation :* Un clic bouton déclenche la fonction Go associée sans rafraîchissement.


* **GMX-302 : Pipeline CSS (Tailwind)**
* *Task :* Intégration d'un processeur JIT pour scanner les classes dans le `.gmx`.
* *Validation :* Le fichier `style.css` est généré avec uniquement les classes utilisées.



---

## 4. Tests de Validation & Définition du "Done" (DoD)

Chaque fonctionnalité doit passer les trois niveaux de validation suivants :

1. **Validation Syntaxique :** Le compilateur GMX doit accepter le code sans erreur de parsing.
2. **Validation de Compilation Go :** Le code produit doit être `fmt`-compliant et compiler sans `unsafe`.
3. **Validation End-to-End (Playwright) :**
* Scénario : "L'utilisateur soumet un formulaire -> La DB est mise à jour -> HTMX met à jour le DOM partiel".
* Critère : Latence < 50ms en local, zéro JS écrit à la main par le dev.



---

## 5. Gestion des Librairies Externes (Stratégie de "Mapping")

Pour éviter la souffrance des imports, GMX utilise un bloc de mapping global ou automatique :

```typescript
import { password as crypto } from "golang.org/x/crypto/bcrypt"
// GMX comprend que 'crypto' dans le code GMX appelle le package Go bcrypt

```

---

**Prochaine étape :** Souhaitez-vous que je génère le squelette de code Go du premier module (le Lexer) pour lancer les agents IA sur le développement ?