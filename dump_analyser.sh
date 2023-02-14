#!/bin/bash

# Function to display error message and exit
function error_exit() {
  echo "$1"
  exit 1
}

echo "Enter the file name: "
read -r file_name

# Check if file exists
if [ ! -f "$file_name" ]; then
  error_exit "Error: The file '$file_name' does not exist."
fi

# Check if file is a valid core dump
file_output=$(file "$file_name")
if echo "$file_output" | grep -q "core file"; then
  echo "Great! The file is a valid core dump, proceeding with analysis."
else
  error_exit "Error: The file is NOT an ELF core dump, please provide a valid core dump file. Make sure the file IS NOT compressed, if so please extract it and then try again!"
fi

# Extract version information from the file
input_string=$(strings "$file_name" | grep -E "/yugabyte/yb-software/yugabyte-|yugabyte_version" | head -n 1) || input_string=""

modified_string=$(echo "$input_string" | awk -F "/" '{for (i=1; i<=NF; i++) if ($i == "yugabyte" && $(i+1) == "yb-software") print $(i+2)}' | sed 's/centos/linux/' | sed 's/$/.tar.gz/')

version=$(echo "$modified_string" | awk -F "-" '{print $2}')

# Generate the URL for the binary
output_string="https://downloads.yugabyte.com/releases/$version/$modified_string"
echo "The final URL is: $output_string"

# Download the binary
download_file="$modified_string"
target_dir="/cases/home/yugabyte/yb-software"

echo "Downloading the binary file to $target_dir/$modified_string"

curl -L -# "$output_string" -o "$target_dir/$modified_string"
if [ $? -eq 0 ]; then
  echo "Download of YB version file succeeded."
else
  error_exit "Error: Download of YB version file failed."
fi

# Extract the binary
tar_file="$target_dir/$modified_string"

echo "Extracting $tar_file in $target_dir"

tar -xzf "$tar_file" -C "$target_dir" --strip-components=0 &>/dev/null

# Execute post install script
post_install="$target_dir/yugabyte-$version/bin/post_install.sh"

if [ -f "$post_install" ]; then
  echo "Executing post_install script to setup the binary as per core dump. Please bare with me!"
  $post_install &>/dev/null
else
  error_exit "Error: $post_install not found."
fi

# Replace the path in the input_string with the new target directory
new_input_string=$(echo "$input_string" | awk -F "/yugabyte/yb-software" '{print "/yugabyte/yb-software"$2}' | sed "s|/yugabyte/yb-software/yugabyte-[^/]*|$target_dir/yugabyte-$version|")
# Use the lldb command with the new input string
lldb -f "$new_input_string" -c $file_name
