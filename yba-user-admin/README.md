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
The intended purpose of this feature is to run THIS program as a child-process of a controlling program that feeds data into STDIN, and processes responses from THIS.

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
        {"email":"u2@eee" ...}
    ]}
  ```

5.  *yba-user-admin* responds with: `["OK":"5 Users added"]` 

6.  ... Similar for **DELETEUSERS**
  
## JSON Stream "LISTUSERS" test

Use this to test the JSON stream function:
Note the `-s` option is used to read from STDIN.
Input is expected in JSON format, so the "LISTUSERS" must arrive with the quotes.
```
$ echo '"LISTUSERS"' |  python   ~/yb-tools/yba-user-admin/yba-user-admin.py  -y https://10.231.0.56/ -a d6465305-2419-4597-ae16-2c0b712ce209  -s
[
{"email":"vanand@yugabyte.com","uuid":"0e0f75ea-7933-4626-a61a-ea9f5add074b","role":"SuperAdmin","created":"2024-02-14T17:36:33Z"}
{"email":"test@somewhere","uuid":"2f6cc436-1f5b-4fc1-8f1d-9e29c17111fc","role":"ReadOnly","created":"2024-11-08T21:45:27Z"}
{"email":"test2@somewhere","uuid":"98422c5b-ee82-41f0-aa72-3022fd2eb965","role":"ReadOnly","created":"2024-11-08T21:54:54Z"}
{"email":"test3@somewhere","uuid":"07b0fd3c-d274-4e41-9436-0a9b1458937f","role":"ReadOnly","created":"2024-11-08T21:56:36Z"}
]
```

## JSON Stream "LISTUSERS"  AND "DELETEUSERS" test
Two JSON commands are sent , and performed in sequence

```
$ echo -e '"LISTUSERS"\n{"DELETEUSERS":[\n{"email":"u1@rrr","role":"ReadOnly","password":"P@12345d67"},\n{"email":"u2@eee","role":"ReadOnly","password":"P#12w34567"}]}' | python   ~/yb-tools/yba-user-admin/yba-user-admin.py  -y https://10.231.0.56/ -a d6465305-2419-4597-ae16-2c0b712ce209 -s
[
{"email":"vanand@yugabyte.com","uuid":"0e0f75ea-7933-4626-a61a-ea9f5add074b","role":"SuperAdmin","created":"2024-02-14T17:36:33Z"}
{"email":"test@somewhere","uuid":"2f6cc436-1f5b-4fc1-8f1d-9e29c17111fc","role":"ReadOnly","created":"2024-11-08T21:45:27Z"}
{"email":"test2@somewhere","uuid":"98422c5b-ee82-41f0-aa72-3022fd2eb965","role":"ReadOnly","created":"2024-11-08T21:54:54Z"}
{"email":"test3@somewhere","uuid":"07b0fd3c-d274-4e41-9436-0a9b1458937f","role":"ReadOnly","created":"2024-11-08T21:56:36Z"}
{"email":"u1@rrr","uuid":"f0d75e9e-8e27-4ac8-a0fb-9b198ac2b248","role":"ReadOnly","created":"2024-11-12T03:23:39Z"}
{"email":"u2@eee","uuid":"a38c53c5-298c-417c-9d26-b6266d182403","role":"ReadOnly","created":"2024-11-12T03:23:39Z"}
]
{"success":true,"Operation":"delete","msg":"2 users deleted"}

```