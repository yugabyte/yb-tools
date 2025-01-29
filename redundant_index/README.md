# Redundant Index Detector

This script identifies redundant (duplicate) indexes in PostgreSQL-compatible databases.

## Overview

When you have multiple indexes on the same table, some may be redundant and safely dropped. For example:

- If you have an index on (a,b,c,d)
- And another index on (a,b,c) INCLUDE (d)
- The second index can make the first redundant

## Usage

Run this script on any PostgreSQL-compatible database using psql/ysqlsh (or any other compatible client). The output includes a table of potential redundant indexes along with the existing indexes that make them redundant.

## Example Usage

It’s recommended to turn on extended display mode to see the full query output:

```sql
yugabyte=# \x
Expanded display is on.
yugabyte=# \i redundant_index_detector.sql
-[ RECORD 1 ]----------------+------------------------------------------------------------------------------------------
Redundant index name         | idx_expr_def2
Existing index name          | idx_expr_def1
Redundant definition         | CREATE INDEX idx_expr_def2 ON test_schema.expr_collations_23 USING lsm (lower(col1) HASH)
Existing definition          | CREATE INDEX idx_expr_def1 ON test_schema.expr_collations_23 USING lsm (lower(col1) HASH)
```

In the example above, "idx_expr_def2" could be considered redundant if "idx_expr_def1" covers the same or superset of columns.

## Columns in the Output

The final query in the script returns the following columns:

| Column                 | Description                                      |
| ---------------------- | ------------------------------------------------ |
| Redundant index name   | Name of the index that could be safely dropped   |
| Existing index name    | Name of the index that makes it redundant        |
| Redundant definition   | Full CREATE INDEX definition of redundant index  |
| Existing definition    | Full CREATE INDEX definition of existing index   |
