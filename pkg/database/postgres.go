package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewPostgreSQLConnection() (*sql.DB, error) {
	// Priorizar DATABASE_URL (Neon)
	databaseURL := os.Getenv("DATABASE_URL")
	
	var dsn string
	var db *sql.DB
	var err error

	if databaseURL != "" {
		// ===== MODO NEON: Usar URL completa =====
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("☁️  Conectando a Neon PostgreSQL...")
		fmt.Printf("   URL: %s...\n", databaseURL[:50])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		
		dsn = databaseURL
	} else {
		// ===== MODO LOCAL: Usar variables individuales =====
		config := PostgresConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "mangaty"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("🔌 Configuración de Base de Datos (Local):")
		fmt.Printf("  Host:     %s\n", config.Host)
		fmt.Printf("  Puerto:   %s\n", config.Port)
		fmt.Printf("  Usuario:  %s\n", config.User)
		fmt.Printf("  DB:       %s\n", config.DBName)
		fmt.Printf("  SSL:      %s\n", config.SSLMode)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
		)
	}

	// Abrir conexión
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("❌ error abriendo conexión: %w", err)
	}

	// Configurar pool de conexiones (importante para Neon)
	db.SetMaxOpenConns(20)   // Neon tiene límite de conexiones
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verificar conexión con reintentos
	fmt.Println("⏳ Verificando conexión...")
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			fmt.Printf("✅ Conexión exitosa (intento %d)\n", i)
			break
		}
		if i < maxRetries {
			fmt.Printf("  ⚠️  Intento %d fallido, reintentando en 2s...\n", i)
			time.Sleep(2 * time.Second)
		} else {
			return nil, fmt.Errorf("❌ error conectando después de %d intentos: %w", maxRetries, err)
		}
	}

	// Verificar extensiones
	if err := checkExtensions(db); err != nil {
		fmt.Printf("⚠️  Advertencia extensiones: %v\n", err)
	}

	// Ejecutar migraciones
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("❌ error ejecutando migraciones: %w", err)
	}

	fmt.Println("✅ Base de datos inicializada correctamente")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return db, nil
}

func checkExtensions(db *sql.DB) error {
	extensions := []string{"uuid-ossp", "vector"}
	for _, ext := range extensions {
		var available bool
		err := db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = $1)", ext,
		).Scan(&available)
		if err != nil {
			return fmt.Errorf("error verificando extensión %s: %w", ext, err)
		}
		if available {
			fmt.Printf("  ✓ Extensión %s disponible\n", ext)
		} else {
			fmt.Printf("  ⚠️  Extensión %s no disponible (puede que ya esté instalada)\n", ext)
		}
	}
	return nil
}

