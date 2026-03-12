import argparse
import gzip
import datetime
import os
import logging
import tabulate
from collections import deque
import json
import re
import colorama
import itertools
import threading
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

parser = argparse.ArgumentParser(
    formatter_class=argparse.RawTextHelpFormatter,
    description=colorama.Fore.CYAN + "Run LNAV Command Script" + colorama.Style.RESET_ALL + """
    
Sample Commands:
1. Filter logs by time range:
   python run_lnav_command.py -t "0923 14:00" -T "0923 15:00"
2. Filter logs by nodes:
   python run_lnav_command.py --nodes n1,n2,n3
3. Filter logs by log types:
   python run_lnav_command.py --types pg,ts
4. Combine filters:
   python run_lnav_command.py -t "0923 14:00" -T "0923 15:00" --nodes n1,n2 --types pg,ms
"""
)
parser.add_argument(
    "-t", "--from_time", metavar="MMDD HH:MM", dest="start_time",
    help=colorama.Fore.YELLOW + "Specify start time in quotes" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "-T", "--to_time", metavar="MMDD HH:MM", dest="end_time",
    help=colorama.Fore.YELLOW + "Specify end time in quotes" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "-d", "--duration", metavar="DURATION",
    help=colorama.Fore.YELLOW + "Specify duration in minutes. Eg: --duration 10m" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "-c", "--context_time", metavar="MMDD HH:MM",
    help=colorama.Fore.YELLOW + "Specify context time in quotes to hide lines before and after the context time" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "-A", "--after_time", metavar="DURATION", default='5m',
    help=colorama.Fore.YELLOW + "Specify duration in minutes to hide lines after the context time. Default is 5 minutes if context time is specified" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "-B", "--before_time", metavar="DURATION", default='10m',
    help=colorama.Fore.YELLOW + "Specify duration in minutes to hide lines before the context time. Default is 10 minutes if context time is specified" + colorama.Style.RESET_ALL
)
parser.add_argument(
    '--types', metavar='LIST',
    help=colorama.Fore.YELLOW + """Comma separated list of log types to include. 
Available types: 
    pg (PostgreSQL), 
    ts (TServer), 
    ms (Master) 
    ybc (YB-Controller) 
Eg: --types pg,ts 
Default is ts,ms,pg""" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "--nodes", metavar="LIST",
    help=colorama.Fore.YELLOW + "Comma separated list of nodes to include. Eg: --nodes n1,n2" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "--rebuild", action="store_true",
    help=colorama.Fore.YELLOW + "Rebuild the log files metadata" + colorama.Style.RESET_ALL
)
parser.add_argument(
    "--debug", action="store_true",
    help=colorama.Fore.YELLOW + "Print debug messages" + colorama.Style.RESET_ALL
)

_NODE_NAME_RE = re.compile(r"/(yb-[^/]*n\d+)/")
_GLOG_PREFIXES = frozenset('IWEF')
_CURRENT_YEAR = datetime.datetime.now().year


def spinner():
    for c in itertools.cycle(['|', '/', '-', '\\']):
        if done:
            break
        sys.stdout.write('\r Building the one time log file metadata' + c)
        sys.stdout.flush()
        time.sleep(0.1)
    sys.stdout.write('\rDone!     \n')

def parse_duration(duration):
    if not duration:
        return None
    try:
        if duration[-1] == 'm':
            return datetime.timedelta(minutes=int(duration[:-1]))
        elif duration[-1] == 'h':
            return datetime.timedelta(hours=int(duration[:-1]))
        elif duration[-1] == 'd':
            return datetime.timedelta(days=int(duration[:-1]))
        else:
            raise ValueError
    except ValueError:
        raise ValueError("Invalid duration format. Use 'm' for minutes, 'h' for hours, and 'd' for days")

