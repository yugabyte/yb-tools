#!/usr/bin/python3
# YBA User list/creation/Deletion
version = "0.08"
from ast import Dict, parse
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
from datetime import datetime

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
    RoleManagement:RoleManagement = None
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
        self.RoleManagement = RoleManagement(self)


    def __set_customeruuid(self):
        customer_list = self.Get(self.yba_url + '/api/v1/customers')
        if len(customer_list) > 1:
            raise SystemExit("ERROR: Multiple Customers Found, Must Specify which Customer UUID to Use!")
        self.customer_uuid = customer_list[0]['uuid']

    def Get(self, targeturl="", raw=False):
        if self.debug:
            print("DEBUG: API Get:"+targeturl + " (RAW="+str(raw)+")")
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
            role =  self.RoleManagement.Get_or_create_role_by_name(u["role"], allow_create=True)
            self.UserList.append(User(uuid=u["uuid"],email=u["email"],creationDate=u["creationDate"],role=role)) # User objects
        return self.UserList

    def Post(self, url:str, data, extra_headers=None, timeout:int=2):
        # avoids no CSRF token error by emptying the cookie jar
        # session.cookies = requests.cookies.RequestsCookieJar()
        if self.debug:
            print("DEBUG: API Post:"+url)
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.auth_token}
        if extra_headers is not None:
            headers.update(extra_headers)
        # avoids no CSRF token error by emptying the cookie jar
        self.session.cookies = requests.cookies.RequestsCookieJar()
        self.raw_response = self.session.post(url, json=data, headers=headers, timeout=timeout, verify=False)
        if self.raw_response.status_code == 200:
            if self.debug:
                print('DEBUG: API Request successful')
        else:
            print(self.raw_response.json())
            self.raw_response.raise_for_status()
        return self.raw_response.text
    
    def Delete(self, url:str, timeout:int=2):
        if self.debug:
            print("DEBUG: API Delete:"+url)
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.auth_token}
        # avoids no CSRF token error by emptying the cookie jar
        self.session.cookies = requests.cookies.RequestsCookieJar()
        self.raw_response = self.session.delete(url, headers=headers, timeout=timeout, verify=False)
        if self.raw_response.status_code == 200:
            if self.debug:
                print('DEBUG: API Request successful')
        else:
            print(self.raw_response.json())
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
    role_list = []

    def __post_init__(self):
        self.yba_api.RoleManagement = self
        self.use_new_authz = self.Fine_grained_RBAC()
        if self.yba_api.debug:
            print("DEBUG: RoleManagement: use_new_authz="+str(self.use_new_authz))
        self.role_by_name  = {}
        if self.use_new_authz:
            role_list = self.yba_api.Get(y.cust_url + "/rbac/role")
            # Populate role_by_name once we know what this looks like 

    def Get_or_create_role_by_name(self,name:str,allow_create:bool=True) -> Role:
        if self.role_by_name.get(name) is None:
            if allow_create:
                if self.yba_api.debug:
                    print("DEBUG: RoleManagement: Creating Role object for "+name)
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

    def Print(self):
        print("USER "+self.email+"("+str(self.uuid)+") Role:"+self.role.name + "  Created:"+self.creationDate  )

    def Create_in_YBA(self):
        y = self.role.mgt.yba_api
        if y.debug:
            print ("DEBUG: Creating User "+ self.email +" in YBA")
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
        if y.debug:
            pprint(response)
        self.uuid = y.raw_response.json()["uuid"]
        self.creationDate = y.raw_response.json()["creationDate"]


    def Delete_from_YBA(self):
        y = self.role.mgt.yba_api
        if y.debug:
            print ("DEBUG: Deleting User "+ self.email +" in YBA")
        
        response = y.Delete(y.cust_url + "/users/" + self.uuid)
        if y.debug:
            pprint(response)

