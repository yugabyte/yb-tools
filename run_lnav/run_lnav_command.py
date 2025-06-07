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
    '--types', metavar='LIST', default='ts,ms,pg',
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
    "--subtypes", metavar="LIST", default="INFO",
    help=colorama.Fore.YELLOW + "Comma separated list of log subtypes to include. Eg: --subtypes INFO,WARN,ERROR,FATAL" + colorama.Style.RESET_ALL
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
    "--full_command", action="store_true",
    help=colorama.Fore.YELLOW + "Print the full command to run" + colorama.Style.RESET_ALL
)

parser.add_argument(
    "--debug", action="store_true",
    help=colorama.Fore.YELLOW + "Print debug messages" + colorama.Style.RESET_ALL
)

parser.add_argument(
    "--files_only", action="store_true",
    help=colorama.Fore.YELLOW + "Print only the filtered file paths, separated by spaces, and exit" + colorama.Style.RESET_ALL
)

# Function to display the rotating spinner
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
    # Validate arguments
    if args.start_time and args.context_time:
        raise ValueError("Cannot specify both start time and context time")
    if args.end_time and args.duration:
        raise ValueError("Cannot specify both end time and duration")
    if args.end_time and args.context_time:
        raise ValueError("Cannot specify both end time and context time")
    if args.context_time and args.duration:
        raise ValueError("Cannot specify both context time and duration")

    # Calculate start and end times
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
            if file.__contains__("log") and file[0] != ".":
                logFiles.append(os.path.join(root, file))
    return logFiles

def getTimeFromLog(line):
    try:
        # Check for glog format (e.g., I0923 14:23:45.123456 12345 file.cc:123] log message)
        if line[0] in ['I', 'W', 'E', 'F']:
            timeFromLogLine = line.split(' ')[0][1:] + ' ' + line.split(' ')[1][:8]
            timestamp = datetime.datetime.strptime(timeFromLogLine, '%m%d %H:%M:%S')
            timestamp = timestamp.replace(year=datetime.datetime.now().year)
        # Check for PostgreSQL log format (e.g., 2023-09-23 14:23:45.123 UTC [12345] LOG:  log message)
        else:
            timeFromLogLine = ' '.join(line.split(' ')[:2])
            timeFromLogLine = timeFromLogLine.split('.')[0]
            timestamp = datetime.datetime.strptime(timeFromLogLine, '%Y-%m-%d %H:%M:%S')
        return timestamp
    except Exception as e:
        raise ValueError(f"Error parsing timestamp from log line: {line} - {e}")

def getFileMetadata(logFile):
    logStartsAt, logEndsAt = None, None
    if logFile.endswith('.gz'):
        try:
            logs = gzip.open(logFile, 'rt')
        except:
            print("Error opening file: " + logFile)
            return None
    else:
        try:
            logs = open(logFile, 'r')
        except:
            print("Error opening file: " + logFile)
            return None
    try:
        # Read first 10 lines to get the start time
        for i in range(10):
            line = logs.readline()
            try:
                logStartsAt = getTimeFromLog(line)
                break
            except ValueError:
                continue
        # Read last 10 lines to get the end time
        last_lines = deque(logs, maxlen=10)
        for line in reversed(last_lines):
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
        logStartsAt = logStartsAt.replace(year=datetime.datetime.now().year)
        logEndsAt = logEndsAt.replace(year=datetime.datetime.now().year)
    except Exception as e:
        print("Error getting metadata for file: " + logFile + " " + str(e))
    
    # Get the log type
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
        
    # Get the subtype if available
    if "INFO" in logFile:
        subType = "INFO"
    elif "WARN" in logFile:
        subType = "WARN"
    elif "ERROR" in logFile:
        subType = "ERROR"
    elif "FATAL" in logFile:
        subType = "FATAL"
    elif "postgres" in logFile:
        subType = "postgres"
    else:
        subType = "unknown"
        
        
    # Get the node name
    # /Users/pgyogesh/logs/log_analyzer_tests/yb-support-bundle-ybu-p01-bpay-20240412151237.872-logs/yb-prod-ybu-p01-bpay-n8/master/logs/yb-master.danpvvy00002.yugabyte.log.INFO.20230521-030902.3601
    nodeNameRegex = r"/(yb-[^/]*n\d+|yb-(master|tserver)-\d+_[^/]+)/"
    nodeName = re.search(nodeNameRegex, logFile)
    if nodeName:
        nodeName = nodeName.group().replace("/","")
    else:
        nodeName = "unknown"
    
    logger.debug(f"Metadata for file: {logFile} - {logStartsAt} - {logEndsAt} - {logType} - {nodeName} - {subType}")
    return {"logStartsAt": logStartsAt, "logEndsAt": logEndsAt, "logType": logType, "nodeName": nodeName , "subType": subType}

