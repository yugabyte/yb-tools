#!/usr/bin/env python3
"""
YugabyteDB Tablet Rebalance Selector

Analyzes tablet distribution across disks on a TServer and recommends which
tablets to force-delete so that remote bootstrap (RBS) re-places them on a
different, less-loaded disk.

IMPORTANT: YugabyteDB places new tablets on the disk with the fewest tablet
count (per-table first, overall count as tiebreaker). It does NOT consider
free disk space. This means blindly deleting a tablet from a full disk can
cause it to be placed right back on the same disk.

This script simulates the placement algorithm to identify tablets that will
actually land on a different disk after RBS.

Collect the input files on the TServer:

    du /mnt/disk0/yb-data/tserver/data/rocksdb/table-*/tablet-* \\
        | grep -v intent | grep -v snapshot > tablet_sizes
    du /mnt/disk1/yb-data/tserver/data/rocksdb/table-*/tablet-* \\
        | grep -v intent | grep -v snapshot >> tablet_sizes
    << ADD MORE DISKs HERE >>

    # Optional — enables disk usage % display and --target-usage:
    df -PT /mnt/disk* > disk_usage

Then run the analysis locally:

    # Minimal (only du output required):
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes

    # With df data (adds usage % display and --target-usage support):
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes --disk-usage disk_usage
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes --disk-usage disk_usage --target-usage 75
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes --disk-usage disk_usage -n 5

    # Analyze a single disk only:
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes --source-disk /mnt/disk0/yb-data

    # Per-disk recommendations independently (not interleaved):
    python tablet_rebalance_selector.py --tablet-sizes tablet_sizes --all-disks
"""

import argparse
import json
import os
import re
import sys
from collections import defaultdict
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Tuple

_TABLET_PATH_RE = re.compile(
    r"^(\d+)\s+"
    r"(.+?)/tserver/data/rocksdb/"
    r"table-([0-9a-fA-F-]+)/"
    r"tablet-([0-9a-fA-F-]+)\s*$"
)


@dataclass
class TabletInfo:
    tablet_id: str
    table_id: str
    disk: str
    size_bytes: int

    @property
    def size_human(self) -> str:
        return _human_size(self.size_bytes)


@dataclass
class DiskInfo:
    path: str
    total_bytes: int = 0
    used_bytes: int = 0
    free_bytes: int = 0
    tablets: List[TabletInfo] = field(default_factory=list)

    @property
    def usage_pct(self) -> float:
        if self.total_bytes == 0:
            return 0.0
        return (self.used_bytes / self.total_bytes) * 100

    @property
    def tablet_data_bytes(self) -> int:
        return sum(t.size_bytes for t in self.tablets)


@dataclass
class MoveRecommendation:
    tablet: TabletInfo
    source_disk: str
    dest_disk: str
    sequence_num: int


def _human_size(nbytes: int) -> str:
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(nbytes) < 1024:
            return f"{nbytes:.1f} {unit}"
        nbytes /= 1024
    return f"{nbytes:.1f} PB"


def _has_capacity(disks: Dict[str, "DiskInfo"]) -> bool:
    """Check whether disk capacity info (from df) is available."""
    return any(d.total_bytes > 0 for d in disks.values())


def parse_tablet_sizes(filepath: str) -> List[Tuple[str, str, str, int]]:
    """
    Parse `du` output file. Returns list of (data_dir, table_id, tablet_id, size_bytes).

    Expected input format (du default uses 1024-byte blocks):
        <blocks>\\t<path>/tserver/data/rocksdb/table-<uuid>/tablet-<uuid>

    Lines with deeper subdirectories (e.g. .../tablet-<uuid>/subdir) are
    automatically filtered out by the regex.
    """
    results = []
    with open(filepath, "r") as f:
        for line in f:
            m = _TABLET_PATH_RE.match(line.rstrip("\n"))
            if not m:
                continue
            size_kb = int(m.group(1))
            data_dir = m.group(2)
            table_id = m.group(3)
            tablet_id = m.group(4)
            results.append((data_dir, table_id, tablet_id, size_kb * 1024))
    return results


