import fs from "fs-extra";
import path from "path";
import { execa } from "execa";

export async function generateProject(projectName, projectPath, config) {
  await fs.ensureDir(projectPath);

  const envContent = generateDotEnv(config);
  await fs.writeFile(path.join(projectPath, ".env"), envContent);
  const gitignoreContent = generateGitignore();
  await fs.writeFile(path.join(projectPath, ".gitignore"), gitignoreContent);
  const readmeContent = generateReadme(projectName, config);
  await fs.writeFile(path.join(projectPath, "README.md"), readmeContent);

  const features = config.features || [config.name];

  for (const featureName of features) {
    const featureDir = path.join(
      projectPath,
      "features",
      featureName.toLowerCase()
    );
    await fs.ensureDir(featureDir);

    const handlerCode = generateHandler(featureName, config.database);
    await fs.writeFile(path.join(featureDir, "handler.go"), handlerCode);

    const routerCode = generateFeatureRouter(
      projectName,
      featureName.toLowerCase(),
      featureName
    );
    await fs.writeFile(path.join(featureDir, "router.go"), routerCode);

    const serviceCode = generateService(featureName);
    await fs.writeFile(path.join(featureDir, "service.go"), serviceCode);

    const repositoryCode = generateRepository(featureName, config.database);
    await fs.writeFile(path.join(featureDir, "repository.go"), repositoryCode);
  }

  const mainCode = generateMainFile(projectName, features);
  await fs.writeFile(path.join(projectPath, "main.go"), mainCode);

  await execa("go", ["mod", "init", projectName], { cwd: projectPath });

  await execa("go", ["get", "github.com/Dziqha/TurboGo@latest"], {
    cwd: projectPath,
  });
}

function generateMainFile(projectName, features) {
  const featureImports = features
    .map((f) => `\t"${projectName}/features/${f.toLowerCase()}"`)
    .join("\n");

  const featureRegistrations = features
    .map((f) => `\t${f.toLowerCase()}.RegisterRoutes(app)`)
    .join("\n");

  return `package main

import (
	"github.com/Dziqha/TurboGo"
${featureImports}
)

func main() {
	app := TurboGo.New()
	
	// Register feature routes
${featureRegistrations}
	
	app.RunServer(":8080")
}
`;
}

function generateHandler(name) {
  return `package ${name.toLowerCase()}

import (
	"github.com/Dziqha/TurboGo/core"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Get(c *core.Context) {
	result := h.service.GetAll()
	c.JSON(200, map[string]interface{}{
		"message": result,
	})
}

func (h *Handler) Create(c *core.Context) {
	// TODO: Parse request body
	result := h.service.Create("sample data")
	c.JSON(201, map[string]interface{}{
		"message": "created",
		"data":    result,
	})
}

func (h *Handler) GetByID(c *core.Context) {
  id := c.Param("id")
  result := h.service.GetByID(id)
  c.JSON(200, map[string]interface{}{
    "message": result,
  })
}

func (h *Handler) Update(c *core.Context) {
  id := c.Param("id")
  // TODO: Parse request body
  result := h.service.Update(id, "sample data")
  c.JSON(200, map[string]interface{}{
    "message": "updated",
    "data":    result,
  })
}

func (h *Handler) Delete(c *core.Context) {
  id := c.Param("id")
  h.service.Delete(id)
  c.JSON(200, map[string]interface{}{
    "message": "deleted",
  })
}
`;
}

function generateService(name) {
  return `package ${name.toLowerCase()}

import "errors"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetAll retrieves all ${name.toLowerCase()} records
func (s *Service) GetAll() string {
	return s.repo.FindAll()
}

// GetByID retrieves a ${name.toLowerCase()} by ID
func (s *Service) GetByID(id string) string {
	return s.repo.FindByID(id)
}

// Create creates a new ${name.toLowerCase()}
func (s *Service) Create(data string) string {
	// TODO: Add validation logic here
	// Example:
	// if data == "" {
	//     return ""
	// }
	return s.repo.Create(data)
}

// Update updates an existing ${name.toLowerCase()}
func (s *Service) Update(id string, data string) string {
	// TODO: Add validation logic here
	// Example:
	// if id == "" || data == "" {
	//     return ""
	// }
	return s.repo.Update(id, data)
}

// Delete deletes a ${name.toLowerCase()} by ID
func (s *Service) Delete(id string) error {
	// TODO: Add business logic validation here
	// Example:
	// if id == "" {
	//     return errors.New("id cannot be empty")
	// }
	
	exists := s.repo.FindByID(id)
	if exists == "" {
		return errors.New("${name.toLowerCase()} not found")
	}
	
	s.repo.Delete(id)
	return nil
}
`;
}

