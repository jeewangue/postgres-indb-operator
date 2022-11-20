package postgres

import (
	"fmt"
)

func getUserQuery(name string) string {
	return fmt.Sprintf("SELECT usename, usesysid FROM pg_user WHERE usename = '%s'", name)
}

func createUserQuery(name string) string {
	return fmt.Sprintf("CREATE ROLE %s WITH "+
		"LOGIN "+
		"NOSUPERUSER "+
		"NOCREATEDB "+
		"NOCREATEROLE "+
		"INHERIT "+
		"NOREPLICATION "+
		"CONNECTION LIMIT -1", name)
}

func setPasswordQuery(name, password string) string {
	return fmt.Sprintf("ALTER ROLE %s PASSWORD '%s'", name, password)
}

func getDatabaseQuery(name string) string {
	return fmt.Sprintf("SELECT datname, datacl FROM pg_database WHERE datname = '%s'", name)
}

func createDatabaseQuery(name string) string {
	return fmt.Sprintf("CREATE DATABASE %s", name)
}

func getRoleQuery(name string) string {
	return fmt.Sprintf("SELECT rolname, oid FROM pg_roles WHERE rolname = '%s'", name)
}

func createRoleQuery(name string) string {
	return fmt.Sprintf("CREATE ROLE  %s", name)
}

func grantRoleToUserQuery(role, user string) string {
	return fmt.Sprintf("GRANT %s TO %s", role, user)
}

func grantOnTablesQuery(privileges, role string) string {
	return fmt.Sprintf("GRANT %s ON ALL TABLES IN SCHEMA public TO %s", privileges, role)
}

func grantReadOnlyOnTablesQuery(role string) string {
	return grantOnTablesQuery("SELECT", role)
}

func grantReadWriteOnTablesQuery(role string) string {
	return grantOnTablesQuery("SELECT, INSERT, UPDATE, DELETE", role)
}

func grantAllOnDatabase(role, dbName string) string {
	return fmt.Sprintf("GRANT ALL ON DATABASE %s TO %s", dbName, role)
}

func grantAllOnPublicQuery(role string) string {
	return fmt.Sprintf("GRANT ALL ON SCHEMA public TO %s", role)
}

func grantUsageOnPublicQuery(role string) string {
	return fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", role)
}

// grantFutureQuery grant access to future tables
func grantFutureQuery(privileges, user, role string) string {
	return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR USER %s IN SCHEMA public GRANT %s ON TABLES TO %s", user, privileges, role)
}

// grantConnectQuery grant connect on database
func grantConnectQuery(role, dbName string) string {
	return fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", dbName, role)
}