#=====================================================================================================
@dataclass
class STDIN_Json_Stream_Processor():
    yba:YBA_API

    def Process_str_cmd(self,cmd:str):
        if self.yba.debug:
            print("DEBUG:Processing string command:"+cmd)
        if cmd == "LISTUSERS":
            for u in self.yba.Get_User_List():
                u.Print();
            return
        
    #=====================================================================================================
    def Process_dict_part(self,cmd:dict):
        if self.yba.debug:
            print("DEBUG:Processing dict command:"+str(cmd))
        if cmd.get("ADDUSERS") is not None:
            if not isinstance(cmd["ADDUSERS"],list):
                raise ValueError("ERROR:ADDUSERS: Could not find user list to add")
            for u_json in cmd["ADDUSERS"]:
                if u_json.get("email") is None:
                    raise ValueError("ERROR: You must specify email when adding a user")
                if u_json.get("role") is None:
                    raise ValueError("ERROR: You must specify role when adding a user="+u_json["email"])
                role = self.yba.RoleManagement.Get_or_create_role_by_name(u_json["role"],allow_create=True)
                usr=User(uuid=None,email=u_json["email"],creationDate=datetime.today().isoformat(),role=role,password=u_json["password"])
                usr.Create_in_YBA()
                usr.Print()

        if cmd.get("DELETEUSERS") is not None:
            if not isinstance(cmd["DELETEUSERS"],list):
                raise ValueError("ERROR:DELETEUSERS: Could not find user list to delete")
            for u_json in cmd["DELETEUSERS"]:
                if u_json.get("email") is None:
                    raise ValueError("ERROR: You must specify email when deleting a user")
                found = False
                for u in self.yba.Get_User_List():
                    if u.email != u_json["email"]:
                        continue
                    found = True
                    u.Delete_from_YBA()
                    break
                if found:
                    print('"OK"')
            
    #=====================================================================================================
    def Run(self):
        """
            to parse a series of json objects from stdin 
        """
        json_found = []  
        # raw_decode expects byte1 to be part of a JSON, so remove whitespace from left
        stdin = sys.stdin.read().lstrip()
        decoder = JSONDecoder()

        while len(stdin) > 0:
            # parsed_json, number of bytes used
            parsed_json, consumed = decoder.raw_decode(stdin)
            # Remove bytes that were consumed in this object ^ 
            if self.yba.debug:
                print("DEBUG:json piece:"+ str(parsed_json) + " ["+ str(type(parsed_json)) + " " + str(consumed) + " bytes]")
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

        if self.yba.debug:
            print("DEBUG:ACCUMULATED JSON FROM STDIN:=====")
            print(json_found)

#=====================================================================================================
#=====================================================================================================
def environ_or_required(key) -> Dict:
    """
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
    args = parser.parse_args()
    if args.debug:
        print("DEBUG: "+datetime.today().isoformat()+" Debugging Enabled")
    
    return args
#=====================================================================================================
#=====================================================================================================
#  M  A  I  N
#=====================================================================================================
args = parse_arguments() 

y = YBA_API(yba_url=args.yba_url, auth_token=args.auth_token, debug=args.debug)

if args.stdin:
    STDIN_Json_Stream_Processor(yba=y).Run()
    sys.exit(0)

if args.make: # AKA create/make
    print("Will create user:"+args.make + " in role:" + args.role )
    if args.role is None:
        raise ValueError("ERROR: You must specify --role when adding a user")
    if args.password is None:
        raise ValueError("ERROR: You must specify --password when adding a user")
    y.RoleManagement.Get_or_create_role_by_name("ReadOnly", allow_create=True) # Manufacture a role 
    role = y.RoleManagement.Get_or_create_role_by_name(args.role,allow_create=False)
    usr=User(uuid=None,email=args.make,creationDate=datetime.today().isoformat(),role=role,password=args.password)
    usr.Create_in_YBA()
    usr.Print()
    sys.exit(0)

if args.remove: # AKA Remove/delete
    print("Attempting to remove user:"+args.remove )
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
    print("Removed "+ args.remove)
    sys.exit(0)

# No args, or args.ls
for u in y.Get_User_List():
    u.Print();

sys.exit(0)