function generateRepository(name, database) {
  const hasDb = database !== null && database !== undefined;

  let dbComment = "";
  let dbField = "";
  let newRepoBody = "";

  if (hasDb) {
    dbComment = `	// db *sql.DB // Uncomment when you setup database connection`;
    newRepoBody = `	// return &Repository{db: db}`;
  } else {
    dbComment = `	// Add database connection here when needed
	// Example: db *sql.DB`;
  }

  return `package ${name.toLowerCase()}

type Repository struct {
${dbComment}
}

func NewRepository() *Repository {
	return &Repository{}
${newRepoBody}
}

func (r *Repository) FindAll() string {
	// TODO: Implement database query
	// Example:
	// rows, err := r.db.Query("SELECT * FROM ${name.toLowerCase()}s")
	// if err != nil {
	//     return ""
	// }
	// defer rows.Close()
	
	return "Hello from ${name} feature"
}

func (r *Repository) FindByID(id string) string {
	// TODO: Implement database query by ID
	// Example:
	// row := r.db.QueryRow("SELECT * FROM ${name.toLowerCase()}s WHERE id = $1", id)
	
	return "Data with ID: " + id
}

func (r *Repository) Create(data string) string {
	// TODO: Implement database insert
	// Example:
	// _, err := r.db.Exec("INSERT INTO ${name.toLowerCase()}s (data) VALUES ($1)", data)
	
	return data
}

func (r *Repository) Update(id string, data string) string {
	// TODO: Implement database update
	// Example:
	// _, err := r.db.Exec("UPDATE ${name.toLowerCase()}s SET data = $1 WHERE id = $2", data, id)
	
	return data
}

func (r *Repository) Delete(id string) {
	// TODO: Implement database delete
	// Example:
	// _, err := r.db.Exec("DELETE FROM ${name.toLowerCase()}s WHERE id = $1", id)
}
`;
}

function generateFeatureRouter(projectName, featureName, name) {
  return `package ${featureName}

import (
	"github.com/Dziqha/TurboGo/core"
)

func RegisterRoutes(router core.Router) {
	// Initialize dependencies
	repo := NewRepository()
	service := NewService(repo)
	handler := NewHandler(service)

	// Register routes
	api := router.Group("/api/${featureName}")
	{
		api.Get("/", handler.Get)                    // GET /api/${featureName}
		api.Post("/", handler.Create)                // POST /api/${featureName}
		api.Get("/:id", handler.GetByID)             // GET /api/${featureName}/:id
		api.Put("/:id", handler.Update)              // PUT /api/${featureName}/:id
		api.Delete("/:id", handler.Delete)           // DELETE /api/${featureName}/:id
	}
}
`;
}

function generateDotEnv(config) {
  let envContent = `# Application Configuration
PORT=8080
APP_NAME=TurboGoApp
ENV=development

`;

  if (config.database) {
    const dbType = config.database.type;

    envContent += `# Database Configuration\n`;

    switch (dbType) {
      case "PostgreSQL":
        envContent += `DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=turbogo_db
DB_USER=postgres
DB_PASSWORD=password
DB_SSLMODE=disable
DB_TIMEZONE=Asia/Jakarta

# Connection String (alternative)
DATABASE_URL=postgres://postgres:password@localhost:5432/turbogo_db?sslmode=disable
`;
        break;

      case "MySQL":
        envContent += `DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=turbogo_db
DB_USER=root
DB_PASSWORD=password
DB_CHARSET=utf8mb4
DB_PARSETIME=true
DB_LOC=Local

# Connection String (alternative)
DATABASE_URL=root:password@tcp(localhost:3306)/turbogo_db?charset=utf8mb4&parseTime=True&loc=Local
`;
        break;

      case "SQLite":
        envContent += `DB_DRIVER=sqlite
DB_PATH=./turbogo.db

# Connection String (alternative)
DATABASE_URL=file:./turbogo.db
`;
        break;

      case "MongoDB":
        envContent += `DB_DRIVER=mongodb
DB_HOST=localhost
DB_PORT=27017
DB_NAME=turbogo_db
DB_USER=
DB_PASSWORD=
DB_AUTH_SOURCE=admin

# Connection String (alternative)
MONGODB_URI=mongodb://localhost:27017/turbogo_db
`;
        break;

      default:
        envContent += `DB_HOST=localhost
DB_PORT=5432
DB_NAME=turbogo_db
DB_USER=postgres
DB_PASSWORD=password
`;
    }
  } else {
    envContent += `# Database Configuration (Not configured)
# Uncomment and configure if you decide to use a database later
# DB_DRIVER=postgres
# DB_HOST=localhost
# DB_PORT=5432
# DB_NAME=turbogo_db
# DB_USER=postgres
# DB_PASSWORD=password
`;
  }

  return envContent;
}

