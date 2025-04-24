/*
Checks if a given table's indexes are consistent by comparing the row counts for each
of its indexes of a given table with the actual table row counts.
Optionally can provide a specific index by name to only check that one.

Example Usage:
  - SELECT * FROM check_indexes('sample_table'::regclass);
     - check all indexes for table 'sample_table'.
  - SELECT * FROM check_indexes('sample_table'::regclass, 'sample_table_idx1');
     - check only the index with name 'sample_table_idx1'

Limitations:
  - Only checks row counts does not compare the table/index row values.
  - Must be run REPEATABLE READ (or SERIALIZABLE) if there are concurrent index operations
    - to ensure index and table snapshots match.
  - For large tables it may be necessary to increase the 'timestamp_history_retention_interval_sec'
    - tserver flag to avoid the SNAPSHOT_TOO_OLD error.
  - Does not support GIN indexes yet.
*/
CREATE OR REPLACE FUNCTION check_indexes(table_oid OID, target_index_name TEXT DEFAULT NULL)
RETURNS TABLE(index_name TEXT, is_valid BOOLEAN, expected_row_count BIGINT, actual_row_count BIGINT) AS $$
DECLARE
    table_name TEXT;
    schema_name TEXT;
    full_table_name TEXT;
    index_name TEXT;
    index_is_primary BOOLEAN;
    index_where_cond TEXT;
    index_am_name TEXT;
    full_table_row_count BIGINT;
    table_row_count BIGINT;
    index_row_count BIGINT;
    target_index_found BOOLEAN;
    indexes RECORD;
    result RECORD;
BEGIN

    -- Find the table's fully qualified name.
    SELECT
        c.relname AS table_name,
        n.nspname AS schema_name
    INTO
        table_name,
        schema_name
    FROM
        pg_class c
    JOIN
        pg_namespace n ON c.relnamespace = n.oid
    WHERE
        c.oid = table_oid;

    IF table_name IS NULL THEN
        RAISE EXCEPTION 'No table found for the given table OID: %', table_oid;
    END IF;
    full_table_name := format('%I.%I', schema_name, table_name);

    IF target_index_name IS NOT NULL THEN
        target_index_found = FALSE;
    END IF;

    -- Get the full table row count
    full_table_name := format('%I.%I', schema_name, table_name);
    EXECUTE format('/*+ SeqScan(%s) */ SELECT COUNT(*) FROM %s', table_name, full_table_name) INTO full_table_row_count;

    -- Get the row count for each index and compare with the expected one.
    FOR indexes IN SELECT
            c.relname AS index_name,
            i.indisprimary AS is_primary,
            pg_get_expr(i.indpred, i.indrelid) AS where_cond,
            a.amname AS am_name
        FROM
            pg_index i
        JOIN
            pg_class c ON c.oid = i.indexrelid
        JOIN
            pg_am a ON a.oid = c.relam
        WHERE
            i.indrelid = table_oid
    LOOP
        index_name := indexes.index_name;
        index_where_cond := indexes.where_cond;
        index_is_primary := indexes.is_primary;
        index_am_name := indexes.am_name;
        -- if there is a target index provided only check that one.
        IF target_index_name IS NOT NULL THEN
            IF index_name = target_index_name THEN
               target_index_found = TRUE;
            ELSE
               CONTINUE;
            END IF;
        END IF;

        IF index_am_name != 'lsm' THEN
            RAISE NOTICE 'Skipping checking index: %, unsupported index access method: %', index_name, index_am_name;
            RETURN QUERY SELECT index_name, NULL::boolean, null::BIGINT, null::BIGINT;
            CONTINUE;
        END IF;

        IF index_is_primary THEN
            -- primary index is same as main table so it is guaranteed to be correct.
            table_row_count := full_table_row_count;
            index_row_count := full_table_row_count;
        ELSIF index_where_cond IS NOT NULL THEN
            -- partial index case
            EXECUTE format('/*+ SeqScan(%s) */ SELECT COUNT(*) FROM %s WHERE %s', table_name, full_table_name, index_where_cond) INTO table_row_count;
            EXECUTE format('/*+ IndexOnlyScan(%s %s) */ SELECT COUNT(*) FROM %s WHERE %s', table_name, index_name, full_table_name, index_where_cond) INTO index_row_count;
        ELSE
            -- regular (non-partial, secondary) index
            EXECUTE format('/*+ IndexOnlyScan(%s %s) */ SELECT COUNT(*) FROM %s', table_name, index_name, full_table_name) INTO index_row_count;
            table_row_count := full_table_row_count;
        END IF;

        RETURN QUERY SELECT index_name, table_row_count = index_row_count, table_row_count, index_row_count;
        IF table_row_count != index_row_count THEN
            RAISE NOTICE 'Invalid Index %', index_name;
        END IF;

    END LOOP;
    IF target_index_name IS NOT NULL AND NOT target_index_found THEN
        RAISE EXCEPTION 'No index found with name % for table with oid %', target_index_name, table_oid;
    END IF;

END;
$$ LANGUAGE plpgsql;