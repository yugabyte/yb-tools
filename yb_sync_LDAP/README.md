# README for yb_snc_ldap.py

This script synchronizes User accounts from Active Directory / LDAP
to Yugabyte CQL or SQL roles.

It has an option to synchronize LDAP users with an external user system, like the YBA users.

As users are added/removed to/from a specified LDAP group,
corresponding roles are added/deleted to YCQL or YSQL.

* Python 3.6 or higher is required
* The script requires a number of python modules which, by default are installed on the YBA node
* The script also requires postgres system libraries to be installed (`libpq.so.5()(64bit)`)

## Run instructions/help

```
usage: yb_sync_ldap.py [-h] [--debug] [--verbose] [--apihost APIHOST] [--apitoken APITOKEN] [--use_https] [--apiport APIPORT] 
                       [--apiuser APIUSER] [--apipassword APIPASSWORD] [--ipv6] [--target_api YCQL|YSQL|EXTERNAL] 
                       [--externalcommand EXTERNALCOMMAND] [--universe_name UNIVERSE_NAME] [--dbhost DBHOST] [--dbuser DBUSER]
                       [--dbpass DBPASS] [--dbname DBNAME] [--db_sslmode {disable,allow,prefer,require,verify-ca,verify-full}] 
                       [--db_certpath DB_CERTPATH] [--db_certkey DB_CERTKEY] --ldapserver LDAPSERVER 
                       --ldapuser LDAPUSER --ldap_password LDAP_PASSWORD --ldap_search_filter LDAP_SEARCH_FILTER 
                       --ldap_basedn   dc=dept,dc=corp.. --ldap_userfield LDAP_USERFIELD --ldap_groupfield LDAP_GROUPFIELD
                        [--ldap_certificate LDAP_CERTIFICATE] [--ldap_tls] [--dryrun] 
                        [--reports COMMA,SEP,RPT...] [--allow_drop_superuser] [--member_map MMAP [MMAP ...]]

YB LDAP sync script, Version 0.50

options:
  -h, --help            show this help message and exit
  --debug               Enable debug logging (including http request logging) for this script
  --verbose             Enable verbose logging for each action taken in this script
  --apihost APIHOST     YBA/YW API Hostname or IP (Defaults to localhost)
  --apitoken APITOKEN   YW API TOKEN - Preferable to use this instead of apiuser+apipassword
  --use_https           YW API http type : Set for https (default is false (http))
  --apiport APIPORT     YW API PORT: Defaults to 9000, which is valid if running inside docker. For external, use 80 or 443
  --apiuser APIUSER     YW API Username
  --apipassword APIPASSWORD
                        YW API Password
  --ipv6                Is system ipv6 based
  --target_api YCQL|YSQL|EXTERNAL
                        Target API: YCQL, YSQL or EXTERNAL
  --externalcommand EXTERNALCOMMAND, --externalcmd EXTERNALCOMMAND
                        command+params to run to external user sync plugin
  --universe_name UNIVERSE_NAME
                        Universe name
  --dbhost DBHOST       Database hostname of IP. Uses a random YB node if not specified.
  --dbuser DBUSER       Database user to connect as
  --dbpass DBPASS       Password for dbuser
  --dbname DBNAME       YSQL database name to connect to
  --db_sslmode {disable,allow,prefer,require,verify-ca,verify-full}
                        SSL mode for YSQL TLS
  --db_certpath DB_CERTPATH
                        SSL certificate path for YSQL TLS
  --db_certkey DB_CERTKEY
                        SSL key path for YSQL TLS
  --ldapserver LDAPSERVER
                        LDAP server address. Should be prefaced with ldap://hostname
  --ldapuser LDAPUSER   LDAP Bind DN for retrieving directory information
  --ldap_password LDAP_PASSWORD
                        LDAP Bind DN password
  --ldap_search_filter LDAP_SEARCH_FILTER
                        LDAP Search filter, like '(&(objectclass=group)(|(samaccountname=grp1)...))'
  --ldap_basedn dc=dept,dc=corp..
                        LDAP BaseDN to search
  --ldap_userfield LDAP_USERFIELD
                        LDAP field to determine user's id to create
  --ldap_groupfield LDAP_GROUPFIELD
                        LDAP field to grab group information (e.g. cn)
  --ldap_certificate LDAP_CERTIFICATE
                        File location that points to LDAP certificate
  --ldap_tls            LDAP Use TLS
  --dryrun              Show list of potential DB role changes, but DO NOT apply them
  --reports COMMA,SEP,RPT...
                        One or a comma separated list of 'tree' reports. Eg: LDAPRAW,LDAPBYUSER,LDAPBYGROUP,DBROLE,DBUPDATES or ALL
  --allow_drop_superuser
                        Allow this code to DROP a superuser role if absent in LDAP
  --member_map MMAP [MMAP ...]
                        Additional YSQL roles to add users to - in the form of [<user regex> <rolename>]

```
## Bash wrapper script
Here is a sample bash script that allows the LDAP sync to be automated if you put this in crontab:

