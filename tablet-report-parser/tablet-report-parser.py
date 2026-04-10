#!/usr/bin/env python3
##########################################################################
## Tablet Report Parser (Python version)
##    Parses the support bundle directory and creates a SQLite DB.
##    Handles: universe-details.json, dump-entities.json, tablet_report.json
## See KB: https://yugabyte.zendesk.com/knowledge/articles/12124512476045/en-us
##########################################################################
import sys
import os
import json
import sqlite3
import base64
import struct
import re
import socket
from datetime import datetime
from urllib.parse import quote

VERSION = "0.52"

USAGE = """\
Tablet Report Parser (Python) {version}
=========================
See KB: https://yugabyte.zendesk.com/knowledge/articles/12124512476045/en-us

The input to this program is a support bundle directory.
The output is a SQLite database containing various reports.

USAGE:
   python3 {prog} <support-bundle-directory/YBA/TabletReport>

This will create a database file named <directory>.sqlite

* The support bundle directory should contain:
    - universe-details.json
    - dump-entities.json
    - tablet_report.json (one per tserver node, in node subdirectories)
"""


# ===================================================================
#  Utility helpers
# ===================================================================

def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)


def to_int(val):
    if val is None:
        return 0
    try:
        return int(val)
    except (ValueError, TypeError):
        try:
            return int(float(val))
        except (ValueError, TypeError):
            return 0


def decode_partition_key(b64_str, default_hex="0000"):
    if not b64_str:
        return f"0x{default_hex}"
    try:
        raw = base64.b64decode(b64_str)
        if not raw:
            return f"0x{default_hex}"
        if len(raw) >= 2:
            val = struct.unpack(">H", raw[:2])[0]
        else:
            val = raw[0] << 8
        return f"0x{val:04x}"
    except Exception:
        return f"0x{default_hex}"


def decode_uuid_b64(b64_str):
    if not b64_str:
        return ""
    try:
        return base64.b64decode(b64_str).hex()
    except Exception:
        return b64_str


def split_json_objects(filepath):
    """Parse a file with concatenated JSON objects (yugatool-style)."""
    objects = []
    current = []
    with open(filepath) as f:
        for line in f:
            current.append(line)
            if re.match(r"^\s?}[\s,]*$", line):
                text = "".join(current).rstrip().rstrip(",")
                try:
                    objects.append(json.loads(text))
                    current = []
                except json.JSONDecodeError:
                    pass
    if current:
        text = "".join(current).strip()
        if text:
            try:
                obj = json.loads(text)
                if isinstance(obj, list):
                    objects.extend(obj)
                else:
                    objects.append(obj)
            except json.JSONDecodeError:
                pass
    return objects


# ===================================================================
#  MetricUnit — human-readable byte formatting
# ===================================================================

class MetricUnit:
    _KILO = 1024
    _ORDER = [" ", "K", "M", "G", "T", "P", "X", "Z", "Y"]
    _UNITS = {}

    @classmethod
    def init(cls):
        cls._UNITS = {cls._ORDER[i]: cls._KILO ** i for i in range(len(cls._ORDER))}

    @classmethod
    def format_kilo(cls, number):
        number = number or 0
        for p in reversed(cls._ORDER):
            if abs(number) >= cls._UNITS[p]:
                val = number / cls._UNITS[p]
                suffix = "" if p == " " else p
                i_part = int(val)
                d_part = int((val - i_part) * 10)
                if d_part > 0:
                    return f"{i_part}.{d_part}{suffix}"
                return f"{i_part}{suffix}"
        return str(int(number))


MetricUnit.init()


# ===================================================================
#  TableInfo — collects per-table statistics & region/zone distribution
# ===================================================================

