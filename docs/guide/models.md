# Models

Les modèles GMX définissent la structure de votre base de données avec validation, relations et hooks automatiques.

## Déclaration de Base

Les modèles sont déclarés dans le bloc `<script>` :

```gmx
<script>
model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}
</script>
```

**Génère :**

```go
type Task struct {
    ID    string `gorm:"primaryKey" json:"id"`
    Title string `json:"title"`
    Done  bool   `gorm:"default:false" json:"done"`
}

func (t *Task) Validate() error {
    if len(t.Title) < 3 {
        return fmt.Errorf("title: minimum length is 3, got %d", len(t.Title))
    }
    if len(t.Title) > 255 {
        return fmt.Errorf("title: maximum length is 255, got %d", len(t.Title))
    }
    return nil
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
    if t.ID == "" {
        t.ID = generateUUID()
    }
    return nil
}
```

## Types de Champs

### Types Primitifs

| Type GMX   | Type Go     | GORM Type   | Usage                    |
|------------|-------------|-------------|--------------------------|
| `uuid`     | `string`    | VARCHAR     | Identifiants uniques     |
| `string`   | `string`    | VARCHAR     | Texte                    |
| `int`      | `int`       | INTEGER     | Nombres entiers          |
| `float`    | `float64`   | REAL        | Nombres décimaux         |
| `bool`     | `bool`      | BOOLEAN     | Vrai/faux                |
| `datetime` | `time.Time` | TIMESTAMP   | Dates et heures          |

### Relations

```gmx
<script>
model User {
  id:    uuid   @pk @default(uuid_v4)
  email: string @unique @email
  posts: Post[] // Relation one-to-many
}

model Post {
  id:     uuid @pk @default(uuid_v4)
  userId: uuid
  user:   User @relation(references: [id])
  title:  string
}
</script>
```

**Génère :**

```go
type User struct {
    ID    string `gorm:"primaryKey" json:"id"`
    Email string `gorm:"unique" json:"email"`
    Posts []Post `json:"posts"`
}

type Post struct {
    ID     string `gorm:"primaryKey" json:"id"`
    UserID string `json:"user_id"`
    User   User   `gorm:"foreignKey:UserID" json:"user"`
    Title  string `json:"title"`
}
```

## Annotations

### Clés Primaires et Defaults

#### `@pk` — Clé Primaire

```gmx
id: uuid @pk
```

Génère : `gorm:"primaryKey"`

#### `@default(value)` — Valeur par Défaut

```gmx
done:      bool     @default(false)
createdAt: datetime @default(now)
id:        uuid     @default(uuid_v4)
```

**`uuid_v4` génère un hook GORM automatique** :

```go
func (t *Task) BeforeCreate(tx *gorm.DB) error {
    if t.ID == "" {
        t.ID = generateUUID()
    }
    return nil
}
```

### Validation

#### `@min(n)` — Longueur/Valeur Minimale

Pour les **strings** : longueur minimale

```gmx
title: string @min(3)
```

Génère :

```go
if len(t.Title) < 3 {
    return fmt.Errorf("title: minimum length is 3, got %d", len(t.Title))
}
```

Pour les **int/float** : valeur minimale

```gmx
age: int @min(18)
```

Génère :

```go
if t.Age < 18 {
    return fmt.Errorf("age: minimum value is 18, got %v", t.Age)
}
```

#### `@max(n)` — Longueur/Valeur Maximale

Pour les **strings** : longueur maximale

```gmx
title: string @max(255)
```

Pour les **int/float** : valeur maximale

```gmx
quantity: int @max(100)
```

#### `@email` — Validation Email

```gmx
email: string @email @unique
```

Génère :

```go
if u.Email != "" && !isValidEmail(u.Email) {
    return fmt.Errorf("email: invalid email format")
}
```

!!!note "Regex Email"
    Le helper `isValidEmail()` utilise une regex standard :
    ```go
    ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$
    ```

### Contraintes GORM

#### `@unique` — Unicité en Base

```gmx
email: string @unique
```

Génère : `gorm:"unique"`

#### `@scoped` — Multi-Tenancy Automatique

```gmx
<script>
model Post {
  tenantId: uuid @scoped
  title:    string
}
</script>
```

**Toutes les queries sont automatiquement scopées** :

```go
// find() ajoute automatiquement WHERE tenant_id = ?
db.Where("tenant_id = ?", ctx.Tenant).First(&post)

// save() injecte automatiquement le tenant_id
post.TenantID = ctx.Tenant
db.Save(&post)
```

### Relations

#### `@relation(references: [field])` — Foreign Key

```gmx
<script>
model Post {
  userId: uuid
  user:   User @relation(references: [id])
}
</script>
```

Génère :

```go
User User `gorm:"foreignKey:UserID" json:"user"`
```