```
#!/bin/bash

# Get Environment Parameters
export TARGETAPI=YSQL
source /path/to/xxx.rc # Set necessary global environmental variables

export UNIVERSE_NAME=UNIVERSE_NAME
export DBNAME='database_name'
export LDAP_SEARCH_FILTER='(&(objectclass=group)(|(samaccountname=my_1st_ldap_group_name)(samaccountname=)(samaccountname=my_2nd_ldap_group_name)(samaccountname=my_3rd_ldap_group_name)))'

python3 /path/to/scripts/yb_sync_ldap.py \
  --apihost $YBA_HOST \
  --apitoken $API_TOKEN \
  --apiport $APIPORT \
  --use_https \
  --target_api $TARGETAPI \
  --universe_name $UNIVERSE_NAME \
  --dbuser $DBUSER \
  --dbpass $DBPASS \
  --ldapserver $LDAPSERVER \
  --ldapuser $LDAPUSER \
  --ldap_password $LDAP_PASSWORD \
  --ldap_search_filter $LDAP_SEARCH_FILTER \
  --ldap_basedn $LDAP_BASEDN \
  --ldap_userfield $LDAP_USERFIELD \
  --ldap_groupfield $LDAP_GROUPFIELD \
  --allow_drop_superuser \
  --member_map 'ad\.*' ldap_people_users \
  --member_map 'svc\.*' ldap_service_users \
  --reports ALL
  #--dryrun
  #--debug
```
## Notes

* Several python modules are required .. see the script for details
* Use `ldapsearch` to verify your LDAP credentials and `search filter`
* It also needs credentials for the YBA API (token and URL)
* It needs the name and credentials for the database you want to sync
* If you want to sync the YBA user list with ldap, use `--target_api=EXTERNAL`
  and `--externalcommand '<path-and options-for-sync-script>'`
  There is a separate companion `yba-user-admin.py` script for this purpose

## parameters Details

Parameters can be passed either from the command-line, via environment variables, or via a ".rc" file.

The .rc file is a configuration file that contains environment variables used by the yb_sync_ldap.py script. It allows you to define parameters in advance, simplifying the script's execution by avoiding the need to pass all arguments via the command line.

The .rc file is not automatically loaded by the script. Instead, it must be sourced manually in the shell or a wrapper script before running the Python script. For example:
`source /path/to/xxx.rc`

Expected Contents of the .rc File
The .rc file should define all required parameters as environment variables. Each line should follow the format:
`export VARIABLE_NAME=value`

Example .rc File:
```
# Yugabyte API Configuration
export APIHOST=localhost
export APITOKEN=your_api_token
export APIPORT=9000
export USE_HTTPS=true

# Database Configuration
export DBHOST=127.0.0.1
export DBUSER=admin
export DBPASS=your_password
export DBNAME=yugabyte

# LDAP Configuration
export LDAPSERVER=ldap://ldap.example.com
export LDAPUSER=cn=admin,dc=example,dc=com
export LDAP_PASSWORD=your_ldap_password
export LDAP_SEARCH_FILTER='(&(objectclass=group)(|(samaccountname=group1)(samaccountname=group2)))'
export LDAP_BASEDN=dc=example,dc=com
export LDAP_USERFIELD=uid
export LDAP_GROUPFIELD=cn

# Additional Options
export TARGET_API=YSQL
export UNIVERSE_NAME=my_universe
export REPORTS=ALL
export ALLOW_DROP_SUPERUSER=false
```

If a variable is not defined in the .rc file or the environment, the script uses default values specified in the argparse configuration.



### General Parameters
--debug
Enables debug logging, including HTTP request logging. Default: False.
Environment Variable: `DEBUG`

--verbose
Enables verbose logging for each action taken by the script. Default: False.
Environment Variable: `VERBOSE`

--dryrun
Displays the list of potential database role changes without applying them. Default: False.
Environment Variable: `DRYRUN`

--reports
Specifies one or more comma-separated reports to generate.
Options: `LDAPRAW`, `LDAPBYUSER`, `LDAPBYGROUP`, `DBROLE`, `DBUPDATES`, or `ALL`. Default: "".
Environment Variable: `REPORTS`
The LDAP reports help visualize the mappiing of users and group-names.

--allow_drop_superuser
Allows the script to drop superuser roles if they are absent in LDAP. Default: False.
Environment Variable: `ALLOW_DROP_SUPERUSER`

