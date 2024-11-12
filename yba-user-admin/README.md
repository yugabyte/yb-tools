&nbsp;
# YBA User Admin
      \ \   )  __ )     \       |   |                                 |            _)
       \   )   __ \    _ \      |   |   __|   _ \   __|     _` |   _` |  __ `__ \   |  __ \
          |    |   |  ___ \     |   | \__ \   __)  |      (    |  (   |  |   |   |  |  |   |
         _|   ____) _/    _\   \___/  ____) \___| _|      \__,_| \__,_| _|  _|  _| _| _|  _|

Command-line to Add / Delete / List user accounts in YBA.

## Prerequisites

This program uses standard Python3 libraries, including @dataclass.

It accesses the YBA API documented in https://api-docs.yugabyte.com/docs/yugabyte-platform/18e7f5bab7963-list-all-users .
You will need the following information:
* YBA URL (the URL used to access the YBA host)
* API token : You can get this from the login user's profile page

## Usage
usage: yba-user-admin.py v 0.06 [-h] [-d] -y YBA_URL [-c CUSTOMER_UUID] -a AUTH_TOKEN [-ls | -rm User@email.addr | -mk User@email.addr] [--role ROLE] [-p PASSWORD] [-s]

This Program operates on YBA users (List/Create/Delete)

```
options:
  -h, --help            show this help message and exit
  -d, --debug
  -y YBA_URL, --yba_url YBA_URL
                        YBAnywhere URL
  -c CUSTOMER_UUID, --customer_uuid CUSTOMER_UUID
                        Customer UUID: Auto-discovered if unspecified
  -a AUTH_TOKEN, --auth_token AUTH_TOKEN
                        YBAnywhere Auth Token
  -ls, --list           List users (This is the default action)
  -rm User@email.addr, --remove User@email.addr, --delete User@email.addr
                        Delete the specified user
  -mk User@email.addr, --make User@email.addr, --create User@email.addr
                        Add a new user. role & password reqd
  --role ROLE           Name of role to apply to new user
  -p PASSWORD, --password PASSWORD
                        Password for new user (>=8 ch, Upcase+num+special)
  -s, --stdin           Read JSON stream from stdin
```

## Basic usage (List users)

python3 yba-user-admin.py  **-y** https://\<yba-host\>/ **-a** \<access-token\> [**-ls**]

## Add (Create) a new user

python3 yba-user-admin.py  **-y** https://\<yba-host\>/ **-a** \<access-token\> **--mk** User@email **--role** ROLE-NAME **--Password** PASSWORD

You can use **--create**  or **--make** instead of **--mk**

## Delete (Remove) a user

python3 yba-user-admin.py  **-y** https://\<yba-host\>/ **-a** \<access-token\> **--rm** User@email 

You can use **--remove** or **--delete** instead of **--rm**

## Advanced: Batch / External command directed operation

The script supports commands sent via an external stream, when the **-s** (**--stdin**) option is used.
In this case, the *yba-user-admin* script will read a **JSON** stream from **STDIN**, and process the commands supplied, and place **JSON** in the output stream **STDOUT**.

A typical sequence would be:

1. External program starts *yba-user_admin* and sets **-y**, **-a** and **-s**.
   It also pipes **STDIN** and **STDOUT** from *yba-user-admin** to allow *Write* and *Read* streams.

2. External program sends `"LISTUSERS"`

3. *yba-user-admin* responds with:
    ```
    [
        {"email":"user1@email1.com","other":"attributes"},
        ...
    ]
    ```

4. External program sends
  ```
     {"ADDUSERS":[
        {"email":"u1@rrr","role":"Admin","password","P@ssword123"},
        {"email":"u2@eee"}
    ]}
  ```

5.  *yba-user-admin* responds with: `["OK":"5 Users added"]` 

6.  ... Similar for **DELETEUSERS**
  
## JSON Stream test

Use this to test the JSON stream function:

```
echo -e '"LISTUSERS"\n{"ADDUSERS":[\n{"email":"u1@rrr","role":"ReadOnly","password":"P@12345d67"},\n{"email":"u2@eee","role":"ReadOnly","password":"P#12w34567"}]}' \
   | python   ~/yb-tools/yba-user-admin/yba-user-admin.py  -y https://Your-YBA-HOST/ -a Your-api-token-value -d -s