class TableInfo:
    collection = {}
    region_zone = {}
    tserver_to_rz = {}
    tablet_by_rz = {}
    tablet_bytes = {}
    bucket_max = 0

    @classmethod
    def reset(cls):
        cls.collection.clear()
        cls.region_zone.clear()
        cls.tserver_to_rz.clear()
        cls.tablet_by_rz.clear()
        cls.tablet_bytes.clear()
        cls.bucket_max = 0

    def __init__(self, t):
        self.table_uuid = t.get("table_uuid", "")
        self.namespace = t.get("namespace", "")
        self.tablename = t.get("tablename", "")
        self.tot_tablet_count = 0
        self.uniq_tablets_estimate = 0
        self.leader_tablets = 0
        self.node_tablet_count = {}
        self.keys_per_tablet = 0
        self.key_range_overlap = 0
        self.unmatched_key_size = 0
        self.comment = ""
        self.sst_tot_bytes = 0
        self.wal_tot_bytes = 0
        self.keyrangelist = []

    @classmethod
    def find_or_new(cls, t):
        name = t.get("table_uuid") or f"{t.get('namespace', '')}:{t.get('tablename', '')}"
        if name not in cls.collection:
            cls.collection[name] = cls(t)
        return cls.collection[name]

    @classmethod
    def register_region_zone(cls, region, zone, uuid):
        if region not in cls.region_zone:
            cls.region_zone[region] = {}
        if zone not in cls.region_zone[region]:
            cls.region_zone[region][zone] = {"TSERVER": []}
        cls.region_zone[region][zone]["TSERVER"].append(uuid)
        cls.tserver_to_rz[uuid] = (region, zone)

    def collect(self, tablet, node_uuid):
        self.tot_tablet_count += 1
        self.node_tablet_count[node_uuid] = self.node_tablet_count.get(node_uuid, 0) + 1
        self.sst_tot_bytes += to_int(tablet.get("sst_size"))
        self.wal_tot_bytes += to_int(tablet.get("wal_size"))

        sk_str = tablet.get("start_key") or "0x0000"
        ek_str = tablet.get("end_key") or "0xffff"
        if sk_str and len(sk_str) > 6:
            sk_str = "0x" + sk_str[-4:]
        if ek_str and len(ek_str) > 6:
            ek_str = "0x" + ek_str[-4:]

        start_key = int(sk_str, 16)
        end_key = int(ek_str, 16)

        if end_key < start_key:
            end_key = start_key

        if self.uniq_tablets_estimate == 0:
            self.keys_per_tablet = end_key - start_key
            if self.keys_per_tablet > 0:
                self.uniq_tablets_estimate = 0xFFFF // self.keys_per_tablet
            else:
                self.uniq_tablets_estimate = -1

        if (tablet.get("lease_status") or "") == "HAS_LEASE":
            self.leader_tablets += 1

        if end_key == 0xFFFF:
            pass
        elif (end_key - start_key) == self.keys_per_tablet:
            pass
        else:
            self.unmatched_key_size += 1

        if self.keys_per_tablet == 0:
            self.key_range_overlap += 1
        else:
            if start_key % self.keys_per_tablet != 0:
                self.key_range_overlap += 1
            idx = start_key // self.keys_per_tablet
            while len(self.keyrangelist) <= idx:
                self.keyrangelist.append(0)
            self.keyrangelist[idx] += 1

        rz = self.__class__.tserver_to_rz.get(node_uuid, (None, None))
        region, zone = rz

        if tablet.get("status") == "TABLET_DATA_TOMBSTONED":
            return

        t_uuid = tablet.get("tablet_uuid", "")
        tbrz = self.__class__.tablet_by_rz
        tbrz.setdefault(t_uuid, {}).setdefault(region, {})
        replicas = tbrz[t_uuid][region].get(zone, 0) + 1
        tbrz[t_uuid][region][zone] = replicas
        self.__class__.tablet_bytes[t_uuid] = to_int(tablet.get("sst_size")) + to_int(tablet.get("wal_size"))
        if replicas > self.__class__.bucket_max:
            self.__class__.bucket_max = replicas

    @classmethod
    def generate_report(cls, conn):
        c = conn.cursor()
        c.execute("""CREATE TABLE tableinfo(
            NAMESPACE TEXT, TABLENAME TEXT, TABLE_UUID TEXT,
            TOT_TABLET_COUNT INTEGER, UNIQ_TABLET_COUNT INTEGER,
            UNIQ_TABLETS_ESTIMATE INTEGER,
            LEADER_TABLETS INTEGER,
            NODE_TABLET_MIN INTEGER, NODE_TABLET_MAX INTEGER,
            KEYS_PER_TABLET INTEGER,
            KEY_RANGE_OVERLAP INTEGER,
            UNMATCHED_KEY_SIZE INTEGER,
            COMMENT TEXT,
            SST_TOT_BYTES INTEGER, WAL_TOT_BYTES INTEGER,
            SST_TOT_HUMAN TEXT, WAL_TOT_HUMAN TEXT,
            SST_RF1_HUMAN TEXT,
            TOT_HUMAN TEXT
        )""")

        c.execute("CREATE TEMP TABLE temp_table_detail AS SELECT * FROM table_detail")

        for tkey in sorted(cls.collection):
            t = cls.collection[tkey]

            unbalanced = ""
            ref_count = None
            for cnt in t.node_tablet_count.values():
                if ref_count is None:
                    ref_count = cnt
                if abs(ref_count - cnt) > 1:
                    unbalanced = "(Unbalanced)"
                    break
            t.comment += unbalanced

            not_found = 0
            for i in range(t.uniq_tablets_estimate):
                if i < len(t.keyrangelist) and t.keyrangelist[i]:
                    continue
                not_found += 1
                if not_found == 1:
                    t.comment += f"[{hex(i * t.keys_per_tablet)} @{i} not found]"
            if not_found > 1:
                t.comment += f"[{not_found} key ranges not found]"

            row = c.execute(
                "SELECT unique_tablet_count FROM temp_table_detail "
                "WHERE table_name=? AND namespace=?",
                (t.tablename, t.namespace),
            ).fetchone()
            uniq_count = row[0] if row else 0

            if t.uniq_tablets_estimate < 0:
                t.uniq_tablets_estimate = uniq_count

            sst_h = MetricUnit.format_kilo(t.sst_tot_bytes)
            wal_h = MetricUnit.format_kilo(t.wal_tot_bytes)
            rf1 = (
                t.sst_tot_bytes * t.uniq_tablets_estimate / t.tot_tablet_count
                if t.tot_tablet_count
                else 0
            )
            sst_rf1_h = MetricUnit.format_kilo(rf1)
            tot_h = MetricUnit.format_kilo(t.wal_tot_bytes + t.sst_tot_bytes)
            ntc = t.node_tablet_count
            n_min = min(ntc.values()) if ntc else 0
            n_max = max(ntc.values()) if ntc else 0

            c.execute(
                "INSERT INTO tableinfo VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                (
                    t.namespace, t.tablename, t.table_uuid,
                    t.tot_tablet_count, uniq_count, t.uniq_tablets_estimate,
                    t.leader_tablets, n_min, n_max,
                    t.keys_per_tablet, t.key_range_overlap, t.unmatched_key_size,
                    t.comment,
                    t.sst_tot_bytes, t.wal_tot_bytes,
                    sst_h, wal_h, sst_rf1_h, tot_h,
                ),
            )

        c.execute("DROP TABLE temp_table_detail")
        c.execute(
            "UPDATE tableinfo SET COMMENT = COMMENT || '[Excess tablets]' "
            "WHERE UNIQ_TABLET_COUNT > UNIQ_TABLETS_ESTIMATE"
        )

        c.execute("""CREATE VIEW large_tables AS
            SELECT namespace, tablename, uniq_tablet_count as uniq_tablets,
                (sst_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as sst_RF1_mb,
                (sst_tot_bytes /tot_tablet_count/1024/1024) as tablet_size_mb,
                round((sst_tot_bytes*uniq_tablet_count/tot_tablet_count /1024.0/1024.0 + 5000) / 10000,1) as recommended_tablets,
                tot_tablet_count / uniq_tablet_count as repl_factor,
                (wal_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as wal_RF1_mb
            FROM tableinfo
            WHERE sst_RF1_mb > 5000
            ORDER by sst_RF1_mb desc""")

        c.execute("""CREATE VIEW table_sizes AS
            SELECT namespace, tablename, uniq_tablet_count as uniq_tablets,
                sst_tot_human as sst_bytes,
                sst_rf1_human as sst_RF1_bytes,
                wal_tot_human as wal_bytes,
                tot_human as total_bytes
            FROM tableinfo
            ORDER BY (sst_tot_bytes) DESC""")

        bmax = cls.bucket_max
        col_defs = "region TEXT, zone TEXT, tservers INTEGER, missing_replicas TEXT"
        for i in range(1, bmax + 1):
            col_defs += f", [{i}_replicas] TEXT"
        col_defs += ", balanced TEXT"
        c.execute(f"CREATE TABLE region_zone_tablets({col_defs})")

        for r in sorted(cls.region_zone):
            for z in sorted(cls.region_zone[r]):
                for t_uuid in cls.tablet_by_rz:
                    cls.tablet_by_rz[t_uuid].setdefault(r, {}).setdefault(z, 0)

                bucket = [0] * (bmax + 1)
                missing_bytes = 0
                replica_count = 0
                for t_uuid in cls.tablet_by_rz:
                    reps = cls.tablet_by_rz[t_uuid].get(r, {}).get(z, 0)
                    if reps == 0:
                        missing_bytes += cls.tablet_bytes.get(t_uuid, 0)
                    if reps <= bmax:
                        bucket[reps] += 1
                    replica_count += 1

                bt = [str(bucket[i]) for i in range(bmax + 1)]
                if missing_bytes:
                    bt[0] = f"{bucket[0]} ({missing_bytes / 1024 ** 3:.1f} GB)"
                if len(bt) > 1:
                    bt[1] = f"{bucket[1]}/{replica_count}"

                balanced = (
                    "YES"
                    if any(replica_count == bucket[i] for i in range(1, bmax + 1))
                    else "NO"
                )
                ts_count = len(cls.region_zone[r][z]["TSERVER"])
                vals = [r, z, ts_count] + bt + [balanced]
                c.execute(
                    f"INSERT INTO region_zone_tablets VALUES({','.join('?' * len(vals))})",
                    vals,
                )

        conn.commit()


