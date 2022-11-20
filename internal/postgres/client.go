package postgres

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
)

type Client struct {
	ctx    context.Context
	logger logr.Logger

	conn *pgx.Conn
}

func NewClient(ctx context.Context, logger logr.Logger, connStr string) (*Client, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		logger.Error(err, "Failed to open database connection")
		return nil, err
	}

	return &Client{
		ctx:    ctx,
		logger: logger,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close(c.ctx)
	} else {
		return fmt.Errorf("Connection is nil")
	}
}

func (c *Client) EnsureDatabase(name string) error {
	var (
		alreadyExists bool = false
		datname       string
		datacl        *string
	)
	err := c.conn.QueryRow(c.ctx, getDatabaseQuery(name)).Scan(&datname, &datacl)
	switch {
	case err == pgx.ErrNoRows:
		alreadyExists = false
	case err != nil:
		c.logger.Error(err, "Failed to query from pg_database")
		return err
	default:
		alreadyExists = true
	}

	if alreadyExists {
		c.logger.Info(fmt.Sprintf("Found database with name '%s' from pg_database", name), "datname", datname, "datacl", datacl)
		return nil
	}
	c.logger.Info(fmt.Sprintf("No database with name %s. Creating...", name))

	if _, err := c.conn.Exec(c.ctx, createDatabaseQuery(name)); err != nil {
		c.logger.Error(err, "Failed to create a database")
		return err
	}
	c.logger.Info("Successfully created a database")

	return nil
}

func (c *Client) EnsureDatabaseAccessRoles(name string) error {
	readonlyRole := name + "_readonly"
	if err := c.EnsureRole(readonlyRole); err != nil {
		return err
	}

	if _, err := c.conn.Exec(c.ctx, grantConnectQuery(readonlyRole, name)); err != nil {
		c.logger.Error(err, "Failed to grant readonly privilege")
		return err
	}
	if _, err := c.conn.Exec(c.ctx, grantUsageQuery(readonlyRole)); err != nil {
		c.logger.Error(err, "Failed to grant readonly privilege")
		return err
	}
	if _, err := c.conn.Exec(c.ctx, grantReadOnlyOnTablesQuery(readonlyRole)); err != nil {
		c.logger.Error(err, "Failed to grant readonly privilege")
		return err
	}
	c.logger.Info("Successfully granted readonly privilege")

	readwriteRole := name + "_readwrite"
	if err := c.EnsureRole(readwriteRole); err != nil {
		return err
	}

	if _, err := c.conn.Exec(c.ctx, grantConnectQuery(readwriteRole, name)); err != nil {
		c.logger.Error(err, "Failed to grant readwrite privilege")
		return err
	}
	if _, err := c.conn.Exec(c.ctx, grantAllQuery(readwriteRole)); err != nil {
		c.logger.Error(err, "Failed to grant readwrite privilege")
		return err
	}
	if _, err := c.conn.Exec(c.ctx, grantReadWriteOnTablesQuery(readwriteRole)); err != nil {
		c.logger.Error(err, "Failed to grant readwrite privilege")
		return err
	}
	c.logger.Info("Successfully granted readwrite privilege")

	return nil
}

func (c *Client) EnsureRole(name string) error {
	var (
		alreadyExists bool = false
		rolname       string
		oid           uint32
	)
	err := c.conn.QueryRow(c.ctx, getRoleQuery(name)).Scan(&rolname, &oid)
	switch {
	case err == pgx.ErrNoRows:
		alreadyExists = false
	case err != nil:
		c.logger.Error(err, "Failed to query from pg_roles")
		return err
	default:
		alreadyExists = true
	}

	if alreadyExists {
		c.logger.Info(fmt.Sprintf("Found role with name '%s' from pg_roles", name), "rolname", rolname, "oid", oid)
		return nil
	}

	c.logger.Info(fmt.Sprintf("No role with name %s. Creating...", name))

	if _, err := c.conn.Exec(c.ctx, createRoleQuery(name)); err != nil {
		c.logger.Error(err, "Failed to create a role")
		return err
	}
	c.logger.Info("Successfully created a role")

	return nil
}

func (c *Client) EnsureUser(name, password string) error {
	var (
		alreadyExists bool = false
		usename       string
		usesysid      uint32
	)
	err := c.conn.QueryRow(c.ctx, getUserQuery(name)).Scan(&usename, &usesysid)
	switch {
	case err == pgx.ErrNoRows:
		alreadyExists = false
	case err != nil:
		c.logger.Error(err, "Failed to query from pg_user")
		return err
	default:
		alreadyExists = true
	}

	if alreadyExists {
		c.logger.Info(fmt.Sprintf("Found user with name '%s' from pg_user", name), "usename", usename, "usesysid", usesysid)
	} else {
		c.logger.Info(fmt.Sprintf("No user with name %s. Creating...", name))

		if _, err := c.conn.Exec(c.ctx, createUserQuery(name)); err != nil {
			c.logger.Error(err, "Failed to create an user")
			return err
		}
		c.logger.Info("Successfully created an user")
	}

	if _, err := c.conn.Exec(c.ctx, setPasswordQuery(name, password)); err != nil {
		c.logger.Error(err, "Failed to set password for an user")
		return err
	}
	c.logger.Info("Successfully set password for the user")

	// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.MasterAccounts.html
	if _, err := c.conn.Exec(c.ctx, grantRoleToUserQuery(name, c.conn.Config().User)); err != nil {
		c.logger.Error(err, "Failed to grant user role to root")
		return err
	}
	c.logger.Info("Successfully grant user role to root")

	return nil
}

func (c *Client) EnsureReadonlyRoleToUser(dbname, username string) error {
	if _, err := c.conn.Exec(c.ctx, grantRoleToUserQuery(dbname+"_readonly", username)); err != nil {
		c.logger.Error(err, "Failed to grant a role to an user")
		return err
	}

	c.logger.Info("Successfully granted a role to an user")

	return nil
}

func (c *Client) EnsureReadwriteRoleToUser(dbname, username string) error {
	if _, err := c.conn.Exec(c.ctx, grantRoleToUserQuery(dbname+"_readwrite", username)); err != nil {
		c.logger.Error(err, "Failed to grant a role to an user")
		return err
	}

	if _, err := c.conn.Exec(c.ctx, grantFutureQuery("SELECT", username, dbname+"_readonly")); err != nil {
		c.logger.Error(err, "Failed to grant default privilege")
		return err
	}
	c.logger.Info("Successfully granted a role to an user")

	return nil
}