```
Debug output:
```
DEBUG: 2024-11-11T17:52:26.094814 Debugging Enabled
DEBUG: API Get:https://Your-YBA-HOST//api/v1/customers (RAW=False)
DEBUG: API Get:https://Your-YBA-HOST//api/v1/customers/ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f/runtime_config/00000000-0000-0000-0000-000000000000/key/yb.rbac.use_new_authz (RAW=True)
DEBUG: RoleManagement: use_new_authz=False
DEBUG:json piece:LISTUSERS [<class 'str'> 11 bytes]
DEBUG:Processing string command:LISTUSERS
DEBUG: API Get:https://Your-YBA-HOST//api/v1/customers/ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f/users (RAW=False)
DEBUG: RoleManagement: Creating Role object for SuperAdmin
DEBUG: RoleManagement: Creating Role object for ReadOnly
USER vanand@yugabyte.com(0e0f75ea-7933-4626-a61a-ea9f5add074b) Role:SuperAdmin  Created:2024-02-14T17:36:33Z
USER test@somewhere(2f6cc436-1f5b-4fc1-8f1d-9e29c17111fc) Role:ReadOnly  Created:2024-11-08T21:45:27Z
USER test2@somewhere(98422c5b-ee82-41f0-aa72-3022fd2eb965) Role:ReadOnly  Created:2024-11-08T21:54:54Z
USER test3@somewhere(07b0fd3c-d274-4e41-9436-0a9b1458937f) Role:ReadOnly  Created:2024-11-08T21:56:36Z
DEBUG:json piece:{'ADDUSERS': [{'email': 'u1@rrr', 'role': 'ReadOnly', 'password': 'P@12345d67'}, {'email': 'u2@eee', 'role': 'ReadOnly', 'password': 'P#12w34567'}]} [<class 'dict'> 138 bytes]
DEBUG:Processing dict command:{'ADDUSERS': [{'email': 'u1@rrr', 'role': 'ReadOnly', 'password': 'P@12345d67'}, {'email': 'u2@eee', 'role': 'ReadOnly', 'password': 'P#12w34567'}]}
DEBUG: Creating User u1@rrr in YBA
DEBUG: API Post:https://Your-YBA-HOST//api/v1/customers/ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f/users
DEBUG: API Request successful
'{"uuid":"d3389290-964d-4429-aa74-37c237e288e5","customerUUID":"ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f","email":"u1@rrr","creationDate":"2024-11-12T01:52:28Z","role":"ReadOnly","userType":"local","ldapSpecifiedRole":false,"primary":false,"features":{"universe":{"import":"disabled","create":"disabled"},"config":{"infra":"disabled","backup":"disabled"},"menu":{"config":"disabled"},"health":{"configure":"disabled"},"alert":{"list":{"actions":"disabled"},"configuration":{"actions":"disabled"},"destinations":{"actions":"disabled"},"channels":{"actions":"disabled"}},"universes":{"details":{"overview":{"upgradeSoftware":"hidden","editGFlags":"hidden","editUniverse":"hidden","readReplica":"hidden","manageEncryption":"hidden","restartUniverse":"hidden","pausedUniverse":"disabled","deleteUniverse":"hidden","costs":"hidden","systemdUpgrade":"hidden"}},"tableActions":"disabled","actions":"disabled","backup":"disabled"},"administration":{"highAvailability":"hidden"},"main":{"stats":"hidden"},"profile":{"profileInfo":"disabled"}}}'
USER u1@rrr(d3389290-964d-4429-aa74-37c237e288e5) Role:ReadOnly  Created:2024-11-12T01:52:28Z
DEBUG: Creating User u2@eee in YBA
DEBUG: API Post:https://Your-YBA-HOST//api/v1/customers/ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f/users
DEBUG: API Request successful
'{"uuid":"e310d795-53be-480e-8b1f-8edbf654576d","customerUUID":"ac21ccc1-6a1c-4c6a-bf13-acc07d7cfc3f","email":"u2@eee","creationDate":"2024-11-12T01:52:29Z","role":"ReadOnly","userType":"local","ldapSpecifiedRole":false,"primary":false,"features":{"universe":{"import":"disabled","create":"disabled"},"config":{"infra":"disabled","backup":"disabled"},"menu":{"config":"disabled"},"health":{"configure":"disabled"},"alert":{"list":{"actions":"disabled"},"configuration":{"actions":"disabled"},"destinations":{"actions":"disabled"},"channels":{"actions":"disabled"}},"universes":{"details":{"overview":{"upgradeSoftware":"hidden","editGFlags":"hidden","editUniverse":"hidden","readReplica":"hidden","manageEncryption":"hidden","restartUniverse":"hidden","pausedUniverse":"disabled","deleteUniverse":"hidden","costs":"hidden","systemdUpgrade":"hidden"}},"tableActions":"disabled","actions":"disabled","backup":"disabled"},"administration":{"highAvailability":"hidden"},"main":{"stats":"hidden"},"profile":{"profileInfo":"disabled"}}}'
USER u2@eee(e310d795-53be-480e-8b1f-8edbf654576d) Role:ReadOnly  Created:2024-11-12T01:52:29Z
DEBUG:ACCUMULATED JSON FROM STDIN:=====
['LISTUSERS', {'ADDUSERS': [{'email': 'u1@rrr', 'role': 'ReadOnly', 'password': 'P@12345d67'}, {'email': 'u2@eee', 'role': 'ReadOnly', 'password': 'P#12w34567'}]}]
```