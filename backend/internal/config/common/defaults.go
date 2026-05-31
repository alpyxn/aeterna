package common

const (
	DefaultDatabasePath    = "./data/aeterna.db"
	DefaultAllowedOrigins  = "http://localhost:5173"
	DefaultWorkerBaseURL   = "http://localhost:5173"
	DefaultSessionTTLHours = 168
	DefaultRefreshTTLHours = 720
	DefaultLogMaxSize      = 50
	DefaultLogMaxBackups   = 5
	DefaultLogMaxAge       = 14
	DefaultLogCompress     = true

	DefaultDBEncryptionEnabled        = false
	DefaultDBEncryptionAutoMigrate    = true
	DefaultDBEncryptionKDFContextFile = "./secrets/db_kdf_context"
)