def parse_disk_usage(filepath: str) -> Dict[str, Tuple[int, int, int]]:
    """
    Parse `df -PT` output file. Returns {mount_point: (total_bytes, used_bytes, free_bytes)}.

    Expected input format:
        Filesystem     Type  1024-blocks      Used Available Capacity Mounted on
        /dev/sdn       xfs   7331889152 5980921856 1350967296      82% /mnt/disk0
    """
    mounts: Dict[str, Tuple[int, int, int]] = {}
    with open(filepath, "r") as f:
        for line in f:
            parts = line.split()
            if len(parts) < 7 or parts[0] == "Filesystem":
                continue
            try:
                total_kb = int(parts[2])
                used_kb = int(parts[3])
                avail_kb = int(parts[4])
            except (ValueError, IndexError):
                continue
            mount_point = parts[6]
            mounts[mount_point] = (total_kb * 1024, used_kb * 1024, avail_kb * 1024)
    return mounts


def _find_mount_for_dir(data_dir: str, mounts: Dict[str, Tuple[int, int, int]]) -> Optional[str]:
    """Find the longest-matching mount point for a data directory path."""
    best = None
    best_len = 0
    for mp in mounts:
        if data_dir == mp or data_dir.startswith(mp.rstrip("/") + "/"):
            if len(mp) > best_len:
                best = mp
                best_len = len(mp)
    return best


def discover_tablets(
    tablet_sizes_file: str,
    disk_usage_file: Optional[str] = None,
) -> Tuple[Dict[str, DiskInfo], List[TabletInfo]]:
    """Build tablet inventory from pre-collected du output and optional df output."""
    tablet_rows = parse_tablet_sizes(tablet_sizes_file)
    mounts = parse_disk_usage(disk_usage_file) if disk_usage_file else {}

    if not tablet_rows:
        print(f"ERROR: No tablet entries parsed from {tablet_sizes_file}", file=sys.stderr)
        print("Expected format: <size_kb>\\t<path>/.../table-<uuid>/tablet-<uuid>",
              file=sys.stderr)
        sys.exit(1)
    if disk_usage_file and not mounts:
        print(f"ERROR: No mount entries parsed from {disk_usage_file}", file=sys.stderr)
        sys.exit(1)

    disks: Dict[str, DiskInfo] = {}
    all_tablets: List[TabletInfo] = []

    for data_dir, table_id, tablet_id, size_bytes in tablet_rows:
        if data_dir not in disks:
            total_b, used_b, free_b = 0, 0, 0
            if mounts:
                mp = _find_mount_for_dir(data_dir, mounts)
                if mp and mp in mounts:
                    total_b, used_b, free_b = mounts[mp]
                else:
                    print(f"WARNING: No mount point found for {data_dir}", file=sys.stderr)
            disks[data_dir] = DiskInfo(
                path=data_dir,
                total_bytes=total_b,
                used_bytes=used_b,
                free_bytes=free_b,
            )

        tablet_info = TabletInfo(
            tablet_id=tablet_id,
            table_id=table_id,
            disk=data_dir,
            size_bytes=size_bytes,
        )
        disks[data_dir].tablets.append(tablet_info)
        all_tablets.append(tablet_info)

    return disks, all_tablets


def build_counts(
    all_tablets: List[TabletInfo], disks: Dict[str, DiskInfo]
) -> Tuple[Dict[str, Dict[str, int]], Dict[str, int]]:
    """Build per-table-per-disk counts and overall per-disk counts."""
    per_table: Dict[str, Dict[str, int]] = defaultdict(lambda: defaultdict(int))
    overall: Dict[str, int] = defaultdict(int)

    for disk_path in disks:
        overall.setdefault(disk_path, 0)

    for t in all_tablets:
        per_table[t.table_id][t.disk] += 1
        overall[t.disk] += 1

    for table_id in per_table:
        for disk_path in disks:
            per_table[table_id].setdefault(disk_path, 0)

    return dict(per_table), dict(overall)