def filterLogFilesByType(logFileList, logFileMetadata, types, subtypes):
    filteredLogFiles = []
    removedLogFiles = []
    type_map = {"pg": "postgres", "ts": "yb-tserver", "ms": "yb-master", "ybc": "yb-controller"}
    # Support both list and string for types
    if isinstance(types, list):
        type_keys = types
    else:
        type_keys = types.split(",")
    selectedTypes = [type_map[t] for t in type_keys if t in type_map]
    # Support both list and string for subtypes
    if isinstance(subtypes, list):
        subtype_keys = subtypes
    else:
        subtype_keys = subtypes.split(",")
    for logFile in logFileList:
        if (logFileMetadata[logFile]["logType"] in selectedTypes and
            logFileMetadata[logFile]["subType"] in subtype_keys):
            filteredLogFiles.append(logFile)
        else:
            removedLogFiles.append(logFile)
    # Filter hidden files
    filteredLogFiles = [logFile for logFile in filteredLogFiles if not logFile.startswith('.')]
    logger.debug(f"Included files by type and subtype: {filteredLogFiles}")
    logger.debug(f"Removed files by type and subtype: {removedLogFiles}")
    return filteredLogFiles, removedLogFiles

def filterLogFilesByTime(logFileList, logFileMetadata, start_time, end_time):
    filtered_files = []
    removed_files = []
    for logFile in logFileList:
        # Get the start and end time of the log file in datetime format
        log_start = datetime.datetime.strptime(logFileMetadata[logFile]["logStartsAt"], '%Y-%m-%d %H:%M:%S')
        log_end = datetime.datetime.strptime(logFileMetadata[logFile]["logEndsAt"], '%Y-%m-%d %H:%M:%S')
        if log_start >= end_time or log_end <= start_time:
            removed_files.append(logFile)
        else:
            filtered_files.append(logFile)
    logger.debug(f"Included files by time: {filtered_files}")
    logger.debug(f"Removed files by time: {removed_files}")
    return filtered_files, removed_files
    
