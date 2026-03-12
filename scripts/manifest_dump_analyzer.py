#!/usr/bin/env python3
import re
from datetime import datetime, timezone, timedelta
import sys
import argparse
from tabulate import tabulate

def parse_hybrid_time_to_micros(hybrid_time_micros_str):
    """
    Converts a hybrid time (string of microseconds) to an integer.
    Returns None if conversion fails.
    """
    try:
        return int(hybrid_time_micros_str)
    except (ValueError, TypeError):
        return None

def convert_micros_to_datetime(micros):
    """
    Converts microseconds since epoch to a datetime object.
    Returns None if input is None.
    """
    if micros is None:
        return None
    # Assuming hybrid_time is microseconds since epoch
    return datetime.fromtimestamp(micros / 1_000_000, tz=timezone.utc)

def format_datetime_to_readable(dt_obj):
    """
    Formats a datetime object to a human-readable string.
    Returns 'N/A' if input is None.
    """
    if dt_obj is None:
        return 'N/A'
    return dt_obj.strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]

def parse_data(raw_data):
    """
    Parses the raw string data to extract number, total_size,
    smallest hybrid_time (micros), and largest hybrid_time (micros).
    """
    # Regex for splitting records more reliably.
    # This regex looks for '{ number:...' and captures everything until the next '{ number:'
    # or the end of the string. Assumes records are concatenated without extra delimiters.
    records = re.findall(r'({ number:.*?)(?={ number:|$)', raw_data, re.DOTALL)

    parsed_results = []

    for record_str in records:
        # Extract 'number'
        number_match = re.search(r'number: (\d+)', record_str)
        number = int(number_match.group(1)) if number_match else 'N/A'

        # Extract 'total_size'
        total_size_match = re.search(r'total_size: (\d+)', record_str)
        total_size_bytes = int(total_size_match.group(1)) if total_size_match else 0
        total_size_gb = round(total_size_bytes / (1024**3), 4) if total_size_bytes != 0 else 0.0000

        # Extract 'smallest hybrid_time' physical value
        smallest_time_match = re.search(r'smallest: .*?hybrid_time: { physical: (\d+).*?}', record_str, re.DOTALL)
        smallest_hybrid_time_micros = parse_hybrid_time_to_micros(smallest_time_match.group(1)) if smallest_time_match else None

        # Extract 'largest hybrid_time' physical value
        largest_time_match = re.search(r'largest: .*?hybrid_time: { physical: (\d+).*?}', record_str, re.DOTALL)
        largest_hybrid_time_micros = parse_hybrid_time_to_micros(largest_time_match.group(1)) if largest_time_match else None

        parsed_results.append({
            'number': number,
            'total_size_gb': total_size_gb,
            'smallest_hybrid_time_micros': smallest_hybrid_time_micros,
            'largest_hybrid_time_micros': largest_hybrid_time_micros
        })
    return parsed_results

# --- Main execution part ---
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="""
        Parse and analyze YugabyteDB manifest dump output files.

        This script extracts and summarizes information from manifest dump files, including:
        - Number of records
        - Total size (GB)
        - Smallest and largest hybrid times (as UTC datetimes)
        - Data range and age in days
        - Projected removal date (based on TTL)
        - Days left until removal

        Only records with total size > 3GB are shown in the output table.

        Example usage:
          python manifest_dump_parser.py <data_file_path>
        """,
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("data_file_path", help="Path to the manifest dump output file to analyze.")
    parser.add_argument("--size", type=float, default=3.0, help="Minimum size threshold in GB for filtering records (default: 3.0)")
    parser.add_argument("--ttl", type=int, required=True, help="TTL in days to use for removal date calculations (required)")
    args = parser.parse_args()

    file_path = args.data_file_path
    size_threshold = args.size
    TTL_DAYS = args.ttl
    try:
        with open(file_path, 'r') as file:
            data = file.read()
    except FileNotFoundError:
        print(f"Error: The file '{file_path}' was not found.")
        sys.exit(1)
    except Exception as e:
        print(f"An unexpected error occurred while reading the file: {e}")
        sys.exit(1)

    try:
        results = parse_data(data)
        # Process results for display and calculations
        processed_results = []
        current_datetime_utc = datetime.now(timezone.utc)

        for res in results:
            smallest_dt = convert_micros_to_datetime(res['smallest_hybrid_time_micros'])
            largest_dt = convert_micros_to_datetime(res['largest_hybrid_time_micros'])

            # Calculate Data Range in Days
            data_range_days = 'N/A'
            if smallest_dt and largest_dt:
                time_difference_seconds = (largest_dt - smallest_dt).total_seconds()
                data_range_days = round(time_difference_seconds / (3600 * 24), 2) # Convert seconds to days

            # Calculate Data Age in Days
            data_age_days = 'N/A'
            if smallest_dt:
                time_difference_seconds = (current_datetime_utc - smallest_dt).total_seconds()
                data_age_days = round(time_difference_seconds / (3600 * 24), 2) # Convert seconds to days

            # Calculate 'To be removed on' date based on LARGEST Hybrid Time + TTL
            to_be_removed_on_dt = None # Store as datetime object for calculation
            to_be_removed_on_date_str = 'N/A'
            if largest_dt:
                to_be_removed_on_dt = largest_dt + timedelta(days=TTL_DAYS)
                to_be_removed_on_date_str = format_datetime_to_readable(to_be_removed_on_dt)

            # Calculate 'Days left to removed'
            days_left_to_removed = 'N/A'
            if to_be_removed_on_dt:
                time_until_removal_seconds = (to_be_removed_on_dt - current_datetime_utc).total_seconds()
                days_left_to_removed = round(time_until_removal_seconds / (3600 * 24), 2)

            processed_results.append({
                'number': res['number'],
                'total_size_gb': res['total_size_gb'],
                'smallest_hybrid_time': format_datetime_to_readable(smallest_dt),
                'largest_hybrid_time': format_datetime_to_readable(largest_dt),
                'data_range_days': data_range_days,
                'data_age_days': data_age_days,
                'to_be_removed_on': to_be_removed_on_date_str,
                'days_left_to_removed': days_left_to_removed
            })

        # Filter results to include only records where 'Total Size (GB)' is greater than the threshold
        filtered_results = [
            res for res in processed_results
            if isinstance(res['total_size_gb'], (int, float)) and res['total_size_gb'] > size_threshold
        ]

        # Sort filtered results by 'total_size_gb'
        results_sorted = sorted(filtered_results, key=lambda x: x['total_size_gb'])

        # Prepare data for tabulate output
        table_headers = [
            "Record",
            "Number",
            f"Total Size (GB) (> {size_threshold}GB)",
            "Smallest Hybrid Time",
            "Largest Hybrid Time",
            "Data Range (Days)",
            "Data Age (Days)",
            "To be removed on",
            "Days left to removed"
        ]
        table_data = []
        for i, result in enumerate(results_sorted):
            table_data.append([
                i + 1,
                result['number'],
                f"{result['total_size_gb']:.4f}",
                result['smallest_hybrid_time'],
                result['largest_hybrid_time'],
                result['data_range_days'],
                result['data_age_days'],
                result['to_be_removed_on'],
                result['days_left_to_removed']
            ])

        print(tabulate(table_data, headers=table_headers, tablefmt="pipe"))
    except Exception as e:
        print(f"An unexpected error occurred: {e}")
        sys.exit(1)