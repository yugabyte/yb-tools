#!/usr/bin/env python3
"""
YBA Task Log Parser

Parses YugabyteDB Anywhere (YBA) task logs and displays workflow in a structured table format.
Supports all task types (CreateXClusterConfig, CreateUniverse, CreateBackup, etc.)

Usage:
    python task_log_parser.py < yba_task_log.txt
    python task_log_parser.py --md < yba_task_log.txt        # Markdown output
    python task_log_parser.py --json < yba_task_log.txt      # JSON output
    python task_log_parser.py --no-csv < yba_task_log.txt    # Skip CSV generation
    python task_log_parser.py --summary < yba_task_log.txt   # Show summary statistics
    cat yba_task_log.txt | python task_log_parser.py -f file.log  # Read from file
"""

import re
import sys
import json
import argparse
from collections import defaultdict, OrderedDict
from datetime import datetime
from dataclasses import dataclass, field, asdict
from typing import Optional, List, Dict, Any
from enum import Enum

try:
    from rich.console import Console
    from rich.table import Table
    from rich.panel import Panel
    from rich.text import Text
    RICH_AVAILABLE = True
except ImportError:
    RICH_AVAILABLE = False


class TaskState(Enum):
    """Official YBA TaskInfo.State enum values."""
    CREATED = "Created"
    INITIALIZING = "Initializing"
    RUNNING = "Running"
    SUCCESS = "Success"
    FAILURE = "Failure"
    UNKNOWN = "Unknown"
    ABORT = "Abort"
    ABORTED = "Aborted"
    SKIPPED = "Skipped"  # Custom state for skipped tasks

    @classmethod
    def from_string(cls, state_str: str) -> "TaskState":
        """Convert string to TaskState, handling various formats."""
        if not state_str:
            return cls.UNKNOWN
        normalized = state_str.strip().title()
        for state in cls:
            if state.value.lower() == normalized.lower():
                return state
        return cls.UNKNOWN

    @property
    def is_error(self) -> bool:
        return self in (TaskState.FAILURE, TaskState.ABORTED, TaskState.ABORT)

    @property
    def is_completed(self) -> bool:
        return self in (TaskState.SUCCESS, TaskState.FAILURE, TaskState.ABORTED, TaskState.SKIPPED)

    @property
    def is_in_progress(self) -> bool:
        return self in (TaskState.CREATED, TaskState.INITIALIZING, TaskState.RUNNING, TaskState.ABORT)


