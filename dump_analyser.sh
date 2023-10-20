#!/bin/bash


#Function to show blinkdots for various steps in progress when needed in the script.

function blinkdots() {
  local pid=$1
  local delay=0.5
  while kill -0 $pid 2>/dev/null; do
    printf "."
    sleep $delay
    printf "."
    sleep $delay
    printf "."
    sleep $delay
    printf "\b\b\b   \b\b\b"
    sleep $delay
  done
  printf "\n"
}


# Function to display error message and exit


# Function to display error message and exit

function error_exit() {
  echo "$1"
  exit 1
}

# Enable tab-completion for file names
if [ $# -eq 0 ]; then
  read -e -p "Enter the Core dump file name: " file_name
else
  file_name=$1
fi

# Check if the Core file exists
if [ ! -f "$file_name" ]; then
  error_exit "Error: The file '$file_name' does not exist."
fi

# Check if the Core file is a valid core dump
file_type=$(file "$file_name")
if echo "$file_type" | grep -q "core file"; then
  echo "Great! The core dump file provided is a valid core dump, proceeding with analysis of core dump."
else
  error_exit "Error: The Core dump file is NOT an ELF core dump, please provide a valid core dump file. Make sure the file IS NOT compressed, if so please extract it and then try again!"
fi


# Extract the yugabyte binary version information from the core dump file

echo "Extracting yugabyte binary version information, please wait..."
#strings "$file_name" | grep -E "/yugabyte/yb-software/yugabyte-" | head -n 1 > /tmp/yb_db_version & PID=$!

strings "$file_name" | grep -P "^/.*/yugabyte/yb-software/.*/bin/" | sort | uniq -c | sort -nr | head -1 | awk {'print $2'} > /tmp/yb_db_version & PID=$!


blinkdots $PID
yb_db_version=$(cat /tmp/yb_db_version)
rm /tmp/yb_db_version
echo "Yugabyte binary version extraction completed."
if [ -n "$yb_db_version" ]; then
  yb_db_tar_file=$(echo "$yb_db_version" | awk -F "/" '{for (i=1; i<=NF; i++) if ($i == "yugabyte" && $(i+1) == "yb-software") print $(i+2)}' | sed 's/centos/linux/' | sed 's/$/.tar.gz/')
  version=$(echo "$yb_db_tar_file" | awk -F "-" '{print $2}')
else
  error_exit "Error: Failed to extract Yugabyte binary version information."
fi




# Separator
echo "--------------------------------------------------------"

#Manupulating the above output to desired downloadable tar file name.

yb_db_tar_file=$(echo "$yb_db_version" | awk -F "/" '{for (i=1; i<=NF; i++) if ($i == "yugabyte" && $(i+1) == "yb-software") print $(i+2)}' | sed 's/centos/linux/' | sed 's/$/.tar.gz/')

version=$(echo "$yb_db_tar_file" | awk -F "-" '{print $2}')


# Generate the URL for the yb-db binary

yb_db_tar_url="https://downloads.yugabyte.com/releases/$version/$yb_db_tar_file"
echo "The final URL is: $yb_db_tar_url"


# Download the yb db binary tar file

download_file="$yb_db_tar_file"
yb_db_install_dir="/home/yugabyte/yb-software"

if [ -f "$yb_db_install_dir/$yb_db_tar_file" ]; then
  echo "The file $yb_db_tar_file already exists in $yb_db_install_dir. Skipping the download step."
else
  echo "Downloading the YB version file to $yb_db_install_dir/$yb_db_tar_file"

  curl -L -# "$yb_db_tar_url" -o "$yb_db_install_dir/$yb_db_tar_file"
  if [ $? -eq 0 ]; then
    echo "Download of YB version file succeeded."
  else
    error_exit "Error: Download of YB version file failed."
  fi
fi

# Separator
echo "--------------------------------------------------------"

#Renamed executable dir, means we are going to extract the tar file binary in the same version format as core file.

executable_dir=$(echo "$yb_db_version" | awk -F "/" '{for (i=1; i<=NF; i++) if ($i == "yugabyte" && $(i+1) == "yb-software") print $(i+2)}')


# Extract the yb binary tar file in the "/home/yugabyte/yb-software" dir in case file server

tar_file="$yb_db_install_dir/$yb_db_tar_file"
extracted_directory="$yb_db_install_dir/$executable_dir"

if [ -d "$extracted_directory" ]; then
  echo "$extracted_directory already exists, not extracting again."
else
  echo "Extracting $tar_file in $yb_db_install_dir"
  mkdir -p $yb_db_install_dir/$executable_dir && tar -xzf "$tar_file" --strip-components=1 -C $_ &>/dev/null &
  blinkdots $!
  if [ $? -eq 0 ]; then
    echo "Extracting $tar_file completed."
  else
    error_exit "Error: Failed to extract $tar_file."
  fi
fi

# Separator
echo "--------------------------------------------------------"

# Execute post install script. This will setup the yb-db executable files to work with core and other cluster related stuff.

post_install="$yb_db_install_dir/$executable_dir/bin/post_install.sh"

if [ -f "$post_install" ]; then
  echo "Executing post_install script to setup the binary as per core dump. Please bear with me!"
  $post_install &>/dev/null &
  blinkdots $!
  if [ $? -eq 0 ]; then
    echo "Post-installation setup completed."
  else
    error_exit "Error: Failed to execute post_install script."
  fi
=======
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

# Execute post install script. This will setup the yb-db executable files to work with core and other cluster related stuff.

post_install="$yb_db_install_dir/$executable_dir/bin/post_install.sh"

if [ -f "$post_install" ]; then
  echo "Executing post_install script to setup the binary as per core dump. Please bear with me!"
  $post_install &>/dev/null &
  blinkdots $!
  if [ $? -eq 0 ]; then
    echo "Post-installation setup completed."
  else
    error_exit "Error: Failed to execute post_install script."
  fi
else
  error_exit "Error: $post_install not found."
fi

# Replace the path in the yb_db_version with the new target directory

new_yb_db_version=$(echo "$yb_db_version" | awk -F "/yugabyte/yb-software" '{print "/yugabyte/yb-software"$2}' | sed "s|/yugabyte/yb-software/yugabyte-[^/]*|$yb_db_install_dir/$executable_dir|")

# Use the lldb command with the new input string
# Ask user to enetr available lldb command option for ease.
#The below section is to ask few more user inputs and redirection etc.

# Separator
echo "--------------------------------------------------------"

echo "Select an option for lldb command and press ENTER:"
echo "1. bt"
echo "2. thread backtrace all"
echo "3. Other lldb command"
echo "4. Quit"
read -r option

while [[ ! "$option" =~ ^(1|2|3|4)$ ]]; do
  echo "Error: Invalid option selected. Please select either 1, 2, 3 or 4."
  echo "Select an option for lldb command and press ENTER:"
  echo "1. bt"
  echo "2. thread backtrace all"
  echo "3. Other lldb command"
  echo "4. Quit"
  read -r option
done

# Separator
echo "--------------------------------------------------------"


if [ "$option" == "1" ]; then
  lldb_command="bt"
elif [ "$option" == "2" ]; then
  lldb_command="thread backtrace all"
elif [ "$option" == "3" ]; then
  echo "Enter the lldb command:"
  read -r lldb_command
fi

if [ "$option" != "4" ]; then
  echo "Do you want to redirect the output to a file? (y/n)"
  read -r redirect_output
  while [[ ! "$redirect_output" =~ ^(y|n)$ ]]; do
    echo "Error: Invalid option selected. Please enter either y or n."
    echo "Do you want to redirect the output to a file? (y/n)"
    read -r redirect_output
  done

# Separator
echo "--------------------------------------------------------"


if [ "$redirect_output" == "y" ]; then
  output_file="${file_name}_$(echo "$lldb_command" | tr -s ' ' '_')_analysis.out"
  echo "Output will be saved to $output_file"
  lldb -f "$new_yb_db_version" -c "$file_name" -o "$lldb_command" -o "quit"> "$output_file"
  echo "Analysis complete, the file '$output_file' has been saved."
else
  lldb -f "$new_yb_db_version" -c "$file_name" -o "$lldb_command"
fi
fi
echo "Exiting."
exit 0