function generateGitignore() {
  return `# Binaries
bin/
*.exe
*.out
*.test

# Logs
*.log

# Environment
.env
.env.local

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`;
}

function generateReadme(project, config) {
  let dbSection = "";

  if (config.database) {
    const dbType = config.database.type;
    dbSection = `

## 🗄️ Database Setup (${dbType})

`;

    switch (dbType) {
      case "PostgreSQL":
        dbSection += `### Install PostgreSQL
\`\`\`bash
# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib

# macOS
brew install postgresql
\`\`\`

### Create Database
\`\`\`bash
psql -U postgres
CREATE DATABASE turbogo_db;
\\q
\`\`\`

### Update .env
Configure your PostgreSQL credentials in \`.env\` file.
`;
        break;

      case "MySQL":
        dbSection += `### Install MySQL
\`\`\`bash
# Ubuntu/Debian
sudo apt-get install mysql-server

# macOS
brew install mysql
\`\`\`

### Create Database
\`\`\`bash
mysql -u root -p
CREATE DATABASE turbogo_db;
EXIT;
\`\`\`

### Update .env
Configure your MySQL credentials in \`.env\` file.
`;
        break;

      case "SQLite":
        dbSection += `### SQLite Setup
SQLite tidak memerlukan instalasi server. Database file akan dibuat otomatis saat aplikasi pertama kali dijalankan.

File database akan tersimpan di: \`./turbogo.db\`
`;
        break;

      case "MongoDB":
        dbSection += `### Install MongoDB
\`\`\`bash
# Ubuntu/Debian
sudo apt-get install mongodb

# macOS
brew tap mongodb/brew
brew install mongodb-community
\`\`\`

### Start MongoDB
\`\`\`bash
# Linux
sudo systemctl start mongodb

# macOS
brew services start mongodb-community
\`\`\`

### Update .env
Configure your MongoDB connection in \`.env\` file.
`;
        break;
    }
  } else {
    dbSection = `

## 💡 No Database Configuration

Project ini di-generate **tanpa konfigurasi database**. Anda bisa:

1. **Menggunakan in-memory storage** untuk prototype
2. **Menambahkan database nanti** saat dibutuhkan
3. **Menggunakan external API** sebagai data source

### Menambahkan Database Nanti:

1. Update \`.env\` dengan konfigurasi database
2. Install driver database yang dibutuhkan:
\`\`\`bash
# PostgreSQL
go get github.com/lib/pq

# MySQL
go get github.com/go-sql-driver/mysql

# MongoDB
go get go.mongodb.org/mongo-driver/mongo
\`\`\`
3. Update \`repository.go\` dengan koneksi database
`;
  }

  const features = config.features || [config.name];

  return `# 🚀 ${project}

Generated with [TurboGo CLI](https://github.com/Dziqha/TurboGo)

## 📁 Feature-Based Architecture

Struktur project ini menggunakan **feature-based architecture** dimana setiap feature memiliki komponen lengkap:

\`\`\`
${project}/
├── features/
${features
  .map(
    (f) => `│   ├── ${f.toLowerCase()}/
│   │   ├── handler.go      # HTTP handlers
│   │   ├── service.go       # Business logic
│   │   ├── repository.go    # Data access
│   │   └── router.go        # Route registration`
  )
  .join("\n")}
├── main.go
├── .env
├── .gitignore
└── README.md
\`\`\`

## 🎯 Keuntungan Feature-Based:

- ✅ **Modular**: Setiap feature independen dan self-contained
- ✅ **Scalable**: Mudah menambah feature baru tanpa mengubah struktur lain
- ✅ **Maintainable**: Mudah menemukan dan mengubah kode berdasarkan feature
- ✅ **Testable**: Setiap layer dapat di-test secara terpisah
- ✅ **Team-Friendly**: Multiple developers dapat bekerja pada feature berbeda tanpa konflik
${dbSection}
## 🚦 Cara Menjalankan:

\`\`\`bash
# Install dependencies
go mod tidy

# Run aplikasi
go run .

# Build untuk production
go build -o bin/app
\`\`\`

Server akan berjalan di: \`http://localhost:8080\`

## 📝 Menambah Feature Baru:

### Manual:
1. Buat folder baru di \`features/[nama-feature]\`
2. Tambahkan \`handler.go\`, \`service.go\`, \`repository.go\`, \`router.go\`
3. Register routes di \`main.go\`

### Menggunakan CLI:
\`\`\`bash
create-turbogo add-feature
\`\`\`

## 🔗 API Endpoints:

${features
  .map(
    (f) => `### ${f} Feature