# ===================================================================
#  UniverseInfo — parses universe-details.json
# ===================================================================

class UniverseInfo:
    def __init__(self, filepath):
        with open(filepath) as f:
            self.data = json.load(f)
        self.nodes = []
        for n in self.data.get("nodeDetailsSet", []):
            ci = n.get("cloudInfo", {})
            node = {
                "nodeName": n.get("nodeName", ""),
                "nodeUuid": (n.get("nodeUuid", "") or "").replace("-", ""),
                "isTserver": n.get("isTserver", False),
                "tserverRpcPort": str(n.get("tserverRpcPort", "")),
                "private_ip": ci.get("private_ip", ""),
                "az": ci.get("az", ""),
                "region": ci.get("region", ""),
            }
            self.nodes.append(node)

    def populate_cluster_table(self, conn):
        c = conn.cursor()
        for n in self.nodes:
            try:
                c.execute(
                    "INSERT INTO cluster(type,uuid,ip,port,region,zone,uptime) "
                    "VALUES(?,?,?,?,?,?,?)",
                    (
                        "TSERVER", n["nodeUuid"], n["private_ip"],
                        n["tserverRpcPort"], n["region"], n["az"], "0",
                    ),
                )
            except sqlite3.IntegrityError:
                pass
        conn.commit()


