# Security

GMX génère du code avec des fonctionnalités de sécurité intégrées : protection CSRF, headers HTTP sécurisés, validation automatique, et guards de méthodes HTTP.

## CSRF Protection

### Double Submit Cookie Pattern

GMX implémente automatiquement le pattern "double submit cookie" pour protéger contre les attaques CSRF.

#### Comment ça Fonctionne

1. **Sur les requêtes GET** : Génération et stockage d'un token CSRF
2. **Sur POST/PATCH/DELETE** : Validation du token

**Code généré** :

```go
func handleRoot(w http.ResponseWriter, r *http.Request) {
    // 1. Generate CSRF token
    csrfToken := generateCSRFToken()

    // 2. Set cookie (HttpOnly + SameSite)
    http.SetCookie(w, &http.Cookie{
        Name:     "csrf_token",
        Value:    csrfToken,
        HttpOnly: true,
        SameSite: http.SameSiteStrictMode,
    })

    // 3. Pass token to template
    data := PageData{
        CSRFToken: csrfToken,
        Tasks:     tasks,
    }

    // 4. Render
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    tmpl.Execute(w, data)
}
```

#### Validation sur POST/PATCH/DELETE

```go
func handleCreateTask(w http.ResponseWriter, r *http.Request) {
    // 1. Check HTTP method
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 2. Validate CSRF token
    cookie, err := r.Cookie("csrf_token")
    if err != nil {
        http.Error(w, "CSRF token missing", http.StatusForbidden)
        return
    }

    headerToken := r.Header.Get("X-CSRF-Token")
    if cookie.Value != headerToken {
        http.Error(w, "CSRF validation failed", http.StatusForbidden)
        return
    }

    // 3. Execute business logic
    ctx := &GMXContext{DB: db, Writer: w, Request: r}
    title := r.FormValue("title")
    if err := createTask(ctx, title); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

### HTMX Auto-Injection

GMX génère automatiquement un script qui injecte le token CSRF dans toutes les requêtes HTMX :

```html
<script>
document.body.addEventListener('htmx:configRequest', function(evt) {
  // Extract CSRF token from cookie
  const csrfToken = document.cookie.split('; ')
    .find(row => row.startsWith('csrf_token='))
    ?.split('=')[1];

  // Inject into request headers
  if (csrfToken) {
    evt.detail.headers['X-CSRF-Token'] = csrfToken;
  }
});
</script>
```

**Résultat** : Toutes les requêtes HTMX sont **automatiquement protégées**.

### Token Generation

```go
func generateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

Le token est généré avec `crypto/rand` (cryptographiquement sécurisé).

## HTTP Security Headers

GMX génère automatiquement ces headers sur toutes les réponses :

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
```

### Explication

| Header | Valeur | Protection |
|--------|--------|------------|
| `X-Content-Type-Options` | `nosniff` | Empêche le navigateur de deviner le MIME type |
| `X-Frame-Options` | `DENY` | Bloque l'embedding dans des iframes (clickjacking) |
| `X-XSS-Protection` | `1; mode=block` | Active la protection XSS du navigateur |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limite l'exposition du referer |

!!!note "Content-Security-Policy"
    GMX ne génère pas encore de CSP header. Vous pouvez l'ajouter manuellement dans le code généré pour renforcer la sécurité.

## Method Guards

Tous les handlers vérifient automatiquement la méthode HTTP :

```go
func handleCreateTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ...
}

func handleToggleTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != "PATCH" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ...
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != "DELETE" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ...
}
```

**Résultat** : Impossible d'appeler `deleteTask` avec un GET par exemple.

## Model Validation

### Automatic Validation

La méthode `Validate()` est appelée automatiquement avant `save()` :

```go
func TaskSave(db *gorm.DB, obj *Task) error {
    // 1. Validate first
    if err := obj.Validate(); err != nil {
        return err
    }

    // 2. Then save
    return db.Save(obj).Error
}
```

### Validation Rules

GMX génère des validations pour ces annotations :

```gmx
model Task {
  title: string @min(3) @max(255)
  email: string @email
}
```

**Code généré** :

```go
func (t *Task) Validate() error {
    if len(t.Title) < 3 {
        return fmt.Errorf("title: minimum length is 3, got %d", len(t.Title))
    }
    if len(t.Title) > 255 {
        return fmt.Errorf("title: maximum length is 255, got %d", len(t.Title))
    }
    if t.Email != "" && !isValidEmail(t.Email) {
        return fmt.Errorf("email: invalid email format")
    }
    return nil
}
```

### Email Validation Helper

```go
func isValidEmail(email string) bool {
    re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return re.MatchString(email)
}
```

## Input Sanitization

### HTML Template Escaping

GMX utilise `html/template` qui **échappe automatiquement** le HTML :

```html
<span>{{.Title}}</span>
```

Si `Title` contient `<script>alert('XSS')</script>`, il sera rendu comme :

```html
<span>&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;</span>
```

### URL Encoding

Les paramètres dans les URLs sont également échappés :

```html
<a href="{{route "viewTask"}}?id={{.ID}}">View</a>
```

### SQL Injection Protection

GMX utilise **GORM avec prepared statements**, ce qui protège contre les injections SQL :

```go
// ✅ Safe (parameterized query)
db.First(&task, "id = ?", id)

