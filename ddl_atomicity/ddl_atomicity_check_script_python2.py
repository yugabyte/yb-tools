# -*- coding: UTF8 -*-

from collections import defaultdict
import HTMLParser
import json
import os
import sys
import subprocess

# Initialize error count
errorcount = 0

FNULL = open(os.devnull, 'w')
html_parser = HTMLParser.HTMLParser()

ysqlsh_path = "/home/yugabyte/tserver/bin/ysqlsh"

is_leader = subprocess.check_output(r"curl -s http://localhost:9300/metrics | grep yb_node_is_master_leader\{ | awk '{print $2}'", shell=True).decode('utf-8').strip()
if is_leader == "0":
    print("Not master leader, exiting.")
    exit(0)

# Get webserver interface from master.conf
with open('/home/yugabyte/master/conf/server.conf', 'r') as f:
    master_conf = f.read()
    webserver_interface = master_conf.split('webserver_interface=')[1].split('\n')[0]


# Get table data
tables_output = (json.loads(subprocess.check_output(["curl", "-s", "http://{}:7000/api/v1/tables".format(webserver_interface)]).decode('utf-8')))
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
    pg_class_output = json.loads(subprocess.check_output([ysqlsh_path, "-h", "/tmp/.yb.0.0.0.0:5433/", "-d", dbname, "-t", "-c", "SELECT json_agg(row_to_json(t)) FROM (SELECT relname, oid, relfilenode FROM pg_class WHERE oid >= 16384) t;"]).decode().strip())
    pg_class_oid_tableinfo_dict = {}
    # Use relfilenode if it exists (as the table may be rewritten)
    for table in pg_class_output:
        if table['relfilenode'] != '0':
            pg_class_oid_tableinfo_dict[table['relfilenode']] = table
        else:
            pg_class_oid_tableinfo_dict[table['oid']] = table

    pg_attribute_output = json.loads(subprocess.check_output([ysqlsh_path, "-h", "/tmp/.yb.0.0.0.0:5433/", "-d", dbname, "-t", "-c", "SELECT json_agg(row_to_json(t)) FROM (SELECT attname, attrelid FROM pg_attribute WHERE attrelid >= 16384) t;"]).decode().strip())
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
            print("❌ - Table {} with oid {} and uuid {} does not exist in database {} - ORPHANED TABLE NEEDS TO BE DROPPED".format(tablename, pg_oid, tableid, dbname))
            errorcount += 1
            continue
        
        pg_class_entry = pg_class_oid_tableinfo_dict[yb_pg_table_oid]
        # work-around for versions older than 2024.1, as master UI doesn't populate YSQL table oid
        # on the UI correctly (it is populated with relfilenode oid instead).
        pg_oid = pg_class_entry['oid']
        
        if tablename != pg_class_entry['relname']:
            print("❌ - Table {} with oid {} and uuid {} exists in {} but has a mismatched table name - TABLE NAME NEEDS TO BE FIXED".format(tablename, pg_oid, tableid, dbname))
            errorcount += 1
            continue

        print("✅ - Table {} with oid {} and uuid {} exists in database {} - NO ISSUE".format(tablename, pg_oid, tableid, dbname))

        # Get columns
        table_schema_json = json.loads(subprocess.check_output(["curl", "-s", "http://{}:7000/api/v1/table?id={}".format(webserver_interface, tableid)]).decode())
        columns = [html_parser.unescape(column['column']) for column in table_schema_json["columns"]]
        # Check if each column exists in pg_attribute
        for column in columns:
            if column == "ybrowid" or column == "ybuniqueidxkeysuffix" or column == "ybidxbasectid":
                continue
            if column not in pg_attribute_attrelid_attnames_dict[pg_oid]:
                print("❌ - Column {} does not exist in table {} in database {} - ORPHANED COLUMN NEEDS TO BE DROPPED".format(column, tablename, dbname))
                errorcount += 1
                continue
            print("✅ - Column {} exists in table {} in database {} - NO ISSUE".format(column, tablename, dbname))
            

if errorcount > 0:
    print("\n\n❌❌❌\n❌❌❌ - Issues found on {} objects, repair required.\n❌❌❌".format(errorcount))
    sys.exit(1)