def filterLogFilesByNode(logFileList, logFileMetadata, nodes):
    # Filter files containing the selected nodes
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
    
    choosenTypes = args.types.split(",")
    choosenSubtypes = args.subtypes.split(",")
    
    start_time, end_time = getStartAndEndTimes()
    logFilesMetadataFile = 'log_files_metadata.json'
    if args.files_only and not os.path.isfile(logFilesMetadataFile):
        print("log_files_metadata.json not found. Please build metadata first.", file=sys.stderr)
        sys.exit(1)
    if not os.path.isfile(logFilesMetadataFile):
        # Search for log_files_metadata.json in the child directories
        for root, dirs, files in os.walk(os.getcwd()):
            if logFilesMetadataFile in files:
                user_input = input(f"Found log_files_metadata.json in {root}. Do you want to use this file? (y/n): ")
                if user_input.lower() == 'y':
                    logFilesMetadataFile = os.path.join(root, logFilesMetadataFile)
                    break
    # Only start spinner if not files_only
    if not args.files_only:
        done = False
        spinner_thread = threading.Thread(target=spinner)
        spinner_thread.start()
    else:
        done = True
    if args.rebuild or not os.path.isfile(logFilesMetadataFile):
        logFileList = getLogFilesFromCurrentDir()
        logFilesMetadata = {}
        for logFile in logFileList:
            metadata = getFileMetadata(logFile)
            if metadata:
                # Add metadata to the dictionary
                logFilesMetadata[logFile] = metadata
        # Save the metadata to a file
        with open(logFilesMetadataFile, 'w') as f:
            json.dump(logFilesMetadata, f, default=str)
    with open(logFilesMetadataFile, 'r') as f:
        logFilesMetadata = json.load(f)
    # Get the list of log files to process based on arguments
    logFilesToProcess = list(logFilesMetadata.keys())
    # Only join spinner thread if it was started
    if 'spinner_thread' in locals():
        done = True
        spinner_thread.join()
    
    # Filter by nodes
    logger.debug(f"Filtering by nodes: {args.nodes}")
    if args.nodes:
        filteredFiles, removedFiles = filterLogFilesByNode(logFilesToProcess, logFilesMetadata, args.nodes)
        logFilesToProcess = [file for file in logFilesToProcess if file not in removedFiles]
    
    # Filter by log types
    logger.debug(f"Filtering by types: {args.types} with subtypes: {args.subtypes}")
    if args.types:
        filteredFiles, removedFiles = filterLogFilesByType(logFilesToProcess, logFilesMetadata, choosenTypes, choosenSubtypes)
        logFilesToProcess = [file for file in logFilesToProcess if file not in removedFiles]
    
    # Filter by start and end time
    if start_time:
        start_time = start_time.replace(year=datetime.datetime.now().year)
        end_time = end_time.replace(year=datetime.datetime.now().year)
        logger.debug(f"Filtering by time: {start_time} - {end_time}")
        filteredFiles, removedFiles = filterLogFilesByTime(logFilesToProcess, logFilesMetadata, start_time, end_time)
        logFilesToProcess = [file for file in logFilesToProcess if file not in removedFiles]

    # Move --files_only check here, before any print statements
    if args.files_only:
        print(' '.join(logFilesToProcess))
        sys.exit(0)

    logFilesRemoved = [file for file in logFilesMetadata.keys() if file not in logFilesToProcess]
    
    # Create a table for removed files by time with start and end times and their type
    table = []
    if args.debug:
        print('====================Skipped Files====================')    
        for file in logFilesRemoved:
            table.append([file[-100:], logFilesMetadata[file]["logStartsAt"], logFilesMetadata[file]["logEndsAt"], logFilesMetadata[file]["logType"], logFilesMetadata[file]["nodeName"]])
        print(tabulate.tabulate(table, headers=["File", "Start Time", "End Time", "Type", "Node Name"], tablefmt="simple_grid"))
        print('====================Included Files====================')   
    # Create a table for files included in the analysis
    
    table = []
    for file in logFilesToProcess:
        table.append([file[-100:], logFilesMetadata[file]["logStartsAt"], logFilesMetadata[file]["logEndsAt"], logFilesMetadata[file]["logType"], logFilesMetadata[file]["subType"], logFilesMetadata[file]["nodeName"]])
    table.sort(key=lambda x: (x[4], x[1]))  # Sort by Node Name, then Start Time
    print(tabulate.tabulate(table, headers=["File", "Start Time", "End Time", "Type", "Subtype", "Node Name"], tablefmt="simple_grid"))

    
    # Print Summary, print ==summary of what is included==
    print("==========Summary of Included Files===========")
    print(f"Total Files: {len(logFilesMetadata)} - Included: {len(logFilesToProcess)}")
    print(f"Start Time: {start_time}")
    print(f"End Time: {end_time}")
    logTypes = sorted(set([logFilesMetadata[file]["logType"] for file in logFilesToProcess]))
    print(f"Log Types: {', '.join(logTypes)}")
    nodes = sorted(set([logFilesMetadata[file]["nodeName"] for file in logFilesToProcess]))
    print(f"Nodes: {', '.join(nodes)}")
    # Log missing for the following nodes
    # Postgres logs
    colorama.init(autoreset=True)
    isLogMissing = False
    if "pg" in choosenTypes:
        missingNodes = [node for node in nodes if not any(node in file for file in logFilesToProcess if logFilesMetadata[file]["logType"] == "postgres")]
        if missingNodes:
            print(colorama.Fore.RED + f"Postgres logs missing for nodes: {', '.join(missingNodes)}")
            isLogMissing = True
    # TServer logs
    if "ts" in choosenTypes:
        missingNodes = [node for node in nodes if not any(node in file for file in logFilesToProcess if logFilesMetadata[file]["logType"] == "yb-tserver")]
        if missingNodes:
            print(colorama.Fore.RED + f"TServer logs missing for nodes: {', '.join(missingNodes)}")
            isLogMissing = True
    # Master logs
    if "ms" in choosenTypes:
        missingNodes = [node for node in nodes if not any(node in file for file in logFilesToProcess if logFilesMetadata[file]["logType"] == "yb-master")]
        if missingNodes:
            print(colorama.Fore.RED + f"Master logs missing for nodes: {', '.join(missingNodes)}")
            isLogMissing = True
    # Controller logs
    if "ybc" in choosenTypes:
        missingNodes = [node for node in nodes if not any(node in file for file in logFilesToProcess if logFilesMetadata[file]["logType"] == "yb-controller")]
        if missingNodes:
            print(colorama.Fore.RED + f"Controller logs missing for nodes: {', '.join(missingNodes)}")
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
        # Add the command to run
        if args.full_command:
            print(colorama.Fore.GREEN + ' '.join(Command))
        else:
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