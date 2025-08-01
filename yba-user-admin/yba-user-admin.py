#!/usr/bin/python3
# YBA User list/creation/Deletion
version = "0.15"
"""
Command-line management of YBA users
====================================

You can CREATE, DELETE, LIST users by passing the arguments:
        -mk     -rm     -ls
use --help to get more useful info.

This uses the YBA API to manage users, so you need the following:
* API_TOKEN  : You can get/generate this  from the "user profile" in the UI
* YBA_HOST   : THE URL used to access the YBA .. includes http:// or https://

This requires @dataclass decorators. See https://docs.python.org/3/library/dataclasses.html

The "--stdin" or "-s" option allows you to pass a JSON stream of commands to the program.
The JSON stream can contain a list of commands, or a single command.
The commands are:
* LISTUSERS: List all users in the YBA
* ADDUSERS: Add users to the YBA. This is a dict of email:role pairs, eg:
    {"ADDUSERS": {" 
* DELETEUSERS: Delete users from the YBA. This is a list of emails, eg:
    {"DELETEUSERS": ["

* ADDUSERS and DELETEUSERS can be combined in a single JSON object, eg:

To TEST this, you can use the following commands:
    echo '["LISTUSERS"]'  | ./yba-user-admin.py -s  --log my-local.log --debug --yba_url http://localhost:7000 --auth_token 1234

    echo -e '"LISTUSERS"\n{"ADDUSERS": {"u12@rrr":"ReadOnly","u13@rrr":"ReadOnly"},"PASSWORD":"Some_Pa*ss"}' | python3 ./yba-user-admin.py -s  --log junk.1 ..

"""
from ast import Dict, arg, parse
import requests
import urllib3
import json
import argparse
import re
import sys
import os
from dataclasses import dataclass,field
#from typing import Optional
from typing import List
from json import JSONDecoder, JSONEncoder
from pprint import pprint
from pprint import pformat
from datetime import datetime, timezone
import logging

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class RoleManagement: # Forward declaration to help with circular reference
    pass

@dataclass
class YBA_API():
    yba_url: str
    auth_token: str = field(default= None)
    customer_uuid: str =field(default= None)
    raw_response: str = field(default= None)
    debug:bool = False
    session = None
    roleManagement:RoleManagement = None
    UserList = []

    def __post_init__(self):
        self.yba_url  = self.yba_url.strip("/")
        self.raw_response = None
        self.session = requests.Session()
        self.session.verify = False
        if self.customer_uuid is None:
            self.__set_customeruuid()

        self.cust_url = self.yba_url + '/api/v1/customers/' + self.customer_uuid
        self.univ_url = self.cust_url + '/universes'
        self.roleManagement = RoleManagement(self)


    def __set_customeruuid(self):
        customer_list = self.Get(self.yba_url + '/api/v1/customers')
        if len(customer_list) > 1:
            raise SystemExit("ERROR: Multiple Customers Found, Must Specify which Customer UUID to Use!")
        self.customer_uuid = customer_list[0]['uuid']

    def Get(self, targeturl="", raw=False):
        logging.debug("DEBUG: API Get:"+targeturl + " (RAW="+str(raw)+")")
        self.raw_response = self.session.get(targeturl, headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.auth_token},
                                         verify=False)
        self.raw_response.raise_for_status()
        # Convert response data to dict
        if raw:
            return self.raw_response.text
        return json.loads(self.raw_response.text)
    
    def Get_User_List(self):
        user_json = self.Get(self.cust_url+"/users")
        self.UserList = [] # Zap it 
        for u in user_json:
            role =  self.roleManagement.Get_or_create_role_by_name(u["role"], allow_create=True)
            self.UserList.append(User(uuid=u["uuid"],email=u["email"],creationDate=u["creationDate"],role=role)) # User objects
        return self.UserList

    def Post(self, url:str, data, extra_headers=None, timeout:int=2):
        # avoids no CSRF token error by emptying the cookie jar
        # session.cookies = requests.cookies.RequestsCookieJar()
        logging.debug("DEBUG: API Post:"+url)
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.auth_token}
        if extra_headers is not None:
            headers.update(extra_headers)
        # avoids no CSRF token error by emptying the cookie jar
        self.session.cookies = requests.cookies.RequestsCookieJar()
        self.raw_response = self.session.post(url, json=data, headers=headers, timeout=timeout, verify=False)
        if self.raw_response.status_code == 200:
            logging.debug('DEBUG: API Request successful')
        else:
            print(self.raw_response.text)
            self.raw_response.raise_for_status()
        return self.raw_response.text
    
    def Delete(self, url:str, timeout:int=2):
        logging.debug("DEBUG: API Delete:"+url)
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.auth_token}
        # avoids no CSRF token error by emptying the cookie jar
        self.session.cookies = requests.cookies.RequestsCookieJar()
        self.raw_response = self.session.delete(url, headers=headers, timeout=timeout, verify=False)
        if self.raw_response.status_code == 200:
            logging.debug('DEBUG: API Request successful')
        else:
            print(self.raw_response.text)
            self.raw_response.raise_for_status()
        return self.raw_response.text