def simulate_placement(
    table_id: str,
    source_disk: str,
    per_table_counts: Dict[str, Dict[str, int]],
    overall_counts: Dict[str, int],
) -> str:
    """
    Simulate GetAndRegisterDataAndWalDir after deleting a tablet of table_id
    from source_disk. Returns the disk path where the tablet would be placed.

    Algorithm (from ts_tablet_manager.cc):
      1. Pick disk with fewest tablets for this table.
      2. Break ties by fewest tablets overall.
    """
    sim_table = dict(per_table_counts.get(table_id, {}))
    sim_overall = dict(overall_counts)

    sim_table[source_disk] = sim_table.get(source_disk, 1) - 1
    sim_overall[source_disk] = sim_overall.get(source_disk, 1) - 1

    best_disk = None
    best_table_count = float("inf")
    best_overall_count = float("inf")

    for disk, tc in sim_table.items():
        oc = sim_overall.get(disk, 0)
        if tc < best_table_count or (
            tc == best_table_count and oc < best_overall_count
        ):
            best_disk = disk
            best_table_count = tc
            best_overall_count = oc

    return best_disk


def find_movable_tablets(
    source_disk: str,
    disks: Dict[str, DiskInfo],
    per_table_counts: Dict[str, Dict[str, int]],
    overall_counts: Dict[str, int],
    max_recommendations: int,
    target_usage_pct: Optional[float] = None,
) -> List[MoveRecommendation]:
    """
    Find tablets on source_disk that, when force-deleted, will be re-placed
    on a different disk by the count-based algorithm.

    Simulates moves sequentially: after each recommended move, counts are
    updated so subsequent predictions account for earlier moves.

    If target_usage_pct is set, stops recommending once cumulative freed space
    would bring the source disk's usage below that percentage.
    """
    sim_per_table = {
        tid: dict(dcounts) for tid, dcounts in per_table_counts.items()
    }
    sim_overall = dict(overall_counts)

    candidates = sorted(
        disks[source_disk].tablets, key=lambda t: t.size_bytes, reverse=True
    )

    disk_info = disks[source_disk]
    target_freed_bytes = None
    if target_usage_pct is not None and disk_info.total_bytes > 0:
        current_used = disk_info.used_bytes
        target_used = disk_info.total_bytes * (target_usage_pct / 100.0)
        if current_used > target_used:
            target_freed_bytes = current_used - target_used

    recommendations: List[MoveRecommendation] = []
    moved_tablet_ids = set()
    cumulative_freed = 0

    for tablet in candidates:
        if len(recommendations) >= max_recommendations:
            break
        if target_freed_bytes is not None and cumulative_freed >= target_freed_bytes:
            break
        if tablet.tablet_id in moved_tablet_ids:
            continue

        dest = simulate_placement(
            tablet.table_id, source_disk, sim_per_table, sim_overall
        )

        if dest is None or dest == source_disk:
            continue

        seq = len(recommendations) + 1
        recommendations.append(MoveRecommendation(
            tablet=tablet,
            source_disk=source_disk,
            dest_disk=dest,
            sequence_num=seq,
        ))
        moved_tablet_ids.add(tablet.tablet_id)
        cumulative_freed += tablet.size_bytes

        sim_per_table[tablet.table_id][source_disk] -= 1
        sim_per_table[tablet.table_id][dest] += 1
        sim_overall[source_disk] -= 1
        sim_overall[dest] += 1

    return recommendations


def find_unmovable_largest(
    source_disk: str,
    disks: Dict[str, DiskInfo],
    per_table_counts: Dict[str, Dict[str, int]],
    overall_counts: Dict[str, int],
    limit: int = 5,
) -> List[Tuple[TabletInfo, str]]:
    """
    Return the largest tablets on source_disk that CANNOT be moved (would land
    back on the same disk), along with a short reason.
    """
    results = []
    candidates = sorted(
        disks[source_disk].tablets, key=lambda t: t.size_bytes, reverse=True
    )
    for tablet in candidates:
        dest = simulate_placement(
            tablet.table_id, source_disk, per_table_counts, overall_counts
        )
        if dest == source_disk:
            table_counts = per_table_counts.get(tablet.table_id, {})
            src_count = table_counts.get(source_disk, 0)
            min_other = min(
                (c for d, c in table_counts.items() if d != source_disk),
                default=src_count,
            )
            if src_count - 1 <= min_other:
                reason = (
                    f"per-table count on source ({src_count}) <= other disks "
                    f"min ({min_other}) after deletion"
                )
            else:
                reason = "overall count tiebreaker favors source disk"
            results.append((tablet, reason))
            if len(results) >= limit:
                break
    return results


