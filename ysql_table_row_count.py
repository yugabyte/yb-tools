# pip install psycopg2

import threading
import time
import random
import os
from functools import partial
import argparse
import psycopg2

from multiprocessing.dummy import Pool as ThreadPool

threads = []

def check_partition_row_count(schema_name, table_name, p_columns, cursors, l_bound):

  tid = threading.current_thread().ident
  if tid not in threads:
  	threads.append(tid)
  	
  index = threads.index(tid)
  stmt = ("SELECT count(*) as rows FROM {} " +
          "WHERE yb_hash_code({}) >= %s " +
          "AND yb_hash_code({}) <= %s").format(
                                                table_name,
                                                p_columns,
                                                p_columns)

  range_size = int((64*1024)/num_tasks_per_table)
  u_bound = l_bound + range_size - 1
  cursor = cursors[index]
  results = cursor.execute(stmt, (int(l_bound), int(u_bound)))
  assert(cursor.rowcount == 1)
  row_cnt = cursor.fetchall()[0][0]
  return row_cnt

def check_table_row_counts(schema_name, table_name):
    print("Checking row counts for: " + schema_name + "." + table_name);
    results = session.execute("SELECT a.attname " +
"FROM   pg_index i " + 
"JOIN   pg_attribute a ON a.attrelid = i.indrelid " + 
                     "AND a.attnum = ANY(i.indkey) " + 
"WHERE  i.indrelid = %s::regclass " +
"AND    i.indisprimary AND (i.indoption[array_position(i.indkey, a.attnum)] & 4 <> 0);",
                               (table_name,))

    # Add the partition columns to an array sorted by the
    # position of the column in the primary key.
    partition_columns = [''] * 256
    num_partition_columns = 0
    results = session.fetchall()
    if len(results) == 0:
    	return
    for row in results:
        partition_columns[num_partition_columns] = row[0]
        num_partition_columns = num_partition_columns + 1
    del partition_columns[num_partition_columns:] # remove extra null elements from array

    p_columns = ",".join(partition_columns)
    print("Partition columns for " + schema_name + "." + table_name + ": (" + p_columns + ")");
    print("Performing {} checks for {}.{}".format(num_tasks_per_table, schema_name, table_name))

    range_size = int((64*1024)/num_tasks_per_table)
    l_bounds = []
    for idx in range(num_tasks_per_table):
        l_bound = int(idx * range_size)
        l_bounds.append(l_bound)


    pool = ThreadPool(num_parallel_tasks)
    cursors = []
    indices = []
    for i in range(num_parallel_tasks):
      t_cluster = psycopg2.connect("host=127.0.0.1 port=5433 dbname=yugabyte user=yugabyte password=yugabyte")
      indices.append(i)
      cursors.append(t_cluster.cursor())

    t1 = time.time()
    row_counts = pool.map(partial(check_partition_row_count,
                                  schema_name,
                                  table_name,
                                  p_columns, cursors),
                          l_bounds)
    t2 = time.time()

    print("====================")
    print("Total Time: %s ms" % ((t2 - t1) * 1000))
    print("====================")

    # cleanup
    for i in range(num_parallel_tasks):
    	cursors[i].close()
    total_row_cnt = 0
    del threads[:]

    for idx in range(len(row_counts)):
      total_row_cnt = total_row_cnt + row_counts[idx]

    print("Total Row Count for {}.{} = {}".format(schema_name, table_name, total_row_cnt))
    print("--------------------------------------------------------")

def check_schema_table_row_counts(schema_name):
    print("Checking table row counts for schema: " + schema_name + " in database: " + dbname);
    print("--------------------------------------------------------")
    tables = []
    results = session.execute("select tablename from pg_tables where schemaname = %s", (schema_name,
    ));
    for row in session.fetchall():
       check_table_row_counts(schema_name, row[0])

parser = argparse.ArgumentParser(description='get row counts of a table using parallel driver')

# Main
parser.add_argument('--cluster', help="ip or hostname of cluster", default='127.0.0.1')
parser.add_argument('--portname', help="portname of cluster", default='5433')
parser.add_argument('--username', help="username of cluster", default='yugabyte')
parser.add_argument('--password', help="password of cluster", default='yugabyte')
parser.add_argument('--dbname', help="name of database to count", default='yugabyte')
parser.add_argument('--schemaname', help="schema to count on", default='public')
parser.add_argument('--tasks', help="number of tasks per table", default=4096)
parser.add_argument('--parallel', help="number of parallel tasks", default=8)

args = parser.parse_args()
cluster = psycopg2.connect("host={} port={} dbname={} user={} password={}".format(args.cluster, args.portname, args.dbname, args.username, args.password))
dbname = args.dbname
num_tasks_per_table = args.tasks
num_parallel_tasks = args.parallel
schema_name = args.schemaname
session = cluster.cursor()

check_schema_table_row_counts(schema_name)