- \`GET /api/${f.toLowerCase()}\` - Get all ${f.toLowerCase()}
- \`POST /api/${f.toLowerCase()}\` - Create new ${f.toLowerCase()}
- \`GET /api/${f.toLowerCase()}/:id\` - Get ${f.toLowerCase()} by ID
- \`PUT /api/${f.toLowerCase()}/:id\` - Update ${f.toLowerCase()}
- \`DELETE /api/${f.toLowerCase()}/:id\` - Delete ${f.toLowerCase()}`
  )
  .join("\n\n")}

## 🏗️ Architecture Layers:

1. **Handler Layer**: Menangani HTTP requests dan responses
2. **Service Layer**: Business logic dan validasi
3. **Repository Layer**: Akses database dan data persistence

## 📚 Resources:

- [TurboGo Documentation](https://github.com/Dziqha/TurboGo)
- [Go Documentation](https://golang.org/doc/)
${config.database ? `- [${config.database.type} Documentation](${getDbDocUrl(config.database.type)})` : ""}

## 🤝 Contributing:

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License:

This project is licensed under the MIT License.
`;
}

function getDbDocUrl(dbType) {
  const urls = {
    PostgreSQL: "https://www.postgresql.org/docs/",
    MySQL: "https://dev.mysql.com/doc/",
    SQLite: "https://www.sqlite.org/docs.html",
    MongoDB: "https://docs.mongodb.com/",
  };
  return urls[dbType] || "#";
}

export async function addFeature(featureName, projectName, featureDir) {
  await fs.ensureDir(featureDir);

  const handlerCode = generateHandler(featureName, null);
  await fs.writeFile(path.join(featureDir, "handler.go"), handlerCode);

  const serviceCode = generateService(featureName);
  await fs.writeFile(path.join(featureDir, "service.go"), serviceCode);

  const repositoryCode = generateRepository(featureName, null);
  await fs.writeFile(path.join(featureDir, "repository.go"), repositoryCode);

  const routerCode = generateFeatureRouter(
    projectName,
    featureName.toLowerCase(),
    featureName
  );
  await fs.writeFile(path.join(featureDir, "router.go"), routerCode);
}

export async function updateReadmeWithFeature(readmePath, featureName) {
  let readmeContent = await fs.readFile(readmePath, "utf-8");

  const featureLower = featureName.toLowerCase();

  const structurePattern = /(├── features\/\n)([\s\S]*?)(├── main\.go)/;
  const structureMatch = readmeContent.match(structurePattern);

  if (structureMatch) {
    const featuresStart = structureMatch[1];
    const existingFeatures = structureMatch[2];
    const mainGoLine = structureMatch[3];

    if (existingFeatures.includes(`├── ${featureLower}/`)) {
      console.log(`Feature ${featureName} already exists in README structure`);
    } else {
      const newFeatureStructure = `│   ├── ${featureLower}/
│   │   ├── handler.go      # HTTP handlers
│   │   ├── service.go       # Business logic
│   │   ├── repository.go    # Data access
│   │   └── router.go        # Route registration
`;

      readmeContent = readmeContent.replace(
        structurePattern,
        `${featuresStart}${existingFeatures}${newFeatureStructure}${mainGoLine}`
      );
    }
  }

  const newEndpointsSection = `### ${featureName} Feature
- \`GET /api/${featureLower}\` - Get all ${featureLower}
- \`POST /api/${featureLower}\` - Create new ${featureLower}
- \`GET /api/${featureLower}/:id\` - Get ${featureLower} by ID
- \`PUT /api/${featureLower}/:id\` - Update ${featureLower}
- \`DELETE /api/${featureLower}/:id\` - Delete ${featureLower}`;

  if (readmeContent.includes(`### ${featureName} Feature`)) {
    console.log(`Feature ${featureName} endpoints already exists in README`);
  } else if (readmeContent.includes("## 🔗 API Endpoints:")) {
    readmeContent = readmeContent.replace(
      /(## 🔗 API Endpoints:\n\n)/,
      `$1${newEndpointsSection}\n\n`
    );
  }

  await fs.writeFile(readmePath, readmeContent);
  return true;
}