// ❌ Dangerous (would be vulnerable if constructed manually)
// db.Raw("SELECT * FROM tasks WHERE id = " + id)
```

## Error Masking

!!!warning "Limitation Actuelle"
    GMX expose actuellement les erreurs internes au client :

    ```go
    if err := toggleTask(ctx, id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)  // ❌ Expose l'erreur
        return
    }
    ```

    **Recommandation** : Modifier le code généré pour logger et masquer :

    ```go
    if err := toggleTask(ctx, id); err != nil {
        log.Printf("toggleTask error: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    ```

## Multi-Tenancy avec `@scoped`

### Scoped Queries

Les modèles avec `@scoped` sont automatiquement filtrés par tenant :

```gmx
model Post {
  tenantId: uuid @scoped
  title:    string
}
```

**Génère** :

```go
func PostAll(db *gorm.DB, tenantID string) ([]Post, error) {
    var objs []Post
    if err := db.Where("tenant_id = ?", tenantID).Find(&objs).Error; err != nil {
        return nil, err
    }
    return objs, nil
}

func PostSave(db *gorm.DB, obj *Post, tenantID string) error {
    if err := obj.Validate(); err != nil {
        return err
    }
    obj.TenantID = tenantID  // Auto-inject
    return db.Save(obj).Error
}
```

**Résultat** : Isolation complète entre tenants.

## Cookie Security

GMX configure les cookies CSRF avec les bonnes options :

```go
http.SetCookie(w, &http.Cookie{
    Name:     "csrf_token",
    Value:    csrfToken,
    HttpOnly: true,                        // ✅ Pas accessible en JavaScript
    SameSite: http.SameSiteStrictMode,    // ✅ Protection CSRF supplémentaire
    // Secure: true,                       // ❌ TODO: Activer en production (HTTPS)
})
```

!!!warning "HTTPS en Production"
    Ajoutez manuellement `Secure: true` dans le code généré pour forcer HTTPS en production.

## Password Hashing (Future)

!!!warning "Non Implémenté"
    GMX ne génère pas encore de hashing automatique pour les mots de passe.

    **Workaround** : Utiliser `bcrypt` manuellement dans le code généré :

    ```go
    import "golang.org/x/crypto/bcrypt"

    // Before save
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    user.Password = string(hashedPassword)
    ```

## Rate Limiting (Future)

!!!warning "Non Implémenté"
    Pas de rate limiting intégré. Recommandation : utiliser un middleware comme [tollbooth](https://github.com/didip/tollbooth).

## Best Practices

### ✅ Do

- ✅ Utiliser HTTPS en production
- ✅ Activer `Secure: true` sur les cookies
- ✅ Valider tous les inputs avec `@min`, `@max`, `@email`
- ✅ Logger les erreurs côté serveur
- ✅ Masquer les erreurs internes au client
- ✅ Utiliser des UUIDs pour les IDs publiques
- ✅ Activer le multi-tenancy avec `@scoped` si nécessaire

### ❌ Don't

- ❌ Ne pas exposer les erreurs SQL au client
- ❌ Ne pas désactiver la validation CSRF
- ❌ Ne pas stocker les mots de passe en clair
- ❌ Ne pas utiliser des IDs auto-incrémentés exposés publiquement
- ❌ Ne pas oublier de valider les inputs utilisateur
- ❌ Ne pas skip la validation dans `save()`

## Security Checklist

### Avant le Déploiement

- [ ] HTTPS activé
- [ ] Cookies avec `Secure: true`
- [ ] CSRF protection validée
- [ ] Security headers configurés
- [ ] Validation des modèles testée
- [ ] Erreurs loggées et masquées
- [ ] Variables d'environnement sécurisées (pas de `.env` en prod)
- [ ] Rate limiting configuré (si nécessaire)
- [ ] Backups de la base de données configurés

### Tests de Sécurité

```bash
# Test CSRF protection
curl -X POST http://localhost:8080/createTask \
  -d "title=test" \
  # ❌ Devrait échouer (pas de token)

# Test method guards
curl -X GET http://localhost:8080/deleteTask?id=123
# ❌ Devrait échouer (method not allowed)

# Test validation
curl -X POST http://localhost:8080/createTask \
  -H "X-CSRF-Token: valid-token" \
  -d "title=ab"
# ❌ Devrait échouer (titre trop court)
```

## Audit Report

Voir `AUDIT_REPORT.md` pour une analyse complète de la sécurité du compilateur GMX.

**Points critiques identifiés** :

- ⚠️ Erreurs internes exposées au client
- ⚠️ Pas de validation d'input dans les handlers (uniquement dans les modèles)
- ✅ CSRF protection fonctionnelle
- ✅ Security headers générés
- ✅ Method guards en place
- ✅ Validation de modèles automatique

## Prochaines Étapes

- **[Contributing](../contributing/architecture.md)** — Contribuer à la sécurité du compilateur
- **[Testing](../contributing/testing.md)** — Tester les fonctionnalités de sécurité
