#!/usr/bin/env python3

###################################
# DEPRECATED
###################################
# This tool is deprecated. Please look at the ycrc tool which provides the same
# functionality with user authentication and TLS support instead


# pip install yb-cassandra-driver

from cassandra.cluster import Cluster
from cassandra.cluster import ResultSet
import time
import random
import argparse
import os
from functools import partial

from multiprocessing.dummy import Pool as ThreadPool

# Which cluster? Which keyspace? How many sub-tasks? Max parallelism?
#cluster = Cluster(['127.0.0.1'])
# cluster = Cluster(['172.151.30.71', '172.151.28.193'])
#keyspace_name="ybdemo_keyspace"
#num_tasks_per_table=4096
#num_parallel_tasks=8


def check_partition_row_count(keyspace_name, table_name, p_columns, l_bound):
  stmt = ("SELECT count(*) as rows FROM {}.{} " +
          "WHERE partition_hash({}) >= ? " +
          "AND partition_hash({}) <= ?").format(keyspace_name,
                                                table_name,
                                                p_columns,
                                                p_columns)

  range_size = int((64*1024)/num_tasks_per_table)
  u_bound = l_bound + range_size - 1

  prepared_stmt = session.prepare(stmt)
  results = session.execute(prepared_stmt, (int(l_bound), int(u_bound)))
  row_cnt = results[0].rows
  print("Row Count for {}.{} partition({}, {}) = {}".format(keyspace_name, table_name, l_bound, u_bound, row_cnt))
  return row_cnt

def check_table_row_counts(keyspace_name, table_name):
    print("Checking row counts for: " + keyspace_name + "." + table_name);
    results = session.execute(("SELECT column_name, position  " +
                               "FROM system_schema.columns " +
                               "WHERE keyspace_name = %s AND table_name = %s " +
                               """AND kind='partition_key' """),
                               (keyspace_name, table_name))

    # Add the partition columns to an array sorted by the
    # position of the column in the primary key.
    partition_columns = [''] * 256
    num_partition_columns = 0
    for row in results:
        partition_columns[row.position] = row.column_name
        num_partition_columns = num_partition_columns + 1
    del partition_columns[num_partition_columns:] # remove extra null elements from array

    p_columns = ",".join(partition_columns)
    print("Partition columns for " + keyspace_name + "." + table_name + ": (" + p_columns + ")");
    print("Performing {} checks for {}.{}".format(num_tasks_per_table, keyspace_name, table_name))

    range_size = int((64*1024)/num_tasks_per_table)
    l_bounds = []
    for idx in range(num_tasks_per_table):
        l_bound = int(idx * range_size)
        l_bounds.append(l_bound)


    pool = ThreadPool(num_parallel_tasks)
    t1 = time.time()
    row_counts = pool.map(partial(check_partition_row_count,
                                  keyspace_name,
                                  table_name,
                                  p_columns),
                          l_bounds)
    t2 = time.time()
    print("====================")
    print("Total Time: %s ms" % ((t2 - t1) * 1000))
    print("====================")

    total_row_cnt = 0
    for idx in range(len(row_counts)):
      total_row_cnt = total_row_cnt + row_counts[idx]

    print("Total Row Count for {}.{} = {}".format(keyspace_name, table_name, total_row_cnt))
    print("--------------------------------------------------------")

def check_keyspace_table_row_counts(keyspace_name):
    print("Checking table row counts for keyspace: " + keyspace_name);
    print("--------------------------------------------------------")
    tables = []
    results = session.execute("select table_name from system_schema.tables where keyspace_name = %s",
                              (keyspace_name, ));
    for row in results:
       check_table_row_counts(keyspace_name, row.table_name)

# Main

parser = argparse.ArgumentParser(description='get row counts of a table using parallel driver')

# Which cluster? Which keyspace? How many sub-tasks? Max parallelism?
#cluster = Cluster(['127.0.0.1'])
# cluster = Cluster(['172.151.30.71', '172.151.28.193'])
#keyspace_name="ybdemo_keyspace"
#num_tasks_per_table=4096
#num_parallel_tasks=8


#parser.add_argument('--cluster', help="ip or hostname of cluster", default=['127.0.0.1'], nargs="+", action="append")
parser.add_argument('--cluster', help="ip or hostname of cluster", default='127.0.0.1')
parser.add_argument('--keyspace', help="name of keyspace")
parser.add_argument('--tasks', help="number of tasks per table", default=4096)
parser.add_argument('--parallel', help="number of parallel tasks", default=8)

args=parser.parse_args()

cluster = Cluster([args.cluster])
keyspace_name = args.keyspace
num_tasks_per_table = args.tasks
num_parallel_tasks = args.parallel

session = cluster.connect()
check_keyspace_table_row_counts(keyspace_name)