def find_balanced_moves(
    disks: Dict[str, DiskInfo],
    per_table_counts: Dict[str, Dict[str, int]],
    overall_counts: Dict[str, int],
    max_recommendations: int,
    target_usage_pct: Optional[float] = None,
) -> List[MoveRecommendation]:
    """
    Find movable tablets across ALL overloaded disks to equalize disk load.

    Each iteration picks the largest movable tablet from whichever disk
    currently has the highest effective load. After each simulated move,
    both source and destination effective load are updated before choosing
    the next move.

    When disk capacity info is available (from df), load is measured as disk
    usage percentage.  Otherwise, tablet data bytes per disk are used as the
    load metric and the target is the per-disk average.
    """
    has_capacity = _has_capacity(disks)

    if has_capacity:
        if target_usage_pct is None:
            total_used = sum(d.used_bytes for d in disks.values())
            total_cap = sum(d.total_bytes for d in disks.values())
            target_usage_pct = (total_used / total_cap * 100.0) if total_cap > 0 else 0.0
        effective_load: Dict[str, float] = {p: float(d.used_bytes) for p, d in disks.items()}
        capacity: Dict[str, float] = {p: float(d.total_bytes) for p, d in disks.items()}
        target_load = target_usage_pct

        def get_load(dp: str) -> Optional[float]:
            if capacity[dp] == 0:
                return None
            return effective_load[dp] / capacity[dp] * 100.0
    else:
        effective_load = {p: float(d.tablet_data_bytes) for p, d in disks.items()}
        total_data = sum(effective_load.values())
        target_load = total_data / len(disks) if disks else 0.0

        def get_load(dp: str) -> Optional[float]:
            return effective_load[dp]

    sim_per_table = {
        tid: dict(dcounts) for tid, dcounts in per_table_counts.items()
    }
    sim_overall = dict(overall_counts)

    per_disk_candidates: Dict[str, List[TabletInfo]] = {}
    per_disk_cursor: Dict[str, int] = {}
    for disk_path, disk_info in disks.items():
        per_disk_candidates[disk_path] = sorted(
            disk_info.tablets, key=lambda t: t.size_bytes, reverse=True
        )
        per_disk_cursor[disk_path] = 0

    recommendations: List[MoveRecommendation] = []
    moved_tablet_ids = set()
    exhausted_disks = set()

    while len(recommendations) < max_recommendations:
        best_source = None
        best_val = -1.0
        for dp in disks:
            if dp in exhausted_disks:
                continue
            val = get_load(dp)
            if val is None:
                continue
            if val > target_load and val > best_val:
                best_val = val
                best_source = dp

        if best_source is None:
            break

        candidates = per_disk_candidates[best_source]
        found = False
        while per_disk_cursor[best_source] < len(candidates):
            tablet = candidates[per_disk_cursor[best_source]]
            per_disk_cursor[best_source] += 1

            if tablet.tablet_id in moved_tablet_ids:
                continue

            dest = simulate_placement(
                tablet.table_id, best_source, sim_per_table, sim_overall
            )
            if dest is None or dest == best_source:
                continue

            seq = len(recommendations) + 1
            recommendations.append(MoveRecommendation(
                tablet=tablet,
                source_disk=best_source,
                dest_disk=dest,
                sequence_num=seq,
            ))
            moved_tablet_ids.add(tablet.tablet_id)

            sim_per_table[tablet.table_id][best_source] -= 1
            sim_per_table[tablet.table_id][dest] += 1
            sim_overall[best_source] -= 1
            sim_overall[dest] += 1

            effective_load[best_source] -= tablet.size_bytes
            effective_load[dest] += tablet.size_bytes

            found = True
            break

        if not found:
            exhausted_disks.add(best_source)

    return recommendations