**IMPORTANT** : Le champ FK doit exister (`userId` dans l'exemple).

## Méthodes ORM Générées

Pour chaque modèle, GMX génère automatiquement ces helpers dans le code transpilé :

### `find(id)` — Trouver par ID

```gmx
let task = try Task.find(id)
```

Transpilé en :

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
```

### `all()` — Tout Récupérer

```gmx
let tasks = try Task.all()
```

Transpilé en :

```go
tasks, err := TaskAll(ctx.DB)
if err != nil {
    return err
}
```

### `save()` — Créer ou Mettre à Jour

```gmx
const task = Task{title: "New task"}
try task.save()
```

Transpilé en :

```go
task := &Task{Title: "New task"}
if err := TaskSave(ctx.DB, task); err != nil {
    return err
}
```

### `delete()` — Supprimer

```gmx
let task = try Task.find(id)
try task.delete()
```

Transpilé en :

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
if err := TaskDelete(ctx.DB, task); err != nil {
    return err
}
```

## Exemples Complets

### Modèle Simple

```gmx
<script>
model User {
  id:        uuid    @pk @default(uuid_v4)
  email:     string  @email @unique
  createdAt: datetime
}
</script>
```

### Relation One-to-Many

```gmx
<script>
model Author {
  id:    uuid   @pk @default(uuid_v4)
  name:  string @min(2) @max(100)
  books: Book[]
}

model Book {
  id:       uuid   @pk @default(uuid_v4)
  authorId: uuid
  author:   Author @relation(references: [id])
  title:    string @min(1) @max(255)
  isbn:     string @unique
}
</script>
```

### Multi-Tenant avec Scoped

```gmx
<script>
model Organization {
  id:   uuid   @pk @default(uuid_v4)
  name: string @min(3) @max(100)
}

model Project {
  tenantId: uuid   @scoped
  id:       uuid   @pk @default(uuid_v4)
  orgId:    uuid
  org:      Organization @relation(references: [id])
  title:    string @min(3) @max(255)
}
</script>
```

**Utilisation dans le script** :

```gmx
<script>
func createProject(orgId: uuid, title: string) error {
  const project = Project{orgId: orgId, title: title}
  try project.save()  // tenantId injecté automatiquement
  return render(project)
}

func listProjects() error {
  let projects = try Project.all()  // filtre automatique par tenant
  return render(projects)
}
</script>
```

## Validation Automatique

La méthode `Validate()` est **appelée automatiquement** avant `save()` :

```go
func TaskSave(db *gorm.DB, obj *Task) error {
    if err := obj.Validate(); err != nil {
        return err
    }
    return db.Save(obj).Error
}
```

Vous pouvez aussi l'appeler manuellement dans le script :

```gmx
<script>
func createTask(title: string) error {
  const task = Task{title: title, done: false}

  // Validation manuelle (optionnelle, déjà faite dans save())
  if err := task.Validate(); err != nil {
    return error("Validation failed: " + err.Error())
  }

  try task.save()
  return render(task)
}
</script>
```

## Hooks GORM

### BeforeCreate

Généré automatiquement pour les champs `@default(uuid_v4)` :

```go
func (t *Task) BeforeCreate(tx *gorm.DB) error {
    if t.ID == "" {
        t.ID = generateUUID()
    }
    return nil
}
```

### Hooks Personnalisés (Future)

!!!warning "Non Implémenté"
    Les hooks personnalisés (BeforeUpdate, AfterDelete, etc.) ne sont pas encore supportés. Utilisez GORM directement dans le code généré si nécessaire.

## Migration de Base de Données

GMX génère l'AutoMigrate dans `main()` :

```go
func main() {
    // ...
    db.AutoMigrate(&Task{}, &User{}, &Post{})
    // ...
}
```

**IMPORTANT** : AutoMigrate ne **supprime pas** les colonnes. Pour une migration complète, utilisez un outil comme [golang-migrate](https://github.com/golang-migrate/migrate).

## Bonnes Pratiques

### ✅ Do

- Toujours utiliser `@pk @default(uuid_v4)` pour les IDs
- Ajouter `@min` et `@max` pour les strings
- Utiliser `@email` pour les champs email
- Utiliser `@scoped` pour le multi-tenancy
- Tester la validation avec des cas limites

### ❌ Don't

- Ne pas oublier `@unique` pour les emails
- Ne pas utiliser `int` comme clé primaire (préférer `uuid`)
- Ne pas skip la validation dans le script
- Ne pas créer des relations sans foreign keys

## Limitations Actuelles

| Fonctionnalité | Status |
|----------------|--------|
| Types primitifs (string, int, uuid, bool) | ✅ Implémenté |
| Relations (one-to-many, belongs-to) | ✅ Implémenté |
| Validation (@min, @max, @email) | ✅ Implémenté |
| Multi-tenancy (@scoped) | ✅ Implémenté |
| Defaults (@default) | ✅ Implémenté |
| Many-to-many | ❌ Non implémenté |
| Indexes composites | ❌ Non implémenté |
| Soft deletes | ❌ Non implémenté |
| Hooks personnalisés | ❌ Non implémenté |

## Prochaines Étapes

- **[Script](script.md)** — Utiliser les modèles dans la logique métier
- **[Templates](templates.md)** — Afficher les modèles dans les templates
- **[Security](security.md)** — Validation et sécurité des modèles
