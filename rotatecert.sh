#!/bin/bash

VERSION=0.1.2

set -o nounset
set -o pipefail

AWK=/usr/bin/awk
BASENAME=/usr/bin/basename
CAT=/bin/cat
DATE=/bin/date
ECHO=/bin/echo
GREP=/bin/grep
MV=/bin/mv
OPENSSL=/usr/bin/openssl
RM=/bin/rm
SSH=/usr/bin/ssh
SCP=/usr/bin/scp
SUDO=/usr/bin/sudo
TAR=/bin/tar

# This must be a space-separated list of IP addresses or hostnames. This node list will be used for making SSH connections,
# setting certificate CNs and SANs, and various other important bits of the script, so make sure you have it right!
# TODO: Dynamically get list of node IPs / hostnames
NODELIST="${NODELIST:=10.230.0.3 10.231.0.46 10.232.0.3}"

# TODO: Dynamically retrieve the path to the SSH key and other SSH parameters (e.g. port number)
SSH_KEY="${SSH_KEY:=/opt/yugabyte/yugaware/data/keys/3664ae19-059c-487d-aaab-93ea0e90ca58/yb-dev-ianderson-gcp_3664ae19-059c-487d-aaab-93ea0e90ca58-key.pem}"
SSH_PORT="${SSH_PORT:=22}"
SSH_USER="${SSH_USER:=yugabyte}"

# TODO: Dynamically get platform root cert path
ROOT_CERT_DIR=/opt/yugabyte/yugaware/data/certs/b024bd98-bbba-42ea-97a7-fa4d7bb3116d/973a6b3b-013e-40d3-8270-ef3ab4c0a534
ROOT_CERT=$ROOT_CERT_DIR/ca.root.crt
ROOT_KEY=$ROOT_CERT_DIR/ca.key.pem
ROOT_CSR=$ROOT_CERT_DIR/ca.root.csr

CLIENT_CERT=$ROOT_CERT_DIR/yugabytedb.crt
CLIENT_KEY=$ROOT_CERT_DIR/yugabytedb.key
CLIENT_CSR=$ROOT_CERT_DIR/yugabytedb.csr

ROOT_CERT_DAYS=365
CLIENT_CERT_DAYS=365
NODE_CERT_DAYS=365

REMOTE_NODE_TLS_DIR=yugabyte-tls-config
REMOTE_CLIENT_TLS_DIR=.yugabytedb

RESTART_WAIT_SECONDS=5

CLEANUP=true
DEBUG=true
# TODO: Add support for an expiry threshold flag, so certs are regenerated if the will expire with the next n days
FORCE=false

TS=$($DATE +%s)

WORKDIR=$HOME/rotatecert-$TS

CA_CONF_FILE=$WORKDIR/openssl-ca.cnf
CLIENT_CRT_CONF_FILE=$WORKDIR/openssl-client-crt.cnf
NODE_CRT_CONF_FILE=$WORKDIR/openssl-node-crt.cnf

PLATFORM_CERT_BACKUP_FILE=$WORKDIR/tls-cert-backup-$TS.tgz
REMOTE_CERT_BACKUP_FILE=tls-cert-backup-$TS.tgz

debug () {
  if [[ "$DEBUG" = true ]]; then
    $ECHO "DEBUG: $1" 1>&2
  fi
}

info () {
  $ECHO "INFO: $1"
}

warn () {
  $ECHO "WARN: $1"
}

error () {
  $ECHO "ERROR: $1"
}

fatal () {
  code=$1
  $ECHO "FATAL: $2"
  exit $code
}

