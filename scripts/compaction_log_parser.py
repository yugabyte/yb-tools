#!/usr/bin/env python3
"""
YugabyteDB Compaction Log Parser

Parses TServer logs and extracts compaction information for a specific tablet.
Displays compaction events in a structured table format.

Usage:
    python compaction_log_parser.py -f tserver.log -t <tablet_uuid>
    python compaction_log_parser.py -f "tserver.INFO*" -t <tablet_uuid>
    python compaction_log_parser.py -f "*.log" "*.gz" -t <tablet_uuid>
    python compaction_log_parser.py -f "tserver.INFO.*.gz" -t <tablet_uuid>
    python compaction_log_parser.py -f tserver.log -t <tablet_uuid> --md
    python compaction_log_parser.py -f tserver.log -t <tablet_uuid> --json
    cat tserver.log | python compaction_log_parser.py -t <tablet_uuid>
"""

import re
import sys
import os
import json
import gzip
import argparse
from collections import defaultdict
from datetime import datetime
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any, Tuple
from enum import Enum

try:
    from rich.console import Console
    from rich.table import Table
    from rich.panel import Panel
    RICH_AVAILABLE = True
except ImportError:
    RICH_AVAILABLE = False


class CompactionState(Enum):
    """State of a compaction operation."""
    STARTED = "Started"
    COMPLETED = "Completed"
    FAILED = "Failed"
    UNKNOWN = "Unknown"