def print_balance_recommendations(
    recommendations: List[MoveRecommendation],
    disks: Dict[str, DiskInfo],
    target_usage_pct: Optional[float],
) -> None:
    """Print recommendations from --balance mode."""

    has_capacity = _has_capacity(disks)

    print("=" * 78)
    print("BALANCE RECOMMENDATIONS (across all disks)")
    if target_usage_pct is not None:
        print(f"  Target: bring all disks below {target_usage_pct:.1f}%")
    elif has_capacity:
        total_used = sum(d.used_bytes for d in disks.values())
        total_cap = sum(d.total_bytes for d in disks.values())
        avg_pct = (total_used / total_cap * 100.0) if total_cap > 0 else 0.0
        print(f"  Target: average usage across disks ({avg_pct:.1f}%)")
    else:
        total_data = sum(d.tablet_data_bytes for d in disks.values())
        avg_data = total_data / len(disks) if disks else 0
        print(f"  Target: equalize tablet data across disks (avg {_human_size(avg_data)}/disk)")
    print("=" * 78)

    if not recommendations:
        print()
        print("  *** NO MOVABLE TABLETS FOUND ***")
        print()
        print("  Per-table tablet counts are balanced across all disks, so no")
        print("  tablets can be moved via force-delete + RBS.")
        print()
        print("  Alternative approaches:")
        print("    1. Stop the TServer, physically move tablet directories between")
        print("       disk mount points, and restart the TServer.")
        print("    2. If tablet splits caused the imbalance, consider manual tablet")
        print("       splitting on under-loaded disks to redistribute data.")
        print("    3. Add a new disk to shift the count-based balance point.")
        print()
        return

    space_freed: Dict[str, int] = defaultdict(int)
    space_added: Dict[str, int] = defaultdict(int)

    print()
    print(f"  Found {len(recommendations)} move(s):\n")
    for rec in recommendations:
        src_short = os.path.basename(os.path.dirname(rec.source_disk)) or rec.source_disk
        dest_short = os.path.basename(os.path.dirname(rec.dest_disk)) or rec.dest_disk
        space_freed[rec.source_disk] += rec.tablet.size_bytes
        space_added[rec.dest_disk] += rec.tablet.size_bytes
        print(f"  #{rec.sequence_num}  Tablet: {rec.tablet.tablet_id}")
        print(f"       Table:  {rec.tablet.table_id}")
        print(f"       Size:   {rec.tablet.size_human}")
        print(f"       Move:   {rec.source_disk} ({src_short}) -> {rec.dest_disk} ({dest_short})")
        print()

    print("-" * 78)
    if has_capacity:
        print("  PROJECTED DISK USAGE AFTER ALL MOVES:\n")
        for path in sorted(disks, key=lambda p: disks[p].usage_pct, reverse=True):
            d = disks[path]
            short = os.path.basename(os.path.dirname(path)) or path
            freed = space_freed.get(path, 0)
            added = space_added.get(path, 0)
            new_used = d.used_bytes - freed + added
            new_pct = (new_used / d.total_bytes * 100.0) if d.total_bytes > 0 else 0.0
            delta = new_pct - d.usage_pct
            delta_str = f"+{delta:.1f}" if delta >= 0 else f"{delta:.1f}"
            print(f"    {short:<10s}  {d.usage_pct:5.1f}% -> {new_pct:5.1f}%  ({delta_str}%)"
                  f"  freed: {_human_size(freed)}"
                  f"  added: {_human_size(added)}")
    else:
        print("  PROJECTED TABLET DATA AFTER ALL MOVES:\n")
        for path in sorted(disks, key=lambda p: disks[p].tablet_data_bytes, reverse=True):
            d = disks[path]
            short = os.path.basename(os.path.dirname(path)) or path
            freed = space_freed.get(path, 0)
            added = space_added.get(path, 0)
            old_data = d.tablet_data_bytes
            new_data = old_data - freed + added
            print(f"    {short:<10s}  {_human_size(old_data)} -> {_human_size(new_data)}"
                  f"  ({len(d.tablets)} tablets)"
                  f"  freed: {_human_size(freed)}"
                  f"  added: {_human_size(added)}")
    print()

    print("-" * 78)
    print("  COMMANDS TO EXECUTE (in order):\n")
    print("  # Step 1: Allow RBS for deleted tablets")
    print("  yb-ts-cli --server_address=<tserver>:9100 "
          "set_flag reject_rbs_for_deleted_tablet false\n")
    print("  # IMPORTANT: Execute one delete at a time. Wait for RBS to complete")
    print("  # before running the next delete command.\n")
    for rec in recommendations:
        src_short = os.path.basename(os.path.dirname(rec.source_disk)) or rec.source_disk
        dest_short = os.path.basename(os.path.dirname(rec.dest_disk)) or rec.dest_disk
        print(f"  # Step {rec.sequence_num + 1}: "
              f"Delete tablet {rec.tablet.tablet_id[:12]}... "
              f"({rec.tablet.size_human}, {src_short} -> {dest_short})")
        print(f"  yb-ts-cli --server_address=<tserver>:9100 "
              f"delete_tablet {rec.tablet.tablet_id} "
              f"\"Disk rebalance: move to less-loaded disk\" --force\n")
    print(f"  # Final: Restore the flag")
    print("  yb-ts-cli --server_address=<tserver>:9100 "
          "set_flag reject_rbs_for_deleted_tablet true\n")
    print("  # NOTE: After executing these moves, re-collect tablet_sizes and")
    print("  # disk_usage from the TServer and re-run this script to verify the")
    print("  # new state and get updated recommendations if further rebalancing")
    print("  # is needed.\n")