check_cert_expiry () {
  cert_file=$1
  debug "Checking expiry status of certificate $cert_file"
  # TODO: Check if we can read this file before trying to get the expiry_date from it?
  # $ openssl x509 -in /opt/yugabyte/yugaware/data/certs/b024bd98-bbba-42ea-97a7-fa4d7bb3116d/973a6b3b-013e-40d3-8270-ef3ab4c0a534/ca.root.crt -noout -enddate
  # notAfter=Oct 24 19:16:31 2026 GMT
  expiry_date=$($OPENSSL x509 -in $cert_file -noout -enddate | $AWK -F '=' '{ print $2 }' )
  # Bail out if the openssl command failed or we didn't get back a valid expiry date
  if [[ $? -ne 0 || -z $expiry_date ]]; then
    fatal 2 "Failed to read expiry date of certificate '$cert_file'."
  fi
  debug "Found certificate expiry date '$expiry_date'"
  expiry_epoch=$($DATE -d "$expiry_date" +%s)
  debug "Certificate expiry epoch time is $expiry_epoch"
  debug "Current epoch time is $TS"

  # Might be shorter with a ternary? Does bash support those?
  if [[ $expiry_epoch -lt $TS ]]; then
    debug "Certificate has expired."
    return 1
  else
    debug "Certificate has not expired."
    return 0
  fi
}

do_ssh () {
  node=$1
  cmd=$2
  ssh_cmd="$SUDO $SSH -i $SSH_KEY -p $SSH_PORT ${SSH_USER}@$node $cmd"
  debug "Running $ssh_cmd"
  $ssh_cmd
  # TODO: Error handling
}

gen_client_crt_conf () {
  $CAT << CLCRTCNF > $CLIENT_CRT_CONF_FILE
[req]
req_extensions = v3_req

[v3_req]
basicConstraints = critical, CA:FALSE
authorityKeyIdentifier = keyid:always,issuer:always
subjectKeyIdentifier = hash
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, keyCertSign
CLCRTCNF
}

gen_node_crt_conf () {
  # TODO: Check whether this is a hostname or IP address
  node=$1
  
  debug "Creating OpenSSL configuration file $NODE_CRT_CONF_FILE for node certificate"
  debug "Adding Subject Alternative Name '$node' to the certificate"
  $CAT << NOCRTCNF > $NODE_CRT_CONF_FILE
[req]
req_extensions = v3_req

[v3_req]
basicConstraints = critical, CA:FALSE
authorityKeyIdentifier = keyid:always,issuer:always
subjectKeyIdentifier = hash
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, keyCertSign
subjectAltName = @alt_names

[ alt_names ]
IP.1 = $node
NOCRTCNF
}

install_cert_files () {
  node=$1

  debug "Installing certificate files on node $node"
  # TODO: DRY this out?
  debug "Installing root certificate for node-to-node TLS"
  do_ssh $node "$MV $REMOTE_NODE_TLS_DIR/ca.crt $REMOTE_NODE_TLS_DIR/ca_crt_$TS.old" && scp_put $node $ROOT_CERT "$REMOTE_NODE_TLS_DIR/ca.crt" || return $?
  #if [[ $? -ne 0 ]]; then return $?; fi
  debug "Installing root certificate for client-to-node TLS"
  do_ssh $node "$MV $REMOTE_CLIENT_TLS_DIR/root.crt $REMOTE_CLIENT_TLS_DIR/root_crt_$TS.old" && scp_put $node $ROOT_CERT "$REMOTE_CLIENT_TLS_DIR/root.crt" || return $?
  #if [[ $? -ne 0 ]]; then return $?; fi

  debug "Installing client certificate"
  do_ssh $node "$MV $REMOTE_CLIENT_TLS_DIR/yugabytedb.crt $REMOTE_CLIENT_TLS_DIR/yugabytedb_crt_$TS.old" && scp_put $node $CLIENT_CERT "$REMOTE_CLIENT_TLS_DIR/" || return $?
  #if [[ $? -ne 0 ]]; then return $?; fi
  
  debug "Installing node certificate"
  do_ssh $node "$MV $REMOTE_NODE_TLS_DIR/node.$NODE.crt $REMOTE_NODE_TLS_DIR/node.${NODE}_crt_$TS.old" && scp_put $node "$WORKDIR/node.$NODE.crt" "$REMOTE_NODE_TLS_DIR/" || return $?
  #if [[ $? -ne 0 ]]; then return $?; fi

  debug "Setting ownership and permissions on certificate files"
  do_ssh $node "chmod 0400 $REMOTE_NODE_TLS_DIR/*.crt $REMOTE_NODE_TLS_DIR/*.key $REMOTE_CLIENT_TLS_DIR/*.crt $REMOTE_CLIENT_TLS_DIR/*.key" || return $?
  #if [[ $? -ne 0 ]]; then return $?; fi
}