def getStartAndEndTimes():
    if args.start_time and args.context_time:
        raise ValueError("Cannot specify both start time and context time")
    if args.end_time and args.duration:
        raise ValueError("Cannot specify both end time and duration")
    if args.end_time and args.context_time:
        raise ValueError("Cannot specify both end time and context time")
    if args.context_time and args.duration:
        raise ValueError("Cannot specify both context time and duration")

    if args.start_time and args.end_time:
        start_time = datetime.datetime.strptime(args.start_time, '%m%d %H:%M')
        end_time = datetime.datetime.strptime(args.end_time, '%m%d %H:%M')
    elif args.start_time and args.duration:
        start_time = datetime.datetime.strptime(args.start_time, '%m%d %H:%M')
        end_time = start_time + parse_duration(args.duration)
    elif args.context_time:
        context_time = datetime.datetime.strptime(args.context_time, '%m%d %H:%M')
        start_time = context_time - parse_duration(args.before_time)
        end_time = context_time + parse_duration(args.after_time)
    elif args.start_time and not args.end_time and not args.duration:
        start_time = datetime.datetime.strptime(args.start_time, '%m%d %H:%M')
        end_time = datetime.datetime.strptime('1231 23:59', '%m%d %H:%M')
    elif not args.start_time and args.end_time and not args.duration:
        start_time = datetime.datetime.strptime('0101 00:00', '%m%d %H:%M')
        end_time = datetime.datetime.strptime(args.end_time, '%m%d %H:%M')
    else:
        start_time = datetime.datetime.strptime('0101 00:00', '%m%d %H:%M')
        end_time = datetime.datetime.strptime('1231 23:59', '%m%d %H:%M')
    return start_time, end_time

def getLogFilesFromCurrentDir():
    logFiles = []
    logDirectory = os.getcwd()
    for root, dirs, files in os.walk(logDirectory):
        for file in files:
            if "log" in file or "postgres" in file or "controller" in file and file[0] != ".":
                logFiles.append(os.path.join(root, file))
    return logFiles

def getTimeFromLog(line):
    try:
        if line and line[0] in _GLOG_PREFIXES:
            parts = line.split(' ', 2)
            timeFromLogLine = parts[0][1:] + ' ' + parts[1][:8]
            timestamp = datetime.datetime.strptime(timeFromLogLine, '%m%d %H:%M:%S')
            timestamp = timestamp.replace(year=_CURRENT_YEAR)
        else:
            parts = line.split(' ', 2)
            timeFromLogLine = parts[0] + ' ' + parts[1].split('.')[0]
            timestamp = datetime.datetime.strptime(timeFromLogLine, '%Y-%m-%d %H:%M:%S')
        return timestamp
    except Exception as e:
        raise ValueError(f"Error parsing timestamp from log line: {line} - {e}")


def _read_tail(filepath, num_lines=10, chunk_size=65536):
    """Read last num_lines from a regular file using seek instead of reading the whole file."""
    try:
        with open(filepath, 'rb') as f:
            f.seek(0, 2)
            size = f.tell()
            if size == 0:
                return []
            read_size = min(chunk_size, size)
            f.seek(-read_size, 2)
            data = f.read()
        return data.decode('utf-8', errors='replace').splitlines()[-num_lines:]
    except Exception:
        return []


def getFileMetadata(logFile):
    logStartsAt, logEndsAt = None, None
    is_gz = logFile.endswith('.gz')

    try:
        if is_gz:
            with gzip.open(logFile, 'rt', errors='replace') as f:
                for _ in range(10):
                    line = f.readline()
                    if not line:
                        break
                    try:
                        logStartsAt = getTimeFromLog(line)
                        break
                    except ValueError:
                        continue
                last_lines = deque(f, maxlen=10)
                for line in reversed(last_lines):
                    try:
                        logEndsAt = getTimeFromLog(line)
                        break
                    except ValueError:
                        continue
        else:
            with open(logFile, 'r', errors='replace') as f:
                for _ in range(10):
                    line = f.readline()
                    if not line:
                        break
                    try:
                        logStartsAt = getTimeFromLog(line)
                        break
                    except ValueError:
                        continue
            for line in reversed(_read_tail(logFile)):
                try:
                    logEndsAt = getTimeFromLog(line)
                    break
                except ValueError:
                    continue
    except Exception as e:
        print(f"Error processing file: {logFile} - {e}")
        return None

    if not logStartsAt:
        logStartsAt = datetime.datetime.strptime('0101 00:00', '%m%d %H:%M')
    if not logEndsAt:
        logEndsAt = datetime.datetime.strptime('1231 23:59', '%m%d %H:%M')
    try:
        logStartsAt = logStartsAt.replace(year=_CURRENT_YEAR)
        logEndsAt = logEndsAt.replace(year=_CURRENT_YEAR)
    except Exception as e:
        print("Error getting metadata for file: " + logFile + " " + str(e))

    if "postgres" in logFile:
        logType = "postgres"
    elif "controller" in logFile:
        logType = "yb-controller"
    elif "tserver" in logFile:
        logType = "yb-tserver"
    elif "master" in logFile:
        logType = "yb-master"
    else:
        logType = "unknown"

    match = _NODE_NAME_RE.search(logFile)
    nodeName = match.group(1) if match else "unknown"

    logging.getLogger(__name__).debug(
        f"Metadata for file: {logFile} - {logStartsAt} - {logEndsAt} - {logType} - {nodeName}"
    )
    return {"logStartsAt": logStartsAt, "logEndsAt": logEndsAt, "logType": logType, "nodeName": nodeName}

