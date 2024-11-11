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

2. External program sends `"LISTUSERS":{}`

3. *yba-user-admin* responds with:
    ```
    "USERS":[
        {"email":"user1@email1.com","other":"attributes"},
        ...
    ]
    ```

4. External program sends `"ADDUSERS":[{"email":...}]

5.  *yba-user-admin* responds with: `"OK":"5 Users added"` 

6.  ... Similar for **DELETEUSERS**
  