scp_get () {
  # TODO: Validate parameters
  node=$1
  remote_file=$2
  local_file=$3

  scp_cmd="$SUDO $SCP -i $SSH_KEY -P $SSH_PORT ${SSH_USER}@$node:$remote_file $local_file"
  debug "Running $scp_cmd"
  $scp_cmd
  # TODO: Error handling
}

scp_put () {
  # TODO: Validate parameters
  node=$1
  local_file=$2
  remote_file=$3

  scp_cmd="$SUDO $SCP -i $SSH_KEY -P $SSH_PORT $local_file ${SSH_USER}@$node:$remote_file"
  debug "Running $scp_cmd"
  $scp_cmd
  # TODO: Error handling
}

info "Starting $($BASENAME -- $0) version $VERSION"
debug "Nodelist is $NODELIST"

if [[ $EUID -eq 0 ]]; then
  debug "Running with root privileges. Disabling calls to sudo."
  SUDO=""
fi

mkdir $WORKDIR
if [[ $? -ne 0 ]]; then
  fatal 6 "Failed to create working directory $WORKDIR. Check ownership and permissions."
fi

# Bail out if the "root cert" has multiple certs inside, since that means it's a cert bundle and we're likely dealing with real CA certs
root_cert_count=$($GREP -c "BEGIN" $ROOT_CERT)
if [[ $root_cert_count -gt 1 ]]; then
  fatal 1 "Found $root_cert_count certificates in the certificate bundle. This script only supports self-signed root certificates."
fi

debug "Backing up certificate files on platform node"
$SUDO $TAR -czf $PLATFORM_CERT_BACKUP_FILE $ROOT_CERT_DIR/
if [[ $? -ne 0 ]]; then
  fatal 4 "Failed to back up certificate files on platform node. Cowardly refusing to proceed without a backup."
fi

# Back up node certs and keys
for NODE in $NODELIST
do
  debug "Backing up certificate files on database node $NODE"

  # Back up .yugabytedb directory and yugabyte-tls-config directory
  do_ssh $NODE "$TAR -czf $REMOTE_CERT_BACKUP_FILE $REMOTE_CLIENT_TLS_DIR/* $REMOTE_NODE_TLS_DIR/*"
  if [[ $? -ne 0 ]]; then
    fatal 4 "Failed to back up certificate files on database node '$NODE'. Cowardly refusing to proceed without a backup."
  fi
done

# Verify that the certificate is self-signed
# $ openssl x509 -in /opt/yugabyte/yugaware/data/certs/b024bd98-bbba-42ea-97a7-fa4d7bb3116d/973a6b3b-013e-40d3-8270-ef3ab4c0a534/ca.root.crt -noout -subject
# notAfter=Oct 24 19:16:31 2026 GMT
root_cert_subject=$($OPENSSL x509 -in $ROOT_CERT -noout -subject | $AWK -F '=' '{ print $2 }' )
if [[ $? -ne 0 || -z $root_cert_subject ]]; then
  fatal 2 "Failed to read Subject of certificate '$ROOT_CERT'."
fi
root_cert_issuer=$($OPENSSL x509 -in $ROOT_CERT -noout -issuer | $AWK -F '=' '{ print $2 }' )
if [[ $? -ne 0 || -z $root_cert_issuer ]]; then
  fatal 2 "Failed to read Issuer of certificate '$ROOT_CERT'."
fi

if [[ "$root_cert_subject" != "$root_cert_issuer" ]]; then
  fatal 3 "Root certificate Subject '$root_cert_subject' does not match Issuer '$root_cert_issuer'. This script only supports self-signed root certificates."
else
  debug "Self signed certificate detected. Proceeding."
fi

# We don't want to regenerate the root cert unless we have to
regenerate_root_cert=false

# Check expiration of root cert
check_cert_expiry $ROOT_CERT
if [[ "$?" -eq 1 ]]; then
  debug "Root certificate has expired. Need to regenerate."
  regenerate_root_cert=true
