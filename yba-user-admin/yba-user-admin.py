#!python3
# YBA User list/creation/Deletion
version = "0.05"
from ast import Dict
from multiprocessing import Value
import time
from venv import create
import requests
import urllib3
import json
import argparse
import re
import sys
from dataclasses import dataclass,field
#from typing import Optional
from typing import List
from json import JSONDecoder, JSONEncoder
from pprint import pprint
from datetime import datetime

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

@dataclass
class YBA_API():
    yba_url: str
    auth_token: str = field(default= None)
    customer_uuid: str =field(default= None)
    raw_response: str = field(default= None)
    debug:bool = False
    session = None

    def __post_init__(self):
        self.yba_url  = self.yba_url.strip("/")
        self.raw_response = None
        self.session = requests.Session()
        self.session.verify = False
        if self.customer_uuid is None:
            self.__set_customeruuid()

        self.cust_url = self.yba_url + '/api/v1/customers/' + self.customer_uuid
        self.univ_url = self.cust_url + '/universes'


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
class RoleManagement: # Forward declaration to help with circular reference
    pass

@dataclass ( match_args=False)
class Role:
    yba_api: YBA_API
    mgt: RoleManagement 
    name:str =  field(default=None)
    uuid: str = field(default=None)

#=====================================================================================================
#https://docs.yugabyte.com/preview/yugabyte-platform/administer-yugabyte-platform/anywhere-rbac/#manage-custom-roles
@dataclass ( match_args=False)
class RoleManagement:
    yba_api: YBA_API
    role_by_name: Dict = field(init=False)
    use_new_authz: bool = False
    new_authz_uri:str = "/runtime_config/00000000-0000-0000-0000-000000000000/key/yb.rbac.use_new_authz"

    def __post_init__(self):
        self.use_new_authz = self.Fine_grained_RBAC()
        if self.yba_api.debug:
            print("DEBUG: RoleManagement: use_new_authz="+str(self.use_new_authz))
        self.role_by_name  = {}
        if self.use_new_authz:
            y = self.yba_api
            role_list = y.Get(y.cust_url + "/rbac/role")

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
            use_new_authz_text = self.yba_api.Get(y.cust_url
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
            "role": role.name,
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
def parse_arguments():
    parser = argparse.ArgumentParser(
        prog='yba-user-admin.py v ' + version,
        description='This Program operates on YBA users (List/Create/Delete)',
        epilog='End of help',
        add_help=True )
    parser.add_argument('-d', '--debug', action='store_true')
    parser.add_argument('-y', '--yba_url', required=True, help='YBAnywhere URL')
    parser.add_argument('-c', '--customer_uuid', help='Customer UUID')
    parser.add_argument('-a', '--auth_token', required=True, help='YBAnywhere Auth Token')
    action_group = parser.add_mutually_exclusive_group()
    action_group.add_argument('-ls','--list',action='store_true',default=True,help="List users (This is the default action)")
    action_group.add_argument('-rm','--remove','--delete',metavar='User@email.addr',help="Delete the specified user")
    action_group.add_argument('-mk','--make','--create',metavar='User@email.addr',help="Add a new user. role & password reqd")
    parser.add_argument('--role',help="Name of role to apply to new user")
    parser.add_argument('-p','--password',help="Password for new user (8 ch, Upcase+num+special)")
    parser.add_argument("-s","--stdin",action='store_true',help="Read JSON stream from stdin")
    args = parser.parse_args()
    if args.debug:
        print("DEBUG: "+datetime.today().isoformat()+" Debugging Enabled")
    
    return args
#=====================================================================================================
def Process_JSON_from_stdin():
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
        print("json piece:"+ str(parsed_json) + " ["+ str(consumed) + " bytes]")
        stdin = stdin[consumed:]
        # Save this parsed object
        json_found.append(parsed_json)
        # Remove any whitespace before the next JSON object
        stdin = stdin.lstrip()
    print("ACCUMULATED JSON FROM STDIN:=====")
    print(json_found)

#=====================================================================================================
#  M  A  I  N
#=====================================================================================================
args = parse_arguments() 

y = YBA_API(yba_url=args.yba_url, auth_token=args.auth_token, debug=args.debug)
RM = RoleManagement(y)
RM.Get_or_create_role_by_name("ReadOnly", allow_create=True) # Manufacture a role 

if args.stdin:
    Process_JSON_from_stdin()
    sys.exit(0)

userlist = y.Get(y.cust_url+"/users")
for u in userlist:
    role =  RM.Get_or_create_role_by_name(u["role"], allow_create=True)
    usr=User(uuid=u["uuid"],email=u["email"],creationDate=u["creationDate"],role=role)
    if args.list:
        usr.Print()

if args.make: # AKA create/make
    print("Will create user:"+args.make + " in role:" + args.role )
    if args.role is None:
        raise ValueError("ERROR: You must specify --role when adding a user")
    if args.password is None:
        raise ValueError("ERROR: You must specify --password when adding a user")    
    role = RM.Get_or_create_role_by_name(args.role,allow_create=False)
    usr=User(uuid=None,email=args.make,creationDate=datetime.today().isoformat(),role=role,password=args.password)
    usr.Create_in_YBA()
    usr.Print()

if args.remove: # AKA Remove/delete
    print("Attempting to remove user:"+args.remove )
    found = False
    for u in userlist:
        if u["email"] != args.remove:
            continue
        found = True
        role =  RM.Get_or_create_role_by_name(u["role"], allow_create=True)
        usr=User(uuid=u["uuid"],email=u["email"],creationDate=u["creationDate"],role=role)
        usr.Delete_from_YBA()
        break
    
    if not found:
        raise ValueError("ERROR: Could not find user "+args.rm)

sys.exit(0)