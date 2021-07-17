package postgis

import (
	"context"
	"fmt"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v4"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB creates and a new PostGIS database for testing,
// populates it with OpenStreetMap spatial surrogate data, and
// returns a URL to connect to the database and the running
// Docker container. dockerContext is the path to the directory containing
// the Docker context (e.g. the Dockerfile).
func SetupTestDB(ctx context.Context, t *testing.T, dockerContext string) (string, testcontainers.Container) {
	const (
		dbhost = "localhost"
		dbname = "postgresTC"
		dbuser = "postgres"
		dbport = "5432"
	)

	// Create the Postgres TestContainer
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       dockerContext,
			PrintBuildLog: false,
		},
		ExposedPorts: []string{fmt.Sprintf("%s/tcp", dbport)},
		Env: map[string]string{
			"POSTGRES_DB":               dbname,
			"DBHOST":                    dbhost,
			"DBNAME":                    dbname,
			"DBUSER":                    dbuser,
			"DBPORT":                    dbport,
			"POSTGRES_HOST_AUTH_METHOD": "trust",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the port that is mapped to 5432.
	p, _ := postgresC.MappedPort(ctx, "5432")

	postGISURL := fmt.Sprintf("postgres://%s@%s:%s/%s", dbuser, dbhost, p.Port(), dbname)

	var conn *pgx.Conn
	err = backoff.Retry(func() error {
		conn, err = pgx.Connect(context.Background(), postGISURL)
		if err != nil {
			return err
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS hstore"); err != nil {
		t.Fatal(err)
	}

	// Populate database with OSM data for Honolulu, using lat-lon (EPSG:4326) projection.
	cmd := []string{"osm2pgsql", "-l", "--hstore-all", "--hstore-add-index", "--database=" + dbname, "--host=" + dbhost,
		"--port=" + dbport, "--username=" + dbuser, "--create", "/honolulu_hawaii.osm.pbf"}
	status, err := postgresC.Exec(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("osm2pgsql failed with nonzero status ", status)
	}

	return postGISURL, postgresC
}