def print_summary(
    disks: Dict[str, DiskInfo],
    all_tablets: List[TabletInfo],
    per_table_counts: Dict[str, Dict[str, int]],
    overall_counts: Dict[str, int],
) -> None:
    has_capacity = _has_capacity(disks)
    print("=" * 78)
    print("DISK SUMMARY")
    print("=" * 78)
    sort_key = (lambda p: disks[p].usage_pct) if has_capacity else (lambda p: disks[p].tablet_data_bytes)
    for path in sorted(disks, key=sort_key, reverse=True):
        d = disks[path]
        if has_capacity:
            print(
                f"  {path:<40s}  {d.usage_pct:5.1f}%  "
                f"({_human_size(d.used_bytes)} / {_human_size(d.total_bytes)})  "
                f"{len(d.tablets)} tablets  "
                f"[tablet data: {_human_size(d.tablet_data_bytes)}]"
            )
        else:
            print(
                f"  {path:<40s}  "
                f"{len(d.tablets)} tablets  "
                f"[tablet data: {_human_size(d.tablet_data_bytes)}]"
            )
    print()

    n_tables = len(per_table_counts)
    print(f"Total: {len(all_tablets)} tablets across {len(disks)} disks, {n_tables} tables")
    print()

    print("-" * 78)
    print("PER-TABLE TABLET COUNT BY DISK")
    print("-" * 78)
    disk_paths = sorted(disks.keys())
    short_names = {p: os.path.basename(os.path.dirname(p)) or p for p in disk_paths}
    header = f"  {'Table ID':<40s}" + "".join(f"  {short_names[p]:>8s}" for p in disk_paths)
    print(header)
    for table_id in sorted(per_table_counts.keys()):
        counts = per_table_counts[table_id]
        vals = [counts.get(p, 0) for p in disk_paths]
        if max(vals) - min(vals) == 0:
            marker = ""
        elif max(vals) - min(vals) == 1:
            marker = "  ~"
        else:
            marker = "  *SKEWED*"
        row = f"  {table_id[:40]:<40s}" + "".join(f"  {v:>8d}" for v in vals) + marker
        print(row)
    print()
    print("  Legend: *SKEWED* = count difference >= 2 (movable candidates likely exist)")
    print("         ~        = count difference == 1 (may be movable via tiebreaker)")
    print()


