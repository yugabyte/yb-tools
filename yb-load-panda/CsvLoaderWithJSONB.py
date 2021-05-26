# Python 3.7.7
# create virtual env
# mkdir project; cd project
# python3 -m venv venv/
# source venv/bin/activate
# pip3 install pandas
# pip3 install sqlalchemy
# pip3 install psycopg2
# pip3 install pyyaml
# Do the following if installing psycopg2 throws error:
# export LDFLAGS="-L/usr/local/opt/openssl/lib"
# export CPPFLAGS="-I/usr/local/opt/openssl/include"

# Imports
import pandas as pd
import sqlalchemy as sa
from sqlalchemy import create_engine
import psycopg2
import psycopg2.extras
from psycopg2 import extensions, connect
import time
from sqlalchemy import exc
from sqlalchemy.dialects.postgresql import insert
import yaml

# You can download the sample data file from here:
# curl -s -O https://raw.githubusercontent.com/yugabyte/yugabyte-db/master/src/postgres/src/test/regress/data/airport-codes.csv
#
meta = sa.MetaData()
constraint_name = ""


def get_all_column_names_information_schema(connection_url, connect_args, table):
    column_names = []
    data_types = []
    with psycopg2.connect(connection_url, **connect_args) as connection:
        with connection.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cursor:
            cursor.execute(
                "select column_name, data_type from information_schema.columns where table_name='{}'".format(table))
            return cursor.fetchall()
            # (column_names, data_types) = [(row[0], row[1]) for row in cursor]


def find_constraint_name(connection_url, connect_args, table):
    """constraint"""

    conn = psycopg2.connect(connection_url, **connect_args)
    conn.set_session(autocommit=True)
    cur = conn.cursor()

    query = """select distinct tco.constraint_name
        from information_schema.table_constraints tco
         join information_schema.key_column_usage kcu
              on kcu.constraint_name = tco.constraint_name
                  and kcu.constraint_schema = tco.constraint_schema
        where kcu.table_name = '{}'
              and ( constraint_type = 'PRIMARY KEY');""".format(table)

    #  also handle constraint_type = 'UNIQUE' or perhaps just handling primary will work.
    constraint = ""
    cur.execute(query)

    for row in cur.fetchall():
        constraint = row[0]

    return constraint


def upsert_overwrite(table, conn, keys, data_iter):
    upsert_args = {"constraint": constraint_name}

    for data in data_iter:
        data = {k: data[i] for i, k in enumerate(keys)}
        upsert_args["set_"] = data
        insert_stmt = insert(meta.tables[table.name]).values(**data)
        upsert_stmt = insert_stmt.on_conflict_do_update(**upsert_args)
        conn.execute(upsert_stmt, data)


def upsert_ignore(table, conn, keys, data_iter):
    for data in data_iter:
        data = {k: data[i] for i, k in enumerate(keys)}
        insert_stmt = insert(meta.tables[table.name]).values(**data)
        upsert_stmt = insert_stmt.on_conflict_do_nothing()
        conn.execute(upsert_stmt, data)


def execute_ddl(connection_url, connect_args, ddl):
    conn = psycopg2.connect(connection_url, **connect_args)

    conn.set_session(autocommit=True)
    cur = conn.cursor()
    cur.execute(ddl)

    print("EXECUTED: {}".format(ddl))
    print("====================")


def load_data(connection_url, connect_args,
              table_name, csv_file, checkpoint_file_name, chunksize, on_conflict, delimiter, jsonb_columns):
    # Instantiate sqlachemy.create_engine object
    # engine = create_engine(connection_url, connect_args=connect_args)
    engine = create_engine(connection_url, connect_args=connect_args,
                           execution_options={
                               "isolation_level": "AUTOCOMMIT"
                           }
                           )
    connection = engine.connect().connection

    print("isolation_level:", connection.isolation_level)
    # print the isolation level for ISOLATION_LEVEL_AUTOCOMMIT
    print("ISOLATION_LEVEL_AUTOCOMMIT:", extensions.ISOLATION_LEVEL_DEFAULT)

    # Create an iterable that will read chunksize rows at a time.

    meta.bind = engine
    meta.reflect(views=True)
    checkpoint = 1
    success = False
    dtypes = None
    if not jsonb_columns:
        dtypes = dict((k, sa.types.JSON) for k in jsonb_columns)

    #  Add table name to the checkpoint file name
    with open(checkpoint_file_name, 'a+') as checkpoint_file:
        checkpoint_file.seek(0)
        try:
            checkpoint = int(checkpoint_file.read())
            if checkpoint < 1:
                checkpoint = 1
            print("Read the checkpoint as %d" % checkpoint)
        except ValueError as e:
            print("caught ValueError in checkpoint file. Reading from beginning.")
            print(e)
            checkpoint = 1

        #, add dtype={'value': str} for any specfic column type
        for data_frame in pd.read_csv(csv_file, chunksize=chunksize, delimiter=delimiter, skiprows=[i for i in range(1, checkpoint)]):
            success = False
            try:
                data_frame.to_sql(
                    table_name,  # table name
                    engine,
                    index=False,
                    dtype=dtypes,
                    if_exists="append",  # if the table already exists, append this data
                    method=None
                )
                success = True
            except exc.IntegrityError as ie:
                print("Caught IntegrityError. Going to upsert. ")
                callback = None
                if on_conflict == 'ignore':
                    callback = upsert_ignore
                    print("Will be ignoring the conflicts. The rows value will be same.")
                elif on_conflict == 'overwrite':
                    callback = upsert_overwrite
                    print("Will be overwriting the values on conflict.")
                else:
                    raise ie

                data_frame.to_sql(
                    table_name,  # table name
                    engine,
                    index=False,
                    dtype=dtypes,
                    if_exists="append",  # if the table already exists, append this data
                    method=callback
                )
                success = True
            finally:
                if success:
                    checkpoint = checkpoint + len(data_frame)
                    print("Successfully inserted upto %d line from the file. " % checkpoint)
                    checkpoint_file.seek(0)
                    checkpoint_file.truncate()
                    checkpoint_file.write(str(checkpoint))
                    checkpoint_file.flush()

