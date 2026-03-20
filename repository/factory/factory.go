package factory

import (
	"database/sql"
	"errors"
	"fmt"

	"reverse-watch/config"
	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	"reverse-watch/domain/secret"
	"reverse-watch/logging"
	"reverse-watch/repository/private"
	"reverse-watch/repository/public"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type factory struct {
	private *gorm.DB
	public  *gorm.DB
	keygen  secret.KeyGenerator

	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository
}

type Config struct {
	PrivateDB *gorm.DB
	PublicDB  *gorm.DB
	KeyGen    secret.KeyGenerator
}

func NewFactoryWithConfig(cfg *Config) (repository.Factory, error) {
	if cfg == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if cfg.PrivateDB == nil {
		return nil, fmt.Errorf("pivate database is required")
	}
	if cfg.PublicDB == nil {
		return nil, fmt.Errorf("public database is required")
	}
	if cfg.KeyGen == nil {
		return nil, fmt.Errorf("key generator is required")
	}

	return &factory{
		private:     cfg.PrivateDB,
		public:      cfg.PublicDB,
		keygen:      cfg.KeyGen,
		key:         private.NewKeyRepository(cfg.PrivateDB, cfg.KeyGen),
		marketplace: private.NewMarketplaceRepository(cfg.PrivateDB),
		adminAudit:  private.NewAdminAuditRepository(cfg.PrivateDB),
		reversal:    public.NewReversalRepository(cfg.PublicDB),
	}, nil
}

func constructDSN(host, user, password, dbname, sslmode string, port *string) string {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		user,
		password,
		dbname,
		sslmode,
	)
	if port != nil {
		dsn += fmt.Sprintf(" port=%s", *port)
	}
	return dsn
}

func NewFactory(cfg config.Config, keygen secret.KeyGenerator) (repository.Factory, error) {
	var privateDSN, publicDSN string
	switch cfg.Environment {
	case constants.EnvironmentProduction:
		host := fmt.Sprintf("/cloudsql/%s", cfg.Database.Host)
		privateDSN = constructDSN(host, cfg.Database.User, cfg.Database.Password, cfg.Database.PrivateDBName, "disable", nil)
		publicDSN = constructDSN(host, cfg.Database.User, cfg.Database.Password, cfg.Database.PublicDBName, "disable", nil)
	case constants.EnvironmentDevelopment:
		privateDSN = constructDSN(cfg.Database.Host, cfg.Database.User, cfg.Database.Password, cfg.Database.PrivateDBName, cfg.Database.SSLMode, &cfg.Database.Port)
		publicDSN = constructDSN(cfg.Database.Host, cfg.Database.User, cfg.Database.Password, cfg.Database.PublicDBName, cfg.Database.SSLMode, &cfg.Database.Port)
	default:
		return nil, fmt.Errorf("unknown environment: %s", cfg.Environment)
	}

	sqlPrivateDB, err := sql.Open("pgx", privateDSN)
	if err != nil {
		return nil, err
	}

	privateDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlPrivateDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		sqlPrivateDB.Close()
		return nil, fmt.Errorf("failed to open private database: %w", err)
	}

	sqlPublicDB, err := sql.Open("pgx", publicDSN)
	if err != nil {
		sqlPrivateDB.Close()
		return nil, err
	}

	publicDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlPublicDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		sqlPrivateDB.Close()
		sqlPublicDB.Close()
		return nil, fmt.Errorf("failed to open public database: %w", err)
	}

	f, err := NewFactoryWithConfig(&Config{
		PrivateDB: privateDB,
		PublicDB:  publicDB,
		KeyGen:    keygen,
	})
	if err != nil {
		sqlPrivateDB.Close()
		sqlPublicDB.Close()
		return nil, fmt.Errorf("failed to initialize factory: %w", err)
	}

	// Setup private database
	if err := private.MigrateModels(privateDB); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to migrate private models: %w", err)
	}

	if err := private.SeedMarketplaces(privateDB); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to seed marketplaces: %w", err)
	}

	if err := private.SeedAdminAPIKey(privateDB, keygen); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to seed admin API key: %w", err)
	}

	// Setup public database
	if err := public.MigrateModels(publicDB); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to migrate public models: %w", err)
	}

	if err := public.CreateIndexes(publicDB); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return f, nil
}

func (f *factory) Key() repository.KeyRepository {
	return f.key
}

func (f *factory) Marketplace() repository.MarketplaceRepository {
	return f.marketplace
}

func (f *factory) AdminAudit() repository.AdminAuditRepository {
	return f.adminAudit
}

func (f *factory) Reversal() repository.ReversalRepository {
	return f.reversal
}

func (f *factory) Close() error {
	var errs []error
	if err := closeDB(f.private); err != nil {
		logging.Log.Errorf("failed to close private database: %v", err)
		errs = append(errs, fmt.Errorf("private database: %w", err))
	}
	if err := closeDB(f.public); err != nil {
		logging.Log.Errorf("failed to close public database: %v", err)
		errs = append(errs, fmt.Errorf("public database: %w", err))
	}
	return errors.Join(errs...)
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (f *factory) NewPrivateTransaction() repository.PrivateTransaction {
	return newPrivateTransaction(f.private.Begin(), f.keygen)
}

func (f *factory) RunInTransactionPrivate(fn func(repository.PrivateTransaction) error) error {
	return f.private.Transaction(func(gormTx *gorm.DB) error {
		tx := newPrivateTransaction(gormTx, f.keygen)
		return fn(tx)
	})
}

func (f *factory) PrivateDB() *gorm.DB {
	return f.private
}

func (f *factory) NewPublicTransaction() repository.PublicTransaction {
	return newPublicTransaction(f.public.Begin())
}

func (f *factory) RunInTransactionPublic(fn func(repository.PublicTransaction) error) error {
	return f.public.Transaction(func(gormTx *gorm.DB) error {
		tx := newPublicTransaction(gormTx)
		return fn(tx)
	})
}

func (f *factory) PublicDB() *gorm.DB {
	return f.public
}