# ===================================================================
#  Schema — tables, indexes, views (mirrors Perl Create_Tables_and_views)
# ===================================================================

def create_schema(conn, script_name):
    c = conn.cursor()
    start_time = datetime.now().strftime("%Y-%m-%d %H:%M")
    hostname = socket.gethostname()

    c.execute("""CREATE TABLE cluster(
        type, uuid TEXT PRIMARY KEY, ip, port, region, zone, role, uptime)""")

    c.execute("""CREATE TABLE tablet(
        node_uuid, tablet_uuid TEXT, table_name, table_uuid, namespace,
        state, status, start_key, end_key, sst_size INTEGER, wal_size INTEGER,
        cterm, cidx, leader, lease_status)""")
    c.execute("CREATE UNIQUE INDEX tablet_idx ON tablet(node_uuid, tablet_uuid)")

    c.execute("""CREATE VIEW table_detail AS
        SELECT namespace, table_name,
               count(*) as total_tablet_count,
               count(DISTINCT tablet_uuid) as unique_tablet_count,
               count(DISTINCT node_uuid) as nodes
        FROM tablet GROUP BY namespace, table_name""")

    c.execute("""CREATE VIEW tablets_per_node AS
        SELECT node_uuid, ip as node_ip, zone,
               count(*) as tablet_count,
               sum(CASE WHEN status='TABLET_DATA_COPYING' THEN 1 ELSE 0 END) as copying,
               sum(CASE WHEN tablet.status = 'TABLET_DATA_TOMBSTONED' THEN 1 ELSE 0 END) as tombstoned,
               sum(CASE WHEN node_uuid = leader THEN 1 ELSE 0 END) as leaders,
               count(DISTINCT table_name) as table_count
            FROM tablet, cluster
            WHERE cluster.type='TSERVER' and cluster.uuid=node_uuid
            GROUP BY node_uuid, ip, zone
        UNION
        SELECT '~~TOTAL~~',
            '*(All '|| (select count(*) from cluster where type='TSERVER') || ' nodes)*', 'ALL',
           (Select count(*) from tablet),
           (Select count(*) from tablet WHERE status='TABLET_DATA_COPYING'),
           (SELECT count(*) from tablet where status = 'TABLET_DATA_TOMBSTONED'),
           (SELECT count(*) from tablet where node_uuid = leader),
           (SELECT count(DISTINCT table_name) as table_count from tablet)
           ORDER BY 1""")

    c.execute("""CREATE VIEW tablet_replica_detail AS
        SELECT t.namespace, t.table_name, t.table_uuid, t.tablet_uuid,
        sum(CASE WHEN t.status = 'TABLET_DATA_TOMBSTONED' THEN 0 ELSE 1 END) as replicas,
        sum(CASE WHEN t.status = 'TABLET_DATA_TOMBSTONED' THEN 1 ELSE 0 END) as tombstoned,
        sum(CASE when t.node_uuid = leader AND lease_status='HAS_LEASE'
            then 1 else 0 END ) as leader_count
        from tablet t
        GROUP BY t.namespace, t.table_name, t.table_uuid, t.tablet_uuid""")

    c.execute("""CREATE VIEW tablet_replica_summary AS
        SELECT replicas, count(*) as tablet_count
        FROM tablet_replica_detail GROUP BY replicas""")

    c.execute("""CREATE VIEW leaderless AS
        SELECT t.tablet_uuid, replicas, t.namespace, t.table_name,
               node_uuid, status, ip, leader_count
        from tablet t, cluster, tablet_replica_detail trd
        WHERE cluster.type='TSERVER' AND cluster.uuid=node_uuid
              AND t.tablet_uuid=trd.tablet_uuid
              AND t.status != 'TABLET_DATA_TOMBSTONED'
              AND trd.leader_count !=1""")

    c.execute("""CREATE VIEW delete_leaderless_be_careful AS
        SELECT '$HOME/tserver/bin/yb-ts-cli delete_tablet '|| tablet_uuid
               ||' -certs_dir_name $TLSDIR -server_address '||ip
               ||':9100  $REASON_tktnbr'
               AS generated_delete_command
        FROM leaderless""")

    c.execute("""CREATE VIEW UNSAFE_Leader_create AS
        SELECT  '$HOME/tserver/bin/yb-ts-cli --server_address='|| ip ||':'||port
            || ' unsafe_config_change ' || t.tablet_uuid
            || ' ' || node_uuid
            || ' -certs_dir_name $TLSDIR;sleep 10;#'
            || trd.replicas || ' replica(s)'
            AS cmd_to_run
        from tablet t, cluster, tablet_replica_detail trd
        WHERE  cluster.type='TSERVER' AND cluster.uuid=node_uuid
              AND  t.tablet_uuid=trd.tablet_uuid
              AND t.status != 'TABLET_DATA_TOMBSTONED'
              AND trd.leader_count !=1
              AND t.state='RUNNING'""")

    c.execute("""CREATE VIEW large_wal AS
        SELECT table_name, count(*) as tablets,
            sum(CASE WHEN wal_size >= 134217728 THEN 1 ELSE 0 END) as "GE128MB",
            sum(CASE WHEN wal_size >= 100663296 AND wal_size < 134217728 THEN 1 ELSE 0 END) as "GE96MB",
            sum(CASE WHEN wal_size >= 68157440  AND wal_size < 100663296 THEN 1 ELSE 0 END) as "GE65MB",
            sum(CASE WHEN wal_size < 68157440 THEN 1 ELSE 0 END) as "LT65MB"
        FROM tablet
        GROUP BY table_name
        ORDER BY GE128MB desc, GE96MB desc""")

    c.execute("""CREATE VIEW unbalanced_tables AS
        SELECT t.namespace , t.table_name, total_tablet_count,
              unique_tablet_count, nodes,
               (SELECT tablet_uuid
                  FROM tablet x
                  WHERE x.namespace =t.namespace   and  x.table_name =t.table_name
                     and (x.sst_size +x.wal_size ) = max_tablet_size
                 LIMIT 1) as large_tablet,
              round(min_tablet_size/1024.0/1024.0,1) as min_tablet_mb,
              round(max_tablet_size/1024.0/1024.0,1) as max_tablet_mb
        FROM
          (SELECT  namespace, table_name, count(*) as total_tablet_count,
             count(DISTINCT tablet_uuid) as unique_tablet_count,
             count(DISTINCT node_uuid) as nodes,
             round(max(sst_size + wal_size)/min(sst_size+wal_size+0.1),1) as heat_level,
             max(sst_size + wal_size) as max_tablet_size,
             min(sst_size+wal_size) as min_tablet_size
          FROM tablet t
          WHERE sst_size + wal_size > 0
          GROUP BY namespace, table_name
          HAVING heat_level > 2.5
          ORDER BY heat_level desc) t""")

    c.execute("""CREATE VIEW region_zone_distribution AS
        SELECT namespace, region, zone,
               count(*) as tablets,
               count(DISTINCT tablet_uuid) as uniq_tablet,
               sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
               count(DISTINCT table_name) as tables,
               count(DISTINCT node_uuid) as tservers,
               (SELECT count(*) from cluster c1
                WHERE type='MASTER' and c1.zone=c.zone and c1.region=c.region) as masters
        FROM tablet, cluster c
        WHERE c.uuid=node_uuid
        GROUP  BY namespace, region, zone
        UNION
        SELECT namespace, region, '~'||namespace||' Total~',
               count(*) as tablets,
               count(DISTINCT tablet_uuid) as uniq_tablet,
               sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
               count(DISTINCT table_name) as tables,
               count(DISTINCT node_uuid) as tservers,
               (SELECT count(*) from cluster c1
                WHERE type='MASTER' and c1.region=c.region) as masters
        FROM tablet, cluster c
        WHERE c.uuid=node_uuid
        GROUP  BY namespace, region
        UNION
        SELECT '~Total~', '~ALL~', '~ALL~',
               count(*) as tablets,
               count(DISTINCT tablet_uuid) as uniq_tablet,
               sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
               count(DISTINCT table_name) as tables,
               count(DISTINCT node_uuid) as tservers,
               (SELECT count(*) from cluster c1 WHERE type='MASTER') as masters
        FROM tablet, cluster c
        WHERE c.uuid=node_uuid
        ORDER BY namespace, region, zone""")

    c.execute("""CREATE VIEW unbalanced_tables_tablet_count_per_size AS
        SELECT
            table_name,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 < 2048 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 <2048 THEN 1 ELSE 0 END) END AS LT2GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) END AS s2GB_3GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) END AS s3GB_4GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) END AS s4GB_6GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) END AS s6GB_8GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) END AS s8GB_10GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) END AS s10GB_12GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) END AS s12GB_14GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) END AS s14GB_16GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) END AS s16GB_20GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) END AS s20GB_24GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) END AS s24GB_28GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) END AS s28GB_32GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) END AS s32GB_36GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) END AS s36GB_40GB,
            CASE WHEN SUM(CASE WHEN sst_size/1024/1024 > 40960 THEN 1 ELSE 0 END) = 0
                 THEN NULL
                 ELSE SUM(CASE WHEN sst_size/1024/1024 > 40960 THEN 1 ELSE 0 END) END AS GT40GB
        FROM tablet
        WHERE lease_status = 'HAS_LEASE'
            AND table_name IN (SELECT table_name FROM unbalanced_tables)
        GROUP BY table_name
        ORDER BY 2 DESC""")

    c.execute(
        "CREATE VIEW version_info AS "
        f"SELECT '{script_name}' as program, '{VERSION}' as version, "
        f"'{start_time}' AS run_on, '{hostname}' as host"
    )

    conn.commit()