@dataclass
class TaskStep:
    """Represents a single task/subtask step in the workflow."""
    name: str
    task_type: str = ""
    status: TaskState = TaskState.CREATED
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    universe_uuid: str = ""
    task_uuid: str = ""
    parent_uuid: str = ""
    position: int = -1
    subtask_group: str = ""
    error_message: str = ""
    raw_log_line: str = ""

    @property
    def duration_seconds(self) -> Optional[float]:
        """Calculate duration in seconds."""
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
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
    def display_name(self) -> str:
        """Get display name with context."""
        name = self.name
        if self.universe_uuid:
            # Truncate UUID for display
            short_uuid = self.universe_uuid[:8] if len(self.universe_uuid) > 8 else self.universe_uuid
            name = f"{name} [{short_uuid}...]"
        return name

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON/CSV output."""
        return {
            "name": self.name,
            "task_type": self.task_type,
            "status": self.status.value,
            "start_time": self.start_time.isoformat() if self.start_time else "",
            "end_time": self.end_time.isoformat() if self.end_time else "",
            "duration_seconds": self.duration_seconds,
            "duration": self.duration_str,
            "universe_uuid": self.universe_uuid,
            "task_uuid": self.task_uuid,
            "subtask_group": self.subtask_group,
            "error_message": self.error_message,
        }


class LogPatterns:
    """Compiled regex patterns for parsing YBA task logs."""

    # Timestamp patterns - YBA uses ISO 8601 format with 'YW' prefix
    TIMESTAMP_YW = re.compile(r'YW (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)')
    TIMESTAMP_ISO = re.compile(r'(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)')
    TIMESTAMP_STANDARD = re.compile(r'(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:,\d+)?)')

    # Task lifecycle patterns (from TaskExecutor.java and DefaultTaskExecutionListener.java)
    # "Adding task #N: TaskName(params)"
    ADD_TASK = re.compile(r'Adding task #(\d+):\s*([^(]+)(?:\(([^)]*)\))?')

    # "About to execute task taskType: X, taskState: Y"
    ABOUT_TO_EXECUTE = re.compile(r'About to execute task\s+(?:taskType\s*:\s*)?(\w+)(?:,\s*taskState\s*:\s*(\w+))?')

    # "Task taskType: X, taskState: Y is completed"
    TASK_COMPLETED = re.compile(r'Task\s+(?:taskType\s*:\s*)?(\w+)(?:,\s*taskState\s*:\s*(\w+))?\s+is completed')

    # "Skipping task taskType: X, taskState: Y beause it is set to not run"
    TASK_SKIPPED = re.compile(r'Skipping task\s+(?:taskType\s*:\s*)?(\w+)(?:,\s*taskState\s*:\s*(\w+))?\s+bea?cause')

    # "Failed to execute task type X UUID Y details Z"
    TASK_FAILED = re.compile(r'Failed to execute task type (\w+)\s+UUID\s+([a-f0-9-]+)\s+details\s*(.*)$', re.IGNORECASE)

    # "hit error : SubTaskGroup X ... failed."
    SUBTASK_GROUP_FAILED = re.compile(r'hit error\s*:\s*SubTaskGroup\s+(\S+).*failed', re.IGNORECASE)

    # "Adding SubTaskGroup #N: name"
    ADD_SUBTASK_GROUP = re.compile(r'Adding SubTaskGroup #(\d+):\s*(\S+)')

    # "Locked universe UUID"
    LOCKED_UNIVERSE = re.compile(r'Locked universe\s+([a-f0-9-]+)', re.IGNORECASE)

    # "Unlocking universe UUID"
    UNLOCKING_UNIVERSE = re.compile(r'Unlocking universe\s+([a-f0-9-]+)', re.IGNORECASE)

    # "Received X request"
    RECEIVED_REQUEST = re.compile(r'Received\s+(.+?)\s+request')

    # Task execution with TaskInfo toString format: "taskType: X, taskState: Y"
    TASK_INFO_FORMAT = re.compile(r'taskType:\s*(\w+),\s*taskState:\s*(\w+)')

    # Error patterns
    ERROR_PATTERN = re.compile(r'(?:ERROR|FATAL|Exception|Error)\s*[:\-]?\s*(.+)', re.IGNORECASE)
    EXCEPTION_PATTERN = re.compile(r'(\w+(?:Exception|Error)):\s*(.+)')

    # UUID pattern
    UUID_PATTERN = re.compile(r'[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}', re.IGNORECASE)

    # Completed task with elapsed time
    COMPLETED_WITH_TIME = re.compile(r'Completed task\s+(\w+)\s+in\s+(\d+)ms')

    # Task waited time
    TASK_WAITED = re.compile(r'Task\s+(\w+)\s+waited for\s+(\d+)ms')

    # One or more SubTaskGroups failed
    SUBTASK_GROUPS_FAILED = re.compile(r'One or more SubTaskGroups? failed', re.IGNORECASE)


class YBALogParser:
    """Parser for YBA task logs."""

    def __init__(self):
        self.steps: List[TaskStep] = []
        self.current_universe: Optional[str] = None
        self.current_task_uuid: Optional[str] = None
        self.subtask_groups: Dict[str, List[TaskStep]] = defaultdict(list)
        self.task_index: Dict[str, TaskStep] = {}  # For quick lookup by task type
        self.errors: List[str] = []
        self.warnings: List[str] = []

    def parse_timestamp(self, line: str) -> Optional[datetime]:
        """Extract timestamp from log line, supporting multiple formats."""
        # Try YW format first (most common in YBA logs)
        match = LogPatterns.TIMESTAMP_YW.search(line)
        if match:
            try:
                return datetime.strptime(match.group(1), '%Y-%m-%dT%H:%M:%S.%fZ')
            except ValueError:
                pass

        # Try standard ISO format
        match = LogPatterns.TIMESTAMP_ISO.search(line)
        if match:
            ts_str = match.group(1)
            for fmt in ['%Y-%m-%dT%H:%M:%S.%fZ', '%Y-%m-%dT%H:%M:%SZ', '%Y-%m-%dT%H:%M:%S.%f', '%Y-%m-%dT%H:%M:%S']:
                try:
                    return datetime.strptime(ts_str, fmt)
                except ValueError:
                    continue

        # Try standard log format
        match = LogPatterns.TIMESTAMP_STANDARD.search(line)
        if match:
            ts_str = match.group(1).replace(',', '.')
            for fmt in ['%Y-%m-%d %H:%M:%S.%f', '%Y-%m-%d %H:%M:%S']:
                try:
                    return datetime.strptime(ts_str, fmt)
                except ValueError:
                    continue

        return None

    def find_task_by_type(self, task_type: str, status_filter: List[TaskState] = None) -> Optional[TaskStep]:
        """Find the most recent task matching the type and optional status filter."""
        for step in reversed(self.steps):
            if step.task_type == task_type or step.name.startswith(task_type):
                if status_filter is None or step.status in status_filter:
                    return step
        return None

    def find_task_by_name_prefix(self, prefix: str, status_filter: List[TaskState] = None) -> Optional[TaskStep]:
        """Find the most recent task whose name starts with prefix."""
        for step in reversed(self.steps):
            if step.name.startswith(prefix) or step.task_type.startswith(prefix):
                if status_filter is None or step.status in status_filter:
                    return step
        return None

    def parse_line(self, line: str) -> None:
        """Parse a single log line and update state."""
        timestamp = self.parse_timestamp(line)

        # Check for universe lock/unlock
        lock_match = LogPatterns.LOCKED_UNIVERSE.search(line)
        if lock_match:
            self.current_universe = lock_match.group(1)
            step = TaskStep(
                name="Lock Universe",
                task_type="LockUniverse",
                status=TaskState.SUCCESS,
                start_time=timestamp,
                end_time=timestamp,
                universe_uuid=self.current_universe,
                raw_log_line=line
            )
            self.steps.append(step)
            return

        unlock_match = LogPatterns.UNLOCKING_UNIVERSE.search(line)
        if unlock_match:
            step = TaskStep(
                name="Unlock Universe",
                task_type="UnlockUniverse",
                status=TaskState.SUCCESS,
                start_time=timestamp,
                end_time=timestamp,
                universe_uuid=unlock_match.group(1),
                raw_log_line=line
            )
            self.steps.append(step)
            return

        # Check for received request
        request_match = LogPatterns.RECEIVED_REQUEST.search(line)
        if request_match:
            request_type = request_match.group(1).strip()
            step = TaskStep(
                name=f"Received {request_type} request",
                task_type="RequestReceived",
                status=TaskState.SUCCESS,
                start_time=timestamp,
                end_time=timestamp,
                raw_log_line=line
            )
            self.steps.append(step)
            return

        # Check for adding subtask group
        add_group_match = LogPatterns.ADD_SUBTASK_GROUP.search(line)
        if add_group_match:
            group_name = add_group_match.group(2)
            step = TaskStep(
                name=f"SubTaskGroup: {group_name}",
                task_type="SubTaskGroup",
                status=TaskState.CREATED,
                start_time=timestamp,
                subtask_group=group_name,
                universe_uuid=self.current_universe or "",
                raw_log_line=line
            )
            self.steps.append(step)
            self.subtask_groups[group_name].append(step)
            return

        # Check for adding task
        add_match = LogPatterns.ADD_TASK.search(line)
        if add_match:
            position = int(add_match.group(1))
            task_name = add_match.group(2).strip()
            params = add_match.group(3) or ""

            # Extract UUID from params if present
            task_uuid = ""
            uuid_match = LogPatterns.UUID_PATTERN.search(params)
            if uuid_match:
                task_uuid = uuid_match.group(0)

            step = TaskStep(
                name=task_name,
                task_type=task_name,
                status=TaskState.CREATED,
                start_time=timestamp,
                position=position,
                task_uuid=task_uuid,
                universe_uuid=self.current_universe or "",
                raw_log_line=line
            )
            self.steps.append(step)
            self.task_index[task_name] = step
            return

        # Check for about to execute
        about_match = LogPatterns.ABOUT_TO_EXECUTE.search(line)
        if about_match:
            task_type = about_match.group(1)
            state = about_match.group(2) if about_match.group(2) else "Running"

            step = self.find_task_by_type(task_type, [TaskState.CREATED, TaskState.INITIALIZING])
            if step:
                step.status = TaskState.RUNNING
                if timestamp:
                    step.start_time = timestamp
            else:
                # Create a new step if not found
                step = TaskStep(
                    name=task_type,
                    task_type=task_type,
                    status=TaskState.RUNNING,
                    start_time=timestamp,
                    universe_uuid=self.current_universe or "",
                    raw_log_line=line
                )
                self.steps.append(step)
            return

        # Check for task completed
        completed_match = LogPatterns.TASK_COMPLETED.search(line)
        if completed_match:
            task_type = completed_match.group(1)
            state_str = completed_match.group(2) if completed_match.group(2) else "Success"
            state = TaskState.from_string(state_str)

            step = self.find_task_by_type(task_type, [TaskState.CREATED, TaskState.INITIALIZING, TaskState.RUNNING])
            if step:
                step.status = state
                if timestamp:
                    step.end_time = timestamp
            return

        # Check for task skipped
        skipped_match = LogPatterns.TASK_SKIPPED.search(line)
        if skipped_match:
            task_type = skipped_match.group(1)

            step = self.find_task_by_type(task_type, [TaskState.CREATED, TaskState.INITIALIZING, TaskState.RUNNING])
            if step:
                step.status = TaskState.SKIPPED
                if timestamp:
                    step.end_time = timestamp
            return

        # Check for task failed
        failed_match = LogPatterns.TASK_FAILED.search(line)
        if failed_match:
            task_type = failed_match.group(1)
            task_uuid = failed_match.group(2)
            details = failed_match.group(3)

            step = self.find_task_by_type(task_type, [TaskState.CREATED, TaskState.INITIALIZING, TaskState.RUNNING])
            if step:
                step.status = TaskState.FAILURE
                step.task_uuid = task_uuid
                step.error_message = details[:200] if details else ""
                if timestamp:
                    step.end_time = timestamp
            self.errors.append(f"Task {task_type} failed: {details[:100] if details else 'Unknown error'}")
            return

        # Check for subtask group failed
        subtask_failed_match = LogPatterns.SUBTASK_GROUP_FAILED.search(line)
        if subtask_failed_match:
            group_name = subtask_failed_match.group(1)

            # Mark all tasks in this group as failed
            if group_name in self.subtask_groups:
                for step in self.subtask_groups[group_name]:
                    if step.status.is_in_progress:
                        step.status = TaskState.FAILURE
                        if timestamp:
                            step.end_time = timestamp

            # Also try to find by name prefix
            step = self.find_task_by_name_prefix(group_name, [TaskState.CREATED, TaskState.INITIALIZING, TaskState.RUNNING])
            if step:
                step.status = TaskState.FAILURE
                if timestamp:
                    step.end_time = timestamp
            self.errors.append(f"SubTaskGroup {group_name} failed")
            return

        # Check for completed with time
        completed_time_match = LogPatterns.COMPLETED_WITH_TIME.search(line)
        if completed_time_match:
            task_name = completed_time_match.group(1)
            elapsed_ms = int(completed_time_match.group(2))

            step = self.find_task_by_name_prefix(task_name)
            if step and step.start_time and not step.end_time:
                # Calculate end time from elapsed
                from datetime import timedelta
                step.end_time = step.start_time + timedelta(milliseconds=elapsed_ms)
                if step.status.is_in_progress:
                    step.status = TaskState.SUCCESS
            return

        # Check for general errors
        if 'ERROR' in line.upper() or 'Exception' in line:
            error_match = LogPatterns.EXCEPTION_PATTERN.search(line)
            if error_match:
                self.errors.append(f"{error_match.group(1)}: {error_match.group(2)[:100]}")

    def parse(self, lines: List[str]) -> List[TaskStep]:
        """Parse all log lines and return task steps."""
        for line in lines:
            line = line.rstrip('\n\r')
            if line:
                self.parse_line(line)

        # Post-process: mark any still-running tasks as unknown if no end time
        for step in self.steps:
            if step.status.is_in_progress and not step.end_time:
                # Leave as is - might be genuinely still running
                pass

        return self.steps

    def get_summary(self) -> Dict[str, Any]:
        """Generate summary statistics."""
        total = len(self.steps)
        by_status = defaultdict(int)
        total_duration = 0.0
        task_types = set()

        for step in self.steps:
            by_status[step.status.value] += 1
            if step.duration_seconds:
                total_duration += step.duration_seconds
            task_types.add(step.task_type)

        return {
            "total_steps": total,
            "by_status": dict(by_status),
            "total_duration_seconds": total_duration,
            "total_duration_str": TaskStep(name="", start_time=datetime.now(),
                                           end_time=datetime.now()).duration_str if total_duration == 0
                                   else f"{total_duration:.2f}s",
            "unique_task_types": len(task_types),
            "errors": self.errors,
            "warnings": self.warnings,
        }


def format_rich_table(steps: List[TaskStep], title: str = "YBA Task Workflow") -> None:
    """Output using rich library for terminal display."""
    if not RICH_AVAILABLE:
        print("Rich library not available. Install with: pip install rich")
        format_plain_table(steps, title)
        return

    console = Console()
    table = Table(title=title, show_lines=True, expand=True)

    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Step/Task Name", style="cyan", no_wrap=False, max_width=50)
    table.add_column("Status", style="bold", width=10, justify="center")
    table.add_column("Start Time", style="dim", width=24)
    table.add_column("End Time", style="dim", width=24)
    table.add_column("Duration", justify="right", width=12)

    for i, step in enumerate(steps, 1):
        status = step.status.value
        if step.status.is_error:
            status_style = "red bold"
        elif step.status == TaskState.SUCCESS:
            status_style = "green"
        elif step.status == TaskState.SKIPPED:
            status_style = "yellow"
        elif step.status.is_in_progress:
            status_style = "blue"
        else:
            status_style = ""

        start_str = step.start_time.strftime('%Y-%m-%d %H:%M:%S') if step.start_time else ""
        end_str = step.end_time.strftime('%Y-%m-%d %H:%M:%S') if step.end_time else ""

        table.add_row(
            str(i),
            step.display_name,
            f"[{status_style}]{status}[/{status_style}]" if status_style else status,
            start_str,
            end_str,
            step.duration_str
        )

    console.print(table)


def format_plain_table(steps: List[TaskStep], title: str = "YBA Task Workflow") -> None:
    """Output as plain text table."""
    print(f"\n{title}")
    print("=" * 120)
    header = f"{'#':>4} | {'Step/Task Name':<45} | {'Status':<10} | {'Start Time':<20} | {'End Time':<20} | {'Duration':>10}"
    print(header)
    print("-" * 120)

    for i, step in enumerate(steps, 1):
        start_str = step.start_time.strftime('%Y-%m-%d %H:%M:%S') if step.start_time else ""
        end_str = step.end_time.strftime('%Y-%m-%d %H:%M:%S') if step.end_time else ""
        name = step.display_name[:45]
        print(f"{i:>4} | {name:<45} | {step.status.value:<10} | {start_str:<20} | {end_str:<20} | {step.duration_str:>10}")

    print("=" * 120)


def format_markdown_table(steps: List[TaskStep], title: str = "YBA Task Workflow") -> None:
    """Output as Markdown table."""
    print(f"# {title}\n")
    print("| # | Step/Task Name | Status | Start Time | End Time | Duration |")
    print("|---|----------------|--------|------------|----------|----------|")

    for i, step in enumerate(steps, 1):
        start_str = step.start_time.strftime('%Y-%m-%dT%H:%M:%S') if step.start_time else ""
        end_str = step.end_time.strftime('%Y-%m-%dT%H:%M:%S') if step.end_time else ""
        status_icon = "✅" if step.status == TaskState.SUCCESS else "❌" if step.status.is_error else "⏭️" if step.status == TaskState.SKIPPED else "🔄"
        print(f"| {i} | {step.display_name} | {status_icon} {step.status.value} | {start_str} | {end_str} | {step.duration_str} |")


def format_json(steps: List[TaskStep], summary: Dict[str, Any]) -> None:
    """Output as JSON."""
    output = {
        "steps": [step.to_dict() for step in steps],
        "summary": summary
    }
    print(json.dumps(output, indent=2, default=str))


def save_csv(steps: List[TaskStep], filename: str = "task_workflow.csv") -> None:
    """Save results to CSV file."""
    import csv
    with open(filename, 'w', newline='') as csvfile:
        fieldnames = ["#", "Step/Task Name", "Task Type", "Status", "Start Time", "End Time",
                      "Duration (s)", "Duration", "Universe UUID", "Task UUID", "SubTask Group", "Error"]
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        for i, step in enumerate(steps, 1):
            writer.writerow({
                "#": i,
                "Step/Task Name": step.name,
                "Task Type": step.task_type,
                "Status": step.status.value,
                "Start Time": step.start_time.isoformat() if step.start_time else "",
                "End Time": step.end_time.isoformat() if step.end_time else "",
                "Duration (s)": step.duration_seconds or "",
                "Duration": step.duration_str,
                "Universe UUID": step.universe_uuid,
                "Task UUID": step.task_uuid,
                "SubTask Group": step.subtask_group,
                "Error": step.error_message
            })
    print(f"\nResults saved to: {filename}", file=sys.stderr)


def print_summary(summary: Dict[str, Any]) -> None:
    """Print summary statistics."""
    if RICH_AVAILABLE:
        console = Console()
        console.print("\n[bold]Summary Statistics[/bold]")
        console.print(f"  Total Steps: {summary['total_steps']}")
        console.print(f"  Unique Task Types: {summary['unique_task_types']}")
        console.print(f"  Total Duration: {summary['total_duration_seconds']:.2f}s")
        console.print("\n  [bold]Status Breakdown:[/bold]")
        for status, count in sorted(summary['by_status'].items()):
            color = "green" if status == "Success" else "red" if status in ("Failure", "Aborted") else "yellow"
            console.print(f"    [{color}]{status}[/{color}]: {count}")

        if summary['errors']:
            console.print(f"\n  [bold red]Errors ({len(summary['errors'])}):[/bold red]")
            for err in summary['errors'][:5]:  # Show first 5 errors
                console.print(f"    - {err[:80]}...")
    else:
        print("\nSummary Statistics")
        print(f"  Total Steps: {summary['total_steps']}")
        print(f"  Unique Task Types: {summary['unique_task_types']}")
        print(f"  Total Duration: {summary['total_duration_seconds']:.2f}s")
        print("\n  Status Breakdown:")
        for status, count in sorted(summary['by_status'].items()):
            print(f"    {status}: {count}")

        if summary['errors']:
            print(f"\n  Errors ({len(summary['errors'])}):")
            for err in summary['errors'][:5]:
                print(f"    - {err[:80]}...")


def main():
    parser = argparse.ArgumentParser(
        description='Parse YBA task logs and display workflow in table format.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python task_log_parser.py < yba_task_log.txt
  python task_log_parser.py --md < yba_task_log.txt
  python task_log_parser.py -f application.log --json
  cat yba_task_log.txt | python task_log_parser.py --summary
        """
    )
    parser.add_argument('--md', '--markdown', action='store_true',
                        help='Output in Markdown table format')
    parser.add_argument('--json', action='store_true',
                        help='Output in JSON format')
    parser.add_argument('--plain', action='store_true',
                        help='Output in plain text table (no colors)')
    parser.add_argument('--no-csv', action='store_true',
                        help='Skip saving CSV file')
    parser.add_argument('--csv', type=str, default='task_workflow.csv',
                        help='CSV output filename (default: task_workflow.csv)')
    parser.add_argument('--summary', action='store_true',
                        help='Show summary statistics')
    parser.add_argument('-f', '--file', type=str,
                        help='Read from file instead of stdin')
    parser.add_argument('--title', type=str, default='YBA Task Workflow',
                        help='Table title')

    args = parser.parse_args()

    # Read input
    if args.file:
        try:
            with open(args.file, 'r') as f:
                lines = f.readlines()
        except FileNotFoundError:
            print(f"Error: File not found: {args.file}", file=sys.stderr)
            sys.exit(1)
        except IOError as e:
            print(f"Error reading file: {e}", file=sys.stderr)
            sys.exit(1)
    else:
        lines = sys.stdin.readlines()

    if not lines:
        print("No input provided. Use -f <file> or pipe input via stdin.", file=sys.stderr)
        sys.exit(1)

    # Parse logs
    log_parser = YBALogParser()
    steps = log_parser.parse(lines)

    if not steps:
        print("No task steps found in the log.", file=sys.stderr)
        sys.exit(0)

    # Sort steps by start time
    steps = sorted(steps, key=lambda x: (x.start_time or datetime.min, x.position))

    # Get summary
    summary = log_parser.get_summary()

    # Output based on format
    if args.json:
        format_json(steps, summary)
    elif args.md:
        format_markdown_table(steps, args.title)
        if args.summary:
            print("\n## Summary")
            print(f"- Total Steps: {summary['total_steps']}")
            print(f"- Success: {summary['by_status'].get('Success', 0)}")
            print(f"- Failed: {summary['by_status'].get('Failure', 0) + summary['by_status'].get('Aborted', 0)}")
    elif args.plain or not RICH_AVAILABLE:
        format_plain_table(steps, args.title)
        if args.summary:
            print_summary(summary)
    else:
        format_rich_table(steps, args.title)
        if args.summary:
            print_summary(summary)

    # Save CSV unless disabled
    if not args.no_csv and not args.json:
        save_csv(steps, args.csv)


if __name__ == '__main__':
    main()