### Yugabyte API Parameters
--apihost
The hostname or IP address of the Yugabyte API. Default: localhost.
Environment Variable: `APIHOST`

--apitoken
The API token for authenticating with the Yugabyte API.
Environment Variable: `APITOKEN`
You can obtain this from the YBA UI, from the "User Profile".

--apiuser
The username for authenticating with the Yugabyte API.
Environment Variable: `APIUSER`

--apipassword
The password for authenticating with the Yugabyte API.
Environment Variable: `APIPASSWORD`

--apiport
The port for the Yugabyte API. Default: 9000.
Environment Variable: `APIPORT`

--use_https
Enables HTTPS for API communication. Default: False.
Environment Variable: `USE_HTTPS`

### Database Parameters
--target_api
Required. Specifies the target API for synchronization.
Options: `YCQL`, `YSQL`, or `EXTERNAL`. Default: `YCQL`.
Environment Variable: `TARGET_API`

--universe_name
The name of the Yugabyte universe to synchronize. (Not used when target_api is `EXTERNAL`)
Environment Variable: `UNIVERSE_NAME`

--dbhost
The hostname or IP address of the database. If not specified, a random node from the universe is used.
Environment Variable: `DBHOST`

--dbuser
The database user with privileges to create/modify roles.
Environment Variable: `DBUSER`

--dbpass
The password for the database user.
Environment Variable: `DBPASS`

--dbname
The name of the YSQL database to connect to. Default: yugabyte.
Environment Variable: `DBNAME`

--db_sslmode
The SSL mode for YSQL connections. Options: disable, allow, prefer, require, verify-ca, verify-full.
Environment Variable: `DB_SSL_MODE`

--db_certpath
The path to the SSL certificate for YSQL connections.
Environment Variable: `DB_CERTPATH`

--db_certkey
The path to the SSL key for YSQL connections.
Environment Variable: `DB_CERTKEY`

### LDAP Parameters
Please confirm the validity of the values you specify here, by using `LDAPSEARCH`.

--ldapserver
The LDAP server address, prefixed with ldap:// or ldaps://.
Environment Variable: `LDAPSERVER`

--ldapuser
The LDAP Bind DN for retrieving directory information.
Environment Variable: `LDAPUSER`

--ldap_password
The password for the LDAP Bind DN.
Environment Variable: `LDAP_PASSWORD`

--ldap_search_filter
The LDAP search filter, e.g., (&(objectclass=group)(|(samaccountname=grp1)...)).
Environment Variable: `LDAP_SEARCH_FILTER`

--ldap_basedn
The Base DN for LDAP searches, e.g., dc=dept,dc=corp.
Environment Variable: `LDAP_BASEDN`

--ldap_userfield
The LDAP field used to determine the user's ID. Typically `samAccountName` 
Environment Variable: `LDAP_USERFIELD`

--ldap_groupfield
The LDAP field used to retrieve group information, e.g., cn or samAccountName.
Environment Variable: `LDAP_GROUPFIELD`

--ldap_certificate
The path to the LDAP server's certificate file.
Environment Variable: `LDAP_CERTIFICATE`

--ldap_tls
Enables TLS for LDAP connections. Default: False.
Environment Variable: `LDAP_TLS`

--member_map MMAP [MMAP ...]
Defines mapping for role names from LDAP to the target (SQL,CQL or EXTERNAL) role.
Specified  in the form of <ldap regex> <rolename> 
The <ldap regex> identifies the group name in LDAP - it can be a regex.
THe rolename is the role in the target API.
In the case of EXTERNAL, the `rolename` can be a regex containing a replacement string.
Eg: `--member map 'ad\.(.+)' '\1@company,com'`   : This converts `ad.firstname.last` to `firstname.last@company.com`


--ldap_testvalue (Hidden)
A stringified LDAP search result for testing/debugging purposes only. NOT for normal use.
When specified, the code will NOT connect to LDAP - but will use this value.
Environment Variable: `LDAP_TESTVALUE`

### External Plugin Parameters
The "External plugin" is used as a plug-in to synchronize LDAP to any external user account system.
In this example, the YBA user accounts/roles can be updated based on corresponding LDAP groups by using the `yba-user-admin.py` script.
The basic idea is that THIS script requests the list of user accounts from the external system, then fetches and matches LDAP accounts, and calls
the external script again with a list of changes to be made (ADDUSER/DELETEUSER). When ADDing, the mapped role is also specified.

--externalcommand
The command and parameters to run an external user sync plugin. Required if `--target_api` is set to `EXTERNAL`.
Environment Variable: None
eg: `--externalcommand "python /path/to/yba-user-admin.py -s"`