def filterLogFilesByType(logFileList, logFileMetadata, types):
    filteredLogFiles = []
    removedLogFiles = []
    type_map = {"pg": "postgres", "ts": "yb-tserver", "ms": "yb-master", "ybc": "yb-controller"}

    selectedTypes = frozenset(type_map[t] for t in types.split(",") if t in type_map)
    for logFile in logFileList:
        if logFileMetadata[logFile]["logType"] in selectedTypes:
            filteredLogFiles.append(logFile)
        else:
            removedLogFiles.append(logFile)
    filteredLogFiles = [logFile for logFile in filteredLogFiles if not logFile.startswith('.')]
    logging.getLogger(__name__).debug(f"Included files by type: {filteredLogFiles}")
    logging.getLogger(__name__).debug(f"Removed files by type: {removedLogFiles}")
    return filteredLogFiles, removedLogFiles

def filterLogFilesByTime(logFileList, logFileMetadata, start_time, end_time):
    filtered_files = []
    removed_files = []
    for logFile in logFileList:
        log_start = datetime.datetime.strptime(logFileMetadata[logFile]["logStartsAt"], '%Y-%m-%d %H:%M:%S')
        log_end = datetime.datetime.strptime(logFileMetadata[logFile]["logEndsAt"], '%Y-%m-%d %H:%M:%S')
        if log_start >= end_time or log_end <= start_time:
            removed_files.append(logFile)
        else:
            filtered_files.append(logFile)
    logging.getLogger(__name__).debug(f"Included files by time: {filtered_files}")
    logging.getLogger(__name__).debug(f"Removed files by time: {removed_files}")
    return filtered_files, removed_files

def filterLogFilesByNode(logFileList, logFileMetadata, nodes):
    filtered_files = []
    removed_files = []
    nodes = nodes.split(",")
    for logFile in logFileList:
        if any(node in logFileMetadata[logFile]["nodeName"] for node in nodes):
            filtered_files.append(logFile)
        else:
            removed_files.append(logFile)
    return filtered_files, removed_files