# ===================================================================
#  Process dump-entities.json
# ===================================================================

def process_entities(filepath, conn, universe):
    eprint(f"  Processing {filepath} as ENTITIES file...")
    with open(filepath) as f:
        data = json.load(f)

    c = conn.cursor()

    c.execute("CREATE TABLE ENT_KEYSPACE (id TEXT PRIMARY KEY,name,type)")
    c.execute("CREATE TABLE ENT_TABLE (id TEXT PRIMARY KEY, keyspace_id,state, table_name)")
    c.execute(
        "CREATE TABLE ENT_TABLET "
        "(id TEXT ,table_id,state,is_leader,server_uuid,server_addr,type)"
    )

    c.execute("""CREATE VIEW IF NOT EXISTS extra_Tablets_detail as
        SELECT 'entities' as source, server_uuid as node, id as tablet FROM ENT_TABLET
        EXCEPT
        SELECT 'entities' as source,node_uuid,tablet_uuid FROM tablet
       UNION ALL
        SELECT 'tserv' as source, node_uuid,tablet_uuid FROM tablet
        EXCEPT
        SELECT 'tserv' as source, server_uuid as node,id as tablet from ENT_TABLET""")

    c.execute("""CREATE VIEW IF NOT EXISTS extra_Tablets_summary as
       SELECT source,node,ip,region,count(*) FROM extra_Tablets_detail,cluster
       WHERE node=uuid
       GROUP BY source,node,ip,region""")

    for ks in data.get("keyspaces", []):
        try:
            c.execute(
                "INSERT INTO ENT_KEYSPACE (id,name,type) VALUES(?,?,?)",
                (ks.get("keyspace_id", ""), ks.get("keyspace_name", ""),
                 ks.get("keyspace_type", "")),
            )
        except sqlite3.IntegrityError:
            pass

    for tbl in data.get("tables", []):
        try:
            c.execute(
                "INSERT INTO ENT_TABLE (id, keyspace_id,state, table_name) VALUES(?,?,?,?)",
                (tbl.get("table_id", ""), tbl.get("keyspace_id", ""),
                 tbl.get("state", ""), tbl.get("table_name", "")),
            )
        except sqlite3.IntegrityError:
            pass

    tserver_map = {}
    for t in data.get("tablets", []):
        leader = t.get("leader", "")
        for r in t.get("replicas", []):
            is_ldr = 1 if r.get("server_uuid", "") == leader else 0
            c.execute(
                "INSERT INTO ENT_TABLET "
                "(id,table_id,state,is_leader,server_uuid,server_addr,type) "
                "VALUES(?,?,?,?,?,?,?)",
                (t.get("tablet_id", ""), t.get("table_id", ""), t.get("state", ""),
                 is_ldr, r.get("server_uuid", ""), r.get("addr", ""),
                 r.get("type", "")),
            )
            tserver_map[r.get("addr", "")] = r.get("server_uuid", "")

    if universe:
        for addr, ts_uuid in tserver_map.items():
            parts = addr.split(":")
            if len(parts) != 2:
                continue
            ip, port = parts
            for node in universe.nodes:
                if ip == node["private_ip"] and port == str(node["tserverRpcPort"]):
                    node["nodeUuid"] = ts_uuid
                    break
        universe.populate_cluster_table(conn)

    conn.commit()