#=====================================================================================================

@dataclass
class Role:
    yba_api: YBA_API
    mgt: RoleManagement 
    name:str =  field(default=None)
    uuid: str = field(default=None)

#=====================================================================================================
#https://docs.yugabyte.com/preview/yugabyte-platform/administer-yugabyte-platform/anywhere-rbac/#manage-custom-roles
@dataclass
class RoleManagement:
    yba_api: YBA_API
    role_by_name: Dict = field(init=False)
    use_new_authz: bool = False
    new_authz_uri:str = "/runtime_config/00000000-0000-0000-0000-000000000000/key/yb.rbac.use_new_authz"


    def __post_init__(self):
        self.yba_api.roleManagement = self
        self.use_new_authz = self.Fine_grained_RBAC()
        logging.debug("DEBUG: RoleManagement: use_new_authz="+str(self.use_new_authz))
        self.role_by_name  = {}
        #if self.use_new_authz:
        role_list = self.yba_api.Get(self.yba_api.cust_url + "/rbac/role")
        for r in role_list:
            self.role_by_name[ r["name"] ] = Role(mgt=self,yba_api=self.yba_api,name=r["name"],uuid=r["roleUUID"])


    def Get_or_create_role_by_name(self,name:str,allow_create:bool=True) -> Role:
        if self.role_by_name.get(name) is None:
            if allow_create:
                logging.debug("DEBUG: RoleManagement: Creating Role object for "+name)
                self.role_by_name[name] = Role(mgt=self,yba_api=self.yba_api,name=name) # Returns new object 
            else:
                raise  ValueError('No such role:'+name)
        return self.role_by_name[name]

    def Fine_grained_RBAC(self,set_to:bool = None) -> bool:
        if set_to is None: # get & return current value
            use_new_authz_text = self.yba_api.Get(self.yba_api.cust_url
                                                + self.new_authz_uri
                                                ,raw=True)
            return use_new_authz_text.upper() == "TRUE"
        #curl --request PUT \
        #    --url http://{yba_host:port}/api/v1/customers/{customerUUID}/runtime_config/00000000-0000-0000-0000-000000000000/key/yb.rbac.use_new_authz \
        #    --header 'Content-Type: text/plain' \
        #    --header 'X-AUTH-YW-API-TOKEN: {api_token}' \
        #    --data 'true'
        resp = self.yba_api.Post(y.cust_url + self.new_authz_uri,
                                 data=str(set_to).lower,
                                 extra_headers={"Content-Type": "text/plain"})
        return set_to   # SET the value
