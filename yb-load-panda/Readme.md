This is a python script based on Panda library to load CSV data to a table in YugabyteDB. This is to alleviate the 
problem of `COPY` command failing and in turn requiring a need to truncate
the table and run `COPY` again. It handles duplicates by either ignoring the new value
or overwriting the oldvalue with newone.
In case of failures, there is no need to truncate the table saving lots of time and effort.
It also maintains a checkpoint file in case you want to load large tables so that you don't upsert the same data.

The table must be created beforehand in YugabyteDB.

The two options in case of duplicates, `overwrite` or `ignore`. These can be controlled
by the input file option.

Currently, it also supports JSONB data types as well as SSLs if configured.

The input is a `yaml` file in following format:

```
dbname: yugabyte
user: yugabyte
password: yugabyte
host: 127.0.0.1
port: 5433
csv_file: altid.csv #airport-codes.csv #
table_name: alternate_ids #airports #
checkpoint_file_name: checkpoint.txt
on_conflict: overwrite  # fail/overwrite/ignore
chunksize: 16
delimiter: '|'
ssl: False
ssl_mode: 'prefer' #'verify-ca'
# root_cert: '/Users/suranjan/Downloads/root.txt'
sslcert: '/Users/suranjan/Downloads/cert.txtt'
sslkey: '/Users/suranjan/Downloads/key.txtt'
```


It will be useful when someone has to quickly try loading data from CSV. The script 
too can be modified quickly to do data-mapping etc.