# ===================================================================
#  Process tablet_report.json
# ===================================================================

def process_tablet_report(filepath, conn, universe):
    eprint(f"  Processing {filepath} as TABLET REPORT file...")
    objects = split_json_objects(filepath)
    c = conn.cursor()

    for data in objects:
        msg = data.get("msg", "cluster")
        msg_clean = re.sub(r"[^\w\s].*", "", msg).strip()

        if msg_clean == "cluster":
            uuid = data.get("clusterUuid", "")
            zone = data.get("zone") or f"Version:{data.get('version', '')}"
            try:
                c.execute(
                    "INSERT INTO cluster(type,uuid,zone) VALUES(?,?,?)",
                    ("CLUSTER", uuid, zone),
                )
            except sqlite3.IntegrityError:
                pass

        elif msg_clean == "Masters":
            for m in data.get("content", []):
                uuid = decode_uuid_b64(
                    m.get("instanceId", {}).get("permanentUuid", "")
                )
                reg = m.get("registration", {})
                rpcs = reg.get("privateRpcAddresses", [])
                rpc = rpcs[0] if rpcs else {}
                ci = reg.get("cloudInfo", {})
                try:
                    c.execute(
                        "INSERT INTO cluster(type,uuid,ip,port,region,zone,role) "
                        "VALUES(?,?,?,?,?,?,?)",
                        ("MASTER", uuid, rpc.get("host", ""),
                         str(rpc.get("port", "")),
                         ci.get("placementRegion", ""),
                         ci.get("placementZone", ""),
                         m.get("role", "")),
                    )
                except sqlite3.IntegrityError:
                    pass

        elif msg_clean == "Tablet Servers":
            for t in data.get("content", []):
                uuid = decode_uuid_b64(
                    t.get("instance_id", {}).get("permanentUuid", "")
                )
                common = t.get("registration", {}).get("common", {})
                rpcs = common.get("privateRpcAddresses", [])
                rpc = rpcs[0] if rpcs else {}
                ci = common.get("cloudInfo", {})
                host = rpc.get("host", "")
                port = str(rpc.get("port", ""))
                region = ci.get("placementRegion", "")
                zone = ci.get("placementZone", "")
                uptime = str(t.get("metrics", {}).get("uptimeSeconds", "") or "")
                try:
                    c.execute(
                        "INSERT INTO cluster(type,uuid,ip,port,region,zone,uptime) "
                        "VALUES(?,?,?,?,?,?,?)",
                        ("TSERVER", uuid, host, port, region, zone, uptime),
                    )
                except sqlite3.IntegrityError:
                    pass
                TableInfo.register_region_zone(region, zone, uuid)

        elif msg_clean.startswith("Tablet Report") or msg_clean.startswith(
            "Tablet report"
        ):
            host_uuid = None
            m = re.search(
                r'\[host:"([^"]+)"\s+port:(\d+).*?\]\s*\((\w+)', msg
            )
            if m:
                host_uuid = m.group(3)
            elif universe:
                best_match = None
                for node in universe.nodes:
                    if node["nodeName"] and node["nodeName"] in filepath:
                        if best_match is None or len(node["nodeName"]) > len(best_match["nodeName"]):
                            best_match = node
                if best_match:
                    host_uuid = best_match["nodeUuid"]
                    TableInfo.register_region_zone(
                        best_match["region"], best_match["az"], host_uuid
                    )
            if not host_uuid:
                eprint(
                    f"  WARNING: Could not determine tserver for {filepath}, "
                    "skipping tablet section"
                )
                continue

            for t in data.get("content", []):
                ts = t.get("tablet", {}).get("tablet_status", {})
                cs = t.get("consensus_state", {})
                cstate = cs.get("cstate", {})
                partition = ts.get("partition", {})

                sk = decode_partition_key(
                    partition.get("partitionKeyStart", ""), "0000"
                )
                ek = decode_partition_key(
                    partition.get("partitionKeyEnd", ""), "ffff"
                )

                sst = to_int(ts.get("sstFilesDiskSize"))
                wal = to_int(ts.get("walFilesDiskSize"))

                vals = {
                    "tablet_uuid": ts.get("tabletId", ""),
                    "tablename": ts.get("tableName", ""),
                    "table_uuid": ts.get("tableId", ""),
                    "namespace": ts.get("namespaceName", ""),
                    "state": ts.get("state", ""),
                    "status": ts.get("tabletDataState", ""),
                    "start_key": sk,
                    "end_key": ek,
                    "sst_size": sst,
                    "wal_size": wal,
                    "cterm": str(cstate.get("currentTerm", "") or ""),
                    "cidx": str(
                        cstate.get("config", {}).get("opidIndex", "") or ""
                    ),
                    "leader": cstate.get("leaderUuid", "") or "",
                    "lease_status": cs.get("leaderLeaseStatus", "") or "",
                }

                try:
                    c.execute(
                        "INSERT INTO tablet(node_uuid, tablet_uuid, table_name, "
                        "table_uuid, namespace, state, status, start_key, end_key, "
                        "sst_size, wal_size, cterm, cidx, leader, lease_status) "
                        "VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                        (
                            host_uuid, vals["tablet_uuid"], vals["tablename"],
                            vals["table_uuid"], vals["namespace"], vals["state"],
                            vals["status"], vals["start_key"], vals["end_key"],
                            vals["sst_size"], vals["wal_size"], vals["cterm"],
                            vals["cidx"], vals["leader"], vals["lease_status"],
                        ),
                    )
                    TableInfo.find_or_new(vals).collect(vals, host_uuid)
                except sqlite3.IntegrityError:
                    pass

    conn.commit()