#=====================================================================================================
@dataclass
class User():
    uuid:str
    email:str
    creationDate: str
    role: Role; # Role object, eg. representing 'SuperAdmin'
    password: str = None
    #'userType': 'local', 'ldapSpecifiedRole': False, 'primary': True, 'features': 

    def Print(self,json:bool=False,leading_comma:bool=False):
        if json:
            print( (',' if leading_comma else ' ')
                    +'{"email":"'+ self.email+'","uuid":"'
                    +str(self.uuid)+'","role":"'+self.role.name + '","created":"'+self.creationDate +'"}')
            return
        logging.info("USER "+self.email+"("+str(self.uuid)+") Role:"+self.role.name + "  Created:"+self.creationDate  )


    def Create_in_YBA(self):
        y = self.role.mgt.yba_api
        logging.debug("DEBUG: Creating User "+ self.email +" in YBA")
        if self.role.mgt.use_new_authz:
            raise ValueError("Create user with new authz is not implemented")

        payload = {
            "email": self.email,
            "password": self.password,
            "confirmPassword": self.password,
            "role": self.role.name,
            "timezone":  "America/New_York" # Fake it out 
        }
        response = y.Post(y.cust_url + "/users", data=payload)
        logging.info(response)
        self.uuid = y.raw_response.json()["uuid"]
        self.creationDate = y.raw_response.json()["creationDate"]


    def Delete_from_YBA(self):
        y = self.role.mgt.yba_api
        logging.info ("Deleting User "+ self.email +" in YBA")
        
        response = y.Delete(y.cust_url + "/users/" + self.uuid)
        logging.info (pformat(response))