if __name__ == "__main__":
    args = parser.parse_args()
    logger = logging.getLogger(__name__)
    log_format = '%(asctime)s - %(name)s - %(levelname)s - %(funcName)s - %(message)s'
    logging.basicConfig(level=logging.DEBUG if args.debug else logging.INFO, format=log_format)
    if args.debug:
        logger.debug('Debug mode enabled')

    choosenTypes = args.types.split(",") if args.types else ["pg", "ts", "ms", "ybc"]

    start_time, end_time = getStartAndEndTimes()
    logFilesMetadataFile = 'log_files_metadata.json'
    if not os.path.isfile(logFilesMetadataFile):
        for root, dirs, files in os.walk(os.getcwd()):
            if logFilesMetadataFile in files:
                user_input = input(f"Found log_files_metadata.json in {root}. Do you want to use this file? (y/n): ")
                if user_input.lower() == 'y':
                    logFilesMetadataFile = os.path.join(root, logFilesMetadataFile)
                    break
    done = False
    spinner_thread = threading.Thread(target=spinner)
    spinner_thread.start()
    if args.rebuild or not os.path.isfile(logFilesMetadataFile):
        logFileList = getLogFilesFromCurrentDir()
        logFilesMetadata = {}
        with ThreadPoolExecutor(max_workers=min(32, (os.cpu_count() or 4) + 4)) as executor:
            futures = {executor.submit(getFileMetadata, f): f for f in logFileList}
            for future in as_completed(futures):
                logFile = futures[future]
                try:
                    metadata = future.result()
                    if metadata:
                        logFilesMetadata[logFile] = metadata
                except Exception as e:
                    print(f"Error processing {logFile}: {e}")
        with open(logFilesMetadataFile, 'w') as f:
            json.dump(logFilesMetadata, f, default=str)
    with open(logFilesMetadataFile, 'r') as f:
        logFilesMetadata = json.load(f)
    logFilesToProcess = list(logFilesMetadata.keys())
    done = True
    spinner_thread.join()

    logger.debug(f"Filtering by nodes: {args.nodes}")
    if args.nodes:
        logFilesToProcess, _ = filterLogFilesByNode(logFilesToProcess, logFilesMetadata, args.nodes)

    logger.debug(f"Filtering by types: {args.types}")
    if args.types:
        logFilesToProcess, _ = filterLogFilesByType(logFilesToProcess, logFilesMetadata, args.types)

    if start_time:
        start_time = start_time.replace(year=datetime.datetime.now().year)
        end_time = end_time.replace(year=datetime.datetime.now().year)
        logger.debug(f"Filtering by time: {start_time} - {end_time}")
        logFilesToProcess, _ = filterLogFilesByTime(logFilesToProcess, logFilesMetadata, start_time, end_time)

    logFilesRemoved = set(logFilesMetadata.keys()) - set(logFilesToProcess)

    table = []
    if args.debug:
        print('====================Skipped Files====================')
        for file in logFilesRemoved:
            table.append([file[-100:], logFilesMetadata[file]["logStartsAt"], logFilesMetadata[file]["logEndsAt"], logFilesMetadata[file]["logType"], logFilesMetadata[file]["nodeName"]])
        print(tabulate.tabulate(table, headers=["File", "Start Time", "End Time", "Type", "Node Name"], tablefmt="simple_grid"))
        print('====================Included Files====================')

    table = []
    for file in logFilesToProcess:
        table.append([file[-100:], logFilesMetadata[file]["logStartsAt"], logFilesMetadata[file]["logEndsAt"], logFilesMetadata[file]["logType"], logFilesMetadata[file]["nodeName"]])
    table.sort(key=lambda x: (x[4], x[1]))
    print(tabulate.tabulate(table, headers=["File", "Start Time", "End Time", "Type", "Node Name"], tablefmt="simple_grid"))

    print("==========Summary of Included Files===========")
    print(f"Total Files: {len(logFilesMetadata)} - Included: {len(logFilesToProcess)}")
    print(f"Start Time: {start_time}")
    print(f"End Time: {end_time}")
    logTypes = sorted(set(logFilesMetadata[file]["logType"] for file in logFilesToProcess))
    print(f"Log Types: {', '.join(logTypes)}")
    nodes = sorted(set(logFilesMetadata[file]["nodeName"] for file in logFilesToProcess))
    print(f"Nodes: {', '.join(nodes)}")

    colorama.init(autoreset=True)
    isLogMissing = False

    type_to_nodes = {}
    for f in logFilesToProcess:
        lt = logFilesMetadata[f]["logType"]
        nn = logFilesMetadata[f]["nodeName"]
        type_to_nodes.setdefault(lt, set()).add(nn)

    nodes_set = set(nodes)
    type_checks = [
        ("pg", "postgres", "Postgres"),
        ("ts", "yb-tserver", "TServer"),
        ("ms", "yb-master", "Master"),
        ("ybc", "yb-controller", "Controller"),
    ]
    for type_key, log_type, label in type_checks:
        if type_key in choosenTypes:
            present_nodes = type_to_nodes.get(log_type, set())
            missingNodes = sorted(nodes_set - present_nodes)
            if missingNodes:
                print(colorama.Fore.RED + f"{label} logs missing for nodes: {', '.join(missingNodes)}")
                isLogMissing = True

    if isLogMissing:
        print(colorama.Fore.YELLOW + "WARNING: If missing logs are reported and if it is suspicious, please check the logs manually.")

    Command = []
    print('====================Command====================')
    if len(logFilesToProcess) > 0:
        mainCommand = 'lnav'
        if start_time:
            mainCommand += f" -c ':hide-lines-before {start_time.strftime('%Y-%m-%d %H:%M:%S')}'"
        if end_time:
            mainCommand += f" -c ':hide-lines-after {end_time.strftime('%Y-%m-%d %H:%M:%S')}'"
        if args.context_time:
            context_time = datetime.datetime.strptime(args.context_time, '%m%d %H:%M').replace(year=datetime.datetime.now().year)
            mainCommand += f" -c ':goto {context_time.strftime('%Y-%m-%d %H:%M:%S')}'"
        Command.append(mainCommand)
        Command.extend(logFilesToProcess)
        print(colorama.Fore.GREEN + ' '.join(Command)[:1000] + "...[truncated]")
        try:
            print('')
            input("Press Enter to run the command or Ctrl+C to exit")
            os.system(' '.join(Command))
        except KeyboardInterrupt:
            print("Exiting...")
            exit(0)
    else:
        print("No files to process")
