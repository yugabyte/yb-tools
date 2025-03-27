# Run LNAV Command Script

This script is designed to process and analyze log files using the `lnav` tool. It provides various filtering options to narrow down the logs based on time, log types, nodes, and more. The script also supports rebuilding log metadata and generating commands for `lnav` to visualize the logs effectively.

> **Note:** On the Lincoln server, you can execute this script using the `run_lnav` bash function.

## Features

- Filter logs by:
    - Start and end time
    - Duration
    - Context time
    - Log types (PostgreSQL, TServer, Master, YB-Controller)
    - Nodes
- Automatically rebuild log metadata if required.
- Display included and excluded log files in a tabular format.
- Generate and execute `lnav` commands for log analysis.
- Debug mode for detailed logging.

## Prerequisites

- Python 3.x
- `lnav` installed on your system
- Required Python libraries:
    - `argparse`
    - `colorama`
    - `tabulate`
    - `json`
    - `re`
    - `datetime`
    - `itertools`
    - `threading`

## Usage

Run the script with the following options:

```bash
python run_lnav_command.py [OPTIONS]
```

### Options

| Option                  | Description                                                                                     |
|-------------------------|-------------------------------------------------------------------------------------------------|
| `-t, --from_time`       | Specify the start time in `MMDD HH:MM` format.                                                 |
| `-T, --to_time`         | Specify the end time in `MMDD HH:MM` format.                                                   |
| `-d, --duration`        | Specify the duration in minutes (e.g., `10m`, `2h`, `1d`).                                     |
| `-c, --context_time`    | Specify the context time in `MMDD HH:MM` format to hide lines before and after the context.    |
| `-A, --after_time`      | Specify the duration to hide lines after the context time (default: `5m`).                     |
| `-B, --before_time`     | Specify the duration to hide lines before the context time (default: `10m`).                   |
| `--types`               | Comma-separated list of log types to include (e.g., `pg,ts,ms,ybc`). Default: `ts,ms,pg`.      |
| `--nodes`               | Comma-separated list of nodes to include (e.g., `n1,n2`).                                      |
| `--rebuild`             | Rebuild the log files metadata.                                                                |
| `--debug`               | Enable debug mode for detailed logging.                                                        |

### Example Commands

1. Filter logs between specific times:
     ```bash
     python run_lnav_command.py --from_time "0923 14:00" --to_time "0923 16:00"
     ```

2. Filter logs for specific nodes and log types:
     ```bash
     python run_lnav_command.py --nodes n1,n2 --types pg,ts
     ```

3. Rebuild log metadata and analyze logs:
     ```bash
     python run_lnav_command.py --rebuild
     ```

4. Analyze logs with a specific context time:
     ```bash
     python run_lnav_command.py --context_time "0923 15:00"
     ```

## Output

- The script displays included and excluded log files in a tabular format.
- It generates a summary of the included files, including:
    - Total files
    - Start and end times
    - Log types
    - Nodes
- If logs are missing for specific nodes or types, a warning is displayed.
- The generated `lnav` command is printed and can be executed directly.

## Debugging

Enable debug mode using the `--debug` flag to view detailed logs of the script's execution.

## Notes

- Ensure that the `lnav` tool is installed and available in your system's PATH.
- The script automatically searches for `log_files_metadata.json` in the current or child directories. If not found, it rebuilds the metadata.