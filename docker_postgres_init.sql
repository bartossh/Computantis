CREATE DATABASE computantis
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

\c computantis

CREATE TYPE address_access_level AS ENUM ('suspended', 'standard', 'trusted', 'admin');

CREATE TABLE IF NOT EXISTS addresses (
   id serial PRIMARY KEY,
   public_key VARCHAR ( 64 ) UNIQUE NOT NULL,
   access_level address_access_level NOT NULL
);

CREATE INDEX address_public_key ON addresses USING HASH (public_key);
CREATE INDEX address_access_level ON addresses USING HASH (access_level);

CREATE TYPE transaction_status AS ENUM ('awaited', 'temporary', 'permanent', 'rejected');
CREATE TABLE IF NOT EXISTS transactions(
   id serial PRIMARY KEY,
   created_at BIGINT NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   issuer_address VARCHAR ( 64 ) NOT NULL,
   receiver_address VARCHAR ( 64 ) NOT NULL,
   subject VARCHAR ( 100 ) NOT NULL,
   data BYTEA,
   issuer_signature BYTEA NOT NULL,
   receiver_signature BYTEA NOT NULL,
   status transaction_status NOT NULL,
   block_hash BYTEA
);

CREATE INDEX transaction_hash ON transactions USING HASH (hash);
CREATE INDEX transaction_status ON transactions USING HASH (status);
CREATE INDEX transaction_issuer_address_status ON transactions USING BTREE (issuer_address, status);
CREATE INDEX transaction_receiver_address_status ON transactions USING BTREE (receiver_address, status);
CREATE INDEX transaction_issuer_address_created_at ON transactions USING BTREE (issuer_address, created_at);
CREATE INDEX transaction_receiver_address_created_at ON transactions USING BTREE (receiver_address, created_at);
CREATE INDEX transaction_receiver_address_hash ON transactions USING BTREE (receiver_address, hash);

CREATE TABLE IF NOT EXISTS blocks (
   id serial PRIMARY KEY,
   index BIGINT UNIQUE NOT NULL,
   timestamp BIGINT NOT NULL,
   nonce INTEGER NOT NULL,
   difficulty INTEGER NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   prev_hash BYTEA NOT NULL,
   trx_hashes BYTEA[] NOT NULL
);

CREATE INDEX block_index ON blocks USING HASH (index);
CREATE INDEX block_hash ON blocks USING HASH (hash);
CREATE INDEX block_prev_hash ON blocks USING HASH (prev_hash);
CREATE INDEX block_created_at ON blocks USING BTREE (timestamp);

CREATE TABLE IF NOT EXISTS tokens (
   id serial PRIMARY KEY,
   token VARCHAR (100) UNIQUE NOT NULL,
   valid BOOLEAN NOT NULL,
   expiration_date BIGINT NOT NULL
);

CREATE INDEX token_token ON tokens USING HASH (token);
CREATE INDEX token_expiration_date ON tokens USING BTREE (expiration_date);

CREATE TABLE IF NOT EXISTS logs (
   id serial PRIMARY KEY,
   level VARCHAR ( 10 ) NOT NULL,
   msg VARCHAR ( 256 ) NOT NULL,
   created_at BIGINT NOT NULL
);

CREATE INDEX logs_created_at ON logs USING BTREE (created_at);
CREATE INDEX logs_level_created_at ON logs USING BTREE (level, created_at);

CREATE TABLE IF NOT EXISTS validatorStatus (
   id serial PRIMARY KEY,
   index INTEGER UNIQUE NOT NULL,
   valid BOOLEAN NOT NULL,
   created_at BIGINT NOT NULL,
   FOREIGN KEY (index) REFERENCES blocks (index)
);

CREATE INDEX validator_index ON validatorStatus USING HASH (index);
CREATE INDEX validator_created_at ON validatorStatus USING BTREE (created_at);

CREATE TABLE IF NOT EXISTS nodes (
   id serial PRIMARY KEY,
   node VARCHAR ( 64 ) UNIQUE NOT NULL,
   websocket VARCHAR ( 256 ) UNIQUE NOT NULL
);

CREATE INDEX nodes_index ON nodes USING HASH (node);

CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $$

    DECLARE 
        data json;
        notification json;
    
    BEGIN
    
        IF (TG_OP = 'DELETE') THEN
            data = row_to_json(OLD);
        ELSE
            data = row_to_json(NEW);
        END IF;

        notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'data', data);
        
                        
        PERFORM pg_notify('events',notification::text);
        
        RETURN NULL; 
    END;
    
$$ LANGUAGE plpgsql;

CREATE TABLE blockchainLocks (
    id serial PRIMARY KEY,
    timestamp BIGINT NOT NULL,
    node VARCHAR ( 100 ) NOT NULL
);

CREATE INDEX blockchainLocks_timestamp ON blockchainLocks USING BTREE (timestamp);
CREATE INDEX blockchainLocks_node ON blockchainLocks USING HASH (node);

CREATE TRIGGER blockchainLocks_notify_event
AFTER INSERT OR UPDATE OR DELETE ON blockchainLocks
    FOR EACH ROW EXECUTE PROCEDURE notify_event();


CREATE USER computantis WITH ENCRYPTED PASSWORD 'computantis';

GRANT ALL PRIVILEGES ON DATABASE computantis TO computantis;
GRANT ALL PRIVILEGES ON SCHEMA public TO computantis;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO computantis;
GRANT ALL ON ALL TABLES IN SCHEMA public TO computantis;
GRANT All PRIVILEGES ON FUNCTION notify_event() TO computantis;