func runMigrations(db *sql.DB) error {
	fmt.Println("📊 Ejecutando migraciones...")

	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "uuid-ossp",
			sql:  `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		},
		{
			name: "vector",
			sql:  `CREATE EXTENSION IF NOT EXISTS vector`,
		},
		{
			name: "categories",
			sql: `CREATE TABLE IF NOT EXISTS categories (
				id SERIAL PRIMARY KEY,
				name VARCHAR(50) UNIQUE NOT NULL,
				description TEXT
			)`,
		},
		{
			name: "seed_categories",
			sql: `INSERT INTO categories (name, description) VALUES 
				('Acción', 'Historias con mucha adrenalina y combates'),
				('Fantasía', 'Mundos mágicos y criaturas legendarias'),
				('Sci-Fi', 'Tecnología avanzada y viajes espaciales'),
				('Romance', 'Historias centradas en relaciones amorosas'),
				('Terror', 'Contenido diseñado para asustar o inquietar'),
				('Comedia', 'Historias divertidas y situaciones cómicas'),
				('Drama', 'Historias con conflictos emocionales profundos'),
				('Aventura', 'Viajes épicos y descubrimientos')
			ON CONFLICT (name) DO NOTHING`,
		},
		{
			name: "users",
			sql: `CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				username VARCHAR(50) UNIQUE NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "roles",
			sql: `CREATE TABLE IF NOT EXISTS roles (
				id SERIAL PRIMARY KEY,
				name VARCHAR(20) UNIQUE NOT NULL,
				description TEXT
			)`,
		},
		{
			name: "user_roles",
			sql: `CREATE TABLE IF NOT EXISTS user_roles (
				user_id UUID REFERENCES users(id) ON DELETE CASCADE,
				role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
				granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				granted_by UUID REFERENCES users(id),
				PRIMARY KEY (user_id, role_id)
			)`,
		},
		{
			name: "seed_roles",
			sql: `INSERT INTO roles (name, description) VALUES 
				('lector', 'Puede leer cómics y desbloquear capítulos'),
				('autor', 'Puede crear cómics y publicar capítulos'),
				('administrador', 'Control total de la plataforma')
			ON CONFLICT (name) DO NOTHING`,
		},
		{
			name: "user_roles_view",
			sql: `CREATE OR REPLACE VIEW user_roles_view AS
				SELECT u.id as user_id, u.username, u.email, 
					   array_agg(r.name) as roles
				FROM users u
				LEFT JOIN user_roles ur ON u.id = ur.user_id
				LEFT JOIN roles r ON ur.role_id = r.id
				GROUP BY u.id, u.username, u.email`,
		},
		{
			name: "wallets",
			sql: `CREATE TABLE IF NOT EXISTS wallets (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				balance INT NOT NULL DEFAULT 0 CHECK (balance >= 0),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "comics",
			sql: `CREATE TABLE IF NOT EXISTS comics (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				title VARCHAR(150) NOT NULL,
				synopsis TEXT NOT NULL,
				embedding_vector VECTOR(768),
				published_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "comic_categories",
			sql: `CREATE TABLE IF NOT EXISTS comic_categories (
				comic_id UUID REFERENCES comics(id) ON DELETE CASCADE,
				category_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
				PRIMARY KEY (comic_id, category_id)
			)`,
		},
		{
			name: "comic_covers",
			sql: `CREATE TABLE IF NOT EXISTS comic_covers (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				comic_id UUID UNIQUE NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
				cloudinary_public_id TEXT NOT NULL,
				image_url TEXT NOT NULL,
				secure_url TEXT NOT NULL,
				thumbnail_url TEXT,
				medium_url TEXT,
				high_url TEXT,
				width INT DEFAULT 0,
				height INT DEFAULT 0,
				format VARCHAR(10) DEFAULT 'jpg',
				bytes BIGINT DEFAULT 0,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "chapters",
			sql: `CREATE TABLE IF NOT EXISTS chapters (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				comic_id UUID NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
				chapter_number INT NOT NULL,
				title VARCHAR(150),
				price_tycoins INT NOT NULL DEFAULT 0 CHECK (price_tycoins >= 0),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(comic_id, chapter_number)
			)`,
		},
		{
			name: "chapter_pages",
			sql: `CREATE TABLE IF NOT EXISTS chapter_pages (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				chapter_id UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
				page_number INT NOT NULL,
				cloudinary_public_id TEXT NOT NULL,
				image_url TEXT NOT NULL,
				secure_url TEXT NOT NULL,
				width INT DEFAULT 0,
				height INT DEFAULT 0,
				format VARCHAR(10) DEFAULT 'jpg',
				bytes BIGINT DEFAULT 0,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(chapter_id, page_number)
			)`,
		},
		{
			name: "transactions",
			sql: `CREATE TABLE IF NOT EXISTS transactions (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				wallet_id UUID NOT NULL REFERENCES wallets(id),
				chapter_id UUID NOT NULL REFERENCES chapters(id),
				amount INT NOT NULL,
				transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('unlock', 'reward', 'refund')),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "unlocked_chapters",
			sql: `CREATE TABLE IF NOT EXISTS unlocked_chapters (
				user_id UUID NOT NULL REFERENCES users(id),
				chapter_id UUID NOT NULL REFERENCES chapters(id),
				transaction_id UUID REFERENCES transactions(id),
				unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY(user_id, chapter_id)
			)`,
		},
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration.sql); err != nil {
			return fmt.Errorf("error en migración '%s': %w", migration.name, err)
		}
		fmt.Printf("  ✓ %s\n", migration.name)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}