#=====================================================================================================
@dataclass
class STDIN_Json_Stream_Processor():
    yba:YBA_API


    def redact_password(self,log_string, redaction_char='*', min_pass_length=4):
        """
        Redacts a password from a log string.
        Parameter "self" is ignored, as this is a static method.
        It identifies 'PASSWORD' (case-insensitive) keys, which may themselves be
        quoted or unquoted. It then looks for a flexible set of delimiters
        (any characters *not* a quote) followed by the password value strictly
        enclosed in single or double quotes.

        Args:
            log_string (str): Input string.
            redaction_char (str): Character to use for redaction.
            min_pass_length (int): Minimum length of the content *inside* the quotes
                                    for it to be considered a password and redacted.

        Returns:
            str: Log string with password redacted.
        """
        # Regex Breakdown:
        # (re.IGNORECASE | re.DOTALL | re.VERBOSE):
        #   re.IGNORECASE: Makes "PASSWORD" case-insensitive.
        #   re.DOTALL: Allows '.' to match any character including newlines.
        #   re.VERBOSE: Allows comments and whitespace in the regex for readability.

        pattern = re.compile(
            r"""
            (                               # Group 1: The full keyword part
                (?:'|")?                      # Optional opening quote for the key itself
                PASSWORD                      # The literal "PASSWORD" keyword
                (?:'|")?                      # Optional closing quote for the key itself
            )
            (                               # Group 2: The flexible delimiter part.
                [^'"]*?                       # Match 0 or more characters that are NOT a single or double quote, non-greedily.
                                            # THIS IS THE CRITICAL CHANGE: It prevents consuming the password's opening quote.
            )
            (['"])                           # Group 3: Captures the opening quote of the *value*.
            ( .*? )                          # Group 4: The password content (non-greedy match for any characters)
            \3                               # Backreference to Group 3, matches the closing quote.
            """,
            re.IGNORECASE | re.DOTALL | re.VERBOSE
        )

        def replacer(match):
            # Extract captured groups
            keyword_full = match.group(1)
            delimiter_part = match.group(2)
            quote_char = match.group(3)
            password_content = match.group(4)

            if len(password_content) < min_pass_length:
                return match.group(0) # Return original full match if password is too short

            redacted_password_content = redaction_char * len(password_content)
            
            # Reconstruct the string: original key + original delimiter + quote + redacted password + quote
            return f"{keyword_full}{delimiter_part}{quote_char}{redacted_password_content}{quote_char}"

        return pattern.sub(replacer, log_string)


    def Process_str_cmd(self,cmd:str):
        logging.debug ("Processing string command:"+ self.redact_password(cmd) )
        if cmd == "LISTUSERS":
            print ("[")
            count = 0
            for u in self.yba.Get_User_List():
                u.Print(json=True,leading_comma=( count > 0))
                count +=1
            print ("]")
            logging.info("Listed "+str(count)+" users")
            return


    def Process_dict_part(self,cmd:dict):
        logging.debug ("Processing dict command:"+self.redact_password(str(cmd)))
        adduser_count = 0
        deleteuser_count = 0
        # Expecting a command like:
        # {"ADDUSERS": {"u12@rrr":"ReadOnly","u13@rrr":"ReadOnly"},"PASSWORD":"Some_Pa*ss"}
        # or {"DELETEUSERS": ["u12@rrr","u13@rrr"]} 
        if cmd.get("ADDUSERS") is not None:
            if not isinstance(cmd["ADDUSERS"],dict):
                raise ValueError("ERROR:ADDUSERS: Could not find user dict to add")

            for email in cmd["ADDUSERS"].keys():
                role = cmd["ADDUSERS"][email]
                if role is None:
                    raise ValueError("ERROR: You must specify role when adding a user="+ email)
                if isinstance(role,list):
                    raise ValueError("ERROR: Attempt to add a role LIST, instead of a simple value for user "+email)
                YBArole = self.yba.roleManagement.Get_or_create_role_by_name(role,allow_create=True)
                new_password = cmd.get("PASSWORD")
                if new_password is None:
                    new_password = "*UnSpecified*123" 
                usr=User(uuid=None,email=email,creationDate=datetime.today().isoformat(),role=YBArole,password=new_password)
                usr.Create_in_YBA()
                logging.info("Added user "+email)
                #usr.Print(json=True)
                adduser_count+=1


        if cmd.get("DELETEUSERS") is not None:
            if not isinstance(cmd["DELETEUSERS"],list):
                raise ValueError("ERROR:DELETEUSERS: Could not find user list to delete")

            for email in cmd["DELETEUSERS"]:
                if email is None:
                    raise ValueError("ERROR: You must specify email when deleting a user")
                for u in self.yba.Get_User_List():
                    if u.email != email:
                        continue
                    deleteuser_count += 1
                    u.Delete_from_YBA()
                    logging.info("Deleted user "+email)
                    break
            if (deleteuser_count + adduser_count) == 0:
                print('{"success":false,"added":'+ adduser_count + ',"deleted":'+deleteuser_count +'}' )
            else:
                print('{"success":true,"added":'+ str(adduser_count) + ',"deleted":'+str(deleteuser_count) +'}' )
            
    #============ R U N   M e t h o d ===
    def Run(self):
        """
            to parse a series of json objects from stdin 
        """
        json_found = []  
        # raw_decode expects byte1 to be part of a JSON, so remove whitespace from left
        stdin = sys.stdin.read().lstrip()
        decoder = JSONDecoder()

        while len(stdin) > 0:
            logging.debug ("input json string:"+ self.redact_password(str(stdin)))
            # parsed_json, number of bytes used
            parsed_json, consumed = decoder.raw_decode(stdin)
            # Remove bytes that were consumed in this object ^ 
            logging.debug ("DEBUG:json piece:"+ str(parsed_json) + " ["
                            + self.redact_password(str(type(parsed_json)))
                            + " " + str(consumed) + " bytes]")
            stdin = stdin[consumed:]
            # Process what we just got ..
            if isinstance(parsed_json, str):
                self.Process_str_cmd(parsed_json)
            elif isinstance(parsed_json, list):
                for part in parsed_json:
                    if isinstance(part, str):
                        self.Process_str_cmd(part)
                    else:
                        self.Process_dict_part(part)
            elif isinstance(parsed_json, dict):
                self.Process_dict_part(parsed_json)
            else:
                raise ValueError("ERROR:Unexpected input type on STDIN:"+str(type(parsed_json))+"="+str(parsed_json))
            # Save this parsed object
            json_found.append(parsed_json)
            # Remove any whitespace before the next JSON object
            stdin = stdin.lstrip()

        logging.info ("DEBUG:ACCUMULATED JSON FROM STDIN:=====")
        logging.info (self.redact_password(pformat(json_found)))