@dataclass
class CompactionEvent:
    """Represents a single compaction event."""
    job_id: int = -1
    timestamp: Optional[datetime] = None
    end_timestamp: Optional[datetime] = None
    state: CompactionState = CompactionState.UNKNOWN
    
    # Input details
    input_files_count: int = 0
    input_files_by_level: Dict[int, int] = field(default_factory=dict)
    input_file_numbers: List[int] = field(default_factory=list)
    input_size_bytes: int = 0
    input_size_mb: float = 0.0
    input_records: int = 0
    
    # Output details
    output_files_count: int = 0
    output_file_numbers: List[int] = field(default_factory=list)
    output_level: int = -1
    output_size_bytes: int = 0
    output_size_mb: float = 0.0
    output_records: int = 0
    
    # Performance metrics
    duration_micros: int = 0
    read_mb_per_sec: float = 0.0
    write_mb_per_sec: float = 0.0
    records_dropped: int = 0
    
    # Additional info
    compaction_reason: str = ""
    compaction_type: str = ""  # Type of compaction (Manual, PostSplit, Scheduled, Level, Universal, etc.)
    is_full_compaction: bool = False
    score: float = 0.0
    base_level: int = -1
    column_family: str = ""
    tablet_id: str = ""
    
    # Raw log lines for debugging
    raw_lines: List[str] = field(default_factory=list)

    @property
    def duration_seconds(self) -> Optional[float]:
        """Calculate duration in seconds."""
        if self.duration_micros > 0:
            return self.duration_micros / 1_000_000.0
        if self.timestamp and self.end_timestamp:
            return (self.end_timestamp - self.timestamp).total_seconds()
        return None

    @property
    def duration_str(self) -> str:
        """Format duration as human-readable string."""
        duration = self.duration_seconds
        if duration is None:
            return ""
        if duration < 1:
            return f"{duration * 1000:.0f}ms"
        elif duration < 60:
            return f"{duration:.2f}s"
        elif duration < 3600:
            mins = int(duration // 60)
            secs = duration % 60
            return f"{mins}m {secs:.1f}s"
        else:
            hours = int(duration // 3600)
            mins = int((duration % 3600) // 60)
            return f"{hours}h {mins}m"

    @property
    def input_files_summary(self) -> str:
        """Format input files by level."""
        if self.input_files_by_level:
            parts = [f"{count}@L{level}" for level, count in sorted(self.input_files_by_level.items())]
            return " + ".join(parts)
        return str(self.input_files_count) if self.input_files_count > 0 else ""

    @property
    def input_size_str(self) -> str:
        """Format input size."""
        return format_size(self.input_size_bytes) if self.input_size_bytes > 0 else (
            f"{self.input_size_mb:.2f} MB" if self.input_size_mb > 0 else ""
        )

    @property
    def output_size_str(self) -> str:
        """Format output size."""
        return format_size(self.output_size_bytes) if self.output_size_bytes > 0 else (
            f"{self.output_size_mb:.2f} MB" if self.output_size_mb > 0 else ""
        )

    @property
    def input_files_str(self) -> str:
        """Format input file numbers as SST file names."""
        if self.input_file_numbers:
            return ", ".join(f"{num:06d}.sst" for num in self.input_file_numbers)
        return ""

    @property
    def output_files_str(self) -> str:
        """Format output file numbers as SST file names."""
        if self.output_file_numbers:
            return ", ".join(f"{num:06d}.sst" for num in self.output_file_numbers)
        return ""

    @property
    def compaction_type_str(self) -> str:
        """Get human-readable compaction type."""
        # If explicitly set, use it
        if self.compaction_type:
            return self.compaction_type
        
        # Derive from compaction_reason if available
        reason_map = {
            'kUnknown': 'Unknown',
            'kLevelL0FilesNum': 'Level (L0)',
            'kLevelMaxLevelSize': 'Level (Size)',
            'kUniversalSizeAmplification': 'Universal (Amp)',
            'kUniversalSizeRatio': 'Universal (Ratio)',
            'kUniversalSortedRunNum': 'Universal (Runs)',
            'kUniversalDirectDeletion': 'Universal (Delete)',
            'kFIFOMaxSize': 'FIFO',
            'kManualCompaction': 'Manual',
            'kFilesMarkedForCompaction': 'Marked',
            'kAdminCompaction': 'Admin',
            'kScheduledFullCompaction': 'Scheduled',
            'kPostSplitCompaction': 'Post-Split',
            # Simplified versions without 'k' prefix
            'ManualCompaction': 'Manual',
            'PostSplitCompaction': 'Post-Split',
            'AdminCompaction': 'Admin',
            'ScheduledFullCompaction': 'Scheduled',
        }
        
        if self.compaction_reason:
            return reason_map.get(self.compaction_reason, self.compaction_reason)
        
        # Infer from other attributes
        if self.is_full_compaction:
            return 'Full'
        
        # Check input levels to determine type
        if self.input_files_by_level:
            levels = list(self.input_files_by_level.keys())
            if len(levels) == 1 and levels[0] == 0:
                return 'Level (L0)'
            elif len(levels) > 1:
                return 'Level'
        
        return ''

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON/CSV output."""
        return {
            "job_id": self.job_id,
            "timestamp": self.timestamp.isoformat() if self.timestamp else "",
            "end_timestamp": self.end_timestamp.isoformat() if self.end_timestamp else "",
            "state": self.state.value,
            "compaction_type": self.compaction_type_str,
            "input_files": self.input_files_summary,
            "input_file_names": self.input_files_str,
            "input_file_numbers": self.input_file_numbers,
            "input_files_count": self.input_files_count,
            "input_size_bytes": self.input_size_bytes,
            "input_size": self.input_size_str,
            "input_records": self.input_records,
            "output_file_names": self.output_files_str,
            "output_file_numbers": self.output_file_numbers,
            "output_files_count": self.output_files_count,
            "output_level": self.output_level,
            "output_size_bytes": self.output_size_bytes,
            "output_size": self.output_size_str,
            "output_records": self.output_records,
            "duration_seconds": self.duration_seconds,
            "duration": self.duration_str,
            "read_mb_per_sec": self.read_mb_per_sec,
            "write_mb_per_sec": self.write_mb_per_sec,
            "records_dropped": self.records_dropped,
            "is_full_compaction": self.is_full_compaction,
            "compaction_reason": self.compaction_reason,
            "tablet_id": self.tablet_id,
        }


def format_size(size_bytes: int) -> str:
    """Format bytes as human-readable size."""
    if size_bytes < 1024:
        return f"{size_bytes} B"
    elif size_bytes < 1024 * 1024:
        return f"{size_bytes / 1024:.2f} KB"
    elif size_bytes < 1024 * 1024 * 1024:
        return f"{size_bytes / (1024 * 1024):.2f} MB"
    else:
        return f"{size_bytes / (1024 * 1024 * 1024):.2f} GB"


def parse_size(size_str: str) -> int:
    """Parse size string like '1.5M', '256K', '1G' to bytes."""
    size_str = size_str.strip().upper()
    multipliers = {
        'B': 1,
        'K': 1024,
        'KB': 1024,
        'M': 1024 * 1024,
        'MB': 1024 * 1024,
        'G': 1024 * 1024 * 1024,
        'GB': 1024 * 1024 * 1024,
    }
    
    match = re.match(r'^([\d.]+)\s*([KMGB]+)?$', size_str)
    if match:
        value = float(match.group(1))
        unit = match.group(2) or 'B'
        return int(value * multipliers.get(unit, 1))
    return 0


class LogPatterns:
    """Compiled regex patterns for parsing compaction logs."""

    # Timestamp patterns - YB TServer uses glog format: Lmmdd HH:MM:SS.uuuuuu
    TIMESTAMP_GLOG = re.compile(r'^([IWEF])(\d{4})\s+(\d{2}:\d{2}:\d{2}\.\d+)')
    TIMESTAMP_ISO = re.compile(r'(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?)')
    
    # Tablet ID patterns
    TABLET_PREFIX = re.compile(r'T ([a-f0-9]{32})')
    TABLET_UUID = re.compile(r'T ([a-f0-9-]{36})')
    
    # RocksDB log prefix pattern (captures column family name)
    ROCKSDB_PREFIX = re.compile(r'\[(\w+)\]')
    
    # Compaction start patterns
    # "[cfname] [JOB N] Compacting X@L0 + Y@L1 files to LN, score S.SS"
    COMPACTION_START = re.compile(
        r'\[(\w+)\]\s*\[JOB\s+(\d+)\]\s*Compacting\s+(.+?)\s+files to L(\d+)(?:,\s*score\s+([\d.]+))?'
    )
    
    # Alternative: "Compacting X@L0 + Y@L1 files to LN"
    COMPACTION_START_ALT = re.compile(
        r'Compacting\s+((?:\d+@L?\d+\s*\+?\s*)+)\s*files? to L(\d+)'
    )
    
    # Input level summary: "3@0 + 2@3 + 1@4 files to L5" or "3@L0 + 2@L3"
    INPUT_LEVEL_SUMMARY = re.compile(r'(\d+)@L?(\d+)')
    
    # Compaction summary pattern
    # "Compaction start summary: Base version V Base level L, inputs: [file1(size) file2(size)], [file3(size)]"
    COMPACTION_SUMMARY = re.compile(
        r'Compaction start summary:\s*Base version\s+(\d+)\s+Base level\s+(\d+),\s*inputs:\s*\[(.+)\]'
    )
    
    # Simpler inputs pattern: "inputs: [...]" or "input files: [...]"
    INPUTS_PATTERN = re.compile(r'inputs?(?:\s+files)?:\s*\[([^\]]+)\]', re.IGNORECASE)
    
    # File info in summary: "32(118MB)" or "12345(1.5M)" or "12345(1.5 M)" or "12345 (1.5M)"
    # File number can be any length (e.g., 29, 32, 12345)
    # Size unit can be: B, K, KB, M, MB, G, GB
    FILE_INFO = re.compile(r'(\d+)\s*\(\s*([\d.]+)\s*([KMGB]B?|[KMGB]|B)?\s*\)')
    
    # SST file pattern: "000123.sst" or "#000123" or "table #123"
    SST_FILE_PATTERN = re.compile(r'(?:table\s*)?#?(\d+)(?:\.sst)?')
    
    # Generated table pattern
    # "[cfname] [JOB N] Generated table #NNNN: K keys, B bytes [reason] [frontiers]"
    GENERATED_TABLE = re.compile(
        r'\[(\w+)\]\s*\[JOB\s+(\d+)\]\s*Generated table #(\d+):\s*(\d+)\s*keys?,\s*(\d+)\s*bytes?'
    )
    
    # Compacted result pattern
    # "[cfname] [JOB N] Compacted X@L0 + Y@L1 files to LN => B bytes"
    COMPACTED_RESULT = re.compile(
        r'\[(\w+)\]\s*\[JOB\s+(\d+)\]\s*Compacted\s+(.+?)\s*=>\s*(\d+)\s*bytes?'
    )
    
    # Compaction finished stats pattern
    # "[cfname] compacted to: levels, MB/sec: R rd, W wr, level L, files in(X, Y) out(Z) MB in(A, B) out(C), ..."
    COMPACTION_STATS = re.compile(
        r'\[(\w+)\]\s*compacted to:.*?'
        r'files in\((\d+),\s*(\d+)\)\s*out\((\d+)\).*?'
        r'MB in\(([\d.]+),\s*([\d.]+)\)\s*out\(([\d.]+)\)'
    )
    
    # Full stats pattern with more details
    COMPACTION_STATS_FULL = re.compile(
        r'compacted to:.*?'
        r'MB/sec:\s*([\d.]+)\s*rd,\s*([\d.]+)\s*wr.*?'
        r'level\s+(\d+).*?'
        r'files in\((\d+),\s*(\d+)\)\s*out\((\d+)\).*?'
        r'MB in\(([\d.]+),\s*([\d.]+)\)\s*out\(([\d.]+)\).*?'
        r'records in:\s*(\d+),\s*records dropped:\s*(\d+)'
    )
    
    # Event logger JSON patterns (compaction_started, compaction_finished)
    EVENT_COMPACTION_STARTED = re.compile(
        r'"event"\s*:\s*"compaction_started".*?"input_data_size"\s*:\s*(\d+)'
    )
    EVENT_COMPACTION_FINISHED = re.compile(
        r'"event"\s*:\s*"compaction_finished".*?"compaction_time_micros"\s*:\s*(\d+)'
    )
    
    # JSON input files pattern: "files_L0": [123, 456, 789] or "files_L1": [...]
    JSON_INPUT_FILES = re.compile(r'"files_L(\d+)"\s*:\s*\[([^\]]+)\]')
    
    # Extract file numbers from JSON array
    JSON_FILE_NUMBERS = re.compile(r'\d+')
    
    # Compaction aborted
    COMPACTION_ABORTED = re.compile(
        r'\[JOB\s+(\d+)\]\s*Compaction.*?aborted'
    )
    
    # Compaction error
    COMPACTION_ERROR = re.compile(
        r'Compaction error:|Compaction failed'
    )
    
    # Full compaction indicator
    FULL_COMPACTION = re.compile(r'is_full_compaction.*?true|full compaction', re.IGNORECASE)
    
    # Compaction reason (from logs and JSON events)
    COMPACTION_REASON = re.compile(r'compaction_reason["\s:]+(\w+)|reason["\s:]+(\w+)')
    
    # Specific compaction type patterns
    POST_SPLIT_COMPACTION = re.compile(r'post.?split|PostSplitCompaction|kPostSplitCompaction', re.IGNORECASE)
    MANUAL_COMPACTION = re.compile(r'manual.?compaction|ManualCompaction|kManualCompaction|ForceRocksDBCompact', re.IGNORECASE)
    ADMIN_COMPACTION = re.compile(r'admin.?compaction|AdminCompaction|kAdminCompaction', re.IGNORECASE)
    SCHEDULED_COMPACTION = re.compile(r'scheduled.?full|ScheduledFullCompaction|kScheduledFullCompaction', re.IGNORECASE)
    
    # Compaction reason from JSON event
    COMPACTION_REASON_JSON = re.compile(r'"compaction_reason"\s*:\s*"?(\w+)"?')


class CompactionLogParser:
    """Parser for YugabyteDB TServer compaction logs."""

    def __init__(self, tablet_id: Optional[str] = None):
        self.tablet_id = self._normalize_tablet_id(tablet_id) if tablet_id else None
        self.compactions: Dict[int, CompactionEvent] = {}  # job_id -> CompactionEvent
        self.completed_compactions: List[CompactionEvent] = []
        self.current_year = datetime.now().year
        self.errors: List[str] = []
        self.warnings: List[str] = []

    def _normalize_tablet_id(self, tablet_id: str) -> str:
        """Normalize tablet ID by removing dashes."""
        return tablet_id.replace('-', '').lower()

    def _tablet_matches(self, line: str) -> bool:
        """Check if the log line belongs to the target tablet."""
        if not self.tablet_id:
            return True
        
        # Fast path: Quick string check before expensive regex
        # If tablet ID is not in the line at all, skip it
        if self.tablet_id not in line.lower():
            return False
        
        # Check for tablet ID in the log line
        match = LogPatterns.TABLET_PREFIX.search(line)
        if match:
            found_id = match.group(1).lower()
            return found_id == self.tablet_id
        
        match = LogPatterns.TABLET_UUID.search(line)
        if match:
            found_id = self._normalize_tablet_id(match.group(1))
            return found_id == self.tablet_id
        
        # If tablet ID string is in line but not matching our patterns,
        # still return True as it might be a relevant log line
        return True

    def parse_timestamp(self, line: str) -> Optional[datetime]:
        """Extract timestamp from log line."""
        # Try glog format first (Lmmdd HH:MM:SS.uuuuuu)
        match = LogPatterns.TIMESTAMP_GLOG.match(line)
        if match:
            month_day = match.group(2)
            time_str = match.group(3)
            try:
                # glog format: mmdd -> need to add year
                month = int(month_day[:2])
                day = int(month_day[2:])
                time_parts = time_str.split(':')
                hour = int(time_parts[0])
                minute = int(time_parts[1])
                sec_parts = time_parts[2].split('.')
                second = int(sec_parts[0])
                microsecond = int(sec_parts[1][:6].ljust(6, '0')) if len(sec_parts) > 1 else 0
                return datetime(self.current_year, month, day, hour, minute, second, microsecond)
            except (ValueError, IndexError):
                pass

        # Try ISO format
        match = LogPatterns.TIMESTAMP_ISO.search(line)
        if match:
            ts_str = match.group(1)
            for fmt in ['%Y-%m-%dT%H:%M:%S.%f', '%Y-%m-%d %H:%M:%S.%f', 
                        '%Y-%m-%dT%H:%M:%S', '%Y-%m-%d %H:%M:%S']:
                try:
                    return datetime.strptime(ts_str, fmt)
                except ValueError:
                    continue

        return None

    def parse_input_levels(self, input_str: str) -> Tuple[Dict[int, int], int]:
        """Parse input level summary like '3@0 + 2@3 + 1@4' or '3@L0 + 2@L3'."""
        levels = {}
        total = 0
        for match in LogPatterns.INPUT_LEVEL_SUMMARY.finditer(input_str):
            count = int(match.group(1))
            level = int(match.group(2))
            levels[level] = levels.get(level, 0) + count
            total += count
        return levels, total

    def parse_line(self, line: str) -> None:
        """Parse a single log line and update state."""
        # Skip lines not belonging to target tablet
        if not self._tablet_matches(line):
            return

        timestamp = self.parse_timestamp(line)
        
        # Extract tablet ID from line if present
        tablet_id = ""
        tablet_match = LogPatterns.TABLET_PREFIX.search(line) or LogPatterns.TABLET_UUID.search(line)
        if tablet_match:
            tablet_id = tablet_match.group(1)

        # Check for compaction start
        start_match = LogPatterns.COMPACTION_START.search(line)
        if start_match:
            cf_name = start_match.group(1)
            job_id = int(start_match.group(2))
            input_str = start_match.group(3)
            output_level = int(start_match.group(4))
            score = float(start_match.group(5)) if start_match.group(5) else 0.0
            
            levels, total_files = self.parse_input_levels(input_str)
            
            event = self.compactions.get(job_id, CompactionEvent(job_id=job_id))
            event.timestamp = timestamp
            event.state = CompactionState.STARTED
            event.column_family = cf_name
            event.input_files_by_level = levels
            event.input_files_count = total_files
            event.output_level = output_level
            event.score = score
            event.tablet_id = tablet_id
            event.raw_lines.append(line.strip())
            self.compactions[job_id] = event
            return

        # Check for alternative compaction start
        start_alt_match = LogPatterns.COMPACTION_START_ALT.search(line)
        if start_alt_match and 'JOB' not in line:
            input_str = start_alt_match.group(1)
            output_level = int(start_alt_match.group(2))
            levels, total_files = self.parse_input_levels(input_str)
            
            # Create a new event with auto-generated job_id
            job_id = len(self.compactions) + 1000  # Use high number to avoid conflicts
            event = CompactionEvent(
                job_id=job_id,
                timestamp=timestamp,
                state=CompactionState.STARTED,
                input_files_by_level=levels,
                input_files_count=total_files,
                output_level=output_level,
                tablet_id=tablet_id,
            )
            event.raw_lines.append(line.strip())
            self.compactions[job_id] = event
            return

        # Check for compaction summary
        summary_match = LogPatterns.COMPACTION_SUMMARY.search(line)
        if summary_match:
            base_version = int(summary_match.group(1))
            base_level = int(summary_match.group(2))
            inputs_str = summary_match.group(3)
            
            # Parse file info from inputs
            file_numbers = []
            total_size = 0
            for file_match in LogPatterns.FILE_INFO.finditer(inputs_str):
                file_num = int(file_match.group(1))
                size_val = file_match.group(2)
                size_unit = file_match.group(3) or ''
                file_size = parse_size(size_val + size_unit)
                file_numbers.append(file_num)
                total_size += file_size
            
            # Find the most recent started compaction to update
            for job_id in sorted(self.compactions.keys(), reverse=True):
                event = self.compactions[job_id]
                if event.state == CompactionState.STARTED:
                    event.base_level = base_level
                    event.input_file_numbers = file_numbers
                    event.input_size_bytes = total_size
                    if file_numbers:
                        event.input_files_count = len(file_numbers)
                    event.raw_lines.append(line.strip())
                    break
            return
        
        # Check for simpler inputs pattern: "inputs: [...]"
        inputs_match = LogPatterns.INPUTS_PATTERN.search(line)
        if inputs_match:
            inputs_str = inputs_match.group(1)
            file_numbers = []
            total_size = 0
            for file_match in LogPatterns.FILE_INFO.finditer(inputs_str):
                file_num = int(file_match.group(1))
                size_val = file_match.group(2)
                size_unit = file_match.group(3) or ''
                file_size = parse_size(size_val + size_unit)
                file_numbers.append(file_num)
                total_size += file_size
            
            if file_numbers:
                for job_id in sorted(self.compactions.keys(), reverse=True):
                    event = self.compactions[job_id]
                    if event.state == CompactionState.STARTED and not event.input_file_numbers:
                        event.input_file_numbers = file_numbers
                        if total_size > 0:
                            event.input_size_bytes = total_size
                        event.raw_lines.append(line.strip())
                        break
            return

        # Check for generated table
        # Format: "[default] [JOB 3] Generated table #12: 100 keys, 69657 bytes kAdminCompaction frontiers: ..."
        gen_match = LogPatterns.GENERATED_TABLE.search(line)
        if gen_match:
            cf_name = gen_match.group(1)
            job_id = int(gen_match.group(2))
            file_number = int(gen_match.group(3))
            keys = int(gen_match.group(4))
            size_bytes = int(gen_match.group(5))
            
            if job_id in self.compactions:
                event = self.compactions[job_id]
                event.output_file_numbers.append(file_number)
                event.output_files_count = len(event.output_file_numbers)
                event.output_size_bytes += size_bytes
                event.output_records += keys
                event.raw_lines.append(line.strip())
                
                # Extract compaction reason from line (appears after "bytes")
                # E.g., "69657 bytes kAdminCompaction frontiers:"
                if not event.compaction_reason:
                    if 'kAdminCompaction' in line:
                        event.compaction_reason = 'kAdminCompaction'
                        event.compaction_type = 'Admin'
                    elif 'kManualCompaction' in line:
                        event.compaction_reason = 'kManualCompaction'
                        event.compaction_type = 'Manual'
                    elif 'kPostSplitCompaction' in line:
                        event.compaction_reason = 'kPostSplitCompaction'
                        event.compaction_type = 'Post-Split'
                    elif 'kScheduledFullCompaction' in line:
                        event.compaction_reason = 'kScheduledFullCompaction'
                        event.compaction_type = 'Scheduled'
                    elif 'kLevelL0FilesNum' in line:
                        event.compaction_reason = 'kLevelL0FilesNum'
                        event.compaction_type = 'Level (L0)'
                    elif 'kUniversalSizeAmplification' in line:
                        event.compaction_reason = 'kUniversalSizeAmplification'
                        event.compaction_type = 'Universal (Amp)'
                    elif 'kUniversalSizeRatio' in line:
                        event.compaction_reason = 'kUniversalSizeRatio'
                        event.compaction_type = 'Universal (Ratio)'
            return

        # Check for compacted result
        result_match = LogPatterns.COMPACTED_RESULT.search(line)
        if result_match:
            cf_name = result_match.group(1)
            job_id = int(result_match.group(2))
            input_str = result_match.group(3)
            output_bytes = int(result_match.group(4))
            
            if job_id in self.compactions:
                event = self.compactions[job_id]
                event.state = CompactionState.COMPLETED
                event.end_timestamp = timestamp
                event.output_size_bytes = output_bytes
                if not event.input_files_by_level:
                    levels, total = self.parse_input_levels(input_str)
                    event.input_files_by_level = levels
                    event.input_files_count = total
                event.raw_lines.append(line.strip())
            return

        # Check for compaction stats (finished)
        stats_match = LogPatterns.COMPACTION_STATS_FULL.search(line)
        if stats_match:
            read_speed = float(stats_match.group(1))
            write_speed = float(stats_match.group(2))
            output_level = int(stats_match.group(3))
            in_files_non_output = int(stats_match.group(4))
            in_files_output = int(stats_match.group(5))
            out_files = int(stats_match.group(6))
            in_mb_non_output = float(stats_match.group(7))
            in_mb_output = float(stats_match.group(8))
            out_mb = float(stats_match.group(9))
            records_in = int(stats_match.group(10))
            records_dropped = int(stats_match.group(11))
            
            # Find the matching compaction to update
            for job_id in sorted(self.compactions.keys(), reverse=True):
                event = self.compactions[job_id]
                if event.state == CompactionState.STARTED or (
                    event.state == CompactionState.COMPLETED and not event.read_mb_per_sec
                ):
                    event.read_mb_per_sec = read_speed
                    event.write_mb_per_sec = write_speed
                    event.output_level = output_level
                    event.input_files_count = in_files_non_output + in_files_output
                    event.output_files_count = out_files
                    event.input_size_mb = in_mb_non_output + in_mb_output
                    event.output_size_mb = out_mb
                    event.input_records = records_in
                    event.records_dropped = records_dropped
                    event.state = CompactionState.COMPLETED
                    event.end_timestamp = timestamp
                    event.raw_lines.append(line.strip())
                    break
            return

        # Simpler stats pattern
        stats_simple_match = LogPatterns.COMPACTION_STATS.search(line)
        if stats_simple_match and not stats_match:
            cf_name = stats_simple_match.group(1)
            in_files_non_output = int(stats_simple_match.group(2))
            in_files_output = int(stats_simple_match.group(3))
            out_files = int(stats_simple_match.group(4))
            in_mb_non_output = float(stats_simple_match.group(5))
            in_mb_output = float(stats_simple_match.group(6))
            out_mb = float(stats_simple_match.group(7))
            
            # Find the matching compaction
            for job_id in sorted(self.compactions.keys(), reverse=True):
                event = self.compactions[job_id]
                if event.column_family == cf_name and event.state in (CompactionState.STARTED, CompactionState.COMPLETED):
                    event.input_files_count = in_files_non_output + in_files_output
                    event.output_files_count = out_files
                    event.input_size_mb = in_mb_non_output + in_mb_output
                    event.output_size_mb = out_mb
                    event.state = CompactionState.COMPLETED
                    event.end_timestamp = timestamp
                    event.raw_lines.append(line.strip())
                    break
            return

        # Check for compaction aborted
        abort_match = LogPatterns.COMPACTION_ABORTED.search(line)
        if abort_match:
            job_id = int(abort_match.group(1))
            if job_id in self.compactions:
                self.compactions[job_id].state = CompactionState.FAILED
                self.compactions[job_id].end_timestamp = timestamp
                self.compactions[job_id].raw_lines.append(line.strip())
            return

        # Check for compaction error
        if LogPatterns.COMPACTION_ERROR.search(line):
            self.errors.append(f"Compaction error at {timestamp}: {line[:100]}...")
            # Mark any started compaction as failed
            for job_id in sorted(self.compactions.keys(), reverse=True):
                event = self.compactions[job_id]
                if event.state == CompactionState.STARTED:
                    event.state = CompactionState.FAILED
                    event.end_timestamp = timestamp
                    event.raw_lines.append(line.strip())
                    break
            return

        # Check for full compaction
        if LogPatterns.FULL_COMPACTION.search(line):
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].is_full_compaction = True
                    if not self.compactions[job_id].compaction_type:
                        self.compactions[job_id].compaction_type = 'Full'
                    break

        # Check for specific compaction types
        if LogPatterns.POST_SPLIT_COMPACTION.search(line):
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].compaction_type = 'Post-Split'
                    self.compactions[job_id].compaction_reason = 'kPostSplitCompaction'
                    break
        elif LogPatterns.MANUAL_COMPACTION.search(line):
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].compaction_type = 'Manual'
                    self.compactions[job_id].compaction_reason = 'kManualCompaction'
                    break
        elif LogPatterns.ADMIN_COMPACTION.search(line):
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].compaction_type = 'Admin'
                    self.compactions[job_id].compaction_reason = 'kAdminCompaction'
                    break
        elif LogPatterns.SCHEDULED_COMPACTION.search(line):
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].compaction_type = 'Scheduled'
                    self.compactions[job_id].compaction_reason = 'kScheduledFullCompaction'
                    break

        # Check for compaction reason
        reason_match = LogPatterns.COMPACTION_REASON.search(line)
        if reason_match:
            reason = reason_match.group(1) or reason_match.group(2)
            for job_id in sorted(self.compactions.keys(), reverse=True):
                if self.compactions[job_id].state == CompactionState.STARTED:
                    self.compactions[job_id].compaction_reason = reason
                    # Set compaction type from reason if not already set
                    if not self.compactions[job_id].compaction_type:
                        self.compactions[job_id].compaction_type = self.compactions[job_id].compaction_type_str
                    break

        # Check for event logger patterns (JSON)
        if '"event"' in line and 'compaction' in line.lower():
            started_match = LogPatterns.EVENT_COMPACTION_STARTED.search(line)
            if started_match:
                input_size = int(started_match.group(1))
                for job_id in sorted(self.compactions.keys(), reverse=True):
                    if self.compactions[job_id].state == CompactionState.STARTED:
                        self.compactions[job_id].input_size_bytes = input_size
                        break
            
            # Extract input file numbers from JSON: "files_L0": [123, 456, 789]
            for files_match in LogPatterns.JSON_INPUT_FILES.finditer(line):
                level = int(files_match.group(1))
                files_str = files_match.group(2)
                file_numbers = [int(n) for n in LogPatterns.JSON_FILE_NUMBERS.findall(files_str)]
                if file_numbers:
                    for job_id in sorted(self.compactions.keys(), reverse=True):
                        event = self.compactions[job_id]
                        if event.state == CompactionState.STARTED:
                            # Add these file numbers to the input list
                            for fn in file_numbers:
                                if fn not in event.input_file_numbers:
                                    event.input_file_numbers.append(fn)
                            break
            
            finished_match = LogPatterns.EVENT_COMPACTION_FINISHED.search(line)
            if finished_match:
                duration_micros = int(finished_match.group(1))
                for job_id in sorted(self.compactions.keys(), reverse=True):
                    event = self.compactions[job_id]
                    if event.state in (CompactionState.STARTED, CompactionState.COMPLETED):
                        event.duration_micros = duration_micros
                        event.state = CompactionState.COMPLETED
                        event.end_timestamp = timestamp
                        break
        
        # Also check for JSON files pattern outside of event context
        elif 'files_L' in line and '[' in line:
            for files_match in LogPatterns.JSON_INPUT_FILES.finditer(line):
                level = int(files_match.group(1))
                files_str = files_match.group(2)
                file_numbers = [int(n) for n in LogPatterns.JSON_FILE_NUMBERS.findall(files_str)]
                if file_numbers:
                    for job_id in sorted(self.compactions.keys(), reverse=True):
                        event = self.compactions[job_id]
                        if event.state == CompactionState.STARTED:
                            for fn in file_numbers:
                                if fn not in event.input_file_numbers:
                                    event.input_file_numbers.append(fn)
                            break

    def parse(self, lines: List[str]) -> List[CompactionEvent]:
        """Parse all log lines and return compaction events."""
        for line in lines:
            line = line.rstrip('\n\r')
            if line:
                self.parse_line(line)

        # Convert to sorted list
        events = sorted(self.compactions.values(), key=lambda x: (x.timestamp or datetime.min, x.job_id))
        return events

    def get_summary(self) -> Dict[str, Any]:
        """Generate summary statistics."""
        events = list(self.compactions.values())
        completed = [e for e in events if e.state == CompactionState.COMPLETED]
        failed = [e for e in events if e.state == CompactionState.FAILED]
        
        total_input_bytes = sum(e.input_size_bytes for e in completed if e.input_size_bytes > 0)
        total_output_bytes = sum(e.output_size_bytes for e in completed if e.output_size_bytes > 0)
        total_duration = sum(e.duration_seconds or 0 for e in completed)
        
        return {
            "total_compactions": len(events),
            "completed": len(completed),
            "failed": len(failed),
            "in_progress": len([e for e in events if e.state == CompactionState.STARTED]),
            "total_input_bytes": total_input_bytes,
            "total_input_size": format_size(total_input_bytes),
            "total_output_bytes": total_output_bytes,
            "total_output_size": format_size(total_output_bytes),
            "total_duration_seconds": total_duration,
            "full_compactions": len([e for e in events if e.is_full_compaction]),
            "errors": self.errors,
        }


def format_rich_table(events: List[CompactionEvent], title: str = "Compaction Events", show_file_names: bool = True) -> None:
    """Output using rich library for terminal display."""
    if not RICH_AVAILABLE:
        print("Rich library not available. Install with: pip install rich")
        format_plain_table(events, title, show_file_names)
        return

    console = Console()
    table = Table(title=title, show_lines=True, expand=True)

    table.add_column("Start Time", style="dim", width=19)
    table.add_column("Job", justify="right", width=5)
    table.add_column("Type", style="magenta", width=11)
    table.add_column("Input Files", style="cyan", width=12)
    if show_file_names:
        table.add_column("Input File Names", style="dim", no_wrap=False, max_width=30)
    table.add_column("Input Size", justify="right", width=10)
    if show_file_names:
        table.add_column("Output File Names", style="dim", no_wrap=False, max_width=20)
    table.add_column("Output Size", justify="right", width=10)
    table.add_column("Duration", justify="right", width=9)
    table.add_column("Status", width=9)

    for event in events:
        timestamp_str = event.timestamp.strftime('%Y-%m-%d %H:%M:%S') if event.timestamp else ""
        
        status = event.state.value
        if event.state == CompactionState.COMPLETED:
            status_style = "green"
        elif event.state == CompactionState.FAILED:
            status_style = "red"
        elif event.state == CompactionState.STARTED:
            status_style = "yellow"
        else:
            status_style = ""

        row_data = [
            timestamp_str,
            str(event.job_id) if event.job_id >= 0 else "",
            event.compaction_type_str,
            event.input_files_summary,
        ]
        if show_file_names:
            row_data.append(event.input_files_str or "-")
        row_data.append(event.input_size_str)
        if show_file_names:
            row_data.append(event.output_files_str or "-")
        row_data.extend([
            event.output_size_str,
            event.duration_str,
            f"[{status_style}]{status}[/{status_style}]" if status_style else status
        ])

        table.add_row(*row_data)

    console.print(table)


def format_plain_table(events: List[CompactionEvent], title: str = "Compaction Events", show_file_names: bool = True) -> None:
    """Output as plain text table."""
    print(f"\n{title}")
    
    if show_file_names:
        print("=" * 200)
        header = f"{'Start Time':<19} | {'Job':>5} | {'Type':<11} | {'Input Files':<12} | {'Input File Names':<28} | {'Input Size':>10} | {'Output File Names':<18} | {'Output Size':>10} | {'Duration':>9} | {'Status':<9}"
        print(header)
        print("-" * 200)

        for event in events:
            timestamp_str = event.timestamp.strftime('%Y-%m-%d %H:%M:%S') if event.timestamp else ""
            job_str = str(event.job_id) if event.job_id >= 0 else ""
            comp_type = event.compaction_type_str[:11] if event.compaction_type_str else ""
            input_names = (event.input_files_str[:28] if len(event.input_files_str) > 28 else event.input_files_str) or "-"
            output_names = (event.output_files_str[:18] if len(event.output_files_str) > 18 else event.output_files_str) or "-"
            
            print(f"{timestamp_str:<19} | {job_str:>5} | {comp_type:<11} | {event.input_files_summary:<12} | {input_names:<28} | {event.input_size_str:>10} | {output_names:<18} | {event.output_size_str:>10} | {event.duration_str:>9} | {event.state.value:<9}")

        print("=" * 200)
    else:
        print("=" * 140)
        header = f"{'Start Time':<19} | {'Job':>5} | {'Type':<12} | {'Input Files':<16} | {'Input Size':>10} | {'Out Files':>10} | {'Output Size':>10} | {'Duration':>9} | {'Status':<9}"
        print(header)
        print("-" * 140)

        for event in events:
            timestamp_str = event.timestamp.strftime('%Y-%m-%d %H:%M:%S') if event.timestamp else ""
            job_str = str(event.job_id) if event.job_id >= 0 else ""
            out_files = str(event.output_files_count) if event.output_files_count > 0 else ""
            comp_type = event.compaction_type_str[:12] if event.compaction_type_str else ""
            
            print(f"{timestamp_str:<19} | {job_str:>5} | {comp_type:<12} | {event.input_files_summary:<16} | {event.input_size_str:>10} | {out_files:>10} | {event.output_size_str:>10} | {event.duration_str:>9} | {event.state.value:<9}")

        print("=" * 140)


def format_markdown_table(events: List[CompactionEvent], title: str = "Compaction Events", show_file_names: bool = True) -> None:
    """Output as Markdown table."""
    print(f"# {title}\n")
    
    if show_file_names:
        print("| Start Time | Job | Type | Input Files | Input File Names | Input Size | Output File Names | Output Size | Duration | Status |")
        print("|------|-----|------|-------------|------------------|------------|-------------------|-------------|----------|--------|")

        for event in events:
            timestamp_str = event.timestamp.strftime('%Y-%m-%d %H:%M:%S') if event.timestamp else ""
            job_str = str(event.job_id) if event.job_id >= 0 else ""
            input_names = event.input_files_str or "-"
            output_names = event.output_files_str or "-"
            status_icon = "✅" if event.state == CompactionState.COMPLETED else "❌" if event.state == CompactionState.FAILED else "🔄"
            
            print(f"| {timestamp_str} | {job_str} | {event.compaction_type_str} | {event.input_files_summary} | {input_names} | {event.input_size_str} | {output_names} | {event.output_size_str} | {event.duration_str} | {status_icon} {event.state.value} |")
    else:
        print("| Start Time | Job | Type | Input Files | Input Size | Output Files | Output Size | Duration | Status |")
        print("|------|-----|------|-------------|------------|--------------|-------------|----------|--------|")

        for event in events:
            timestamp_str = event.timestamp.strftime('%Y-%m-%d %H:%M:%S') if event.timestamp else ""
            job_str = str(event.job_id) if event.job_id >= 0 else ""
            out_files = str(event.output_files_count) if event.output_files_count > 0 else ""
            status_icon = "✅" if event.state == CompactionState.COMPLETED else "❌" if event.state == CompactionState.FAILED else "🔄"
            
            print(f"| {timestamp_str} | {job_str} | {event.compaction_type_str} | {event.input_files_summary} | {event.input_size_str} | {out_files} | {event.output_size_str} | {event.duration_str} | {status_icon} {event.state.value} |")


def format_json(events: List[CompactionEvent], summary: Dict[str, Any]) -> None:
    """Output as JSON."""
    output = {
        "events": [event.to_dict() for event in events],
        "summary": summary
    }
    print(json.dumps(output, indent=2, default=str))


def save_csv(events: List[CompactionEvent], filename: str = "compaction_events.csv") -> None:
    """Save results to CSV file."""
    import csv
    with open(filename, 'w', newline='') as csvfile:
        fieldnames = ["Start Time", "Job ID", "Type", "Input Files", "Input File Names", "Input Files Count", "Input Size", 
                      "Output File Names", "Output Files Count", "Output Size", "Duration", "Duration (s)", 
                      "Read MB/s", "Write MB/s", "Status", "Full Compaction", "Tablet ID"]
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        for event in events:
            writer.writerow({
                "Start Time": event.timestamp.isoformat() if event.timestamp else "",
                "Job ID": event.job_id if event.job_id >= 0 else "",
                "Type": event.compaction_type_str,
                "Input Files": event.input_files_summary,
                "Input File Names": event.input_files_str,
                "Input Files Count": event.input_files_count,
                "Input Size": event.input_size_str,
                "Output File Names": event.output_files_str,
                "Output Files Count": event.output_files_count,
                "Output Size": event.output_size_str,
                "Duration": event.duration_str,
                "Duration (s)": event.duration_seconds or "",
                "Read MB/s": event.read_mb_per_sec if event.read_mb_per_sec > 0 else "",
                "Write MB/s": event.write_mb_per_sec if event.write_mb_per_sec > 0 else "",
                "Status": event.state.value,
                "Full Compaction": "Yes" if event.is_full_compaction else "No",
                "Tablet ID": event.tablet_id,
            })
    print(f"\nResults saved to: {filename}", file=sys.stderr)


def print_summary(summary: Dict[str, Any]) -> None:
    """Print summary statistics."""
    if RICH_AVAILABLE:
        console = Console()
        console.print("\n[bold]Compaction Summary[/bold]")
        console.print(f"  Total Compactions: {summary['total_compactions']}")
        console.print(f"  [green]Completed[/green]: {summary['completed']}")
        console.print(f"  [red]Failed[/red]: {summary['failed']}")
        console.print(f"  [yellow]In Progress[/yellow]: {summary['in_progress']}")
        console.print(f"  Full Compactions: {summary['full_compactions']}")
        console.print(f"\n  Total Input: {summary['total_input_size']}")
        console.print(f"  Total Output: {summary['total_output_size']}")
        console.print(f"  Total Duration: {summary['total_duration_seconds']:.2f}s")
        
        if summary['errors']:
            console.print(f"\n  [bold red]Errors ({len(summary['errors'])}):[/bold red]")
            for err in summary['errors'][:5]:
                console.print(f"    - {err[:80]}...")
    else:
        print("\nCompaction Summary")
        print(f"  Total Compactions: {summary['total_compactions']}")
        print(f"  Completed: {summary['completed']}")
        print(f"  Failed: {summary['failed']}")
        print(f"  In Progress: {summary['in_progress']}")
        print(f"  Full Compactions: {summary['full_compactions']}")
        print(f"\n  Total Input: {summary['total_input_size']}")
        print(f"  Total Output: {summary['total_output_size']}")
        print(f"  Total Duration: {summary['total_duration_seconds']:.2f}s")


def main():
    parser = argparse.ArgumentParser(
        description='Parse YugabyteDB TServer logs and display compaction events for a specific tablet.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python compaction_log_parser.py -f tserver.log -t <tablet_uuid>
  python compaction_log_parser.py -f "tserver.INFO*" -t <tablet_uuid>
  python compaction_log_parser.py -f *.log *.gz -t <tablet_uuid>
  python compaction_log_parser.py -f "tserver.INFO.*.gz" -t <tablet_uuid>
  python compaction_log_parser.py -f tserver.log -t <tablet_uuid> --md
  python compaction_log_parser.py -f tserver.log -t <tablet_uuid> --json --summary
  cat tserver.log | python compaction_log_parser.py -t <tablet_uuid>

Note: The -t/--tablet option is required. Supports gzip compressed files (.gz).
      When using wildcards, quote the pattern to prevent shell expansion,
      or let the shell expand it (both work).
        """
    )
    parser.add_argument('-t', '--tablet', type=str, required=True,
                        help='Tablet UUID to filter compaction events (required)')
    parser.add_argument('-f', '--file', type=str, nargs='*',
                        help='TServer log file(s) to parse. Supports wildcards (e.g., -f *INFO*) and gzip files (.gz). Reads from stdin if not specified.')
    parser.add_argument('--md', '--markdown', action='store_true',
                        help='Output in Markdown table format')
    parser.add_argument('--json', action='store_true',
                        help='Output in JSON format')
    parser.add_argument('--plain', action='store_true',
                        help='Output in plain text table (no colors)')
    parser.add_argument('--no-csv', action='store_true',
                        help='Skip saving CSV file')
    parser.add_argument('--csv', type=str, default='compaction_events.csv',
                        help='CSV output filename (default: compaction_events.csv)')
    parser.add_argument('--summary', action='store_true',
                        help='Show summary statistics')
    parser.add_argument('--title', type=str, default='Compaction Events',
                        help='Table title')
    parser.add_argument('--year', type=int, default=datetime.now().year,
                        help='Year for glog timestamps (default: current year)')
    parser.add_argument('--no-file-names', action='store_true',
                        help='Hide input/output file names (show only counts)')

    args = parser.parse_args()

    # Read input
    import glob as glob_module
    lines = []
    files_processed = []
    
    if args.file:
        # Expand glob patterns and collect all matching files
        all_files = []
        for pattern in args.file:
            # Try glob expansion
            matches = glob_module.glob(pattern)
            if matches:
                all_files.extend(matches)
            elif os.path.exists(pattern):
                # If no glob match but file exists, use it directly
                all_files.append(pattern)
            else:
                print(f"Warning: No files matching '{pattern}'", file=sys.stderr)
        
        # Remove duplicates while preserving order
        seen = set()
        unique_files = []
        for f in all_files:
            if f not in seen:
                seen.add(f)
                unique_files.append(f)
        
        if not unique_files:
            print("Error: No valid input files found.", file=sys.stderr)
            sys.exit(1)
        
        # Sort files for consistent ordering
        unique_files.sort()
        
        print(f"Found {len(unique_files)} file(s) to process", file=sys.stderr)
        
        # Pre-filter optimization: if tablet ID specified, only keep lines containing it
        tablet_filter = args.tablet.lower() if args.tablet else None
        
        # Read all files with progress
        for i, filepath in enumerate(unique_files, 1):
            try:
                print(f"  [{i}/{len(unique_files)}] Reading {filepath}...", end='', file=sys.stderr, flush=True)
                # Check if file is gzip compressed
                if filepath.endswith('.gz'):
                    with gzip.open(filepath, 'rt', errors='replace') as f:
                        file_lines = f.readlines()
                else:
                    with open(filepath, 'r', errors='replace') as f:
                        file_lines = f.readlines()
                
                # Pre-filter lines if tablet ID specified (much faster than regex later)
                if tablet_filter:
                    file_lines = [l for l in file_lines if tablet_filter in l.lower()]
                
                lines.extend(file_lines)
                files_processed.append(filepath)
                print(f" {len(file_lines)} lines", file=sys.stderr)
            except FileNotFoundError:
                print(f" NOT FOUND", file=sys.stderr)
            except gzip.BadGzipFile:
                print(f" INVALID GZIP", file=sys.stderr)
            except IOError as e:
                print(f" ERROR: {e}", file=sys.stderr)
        
        print(f"Total: {len(lines)} relevant lines from {len(files_processed)} file(s)", file=sys.stderr)
    else:
        lines = sys.stdin.readlines()

    if not lines:
        print("No input provided. Use -f <file(s)> or pipe input via stdin.", file=sys.stderr)
        sys.exit(1)

    # Parse logs
    log_parser = CompactionLogParser(tablet_id=args.tablet)
    log_parser.current_year = args.year
    events = log_parser.parse(lines)

    if not events:
        tablet_msg = f" for tablet {args.tablet}" if args.tablet else ""
        print(f"No compaction events found{tablet_msg}.", file=sys.stderr)
        sys.exit(0)

    # Get summary
    summary = log_parser.get_summary()

    # Add tablet info to title if specified
    title = args.title
    if args.tablet:
        short_tablet = args.tablet[:8] + "..." if len(args.tablet) > 8 else args.tablet
        title = f"{args.title} (Tablet: {short_tablet})"

    # Determine if file names should be shown
    show_file_names = not getattr(args, 'no_file_names', False)

    # Output based on format
    if args.json:
        format_json(events, summary)
    elif args.md:
        format_markdown_table(events, title, show_file_names)
        if args.summary:
            print("\n## Summary")
            print(f"- Total Compactions: {summary['total_compactions']}")
            print(f"- Completed: {summary['completed']}")
            print(f"- Failed: {summary['failed']}")
            print(f"- Total Input: {summary['total_input_size']}")
            print(f"- Total Output: {summary['total_output_size']}")
    elif args.plain or not RICH_AVAILABLE:
        format_plain_table(events, title, show_file_names)
        if args.summary:
            print_summary(summary)
    else:
        format_rich_table(events, title, show_file_names)
        if args.summary:
            print_summary(summary)

    # Save CSV unless disabled
    if not args.no_csv and not args.json:
        save_csv(events, args.csv)


if __name__ == '__main__':
    main()
