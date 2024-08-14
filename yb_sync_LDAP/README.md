# README for yb_snc_ldap.py

This script synchronizes User accounts from Active Directory / LDAP
to Yugabyte CQL or SQL roles.

As users are added/removed to/from a specified LDAP group,
corresponding roles are added/deleted to YCQL or YSQL.

* Python 3.6 or higher is required
* The script requires a number of python modules which, by default are installed on the YBA node
* The script also requires postgres system libraries to be installed (`libpq.so.5()(64bit)`)

## Run instructions/help

```
usage: yb_sync_ldap.py [-h] [--debug] [--apihost APIHOST]
                       [--apitoken APITOKEN] [--use_https] [--apiport APIPORT]
                       [--apiuser APIUSER] [--apipassword APIPASSWORD]
                       [--ipv6] [--target_api {YCQL,YSQL}] --universe_name
                       UNIVERSE_NAME --dbuser DBUSER --dbpass DBPASS
                       [--dbname DBNAME]
                       [--db_sslmode {disable,allow,prefer,require,verify-ca,verify-full}]
                       [--db_certpath DB_CERTPATH] [--db_certkey DB_CERTKEY]
                       --ldapserver LDAPSERVER --ldapuser LDAPUSER
                       --ldap_password LDAP_PASSWORD --ldap_search_filter
                       LDAP_SEARCH_FILTER --ldap_basedn LDAP_BASEDN
                       --ldap_userfield LDAP_USERFIELD --ldap_groupfield
                       LDAP_GROUPFIELD [--ldap_certificate LDAP_CERTIFICATE]
                       [--ldap_tls]

YB LDAP sync script, Version 0.44

optional arguments:
  -h, --help            show this help message and exit
  --debug               Enable debug logging (including http request logging)
                        for this script
  --apihost APIHOST     YBA/YW API Hostname or IP (Defaults to localhost)
  --apitoken APITOKEN   YW API TOKEN - Preferable to use this instead of
                        apiuser+apipassword
  --use_https           YW API http type : Set for https (default is false
                        (http))
  --apiport APIPORT     YW API PORT: Defaults to 9000, which is valid if
                        running inside docker. For external, use 80 or 443
  --apiuser APIUSER     YW API Username
  --apipassword APIPASSWORD
                        YW API Password
  --allow_drop_superuser
                      Allow this code to DROP a superuser role if absent in LDAP
  --ipv6                Is system ipv6 based
  --target_api {YCQL,YSQL}
                        Target API: YCQL or YSQL
  --universe_name UNIVERSE_NAME
                        Universe name
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
                        LDAP server address. Should be prefaced with
                        ldap://hostname
  --ldapuser LDAPUSER   LDAP Bind DN for retrieving directory information
  --ldap_password LDAP_PASSWORD
                        LDAP Bind DN password
  --ldap_search_filter LDAP_SEARCH_FILTER
                        LDAP Search filter
  --ldap_basedn LDAP_BASEDN
                        LDAP BaseDN to search
  --ldap_userfield LDAP_USERFIELD
                        LDAP field to determine user's id to create
  --ldap_groupfield LDAP_GROUPFIELD
                        LDAP field to grab group information (e.g. cn)
  --ldap_certificate LDAP_CERTIFICATE
                        File location that points to LDAP certificate
  --ldap_tls            LDAP Use TLS
  --member_map REGEX ROLE
                        Add users who meet the REGEX in the specified ROLE (NOLOGIN)
  --reports <csv-list>
                        One or a comma separated list of 'tree' reports. 
                        Eg: LDAPRAW,LDAPBYUSER,LDAPBYGROUP,DBROLE,DBUPDATES or ALL
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
