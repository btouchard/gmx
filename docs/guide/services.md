# Services

Les services GMX permettent de configurer des dépendances externes (base de données, SMTP, HTTP APIs) avec chargement automatique des variables d'environnement.

## Types de Services

GMX supporte actuellement 4 types de providers :

| Provider | Usage | Status |
|----------|-------|--------|
| `sqlite` | Base de données SQLite | ✅ Implémenté |
| `postgres` | Base de données PostgreSQL | ✅ Implémenté |
| `smtp` | Serveur email | ✅ Implémenté |
| `http` | API HTTP externe | ✅ Implémenté |

## Database Service

### SQLite

```gmx
service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}
```

**Génère** :

```go
type DatabaseConfig struct {
    Provider string
    Url      string
}

func initDatabase() *DatabaseConfig {
    cfg := &DatabaseConfig{
        Provider: "sqlite",
    }
    cfg.Url = os.Getenv("DATABASE_URL")
    if cfg.Url == "" {
        log.Fatal("missing required env var: DATABASE_URL")
    }
    return cfg
}

// Dans main():
dbCfg := initDatabase()
db, err := gorm.Open(sqlite.Open(dbCfg.Url), &gorm.Config{})
if err != nil {
    log.Fatal("failed to connect database:", err)
}
```

### PostgreSQL

```gmx
service Database {
  provider: "postgres"
  url:      string @env("DATABASE_URL")
}
```

**Génère** :

```go
db, err := gorm.Open(postgres.Open(dbCfg.Url), &gorm.Config{})
```

### Utilisation

```bash
export DATABASE_URL="app.db"  # SQLite
# ou
export DATABASE_URL="postgres://user:pass@localhost/dbname"  # PostgreSQL
```

## SMTP Service (Mailer)

### Configuration

```gmx
service Mailer {
  provider: "smtp"
  host:     string @env("SMTP_HOST")
  pass:     string @env("SMTP_PASS")
  func send(to: string, subject: string, body: string) error
}
```

**Génère** :

```go
// Config struct
type MailerConfig struct {
    Provider string
    Host     string
    Pass     string
}

func initMailer() *MailerConfig {
    cfg := &MailerConfig{
        Provider: "smtp",
    }
    cfg.Host = os.Getenv("SMTP_HOST")
    if cfg.Host == "" {
        log.Fatal("missing required env var: SMTP_HOST")
    }
    cfg.Pass = os.Getenv("SMTP_PASS")
    if cfg.Pass == "" {
        log.Fatal("missing required env var: SMTP_PASS")
    }
    return cfg
}

// Interface
type MailerService interface {
    Send(to string, subject string, body string) error
}

// SMTP Implementation
type mailerImpl struct {
    config *MailerConfig
}

func (m *mailerImpl) Send(to string, subject string, body string) error {
    msg := []byte("To: " + to + "\r\n" +
        "Subject: " + subject + "\r\n" +
        "MIME-Version: 1.0\r\n" +
        "Content-Type: text/plain; charset=\"utf-8\"\r\n" +
        "\r\n" +
        body)

    var auth smtp.Auth
    if m.config.Pass != "" {
        auth = smtp.PlainAuth("", "", m.config.Pass, m.config.Host)
    }

    from := "noreply@localhost"
    addr := m.config.Host

    return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func newMailerService(cfg *MailerConfig) MailerService {
    return &mailerImpl{config: cfg}
}
```

### Champs Optionnels

Vous pouvez ajouter plus de configuration :

```gmx
service Mailer {
  provider: "smtp"
  host:     string @env("SMTP_HOST")
  port:     string @env("SMTP_PORT")
  user:     string @env("SMTP_USER")
  pass:     string @env("SMTP_PASS")
  from:     string @env("SMTP_FROM")
  func send(to: string, subject: string, body: string) error
}
```

GMX détecte automatiquement les champs `user`, `port`, `from` et les utilise dans l'implémentation.

### Utilisation

```bash
export SMTP_HOST="smtp.gmail.com:587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"
export SMTP_FROM="noreply@yourapp.com"
```

## HTTP Service (API Client)

### Configuration

```gmx
service GitHub {
  provider: "http"
  baseUrl:  string @env("GITHUB_API_URL")
  apiKey:   string @env("GITHUB_TOKEN")
}
```

**Génère** :

```go
type GitHubConfig struct {
    Provider string
    BaseUrl  string
    ApiKey   string
}

func initGitHub() *GitHubConfig {
    cfg := &GitHubConfig{
        Provider: "http",
    }
    cfg.BaseUrl = os.Getenv("GITHUB_API_URL")
    if cfg.BaseUrl == "" {
        log.Fatal("missing required env var: GITHUB_API_URL")
    }
    cfg.ApiKey = os.Getenv("GITHUB_TOKEN")
    if cfg.ApiKey == "" {
        log.Fatal("missing required env var: GITHUB_TOKEN")
    }
    return cfg
}

// HTTP Client
type GitHubClient struct {
    config *GitHubConfig
    http   *http.Client
}

func newGitHubClient(cfg *GitHubConfig) *GitHubClient {
    return &GitHubClient{
        config: cfg,
        http:   &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *GitHubClient) Get(path string) (*http.Response, error) {
    req, err := http.NewRequest("GET", c.config.BaseUrl+path, nil)
    if err != nil {
        return nil, err
    }
    if c.config.ApiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
    }
    return c.http.Do(req)
}

func (c *GitHubClient) Post(path string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequest("POST", c.config.BaseUrl+path, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    if c.config.ApiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
    }
    return c.http.Do(req)
}
```