def main():
    # drop_table_ddl = """DROP TABLE IF EXISTS books"""
    # create_table_ddl = """CREATE TABLE IF NOT EXISTS  books(k int primary key, doc jsonb not null)"""
    # create_table_ddl = """CREATE TABLE books(col1 int primary key, col2 varchar(10), col3 int)"""
    # create_index_ddl = """CREATE UNIQUE INDEX u1 on books(col2)"""
    # drop_table_ddl = """DROP TABLE IF EXISTS airports"""
    #
    # create_table_ddl = """CREATE TABLE airports(
    #     ident TEXT,
    #     type TEXT,
    #     name TEXT,
    #     elevation_ft INT,
    #     continent TEXT,
    #     iso_country CHAR(2),
    #     iso_region CHAR(7),
    #     municipality TEXT,
    #     gps_code TEXT,
    #     iata_code TEXT,
    #     local_code TEXT,
    #     coordinates TEXT,
    #     PRIMARY KEY (ident))"""

    # idx_ddl = "CREATE INDEX airport_type_region_idx ON airports((type, iso_region) HASH, ident ASC)"

    global constraint_name

    with open("loader.yaml", 'r') as stream:
        try:
            print("The configuration file is : ")
            cfg = yaml.safe_load(stream)
            print(cfg)
        except yaml.YAMLError as exc:
            print(exc)

    host = cfg["host"]
    dbname = cfg["dbname"]
    user = cfg["user"]
    password = cfg["password"]
    port = int(cfg["port"])
    table_name = cfg["table_name"]
    chunksize = int(cfg["chunksize"])
    csv_file = cfg["csv_file"]
    checkpoint_file_name = cfg["checkpoint_file_name"]
    on_conflict = cfg["on_conflict"]
    ssl_enabled = cfg["ssl"]
    delimiter = cfg["delimiter"]
    connect_args = {}

    if (ssl_enabled):
        ssl_mode = cfg["ssl_mode"]
        # if (ssl_mode == 'verify-ca' or ssl_mode == 'verify-full'):
        sslcert = cfg["sslcert"]
        sslkey = cfg["sslkey"]
        # ssl_key = cfg["ssl_key"]
        # ssl_cert = cfg["ssl_cert"]
        connect_args = {"sslmode": ssl_mode, "sslcert": sslcert, "sslkey": sslkey}


    connection_url = "postgresql://{}:{}@{}:{}/{}".format(user, password, host, port, dbname)
    # execute_ddl(connection_url, connect_args, drop_table_ddl)
    # execute_ddl(connection_url, connect_args, create_table_ddl)

    # find the constraints
    constraint_name = find_constraint_name(connection_url, connect_args, table_name)

    print("The constraints are {}".format(constraint_name))


    col_data_types = get_all_column_names_information_schema(connection_url, connect_args, table_name)
    jsonb_columns = []
    for row in col_data_types:
        if row['data_type'] == 'JSONB' or row['data_type'] == 'jsonb':
            jsonb_columns.append(row['column_name'])
    print("JSONB columns are {}", jsonb_columns)
    print("The connection string is ", connection_url)
    start = time.time()
    load_data(connection_url, connect_args, table_name, csv_file, checkpoint_file_name, chunksize, on_conflict, delimiter, jsonb_columns)
    now = time.time()
    print("Time: %s secs" % (now - start))


if __name__ == "__main__":
    main()