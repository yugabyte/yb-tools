# -*- coding: UTF8 -*-

from collections import defaultdict
import html
import json
import os
import sys
import subprocess
import argparse

# Define command-line arguments
parser = argparse.ArgumentParser(description='Script to validate YSQL tables and columns.')
parser.add_argument('--ysqlsh_path', type=str, default="/home/yugabyte/tserver/bin/ysqlsh", help='Path to ysqlsh executable.')
parser.add_argument('--master_conf_path', type=str, default='/home/yugabyte/master/conf/server.conf', help='Path to master configuration file.')
parser.add_argument('--master_interface_address', type=str, default=None, help='Master UI interface IP. If not provided, will be read from master_conf_path.')
parser.add_argument('--master_interface_port', type=int, default=7000, help='Port for the master UI interface.')
parser.add_argument('--ysql_host', type=str, default="/tmp/.yb.0.0.0.0:5433/", help='Host for ysqlsh.')
parser.add_argument('--master_leader_only', action='store_true', help='Check if the node is the master leader and exit if it is not.')
args = parser.parse_args()

# Initialize error count and message list
errorcount = 0
error_messages = []

FNULL = open(os.devnull, 'w')

# Check if the current node is the master leader
if args.master_leader_only:
    print("Checking if the node is the master leader.")
    is_leader = subprocess.check_output(
        r"curl -s http://localhost:9300/metrics | grep yb_node_is_master_leader\{ | awk '{print $2}'", 
        shell=True
    ).decode('utf-8').strip()
    if is_leader == "0":
        print("Not master leader, exiting.")
        exit(0)

# Get master UI interface
if args.master_interface_address is None:
    # Read from master.conf if master_interface_address is not provided
    with open(args.master_conf_path, 'r') as f:
        master_conf = f.read()
        master_interface_address = master_conf.split('webserver_interface=')[1].split('\n')[0]
else:
    master_interface_address = args.master_interface_address

# Get table data
tables_output = json.loads(
    subprocess.check_output(
        ["curl", "-s", f"http://{master_interface_address}:{args.master_interface_port}/api/v1/tables"]
    ).decode('utf-8')
)
table_data_json = tables_output["user"]
table_data_json += tables_output["index"]


# Initialize a dictionary to store table data by database
db_tables = {}

# Iterate through each line of table data
for table in table_data_json:
    pg_oid = table["ysql_oid"]
    dbname = table["keyspace"]

    # Skip over tables that aren't in YSQL/are hidden
    if pg_oid == "" or table["hidden"]:
        continue
    # Extract table oid
    yb_pg_table_oid = str(int(table["uuid"][-4:], 16))

    # Add table to the database's list in the dictionary
    if dbname not in db_tables:
        db_tables[dbname] = []
    db_tables[dbname].append((table["table_name"], pg_oid, yb_pg_table_oid, table["uuid"]))

# Iterate through each database
for dbname, tables in db_tables.items():
    # Fetch all user tables from pg_class for the database
    pg_class_output = json.loads(
        subprocess.check_output([
            args.ysqlsh_path, 
            "-h", args.ysql_host, 
            "-d", dbname, 
            "-t", 
            "-c", "SELECT json_agg(row_to_json(t)) FROM (SELECT relname, oid, relfilenode FROM pg_class WHERE oid >= 16384) t;"
        ]).decode().strip()
    )
    pg_class_oid_tableinfo_dict = {}
    # Use relfilenode if it exists (as the table may be rewritten)
    for table in pg_class_output:
        if table['relfilenode'] != '0':
            pg_class_oid_tableinfo_dict[table['relfilenode']] = table
        else:
            pg_class_oid_tableinfo_dict[table['oid']] = table

    pg_attribute_output = json.loads(
        subprocess.check_output([
            args.ysqlsh_path, 
            "-h", args.ysql_host, 
            "-d", dbname, 
            "-t", 
            "-c", "SELECT json_agg(row_to_json(t)) FROM (SELECT attname, attrelid FROM pg_attribute WHERE attrelid >= 16384) t;"
        ]).decode().strip()
    )
    pg_attribute_attrelid_attnames_dict = defaultdict(list)
    for attribute in pg_attribute_output:
        pg_attribute_attrelid_attnames_dict[attribute['attrelid']].append(attribute['attname'])

    # Iterate through each table
    for tablename, pg_oid, yb_pg_table_oid, tableid in tables:
        print("\n\n")
        # Check if the table exists in pg_class
        if yb_pg_table_oid not in pg_class_oid_tableinfo_dict:
            # Note: on versions older than 2024.1, the oid in this log will refer to the relfilenode
            # for materialized views.
            error_messages.append(
                f"❌ - Table {tablename} with oid {pg_oid} and uuid {tableid} does not exist in database {dbname} - ORPHANED TABLE NEEDS TO BE DROPPED"
            )
            errorcount += 1
            continue
        
        pg_class_entry = pg_class_oid_tableinfo_dict[yb_pg_table_oid]
        # work-around for versions older than 2024.1, as master UI doesn't populate YSQL table oid
        # on the UI correctly (it is populated with relfilenode oid instead).
        pg_oid = pg_class_entry['oid']
        
        if tablename != pg_class_entry['relname']:
            error_messages.append(
                f"❌ - Table {tablename} with oid {pg_oid} and uuid {tableid} exists in {dbname} but has a mismatched table name: table name in YSQL metadata is {pg_class_entry['relname']} - TABLE NAME NEEDS TO BE FIXED"
            )
            errorcount += 1
        else:
            print(
                f"✅ - Table {tablename} with oid {pg_oid} and uuid {tableid} exists in database {dbname} - NO ISSUE"
            )

        # Get columns
        table_schema_json = json.loads(
            subprocess.check_output([
                "curl", 
                "-s", 
                f"http://{master_interface_address}:{args.master_interface_port}/api/v1/table?id={tableid}"
            ]).decode()
        )
        columns = [html.unescape(column['column']) for column in table_schema_json["columns"]]
        # Check if each column exists in pg_attribute
        for column in columns:
            if column in {"ybrowid", "ybuniqueidxkeysuffix", "ybidxbasectid"}:
                continue
            if column not in pg_attribute_attrelid_attnames_dict[pg_oid]:
                error_messages.append(
                    f"❌ - Column {column} does not exist in table {tablename} in database {dbname} - ORPHANED COLUMN NEEDS TO BE DROPPED"
                )
                errorcount += 1
                continue
            print(
                f"✅ - Column {column} exists in table {tablename} in database {dbname} - NO ISSUE"
            )

# Print collected error messages
if errorcount > 0:
    print("\n\n❌❌❌\n" + "\n".join(error_messages) + "\n❌❌❌")
    print(f"❌❌❌ - Issues found on {errorcount} objects, repair required.\n❌❌❌")
    sys.exit(1)