else
  debug "Root certificate has not expired. Reprieving from regeneration."
fi

# If the root cert doesn't have a Subject Key Identifier, we need to regenerate it
# It would be better to use -ext SubjectKeyIdentifier here but ancient openssl releases
# (1.0.2 and the like) don't support that flag.
root_subject_key_id=$($OPENSSL x509 -in $ROOT_CERT -noout -text | grep -A1 "Subject Key Identifier" )
if [[ $? -ne 0 ]]; then
  fatal 2 "Failed to read certificate '$ROOT_CERT' while checking Subject Key Identifier,"
fi

if [[ -z $root_subject_key_id ]]; then
  debug "Root certificate's Subject Key Identifier is missing. Need to regenerate."
  regenerate_root_cert=true
else
  debug "Root certificate has a Subject Key Identifier. Reprieving from regeneration."
fi

# Generate new root cert if expired or missing critical extensions
if [[ "$regenerate_root_cert" = true ]]; then
  debug "Creating OpenSSL configuration file $CA_CONF_FILE for root certificate generation"
  $CAT << CACNF > $CA_CONF_FILE
[req]
req_extensions = v3_req

[v3_req]
basicConstraints = critical, CA:TRUE, pathlen:1
subjectKeyIdentifier = hash
keyUsage = critical, digitalSignature, nonRepudiation, keyEncipherment, keyCertSign
CACNF

  # Generate certificate signing request from existing root cert
  $SUDO $OPENSSL x509 -x509toreq -in $ROOT_CERT -signkey $ROOT_KEY -out $ROOT_CSR
  # Generate new cert from CSR
  $SUDO $OPENSSL x509 -req -in $ROOT_CSR -signkey $ROOT_KEY -set_serial $($DATE "+%s%3N") -out $ROOT_CERT_DIR/ca.root_new.crt -days $ROOT_CERT_DAYS -sha256 -extensions v3_req -extfile $CA_CONF_FILE
  if [[ "$?" -ne 0 ]]; then
    fatal 5 "Failed to generate new root certificate. Unable to continue."
  fi
  debug "Moving aside old platform root certificate and installing new root certificate"
  $SUDO $MV $ROOT_CERT_DIR/ca.root.crt $ROOT_CERT_DIR/ca.root_crt_$TS.old && $SUDO $MV $ROOT_CERT_DIR/ca.root_new.crt $ROOT_CERT_DIR/ca.root.crt
  if [[ "$?" -ne 0 ]]; then
    fatal 5 "Failed to install new root certificate '$ROOT_CERT_DIR/ca.root.crt'. Unable to continue. Check permissions, restore from backup '$PLATFORM_CERT_BACKUP_FILE' and try again."
  fi
else
  debug "Skipping root certificate generation."
fi

# We don't want to regenerate the client cert unless we have to
regenerate_client_cert=false

if [[ "$regenerate_root_cert" = true ]]; then
  debug "Root certificate has been rotated, so forcing regeneration of client certificate '$CLIENT_CERT'."
  regenerate_client_cert=true
else
  # Check expiration of client cert on platform node
  check_cert_expiry $CLIENT_CERT
  if [[ "$?" -eq 1 ]]; then
    debug "Client certificate '$CLIENT_CERT' has expired. Need to regenerate."
    regenerate_client_cert=true
  else
    debug "Client certificate '$CLIENT_CERT' has not expired. Reprieving from regeneration."
  fi

  # If the client cert was not signed by the current root cert, we need to regenerate it
  $OPENSSL verify -CAfile $ROOT_CERT $CLIENT_CERT
  # TODO: There may be other return codes that indicate a need to regenerate
  if [[ "$?" -eq 2 ]]; then
    regenerate_client_cert=true
  fi
fi