#=====================================================================================================
#=====================================================================================================
def environ_or_required(key) -> Dict:
    """
      Helper for Param parser...
      If ENV var is available, use it
      If not, mark this arg as REQUIRED
    """
    return (
        {'default': os.environ.get(key)} if os.environ.get(key)
        else {'required': True}
    )

def parse_arguments():
    parser = argparse.ArgumentParser(
        prog='yba-user-admin.py v ' + version,
        description='This Program operates on YBA users (List/Create/Delete)',
        epilog='End of help',
        add_help=True )
    parser.add_argument('-d', '--debug', action='store_true', default=os.getenv("DEBUG",False))
    parser.add_argument('-y', '--yba_url', help='YBAnywhere URL', **environ_or_required("YBA_HOST"))
    parser.add_argument('-c', '--customer_uuid', help='Customer UUID: Auto-discovered if unspecified',default=os.getenv("CUST_UUID"))
    parser.add_argument('-a', '--auth_token', help='YBAnywhere Auth Token', **environ_or_required("API_TOKEN"))
    action_group = parser.add_mutually_exclusive_group()
    action_group.add_argument('-ls','--list',action='store_true',default=True,help="List users (This is the default action)")
    action_group.add_argument('-rm','--remove','--delete',metavar='User@email.addr',help="Delete the specified user")
    action_group.add_argument('-mk','--make','--create',metavar='User@email.addr',help="Add a new user. role & password reqd")
    parser.add_argument('--role',help="Name of role to apply to new user")
    parser.add_argument('-p','--password',help="Password for new user (>=8 ch, Upcase+num+special)")
    parser.add_argument("-s","--stdin",action='store_true',help="Read JSON stream from stdin")
    parser.add_argument("-log","--logfile",metavar='log.file.path.and.name',help="Log to this file instead of stdout",default=os.getenv("LOGFILE",None))
    args = parser.parse_args()
    
    return args
#=====================================================================================================
#=====================================================================================================
#  M  A  I  N
#=====================================================================================================
args = parse_arguments()
if args.logfile is None and args.stdin is True:
    args.logfile = "yba-user-admin.log" # Default log file if not specified for STDIN
logging.basicConfig(format='%(asctime)s %(levelname)s %(message)s', datefmt='%H:%M:%S',
                    level= logging.DEBUG if args.debug else  logging.INFO,
                    filename=args.logfile if args.logfile else None, filemode='a' # append
                    )
logging.info(datetime.now(timezone.utc).astimezone().isoformat() + " YBA User admin : v "+version)
logging.debug("Debugging Enabled")

y = YBA_API(yba_url=args.yba_url, auth_token=args.auth_token, customer_uuid=args.customer_uuid, debug=args.debug)

if args.stdin:
    STDIN_Json_Stream_Processor(yba=y).Run()
    sys.exit(0)

if args.make: # AKA create/make
    logging.info("Will create user:"+args.make + " in role:" + args.role )
    if args.role is None:
        raise ValueError("ERROR: You must specify --role when adding a user")
    if args.password is None:
        raise ValueError("ERROR: You must specify --password when adding a user")
    y.roleManagement.Get_or_create_role_by_name("ReadOnly", allow_create=True) # Manufacture a role 
    role = y.roleManagement.Get_or_create_role_by_name(args.role,allow_create=False)
    usr=User(uuid=None,email=args.make,creationDate=datetime.today().isoformat(),role=role,password=args.password)
    usr.Create_in_YBA()
    usr.Print()
    sys.exit(0)

if args.remove: # AKA Remove/delete
    logging.info("Attempting to remove user:"+args.remove )
    userlist = y.Get_User_List()
    found = False
    for u in userlist:
        if u.email != args.remove:
            continue
        found = True
        u.Delete_from_YBA()
        break
    
    if not found:
        raise ValueError("ERROR: Could not find user "+args.remove)
    logging.info("Removed "+ args.remove)
    sys.exit(0)

# No args, or args.ls
for u in y.Get_User_List():
    u.Print();

sys.exit(0)