def print_recommendations(
    source_disk: str,
    recommendations: List[MoveRecommendation],
    unmovable: List[Tuple[TabletInfo, str]],
    disks: Dict[str, DiskInfo],
) -> None:
    print("=" * 78)
    print(f"RECOMMENDATIONS FOR: {source_disk}")
    d = disks[source_disk]
    if d.total_bytes > 0:
        print(f"  Current usage: {d.usage_pct:.1f}% ({len(d.tablets)} tablets)")
    else:
        print(f"  Tablets: {len(d.tablets)}  "
              f"[tablet data: {_human_size(d.tablet_data_bytes)}]")
    print("=" * 78)

    if recommendations:
        cumulative_freed = 0
        src_short = os.path.basename(os.path.dirname(source_disk)) or source_disk
        print()
        print(f"  Found {len(recommendations)} movable tablet(s) (largest first):\n")
        for rec in recommendations:
            cumulative_freed += rec.tablet.size_bytes
            dest_short = os.path.basename(os.path.dirname(rec.dest_disk)) or rec.dest_disk
            print(f"  #{rec.sequence_num}  Tablet: {rec.tablet.tablet_id}")
            print(f"       Table:  {rec.tablet.table_id}")
            print(f"       Size:   {rec.tablet.size_human}")
            print(f"       Move:   {source_disk} ({src_short}) -> {rec.dest_disk} ({dest_short})")
            print(f"       Cumulative space freed from {src_short}: {_human_size(cumulative_freed)}")
            print()

        print("-" * 78)
        print("  COMMANDS TO EXECUTE (in order):\n")
        print("  # Step 1: Allow RBS for deleted tablets")
        print("  yb-ts-cli --server_address=<tserver>:9100 "
              "set_flag reject_rbs_for_deleted_tablet false\n")
        print("  # IMPORTANT: Execute one delete at a time. Wait for RBS to complete")
        print("  # before running the next delete command.\n")
        for rec in recommendations:
            dest_short = os.path.basename(os.path.dirname(rec.dest_disk)) or rec.dest_disk
            print(f"  # Step {rec.sequence_num + 1}: "
                  f"Delete tablet {rec.tablet.tablet_id[:12]}... "
                  f"({rec.tablet.size_human}, {src_short} -> {dest_short})")
            print(f"  yb-ts-cli --server_address=<tserver>:9100 "
                  f"delete_tablet {rec.tablet.tablet_id} "
                  f"\"Disk rebalance: move to less-loaded disk\" --force\n")
        print(f"  # Final: Restore the flag")
        print("  yb-ts-cli --server_address=<tserver>:9100 "
              "set_flag reject_rbs_for_deleted_tablet true\n")
        print("  # NOTE: After executing these moves, re-collect tablet_sizes and")
        print("  # disk_usage from the TServer and re-run this script to verify the")
        print("  # new state and get updated recommendations if further rebalancing")
        print("  # is needed.\n")
    else:
        print()
        print("  *** NO MOVABLE TABLETS FOUND ***")
        print()
        print("  All per-table tablet counts are balanced across disks, so any")
        print("  deleted tablet would be placed right back on this disk by the")
        print("  count-based placement algorithm.")
        print()
        print("  Alternative approaches:")
        print("    1. Stop the TServer, physically move tablet directories between")
        print("       disk mount points, and restart the TServer.")
        print("    2. If tablet splits caused the imbalance, consider manual tablet")
        print("       splitting on under-loaded disks to redistribute data.")
        print("    3. Add a new disk to shift the count-based balance point.")
        print()

    if unmovable:
        print("-" * 78)
        print("  LARGEST NON-MOVABLE TABLETS (would land back on same disk):\n")
        for tablet, reason in unmovable:
            print(f"    {tablet.tablet_id}  {tablet.size_human:>10s}  ({reason})")
        print()


def export_json(
    disks: Dict[str, DiskInfo],
    all_tablets: List[TabletInfo],
    recommendations_by_disk: Dict[str, List[MoveRecommendation]],
) -> str:
    data = {
        "disks": {
            path: {
                "total_bytes": d.total_bytes,
                "used_bytes": d.used_bytes,
                "free_bytes": d.free_bytes,
                "usage_pct": round(d.usage_pct, 2),
                "tablet_count": len(d.tablets),
                "tablet_data_bytes": d.tablet_data_bytes,
            }
            for path, d in disks.items()
        },
        "total_tablets": len(all_tablets),
        "recommendations": {
            disk: [
                {
                    "sequence": r.sequence_num,
                    "tablet_id": r.tablet.tablet_id,
                    "table_id": r.tablet.table_id,
                    "size_bytes": r.tablet.size_bytes,
                    "source_disk": r.source_disk,
                    "dest_disk": r.dest_disk,
                }
                for r in recs
            ]
            for disk, recs in recommendations_by_disk.items()
        },
    }
    return json.dumps(data, indent=2)


