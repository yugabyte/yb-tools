# YugabyteDB Bashrc Script Documentation

## Overview
This bashrc script provides a comprehensive environment setup for YugabyteDB operations, including automatic master address discovery, server type detection, and convenient aliases for database management.

## Version
2025.08.07

## Main Features

### 1. Automatic Master Address Discovery
The script automatically determines master addresses through two methods:

#### Method 1: Configuration File
- **File**: `$HOME/tserver/conf/server.conf`
- **Process**: Extracts `master_addr` parameter using grep and awk
- **Priority**: First choice if file exists

#### Method 2: YBA API Integration
- **Endpoint**: YBA (YugabyteDB Anywhere) API
- **Authentication**: Uses `API_TOKEN` and `YBA_HOST` environment variables
- **Process**: 
  - Gets customer UUID from YBA API
  - Discovers universe UUID based on current IP address
  - Retrieves master addresses from `/metamaster/universe/{UNIVERSE_UUID}` endpoint
  - Returns comma-separated list of master addresses with ports

### 2. Server Type Detection
The script automatically determines if the current node is a master or tablet server:

- **Master Server**: Current IP matches any master IP address
- **Tablet Server**: Current IP doesn't match any master IP address
- **Variable**: `srvrtype` is set to 'M' for master or 'T' for tablet server

### 3. Configuration Management

#### YugabyteDB Config Files
- **Pattern**: `$HOME/yb*.rc` files
- **Loading**: Automatically sourced if they exist
- **Purpose**: Set environment variables like `YBA_HOST`, `API_TOKEN`, `UNIVERSE_UUID`
- **Behavior**: Sources first matching file and exits

#### Required Environment Variables
- `YBA_HOST`: YBA API host URL
- `API_TOKEN`: YBA API authentication token
- `UNIVERSE_UUID`: (Optional) Specific universe UUID

### 4. Environment Setup

#### Path Configuration
- Adds `$HOME/tserver/bin` and `$HOME/scripts` to PATH
- Creates `/home/yugabyte/scripts` directory

#### SSL/TLS Configuration
- `SSL_VERSION`: Set to "TLSv1_2"
- `SSL_CERTFILE`: Points to CA certificate
- `TLSDIR`: TLS configuration directory

#### Prompt Customization
- **Format**: `{USER}@{HOSTNAME}:{IP}:({srvrtype}):({PWD})#`
- **Example**: `yugabyte@node1:192.168.1.100:(M):(/home/yugabyte)#`

### 5. Database Connection Aliases

#### CQL (Cassandra Query Language)
```bash
alias c='cqlsh --ssl `hostname -i` -u cassandra'
```

#### YSQL (YugabyteDB SQL)
```bash
alias p='~/tserver/bin/ysqlsh -h /tmp/.yb.$(hostname -i):5433 -U yugabyte'
```

### 6. Process Management Aliases

#### Process Monitoring
- `pse`: Search processes with grep
- `pseyb`: Show YugabyteDB processes (excluding auto and Java processes)
- `hcheck`: Run health check script

#### Service Status
- `mstat`: Check master service status (systemd or legacy)
- `tstat`: Check tablet server status (systemd or legacy)

### 7. Log Management Aliases

#### Master Logs
- `mconf`: View master configuration
- `minfo/minfot`: View/tail master INFO logs
- `mwarn/mwarnt`: View/tail master WARNING logs
- `merror/merrort`: View/tail master ERROR logs
- `mfatal/mfatalt`: View/tail master FATAL logs

#### Tablet Server Logs
- `tconf`: View tablet server configuration
- `tinfo/tinfot`: View/tail tablet server INFO logs
- `twarn/twarnt`: View/tail tablet server WARNING logs
- `terror/terrort`: View/tail tablet server ERROR logs
- `tfatal/tfatalt`: View/tail tablet server FATAL logs

### 8. Administrative Tools

#### YB Admin
```bash
alias ybadmin='~/tserver/bin/yb-admin -master_addresses $MASTERS -certs_dir_name $TLSDIR'
```

## API Integration Details

### YBA API Endpoints Used
1. **Customers**: `GET /api/v1/customers` - Get customer UUID
2. **Universes**: `GET /api/v1/customers/{customer_uuid}/universes` - List universes
3. **Universe Details**: `GET /metamaster/universe/{universe_uuid}` - Get master addresses

### JQ Queries
- **Customer UUID**: `.[0].uuid`
- **Universe Discovery**: `.[] | select(.universeDetails.nodeDetailsSet | any(.cloudInfo.private_ip == "IP")) | .universeUUID`
- **Master Addresses**: `[.masters[] | "\(.cloudInfo.private_ip):\(.masterRpcPort)"] | join(",")`

## Error Handling

### Validation Checks
- Required variables (`YBA_HOST`, `API_TOKEN`) are validated before API calls
- API responses are checked for null/empty values
- Graceful fallback to configuration file if API fails

### Debug Output
- All API calls include debug logging to stderr
- Master IP discovery process is logged
- Server type detection results are displayed

## Security Features

### SSL/TLS Support
- Uses `-sk` flag for curl (skip certificate verification)
- Configurable SSL version and certificate paths
- Secure API token authentication

### Network Security
- Private IP address detection
- Secure API communication
- Certificate-based authentication for admin tools

## Usage Examples

### Basic Setup
```bash
# Set required environment variables
export YBA_HOST="https://yba.example.com"
export API_TOKEN="your-api-token"

# Source the bashrc
source ~/.bashrc
```

### Configuration File Example
```bash
# Create ~/yb-config.rc
export YBA_HOST="https://yba.example.com"
export API_TOKEN="your-api-token"
export UNIVERSE_UUID="optional-specific-universe-uuid"
```

### Manual Override
```bash
# Override automatic detection
export MASTERS="192.168.1.10:7100,192.168.1.11:7100,192.168.1.12:7100"
```

## Dependencies

### Required Tools
- `curl`: API communication
- `jq`: JSON parsing
- `awk`: Text processing
- `grep`: Pattern matching

### Optional Tools
- `systemctl`: Service management (if using systemd)
- `cqlsh`: Cassandra Query Language shell
- `ysqlsh`: YugabyteDB SQL shell
- `yb-admin`: YugabyteDB administration tool

## Troubleshooting

### Common Issues
1. **API Token Issues**: Check `API_TOKEN` environment variable
2. **Network Connectivity**: Verify `YBA_HOST` is accessible
3. **IP Detection**: Ensure current IP is in universe node list
4. **Configuration File**: Check `$HOME/tserver/conf/server.conf` exists and is readable

### Debug Mode
- All debug messages are sent to stderr
- Check debug output for API responses and variable values
- Use `echo "DEBUG:MASTERS: $MASTERS"` to verify master address discovery

## Best Practices

1. **Environment Variables**: Use configuration files for sensitive data
2. **API Tokens**: Rotate tokens regularly and use minimal permissions
3. **Network Security**: Use HTTPS for YBA_HOST and secure network connections
4. **Logging**: Monitor debug output for troubleshooting
5. **Backup**: Keep configuration files backed up and version controlled
