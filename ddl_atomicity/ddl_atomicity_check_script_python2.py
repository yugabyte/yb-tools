#!/usr/bin/env python2
# -*- coding: utf-8 -*-

from collections import defaultdict
import HTMLParser
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
parser.add_argument('--curl_path', type=str, default='curl', help='Path to curl executable.')
parser.add_argument('--grep_path', type=str, default='grep', help='Path to grep executable.')
parser.add_argument('--awk_path', type=str, default='awk', help='Path to awk executable.')

args = parser.parse_args()

# Initialize error count and message list
errorcount = 0
error_messages = []

FNULL = open(os.devnull, 'w')
html_parser = HTMLParser.HTMLParser()

# Check if the current node is the master leader if the flag is set
if args.master_leader_only:
    print("Checking if the node is the master leader.")
    is_leader = subprocess.check_output(
        "{} -skL http://localhost:9300/metrics | {} yb_node_is_master_leader{{ | {} '{{print $2}}'".format(
            args.curl_path,
            args.grep_path,
            args.awk_path
        ),
        shell=True
    ).decode('utf-8').strip()
    if is_leader == "0":
        print("Not master leader, exiting.")
        sys.exit(0)

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
        [args.curl_path, "-skL", "http://{}:{}/api/v1/tables".format(
            master_interface_address,
            args.master_interface_port
        )]
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

    # Skip over tables that aren't in YSQL/are hidden.
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
        # Check if the table exists in pg_class
        if yb_pg_table_oid not in pg_class_oid_tableinfo_dict:
            # Note: on versions older than 2024.1, the oid in this log will refer to the relfilenode
            # for materialized views.
            error_messages.append(
                "❌ - Table {} with oid {} and uuid {} does not exist in database {} - ORPHANED TABLE NEEDS TO BE DROPPED".format(
                    tablename, pg_oid, tableid, dbname
                )
            )
            errorcount += 1
            continue
        
        pg_class_entry = pg_class_oid_tableinfo_dict[yb_pg_table_oid]
        # work-around for versions older than 2024.1, as master UI doesn't populate YSQL table oid
        # on the UI correctly (it is populated with relfilenode oid instead).
        pg_oid = pg_class_entry['oid']
        
        if tablename != pg_class_entry['relname']:
            error_messages.append(
                "❌ - Table {} with oid {} and uuid {} exists in {} but has a mismatched table name: table name in YSQL metadata is {} - TABLE NAME NEEDS TO BE FIXED".format(
                    tablename, pg_oid, tableid, dbname, pg_class_entry['relname']
                )
            )
            errorcount += 1
        else:
            print(
                "✅ - Table {} with oid {} and uuid {} exists in database {} - NO ISSUE".format(
                    tablename, pg_oid, tableid, dbname
                )
            )

        # Get columns
        try:
            table_schema_json = json.loads(
                subprocess.check_output([
                    args.curl_path, 
                    "-skL", 
                    "http://{}:{}/api/v1/table?id={}".format(
                        master_interface_address,
                        args.master_interface_port,
                        tableid
                    )
                ]).decode()
            )
        except:
            print("Failed to load schema. Exiting.")
            sys.exit(1)

        columns = [html_parser.unescape(column['column']) for column in table_schema_json["columns"]]
        # Check if each column exists in pg_attribute
        for column in columns:
            if column == "ybrowid" or column == "ybuniqueidxkeysuffix" or column == "ybidxbasectid":
                continue
            if column not in pg_attribute_attrelid_attnames_dict[pg_oid]:
                error_messages.append(
                    "❌ - Column {} does not exist in table {} in database {} - ORPHANED COLUMN NEEDS TO BE DROPPED".format(
                        column, tablename, dbname
                    )
                )
                errorcount += 1
                continue
            print(
                "✅ - Column {} exists in table {} in database {} - NO ISSUE".format(
                    column, tablename, dbname
                )
            )
        print("\n\n")

# Print collected error messages
if errorcount > 0:
    print("\n\n❌❌❌\n" + "\n".join(error_messages) + "\n❌❌❌")
    print("❌❌❌ - Issues found on {} objects, repair required.\n❌❌❌".format(errorcount))
    sys.exit(1)