def main():
    parser = argparse.ArgumentParser(
        description="Analyze tablet distribution across TServer disks and recommend "
                    "tablets to force-delete for disk space rebalancing.",
        epilog="YugabyteDB places tablets by count, not by disk space. This script "
               "simulates the placement algorithm to identify tablets that will "
               "actually land on a different disk after remote bootstrap.",
    )
    parser.add_argument(
        "--tablet-sizes",
        required=True,
        metavar="FILE",
        help="File containing `du` output of tablet directories. "
             "Collect with: du /mnt/disk*/yb-data/tserver/data/rocksdb/table-*/tablet-* "
             "| grep -v intent | grep -v snapshot > tablet_sizes",
    )
    parser.add_argument(
        "--disk-usage",
        metavar="FILE",
        default=None,
        help="Optional file containing `df -PT` output.  Enables disk usage %% "
             "display and --target-usage support.  "
             "Collect with: df -PT /mnt/disk* > disk_usage",
    )
    parser.add_argument(
        "--top", "-n",
        type=int,
        default=10,
        help="Maximum number of tablet recommendations per disk (default: 10)",
    )
    parser.add_argument(
        "--source-disk",
        metavar="DISK",
        help="Only analyze this specific disk as the source (use the data dir path, "
             "e.g. /mnt/disk0/yb-data). By default, the most-used disk is chosen.",
    )
    parser.add_argument(
        "--target-usage",
        type=float,
        metavar="PCT",
        help="Stop recommending once enough tablets are identified to bring the "
             "disk(s) below this usage percentage (e.g. 75). In the default "
             "balance mode, defaults to the average usage across all disks.",
    )
    parser.add_argument(
        "--all-disks",
        action="store_true",
        help="Show per-disk recommendations for each disk independently "
             "(not interleaved). By default, moves are balanced across all disks.",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        dest="json_output",
        help="Output results as JSON.",
    )
    args = parser.parse_args()

    if args.target_usage is not None and args.disk_usage is None:
        parser.error("--target-usage requires --disk-usage")

    disks, all_tablets = discover_tablets(args.tablet_sizes, args.disk_usage)

    if len(disks) < 2:
        print("ERROR: Need at least 2 disks in the tablet_sizes data.", file=sys.stderr)
        sys.exit(1)

    if not all_tablets:
        print("ERROR: No tablets found in the provided files.", file=sys.stderr)
        sys.exit(1)

    per_table_counts, overall_counts = build_counts(all_tablets, disks)

    single_disk_mode = args.source_disk or args.all_disks

    if single_disk_mode:
        if args.source_disk:
            source_disk = args.source_disk
            if source_disk not in disks:
                print(f"ERROR: Source disk '{source_disk}' not found in parsed data.",
                      file=sys.stderr)
                print(f"Available disks: {', '.join(sorted(disks.keys()))}", file=sys.stderr)
                sys.exit(1)
            source_disks = [source_disk]
        else:
            if _has_capacity(disks):
                sort_key = lambda p: disks[p].usage_pct
            else:
                sort_key = lambda p: disks[p].tablet_data_bytes
            source_disks = sorted(disks.keys(), key=sort_key, reverse=True)

        recommendations_by_disk: Dict[str, List[MoveRecommendation]] = {}
        unmovable_by_disk: Dict[str, List[Tuple[TabletInfo, str]]] = {}

        for sd in source_disks:
            recs = find_movable_tablets(
                sd, disks, per_table_counts, overall_counts, args.top,
                target_usage_pct=args.target_usage,
            )
            recommendations_by_disk[sd] = recs
            unmovable_by_disk[sd] = find_unmovable_largest(
                sd, disks, per_table_counts, overall_counts, limit=5
            )

        if args.json_output:
            print(export_json(disks, all_tablets, recommendations_by_disk))
            return

        print_summary(disks, all_tablets, per_table_counts, overall_counts)

        for sd in source_disks:
            print_recommendations(
                sd,
                recommendations_by_disk[sd],
                unmovable_by_disk[sd],
                disks,
            )
    else:
        recs = find_balanced_moves(
            disks, per_table_counts, overall_counts, args.top,
            target_usage_pct=args.target_usage,
        )

        if args.json_output:
            print(export_json(disks, all_tablets, {"_balanced": recs}))
            return

        print_summary(disks, all_tablets, per_table_counts, overall_counts)
        print_balance_recommendations(recs, disks, args.target_usage)

    print("=" * 78)
    print("NOTES:")
    print("  - Predictions assume no other tablet creates/deletes happen concurrently.")
    print("  - Execute moves ONE AT A TIME. Wait for RBS to complete before the next")
    print("    delete, so the placement algorithm sees updated counts.")
    print("  - Verify the tablet is healthy (RF replicas present) before deleting.")
    print("  - Avoid deleting leader replicas; step down leadership first.")
    print("  - Do not delete tablets from colocated table groups.")
    print("  - Run during a maintenance window or low-traffic period.")
    print("=" * 78)


if __name__ == "__main__":
    main()