# ===================================================================
#  Summary view (created last, after all tables exist)
# ===================================================================

def create_summary_view(conn, has_entities):
    extra = ",extra_Tablets_summary/detail" if has_entities else ""
    c = conn.cursor()
    c.execute(f"""CREATE VIEW summary_report AS
        SELECT (SELECT count(*) from cluster where type='TSERVER') || ' TSERVERs, '
             || (SELECT count(DISTINCT table_name) FROM tablet) || ' Tables, '
             || (SELECT count(*) from tablet) || ' Tablets loaded.'
            AS Summary_Report
        UNION
         SELECT count(*) || ' leaderless tablets found.(See "leaderless")' FROM leaderless
        UNION
          SELECT sum(GE128MB) || ' wal files in ' ||sum(CASE WHEN GE128MB>0 then 1 else 0 END)
                || ' tables are > 128MB' FROM large_wal
        UNION
         SELECT (SELECT sum(tablet_count) FROM tablet_replica_summary WHERE replicas < 3)
                 || ' tablets have RF < 3. (See "tablet_replica_summary/detail")'
        UNION
           SELECT count(*) || ' tables have unbalanced tablet sizes (see "unbalanced_tables")'
           from unbalanced_tables
        UNION
           SELECT count(*) || ' Zones have unbalanced tablets (See "region_zone_tablets{extra}")'
            from  region_zone_tablets WHERE balanced='NO'
        """)
    conn.commit()


