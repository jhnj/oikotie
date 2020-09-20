DO $$
BEGIN
	IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'johan') THEN
		CREATE ROLE johan;
	END IF;
	ALTER ROLE johan WITH LOGIN PASSWORD 'password' NOSUPERUSER CREATEDB NOCREATEROLE;
END $$;

DROP DATABASE IF EXISTS oikotie;
CREATE DATABASE oikotie OWNER johan;
GRANT ALL ON DATABASE oikotie TO johan;