### Utilisation

```bash
export GITHUB_API_URL="https://api.github.com"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
```

## Annotation `@env`

### Syntaxe

```gmx
fieldName: type @env("ENV_VAR_NAME")
```

### Validation Automatique

GMX génère automatiquement une vérification `log.Fatal()` si la variable est absente :

```go
cfg.Host = os.Getenv("SMTP_HOST")
if cfg.Host == "" {
    log.Fatal("missing required env var: SMTP_HOST")
}
```

### Champs Optionnels (Future)

!!!warning "Non Implémenté"
    Actuellement, **tous** les champs `@env` sont requis. Pour les champs optionnels, utilisez une valeur par défaut dans le code généré.

## Service Methods

### Déclaration

```gmx
service Mailer {
  provider: "smtp"
  host:     string @env("SMTP_HOST")
  pass:     string @env("SMTP_PASS")
  func send(to: string, subject: string, body: string) error
}
```

### Interface Générée

```go
type MailerService interface {
    Send(to string, subject string, body string) error
}
```

### Implémentation

Pour `smtp` provider, GMX génère une implémentation complète avec `net/smtp`.

Pour les autres providers, GMX génère un **stub** :

```go
type mailerStub struct {
    config *MailerConfig
}

func (s *mailerStub) Send(to string, subject string, body string) error {
    log.Printf("[%s] Mailer.Send called (stub)", s.config.Provider)
    return nil
}
```

## Exemples Complets

### Application avec SMTP

```gmx
service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}

service Mailer {
  provider: "smtp"
  host:     string @env("SMTP_HOST")
  pass:     string @env("SMTP_PASS")
  func send(to: string, subject: string, body: string) error
}

model User {
  id:    uuid   @pk @default(uuid_v4)
  email: string @email @unique
}

<script>
func notifyUser(userId: uuid, message: string) error {
  let user = try User.find(userId)

  // TODO: Call mailer service
  // let err = mailer.send(user.email, "Notification", message)
  // if err != nil { return err }

  return nil
}
</script>
```

**Variables d'environnement** :

```bash
export DATABASE_URL="app.db"
export SMTP_HOST="smtp.gmail.com:587"
export SMTP_PASS="your-app-password"
```

### Multi-Database

```gmx
service PrimaryDB {
  provider: "postgres"
  url:      string @env("PRIMARY_DB_URL")
}

service AnalyticsDB {
  provider: "postgres"
  url:      string @env("ANALYTICS_DB_URL")
}
```

!!!warning "Limitation"
    GMX ne génère actuellement qu'**une seule connexion DB** par application. Les services multiples de type database ne sont pas encore complètement supportés.

## Service dans le Script (Future)

!!!warning "Non Implémenté"
    Actuellement, vous ne pouvez pas **appeler directement** les services depuis GMX Script. Vous devez les utiliser dans le code Go généré.

    **Planifié pour une version future** :

    ```gmx
    <script>
    func sendWelcomeEmail(userId: uuid) error {
      let user = try User.find(userId)
      try Mailer.send(user.email, "Welcome!", "Hello, world!")
      return nil
    }
    </script>
    ```

## Bonnes Pratiques

### ✅ Do

- Utiliser `@env` pour tous les secrets
- Nommer les services en PascalCase (`Mailer`, `Database`)
- Vérifier les variables d'environnement avant le déploiement
- Utiliser des noms descriptifs (`SMTP_HOST` plutôt que `HOST`)

### ❌ Don't

- Ne pas hardcoder les credentials
- Ne pas commit les fichiers `.env`
- Ne pas utiliser le même service pour dev et prod sans namespace
- Ne pas oublier de documenter les variables requises

## Variables d'Environnement

### Fichier `.env`

Créez un fichier `.env` à la racine :

```bash
# Database
DATABASE_URL=app.db

# SMTP
SMTP_HOST=smtp.gmail.com:587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=noreply@yourapp.com

# APIs
GITHUB_API_URL=https://api.github.com
GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
```

### Chargement avec `godotenv`

```bash
go get github.com/joho/godotenv
```

Ajoutez dans `main()` :

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()  // Charge .env si présent
    // ...
}
```

### Production

Pour la production, utilisez des variables d'environnement réelles (pas de fichier `.env`) :

```bash
export DATABASE_URL="postgres://prod-db-url"
export SMTP_HOST="smtp.sendgrid.net:587"
./app
```

## Limitations Actuelles

| Fonctionnalité | Status |
|----------------|--------|
| sqlite/postgres providers | ✅ Implémenté |
| smtp provider | ✅ Implémenté |
| http provider | ✅ Implémenté |
| @env annotation | ✅ Implémenté |
| Service methods (interface) | ✅ Implémenté |
| SMTP implementation | ✅ Implémenté |
| HTTP client implementation | ✅ Implémenté |
| Service calls depuis script | ❌ Non implémenté |
| Champs @env optionnels | ❌ Non implémenté |
| Custom providers | ❌ Non implémenté |
| Service dependency injection | ❌ Non implémenté |

## Prochaines Étapes

- **[Security](security.md)** — Sécuriser les credentials et la configuration
- **[Contributing](../contributing/generator.md)** — Voir comment les services sont générés