# ===================================================================
#  Bundle directory scanner
# ===================================================================

def scan_bundle(path):
    universe_file = None
    entities_file = None
    tablet_reports = []

    def _scan(d):
        nonlocal universe_file, entities_file
        try:
            entries = sorted(os.listdir(d))
        except OSError:
            return
        for name in entries:
            if name.startswith(".") or name in ("master", "tserver"):
                continue
            full = os.path.join(d, name)
            if os.path.islink(full):
                continue
            if os.path.isdir(full):
                _scan(full)
            elif os.path.isfile(full):
                if name.endswith("dump-entities.json"):
                    entities_file = full
                elif name.endswith("universe-details.json"):
                    universe_file = full
                elif name.endswith("tablet_report.json"):
                    tablet_reports.append(full)

    _scan(path)
    return universe_file, entities_file, tablet_reports


# ===================================================================
#  Main
# ===================================================================

def main():
    script_name = os.path.basename(sys.argv[0])
    start_time = datetime.now().strftime("%Y-%m-%d %H:%M")

    if len(sys.argv) < 2 or sys.argv[1] in ("-h", "--help", "-help", "help"):
        print(USAGE.format(version=VERSION, prog=script_name))
        sys.exit(1 if len(sys.argv) < 2 else 0)

    bundle_dir = sys.argv[1]
    if not os.path.isdir(bundle_dir):
        eprint(f"ERROR: '{bundle_dir}' is not a directory.")
        sys.exit(1)

    universe_file, entities_file, tablet_reports = scan_bundle(bundle_dir)

    if not tablet_reports and not entities_file:
        eprint(
            f"ERROR: No tablet_report.json or dump-entities.json found in "
            f"'{bundle_dir}'."
        )
        sys.exit(1)

    # Determine output DB filename
    db_name = bundle_dir.rstrip("/")
    for suffix in (".json", ".out", ".gz", ".tar"):
        if db_name.endswith(suffix):
            db_name = db_name[: -len(suffix)]
    db_name += ".sqlite"

    if os.path.exists(db_name):
        mtime = os.path.getmtime(db_name)
        dt = datetime.fromtimestamp(mtime).strftime("%Y-%m-%d")
        rename_to = db_name.replace(".sqlite", f".{dt}.sqlite")
        if os.path.exists(rename_to):
            eprint(
                f"ERROR: Files {db_name} and {rename_to} already exist. "
                "Please cleanup!"
            )
            sys.exit(1)
        eprint(f"WARNING: Renaming existing file {db_name} to {rename_to}.")
        os.rename(db_name, rename_to)

    eprint(f"{start_time} {script_name} version {VERSION}")
    eprint(f"\tReading {bundle_dir},")
    eprint(f"\tcreating/updating sqlite db {db_name}.")

    conn = sqlite3.connect(db_name)

    universe = None
    if universe_file:
        eprint(f"  Processing {universe_file} as universe details...")
        universe = UniverseInfo(universe_file)

    create_schema(conn, script_name)

    has_entities = False
    if entities_file:
        process_entities(entities_file, conn, universe)
        has_entities = True

    for tr in tablet_reports:
        process_tablet_report(tr, conn, universe)

    TableInfo.generate_report(conn)
    create_summary_view(conn, has_entities)

    c = conn.cursor()

    eprint("\n" + "=" * 60)
    eprint("  Available REPORT-NAMEs")
    eprint("=" * 60)
    names = [
        row[0]
        for row in c.execute(
            "SELECT name FROM sqlite_master WHERE type IN ('table','view') "
            "ORDER BY type DESC, name"
        )
    ]
    col_width = max(len(n) for n in names) + 4
    cols = max(1, 60 // col_width)
    for i in range(0, len(names), cols):
        eprint("  " + "".join(n.ljust(col_width) for n in names[i : i + cols]))

    eprint("\n" + "-" * 60)
    eprint("  Summary Report")
    eprint("-" * 60)
    for row in c.execute("SELECT * FROM summary_report"):
        text = str(row[0] or "")
        if text:
            eprint(f"  * {text}")
    eprint("-" * 60)

    conn.close()

    db_path = os.path.abspath(db_name)
    ui_url = f"http://lincoln:5001/tablet-report?db={quote(db_path, safe='')}"

    eprint(
        f"\n  To get a report, run:\n"
        f'  sqlite3 -header -column {db_name} '
        '"SELECT * from <REPORT-NAME>"'
    )
    eprint(f"\n  Open UI: {ui_url}")


if __name__ == "__main__":
    main()