if [[ "$regenerate_client_cert" == true ]]; then
  # The client certificate is signed by the root certificate and is common across all nodes
  debug "Generating new client certificate"
  debug "Creating OpenSSL configuration file for client certificate generation"
  # Specify path here instead of inside function?
  gen_client_crt_conf
  $SUDO $OPENSSL x509 -x509toreq -in $CLIENT_CERT -signkey $CLIENT_KEY -out $CLIENT_CSR
  # Need to sign with sudo since the output file is owned by root
  $SUDO $OPENSSL x509 -req -in $CLIENT_CSR -CA $ROOT_CERT -CAkey $ROOT_KEY -set_serial $($DATE "+%s%3N") -out $ROOT_CERT_DIR/yugabytedb_new.crt  -days $CLIENT_CERT_DAYS -sha256 -extensions v3_req -extfile $CLIENT_CRT_CONF_FILE
  if [[ "$?" -ne 0 ]]; then
    fatal 5 "Failed to generate new client certificate. Unable to continue."
  fi
  # TODO: This message could be improved
  debug "Moving aside old platform client certificate and installing new client certificate"
  $SUDO $MV $ROOT_CERT_DIR/yugabytedb.crt $ROOT_CERT_DIR/yugabytedb_crt_$TS.old && $SUDO $MV $ROOT_CERT_DIR/yugabytedb_new.crt $ROOT_CERT_DIR/yugabytedb.crt
  if [[ "$?" -ne 0 ]]; then
    fatal 5 "Failed to install new client certificate '$ROOT_CERT_DIR/yugabytedb.crt'. Unable to continue. Check permissions, restore from backup '$PLATFORM_CERT_BACKUP_FILE' and try again."
  fi
fi

# Generate new node certs and keys
for NODE in $NODELIST; do
  debug "Generating new CSR for node $NODE based on its existing certificate"
  # TODO: Handle errors here. Also we should possibly move any existing csr aside before attempting to create a new one.
  # TODO: There is an implicit assumption here that the openssl binary is in the same location on the database nodes. This is fragile.
  do_ssh $NODE "$OPENSSL x509 -x509toreq -in $REMOTE_NODE_TLS_DIR/node.$NODE.crt -signkey $REMOTE_NODE_TLS_DIR/node.$NODE.key -out $REMOTE_NODE_TLS_DIR/node.$NODE.csr"
  debug "Collecting CSR from node $NODE"
  scp_get $NODE "$REMOTE_NODE_TLS_DIR/node.$NODE.csr" "$WORKDIR/"
  debug "Regenerating OpenSSL config to add Subject Alt Name(s)"
  gen_node_crt_conf $NODE
  debug "Generating node certificate for node $NODE"
  $OPENSSL x509 -req -in $WORKDIR/node.$NODE.csr -CA $ROOT_CERT -CAkey $ROOT_KEY -set_serial $($DATE "+%s%3N") -out $WORKDIR/node.$NODE.crt -days $NODE_CERT_DAYS -sha256 -extensions v3_req -extfile $NODE_CRT_CONF_FILE
  if [[ "$?" -ne 0 ]]; then
    fatal 5 "Failed to generate new node certificate for node $NODE. Unable to continue."
  fi
done

# Copy new cert files to the nodes
for NODE in $NODELIST
do
  install_cert_files $NODE
  if [[ $? -ne 0 ]]; then
    fatal 5 "Failed to install certificate files on $NODE. Unable to continue. Restore platform certificate files from backup '$PLATFORM_CERT_BACKUP_FILE', restore node certificate files from backup '$REMOTE_CERT_BACKUP_FILE', and try again."
  fi
done

if [[ "$CLEANUP" = true ]]; then
  debug "Cleaning up OpenSSL configuration files"
  $RM -f $CA_CONF_FILE $CLIENT_CRT_CONF_FILE $NODE_CRT_CONF_FILE
  debug "Cleaning up certificate signing requests"
  $SUDO $RM -f $ROOT_CSR $CLIENT_CSR $WORKDIR/node.*.csr
  for NODE in $NODELIST; do
    do_ssh $NODE "$RM -f $REMOTE_NODE_TLS_DIR/node.$NODE.csr"
  done
  debug "Cleaning up certificates in working directory"
  $RM -f $WORKDIR/node.*.crt
fi

# TODO: Check for xcluster and tell customer to suspend replication before rotating?
debug "Please restart master and tserver processes on each node to load the new certificate(s)."
