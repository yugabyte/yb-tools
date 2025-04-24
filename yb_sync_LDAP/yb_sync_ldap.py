#!/usr/bin/env python3
#
# Copyright © 2024 YugaByte, Inc. and Contributors
#
# Licensed under the Polyform Free Trial License 1.0.0 (the "License"); you
# may not use this file except in compliance with the License. You
# may obtain a copy of the License at
#
# https://github.com/YugaByte/yugabyte-db/blob/master/licenses/POLYFORM-FREE-TRIAL-LICENSE-1.0.0.txt
"""
LDAP Sync script for Yugabyte YCQL and YSQL.
"""
import sys
import os
import re
import argparse
import socket
import json
import ssl
import time
import logging
import random
import atexit
import string
import traceback
import requests
import psycopg2
import psycopg2.extras
import ldap
from deepdiff import DeepDiff, Delta
from cassandra.cluster import Cluster, NoHostAvailable  # pylint: disable=no-name-in-module
from cassandra.auth import PlainTextAuthProvider
from cassandra.query import dict_factory  # pylint: disable=no-name-in-module
from cassandra.policies import DCAwareRoundRobinPolicy
from time import gmtime, strftime


VERSION = "0.44"



YW_LOGIN_API = "{}://{}:{}/api/v1/login"
YW_API_TOKEN = "{}://{}:{}/api/v1/customers/{}/api_token"
YW_API_UNIVERSE_LIST = "{}://{}:{}/api/v1/customers/{}/universes"
YW_API_CUSTOMER_LIST = "{}://{}:{}/api/v1/customers"
YW_CERT_API = "{}://{}:{}/api/v1/customers/{}/certificates/{}/download"
YW_API_FIND_UNIVERSE = "{}://{}:{}/api/v1/customers/{}/universes/find/{}"
YW_YQL_SERVER_LIST = "{}://{}:{}/api/v1/customers/{}/universes/{}/yqlservers"
YW_YSQL_SERVER_LIST = "{}://{}:{}/api/v1/customers/{}/universes/{}/ysqlservers"
YCQL_RANDOM_PASSWORD_LENGTH = 16
YCQL_CREATE_ROLE = "CREATE ROLE IF NOT EXISTS \"{}\" WITH"\
                 " SUPERUSER = false AND LOGIN = true AND PASSWORD = '{}';"
YCQL_GRANT_ROLE = "GRANT \"{}\" TO \"{}\";"
YCQL_REVOKE_ROLE = "REVOKE \"{}\" FROM \"{}\";"
YCQL_DROP_ROLE = "DROP ROLE IF EXISTS \"{}\";"
YCQL_CREATE_NOLOGIN_ROLE = "CREATE ROLE \"{}\" WITH LOGIN=false;"
YSQL_CREATE_ROLE = "CREATE ROLE \"{}\" WITH LOGIN PASSWORD '{}';"
YSQL_CREATE_ROLE_IN_ROLES = "CREATE ROLE \"{}\" WITH LOGIN PASSWORD '{}' IN ROLE {};" # Caller must quote role(s)
YSQL_CREATE_NOLOGIN_ROLE = "CREATE ROLE \"{}\" WITH NOLOGIN;"
YSQL_CREATE_NOLOGIN_ROLE_IN_ROLES = "CREATE ROLE \"{}\" WITH NOLOGIN IN ROLE {};"
YSQL_GRANT_ROLE = "GRANT \"{}\" TO \"{}\";"
YSQL_REVOKE_ROLE = "REVOKE \"{}\" FROM \"{}\";"
YSQL_DROP_ROLE = "DROP ROLE IF EXISTS \"{}\";"
YSQL_OWNED_OBJECTS = "SELECT r.rolname as role,count(*) as owned_objects "\
                     "FROM pg_roles r,pg_shdepend d "\
                     "WHERE d.refobjid=r.oid "\
                     "GROUP BY  r.rolname;"
YW_TEMP_DIR = "/tmp"
LDAP_BASE_DATA_DIR = "/opt/yugabyte/yugaware/data"
LDAP_DATA_CACHE_DIR = os.path.join(LDAP_BASE_DATA_DIR, 'cache')
LDAP_DATA_CACHE_CUST_DIR = os.path.join(LDAP_DATA_CACHE_DIR, '{}')
LDAP_DATA_CACHE_UNIV_DIR = os.path.join(LDAP_DATA_CACHE_CUST_DIR, '{}')
LDAP_FILE_DATA = os.path.join(LDAP_BASE_DATA_DIR, 'cache/{}/{}/ldap_data.json')
LDAP_BASE_DIR_ERROR = "The base directory {} does not exist"
YCQL_ROLE_QUERY = "SELECT role, member_of, can_login FROM roles "\
                  "WHERE  role !='cassandra' {}" # param : "AND is_superuser = false" or ""
YSQL_ROLE_QUERY = "SELECT r.rolname as role, r.rolcanlogin as can_login, ARRAY(SELECT b.rolname FROM "\
                  "pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles b "\
                  "ON (m.roleid = b.oid) WHERE m.member=r.oid) as member_of "\
                  "FROM pg_catalog.pg_roles r WHERE r.rolname !~ '^pg_' "\
                  "  AND r.rolname NOT IN ('yugabyte','postgres')"\
                  " {} order by 1;"  # param: "AND r.rolsuper='f'" or ""
UID_RE = r"\['?([A-Za-z0-9_\.@]+)'?\]"


def random_string(length):
    """ Routine to build a random string for temporary directory names. """
    return ''.join(random.choice('abcdefghijklmnopqrstuvwxyz') for i in range(length))


def get_uid_from_ddiff(sq1):
    """ Routine to extract usernames(firstname.lastname)  from DeepDiff dictionary items  like: root['firstname.lastname'] """
    return re.findall(UID_RE, sq1)[0].translate({ord(i): None for i in '][\''})


def generate_random_password():
    """ Routine to generate a random password.  """
    lower = string.ascii_lowercase
    upper = string.ascii_uppercase
    num = string.digits
    symbols = string.punctuation.replace("'", "").replace("\"", "")
    all_chars = lower + upper + num + symbols
    temp = random.sample(all_chars, YCQL_RANDOM_PASSWORD_LENGTH)
    random_pw = "".join(temp)
    return random_pw


class LDAPSyncException(Exception):
    """A YugaByte LDAP sync exception."""


class YBLDAPSync:
    """ The main class """
    def __init__(self):
        """ The init routine for this class."""
        self.args = self.parse_arguments()
        self.host_ipaddr  = self.args.apihost if self.args.apihost else self.get_local_ipaddr()
        #Strip off http http:// or https:// if sent in as the 'http_type' var us used to build the URL
        self.host_ipaddr = self.host_ipaddr.replace('https://', '').replace('http://', '')
        self.ycql_session = None
        self.ysql_session = None
        self.http_type    = 'http'
        self.api_port     = self.args.apiport # Default 9000 (for docker instance)

    @classmethod
    def cleanup_temporary_certfile(cls, filepath):
        """ Cleanup temporary file location."""
        path = os.path.dirname(filepath)
        os.remove(filepath)
        os.rmdir(path)

    @classmethod
    def make_temporary_certfile(cls):
        """ Create temporary file location for cert extracted from API"""
        path = os.path.join(YW_TEMP_DIR, '.{}'.format(random_string(16)))
        filepath = os.path.join(path, 'root.crt')
        os.mkdir(path)
        atexit.register(cls.cleanup_temporary_certfile, filepath)
        return filepath

    def get_local_ipaddr(self):
        """ Get the local ipv4 or ipv6 address of YW API """
        addr_family = socket.AF_INET6 if self.args.ipv6 else socket.AF_INET
        sock = socket.socket(addr_family, socket.SOCK_DGRAM)
        try:
            if os.getenv('YW_TEST_YUGAWARE_UI_SERVICE_HOST') is not None:
                sock.connect((os.getenv('YW_TEST_YUGAWARE_UI_SERVICE_HOST'), 1))
            else:
                sock.connect((socket.gethostname(), 1))
            api_address = sock.getsockname()[0]
        except socket.error:
            api_address = '::1' if self.args.ipv6 else '127.0.0.1'
        finally:
            sock.close()
        return api_address

    @classmethod
    def create_ldap_cache_directory(cls, customeruuid, universeuuid):
        """
        Routine to create the LDAP cache directory on the permanent mount point for YW
        :Param customeruuid  - the customeruuid from the YW API
        :Param universeuuid - the universeuuid from the YW API
        """
        if os.path.isdir(LDAP_BASE_DATA_DIR):
            if not os.path.isdir(LDAP_DATA_CACHE_DIR):
                os.mkdir(LDAP_DATA_CACHE_DIR)
            if not os.path.isdir(LDAP_DATA_CACHE_CUST_DIR.format(customeruuid)):
                os.mkdir(LDAP_DATA_CACHE_CUST_DIR.format(customeruuid))
            if not os.path.isdir(LDAP_DATA_CACHE_UNIV_DIR.format(customeruuid, universeuuid)):
                os.mkdir(LDAP_DATA_CACHE_UNIV_DIR.format(customeruuid, universeuuid))
        else:
            raise LDAPSyncException(LDAP_BASE_DIR_ERROR.format(LDAP_BASE_DATA_DIR))

    @classmethod
    def load_previous_ldap_data(cls, customeruuid, universeuuid):
        """
        Routine to load data from the last invocation of this script. Useful in determining
        whether more processing should occur on this pass, or if no changes to go back to
        sleep until the next execution.
        :Param customeruuid  - the customeruuid from the YW API
        :Param universeuuid - the universeuuid from the YW API
        :Return ldap_data - dictionary containing last data fetched from ldap
        """
        ldap_data = {}
        filepath = LDAP_FILE_DATA.format(customeruuid, universeuuid)
        if os.path.exists(filepath):
            with open(filepath, encoding='utf-8') as ldap_file:
                ldap_data = json.load(ldap_file)
            ldap_file.close()
            logging.info("Loaded {} previous LDAP items from file {}.".format(len(ldap_data),ldap_file))
        else:
            cls.create_ldap_cache_directory(customeruuid, universeuuid)
        return ldap_data

    @classmethod
    def save_ldap_data(cls, ldap_data, customeruuid, universeuuid):
        """
        Routine to write LDAP data to disk so that it can be retrieved on the next run
        for quick comparison.
        :Param ldap_data - the dictionary retrieved from LDAP
        :Param customeruuid - the customeruuid fom the YW API
        :Param universeuuid - the universeuuid from the YW API
        """
        filepath = LDAP_FILE_DATA.format(customeruuid, universeuuid)
        if os.path.exists(filepath):
            os.remove(filepath)
        else:
            cls.create_ldap_cache_directory(customeruuid, universeuuid)
        with open(filepath, 'w', encoding='utf-8') as ldap_file:
            json.dump(ldap_data, ldap_file)
        ldap_file.close()

    @classmethod
    def Pretty_Print(cls,d, indent=0, keyfilter=None):
       if isinstance(d, list):
          for item in d:
            cls.Pretty_Print(item,indent,keyfilter)
          return
       if isinstance(d,tuple):
          for (idx,val) in enumerate(d):
            cls.Pretty_Print(val,indent,keyfilter)
          return
       if isinstance(d, dict):
          if isinstance(keyfilter,str): # Convert str to dict
             keyfilter = {item:True for item in keyfilter.split(",")} # It is now a DICT
          for key, value in d.items():
           if indent==0 or keyfilter==None  or (keyfilter and key in keyfilter):
                print('\t' * indent + ("" if indent ==0 else "--> ") + str(key) + ":") 
                cls.Pretty_Print(value, indent+1,keyfilter)
           else:
              continue # Ignore key  NOT in filter
       else:
           try:
             d=d.decode()
           except:
             None # Ignore exception 
           print('\t' * indent + str(d))
           
    @classmethod
    def Print_Report(cls,hdr,data,keyfilter=None): # Adds header and borders
        logging.debug("Printing report {} with {} items, filtering={}.".format(hdr,len(data),keyfilter))
        print("-"*50)
        row = len(hdr)+2
        h = ''.join(['+'] + ['-' *row] + ['+'])
        result= h + '\n'"| "+hdr+" |"'\n' + h
        print(result)
        cls.Pretty_Print(data,0,keyfilter)
        print("-"*50)

    def get_auth_token(self, username, password):
        """
        Routine that gets an authentication token from YW API.
        TODO: This information should be stored in hashicorp vault.
        :Param username - the YW API username
        :Param password - the YW API password
        :Return the authentication token and customeruuid from YW API
        """
        authtoken = None
        customeruuid = None
        userauth_payload = {}
        yw_url = YW_LOGIN_API.format(self.http_type,self.host_ipaddr, self.api_port)
        userauth_payload['email'] = username
        userauth_payload['password'] = password
        headers = {'Content-type': 'application/json'}
        api_result = requests.post(yw_url, headers=headers, data=json.dumps(userauth_payload))
        data = json.loads(api_result.text)
        if 'error' in data:
            raise LDAPSyncException("Failure to connect to YW API: {}".format(data['error']))
        if 'authToken' in data and 'customerUUID' in data:
            authtoken = data['authToken']
            customeruuid = data['customerUUID']
        else:
            raise LDAPSyncException("Failed to find authtoken or customerUUID in API response")
        return authtoken, customeruuid

    def get_customeruuid(self,apitoken):
        yw_url = YW_API_CUSTOMER_LIST.format(self.http_type,self.host_ipaddr, self.api_port)
        headers = {'Content-type': 'application/json'}
        xapi_headers = {}
        xapi_headers['X-AUTH-YW-API-TOKEN'] = apitoken
        logging.debug("Getting customer list from YBA API {}.".format(yw_url))
        api_result = requests.get(yw_url, headers=xapi_headers, verify=False)
        data = json.loads(api_result.text)
        if 'error' in data:
            raise LDAPSyncException("Failed to receive customer list from API: {}".format(data['error']))
        logging.debug("Customer list returned={}".format(data))
        customerUUID = None
        if 'uuid' in data[0]:
            customerUUID = data[0]['uuid']
        return customerUUID
    
    def get_api_token(self, authtoken, customeruuid):
        """
        Routine that gets the API token from the YW API.
        :Param authtoken - the authtoken from get_auth_token()
        :Param customeruuid - the customer uuid of the YW instance from get_auth_token()
        :Return the apitoken used for all subsequent YW API calls
        """
        xauth_headers = {}
        apitoken = None
        yw_url = YW_API_TOKEN.format(self.http_type,self.host_ipaddr, self.api_port, customeruuid)
        xauth_headers['X-AUTH-TOKEN'] = authtoken
        api_result = requests.put(yw_url, headers=xauth_headers)
        data = json.loads(api_result.text)
        if 'error' in data:
            raise LDAPSyncException("Failed to receive apitoken from API: {}".format(data['error']))
        if 'apiToken' in data:
            apitoken = data['apiToken']
        return apitoken

    def get_universe_details(self, apitoken, customeruuid, universe_name):
        """
        Routine that gets the list of universes from the YW API.
        :Param apitoken - the apitoken from get_api_token()
        :Param customeruuid - the customer uuid of the YW instance
        :Param universe_name - the name of the universe we want to select
        :Return universe - selected universe details
        """
        xapi_headers = {}
        universes = {}
        universe = {}
        yw_url = YW_API_UNIVERSE_LIST.format(self.http_type,self.host_ipaddr, self.api_port, customeruuid)
        xapi_headers['X-AUTH-YW-API-TOKEN'] = apitoken
        api_result = requests.get(yw_url, headers=xapi_headers, verify=False)
        data = json.loads(api_result.text)
        if not data:
            raise LDAPSyncException("No universes in API for customer: {}".format(customeruuid))
        for curr_idx in range(len(data)):
            current_universe = data[curr_idx]['name']
            logging.debug("Universe list has {} {}".format(data[curr_idx]['name'],data[curr_idx]['universeDetails']['universeUUID']))
            universe_details = data[curr_idx]['universeDetails']
            universes[current_universe] = universe_details
        if universe_name in universes:
            universe_data = universes[universe_name]
            universe['universe_data'] = universe_data
        else:
            raise LDAPSyncException("Invalid universe name specified: {}. The following Universe names are valid:{}".format(universe_name,list(universes)))
        if 'universeUUID' in universe_data:
            universe['universeuuid'] = universe_data['universeUUID']
        if 'rootCA' in universe_data:
            universe['rootCA'] = universe_data['rootCA']
        else:
            universe['rootCA'] = None
        if universe_data['nodeDetailsSet']:
            if 'yqlServerRpcPort' in universe_data['nodeDetailsSet'][0]:
                universe['ycql_port'] = universe_data['nodeDetailsSet'][0]['yqlServerRpcPort']
            else:
                universe['ycql_port'] = 9042
            if 'ysqlServerRpcPort' in universe_data['nodeDetailsSet'][0]:
                universe['ysql_port'] = universe_data['nodeDetailsSet'][0]['ysqlServerRpcPort']
            else:
                universe['ysql_port'] = 5433
        universe['node_list'] = []
        for node_idx in range(len(universe_data['nodeDetailsSet'])):
            private_ip = universe_data['nodeDetailsSet'][node_idx]['cloudInfo']['private_ip']
            universe['node_list'].append(private_ip)
        return universe

    def download_root_certificate(self, apitoken, customeruuid, rootuuid):
        """
        Routine to get the CA root certificate from YW API for communicating with the database.
        :Param apitoken - the apitoken from the YW API
        :Param customeruuid - the customer UUID
        :Param rootuuid - the uuid of the root certificate gathered fom the universe details
        :Return certpath -- the path to the certificate
        """
        xapi_headers = {}
        certpath = None
        yw_url = YW_CERT_API.format(self.http_type,self.host_ipaddr, self.api_port, customeruuid, rootuuid)
        xapi_headers['X-AUTH-YW-API-TOKEN'] = apitoken
        api_result = requests.get(yw_url, headers=xapi_headers, verify=False)
        if api_result.content:
            result = json.loads(api_result.content)
            certificate = result['root.crt'].replace("\\n","\n")
            certpath = self.make_temporary_certfile()
            with open(certpath, 'w', encoding='ascii') as certfile:
                certfile.write(certificate)
            certfile.close()
        else:
            raise LDAPSyncException("Failure to retrieve certificate {}".format(rootuuid))
        return certpath

    def connect_to_ysql(self, universe, dbuser, dbpass, dbcert):
        """
        Routine to connect to YSQL database.
        :Param contactlist - list of IP addresses to try and contact
        :Param dbuser - username that has privelege to login and create/modify users
        :Param dbpass - password
        :Param dbcert - dictionary with certificate path if required
        :Return connection
        """
        sslmode = None
        sslrootcert = None
        sslcert = None
        sslkey = None
        conn = None
        #contact_string = ','.join(universe['node_list'])
        # the YB version of psycopg2 supports multi hosts (pip install psycopg2-yugabytedb)
        # but the default does non - use only one conn - since we do only one query
        contact_string = random.choice(universe['node_list'])
        if self.args.dbhost:
            contact_string = self.args.dbhost
        logging.debug("Connecting to PG host {}".format(contact_string))
        if dbcert['sslmode'] is not None:
            sslmode = dbcert['sslmode']
            sslrootcert = dbcert['root_certificate_path']
            sslcert = dbcert['user_certificate_path']
            sslkey = dbcert['user_certificate_key']
        try:
            conn = psycopg2.connect(database=self.args.dbname,
                                    user=dbuser,
                                    password=dbpass,
                                    host=contact_string,
                                    port=universe['ysql_port'],
                                    sslmode=sslmode,
                                    sslrootcert=sslrootcert,
                                    sslcert=sslcert,
                                    sslkey=sslkey)
        except psycopg2.OperationalError as pg_ex:
            raise LDAPSyncException("Failure to connect to YSQL: {}".format(str(pg_ex)))
        finally:
            if conn:
                logging.info("Successfully connected to YSQL.")
        return conn

    @classmethod
    def connect_to_ycql(cls, universe, dbuser, dbpass, dbcert):
        """
        Routine to connect to YCQL database.
        :Param universe - dictionary including list of IP addresses to try and contact
        :Param dbuser - username that has privelege to login and create/modify users
        :Param dbpass - password
        :Param dbcert - dictionary with certificate path if required
        :Return connection
        """
        ssl_context = None
        attempt = 0
        connected = False
        session = None
        auth_provider = PlainTextAuthProvider(username=dbuser, password=dbpass)
        if dbcert['root_certificate_path'] is not None:
            ssl_context = dict(ca_certs = dbcert['root_certificate_path'],
                               cert_reqs = ssl.CERT_REQUIRED,
                               ssl_version = ssl.PROTOCOL_TLSv1_2)

        while not connected and attempt < 10:
            try:
                cluster = Cluster(contact_points=universe['node_list'],
                                  port=universe['ycql_port'],
                                  load_balancing_policy=DCAwareRoundRobinPolicy(),
                                  ssl_options=ssl_context,
                                  auth_provider=auth_provider)
                session = cluster.connect('system_auth')
                connected = True
                logging.info('YCQL API connection successful.')
            except NoHostAvailable:
                logging.warning('YCQL Connection failed: NO_HOST_AVAILABLE attempt %d', attempt)
                attempt += 1
                time.sleep(10)
        return session

    @classmethod
    def ysql_auth_to_dict(self, session, allow_drop_superuser):
        """
        Routine to query the PG catalog for all roles and grants.
        Explicitly queries for users that can login and (conditionally) are not superuser.
        :Param session - an active session from connect_to_ycql
        :Return dbdict - dictionary contain the state of the database
        """
        dbdict = {}
        db_nologin_role = {}
        owned_object_count = {}
        with session:
            owned_cursor = session.cursor(cursor_factory=psycopg2.extras.DictCursor)
            owned_cursor.execute(YSQL_OWNED_OBJECTS)
            rows = owned_cursor.fetchall()
            for row in rows:
                owned_object_count[row['role']] = row['owned_objects']
            owned_cursor.close()
            #
            auth_cursor = session.cursor(cursor_factory=psycopg2.extras.DictCursor)
            auth_cursor.execute(YSQL_ROLE_QUERY.format("" if allow_drop_superuser else "AND r.rolsuper='f'"))
            rows = auth_cursor.fetchall()
            for row in rows:
                if row['can_login']:
                    dbdict[row['role']] = row['member_of']
                else:
                    db_nologin_role[row['role']] = row['member_of']
            auth_cursor.close()
        return dbdict,db_nologin_role,owned_object_count

    @classmethod
    def ycql_auth_to_dict(self, session, allow_drop_superuser):
        """
        Routine to query the system_auth keyspace for all roles and grants.
        Explicitly queries for users that can login and (conditionally) are not superuser.
        TODO: Look at universe Gflags to see if there are excluded users from ycql ldap Gflag
        :Param session - an active session from connect_to_ycql
        :Return dbdict - dictionary contain the state of the database
        """
        dbdict = {}
        db_nologin_role = {}
        session.row_factory = dict_factory
        rows = session.execute(YCQL_ROLE_QUERY.format(
                  "" if allow_drop_superuser else "AND is_superuser = false"))
        for row in rows:
          if row['can_login']:
            dbdict[row['role']] = row['member_of']
          else:
             db_nologin_role[row['role']] = row['member_of']
             
        return dbdict,db_nologin_role

    def yb_init_ldap_conn(self, ldap_uri, ldap_user, ldap_password, ldap_usetls, ldap_certificate):
        """
        Routine to connect to ldap given the parameters documented here.
        :Param ldap_uri -- the ldap server
        :Param ldap_user - the DN of the ldap user
        :Param ldap_password - the password of the ldap user
        :Param ldap_usetls - whether TLS should be used
        :Param ldap_certificate - the path to the certificate
        :Return connection
        """
        if self.args.ldap_testvalue:
           return "Non-null Invalid value" # TEST mode - dont connect 
           
        connect = None
        try:
            connect = ldap.initialize(ldap_uri)
            connect.set_option(ldap.OPT_PROTOCOL_VERSION, 3)
            connect.set_option(ldap.OPT_REFERRALS, 0)
            if ldap_uri.startswith('ldaps://'):
                connect.set_option(ldap.OPT_X_TLS_REQUIRE_CERT, ldap.OPT_X_TLS_ALLOW)
            if ldap_usetls:
                connect.set_option(ldap.OPT_X_TLS, True)
                connect.set_option(ldap.OPT_X_TLS_DEMAND, True)
                if ldap_certificate:
                    connect.set_option(ldap.OPT_X_TLS_CACERTFILE, ldap_certificate)
                connect.set_option(ldap.OPT_X_TLS_REQUIRE_CERT, ldap.OPT_X_TLS_ALLOW)
                connect.set_option(ldap.OPT_X_TLS_NEWCTX, 0)
                connect.start_tls_s()
            connect.simple_bind_s(ldap_user, ldap_password)
        except ldap.SERVER_DOWN as ex:
            raise LDAPSyncException("LDAP server {} is down: {}".format(ldap_uri, str(ex))) from ex
        except ldap.INVALID_CREDENTIALS as ex:
            if connect:
                connect.unbind_s()
            raise LDAPSyncException("Wrong username or password: {}".format(str(ex)))
        except ldap.LDAPError as ex:
            raise LDAPSyncException("LDAP Error: {}: Type: {}".format(str(ex), type(ex)))
        finally:
            if connect:
                logging.info('Connected to LDAP Server %s', ldap_uri)
        return connect

    def query_ldap(self, connect, basedn, search_filter):
        """
        Routine to query ldap directory
        """
        if self.args.ldap_testvalue:
           import ast
           return  dict(ast.literal_eval(self.args.ldap_testvalue)) # TEST mode - returnedd canned value
           
        # Assume SCOPE_SUBTREE
        ldap_result_dict = None
        result = None
        scope = ldap.SCOPE_SUBTREE
        logging.info('query_ldap - basedn %s', basedn)
        logging.info('query_ldap - filter %s', search_filter)
        try:
            result = connect.search_s(basedn, scope, search_filter)
            logging.debug('Raw LDAP results {}'.format(result))
        except ldap.LDAPError as ex:
            raise LDAPSyncException("LDAP Error: {}: Type: {}".format(str(ex), type(ex)))
        finally:
            if result and len(result) > 0:
                ldap_result_dict = dict(result)
                logging.info('query_ldap returned %d items', len(result))
            else:
                if result:
                    logging.warning('query_ldap returned no items')
        return ldap_result_dict

    @classmethod
    def process_ldap_user_list(cls, ldap_raw, userfield, groupfield):
        """
        Routine to process the raw LDAP dictionary into the format of user and list of groups
        similar to how it appears in YCQL and YSQL.
        :Param ldap_raw - the incoming dictionary
        :Param userfield - the userfield name in the dictionary
        :Param groupfield - the groupfield name in the dictionary
        :Return new formatted dictionary
        """
        ldap_dict = {}
        result_count = 0
        logging.info('Processing LDAP User result into dictionary')
        for user_key, group_value in ldap_raw.items():
            logging.debug('Processing ldap item with user_key %s and group_value %s', user_key, group_value)
            user = dict(item.split("=") for item in user_key.split(","))[userfield]
            group_list = []
            if 'memberOf' in group_value:
                for group_idx in range(len(group_value['memberOf'])):
                    group = dict(item.split("=")
                                 for item in
                                 group_value['memberOf'][group_idx].decode().split(","))[groupfield]
                    group_list.append(group)
            else:
                logging.info('No groups found for user {}'.format(user_key))
                continue
            ldap_dict[user] = group_list
            result_count += 1
        logging.info('Processed %d results into dictionary', result_count)
        return ldap_dict

    def process_ldap_group_list(self, ldap_raw, userfield, groupfield):
        """
        Routine to process the raw LDAP dictionary into the format of user and list of groups
        similar to how it appears in YCQL and YSQL.
        :Param ldap_raw - the incoming dictionary
        :Param userfield - the userfield name in the parameters
        :Param groupfield - the groupfield name in the parameters
        :Return new formatted dictionary {user1:[grp1,grp2..], ...}
        """
        ldap_dict = {}
        result_count = 0
        logging.info('Processing LDAP group result into dictionary (process_ldap_group_list)')
        if "LDAPRAW" in self.args.reports or "ALL" in self.args.reports:
           logging.debug("Printing 'LDAP RAW' report with {} LDAP items".format(len(ldap_raw)))
           self.Print_Report("LDAP RAW", ldap_raw,"member,name")
           
        for group_dn, group_att in ldap_raw.items():
            logging.debug('Processing ldap item with Group %s and group_att %s', group_dn, group_att)
            if group_dn == None  or  len(group_dn) < 3:
                continue
            if not type(group_att) is dict:
                logging.info("LDAP Group {} has no attributes. Ignored.".format(group_dn))
                continue
            if not ('member' in group_att  and  len(group_att['member']) > 0):
                logging.warning("LDAP Group {} has no members. Ignored.".format(group_dn))
                continue
            group_dict = dict(item.split("=") for item in group_dn.split(","))
            group = None
            if groupfield in group_dict:
               group = group_dict[groupfield]
            elif groupfield in group_att:
                   group = group_att[groupfield][0].decode()
            else:
                   logging.warning("Did not find '{}' in group atts for group {}. Ignoring group.".format(groupfield,group_dn))
                   continue
            member_list = group_att['member']
            logging.debug("   GROUP {}: MEMBERS {}".format(group,member_list))
            for member in member_list:
               logging.debug ("   Working on member {} of type {};".format(member,type(member)))
               # member_dn= dict( x.split('=') for x in member.decode().split(","))
               # The "CN" part of the name may contain escaped commas, so translate those before split on comma.
               member_dn =  dict( x.split('=') for x in member.decode().replace('\\,','/').split(","))

               logging.debug("    Member:{}; Mem DN={}".format(member,member_dn))
               if userfield not in member_dn:
                  logging.debug("User {} does not contain a {} (userfield). Fetching user details...".format(member, userfield))
                  # We do ldap FETCH of all user atts
                  member_dn = self.query_ldap(self.ldap_connection,
                                            member.decode(),
                                            "(objectCategory=user)")
                  if isinstance(member_dn,(list,bytearray,tuple)): # it is a single-item list that looks like: [(DN,{dn-values})]
                      member_dn = member_dn[0][1]                  # Second item in tuple is a DICT of {user=> atts-dict}
                  if len(member_dn) == 1:
                      member_dn = next(iter(member_dn.values()))   # Extract the first VALUE from dict(atts) (which is also a DICT)
                  if userfield not in member_dn:
                      logging.warning("User {} does not contain a '{}' (userfield). Detail:{}".format(member, userfield,member_dn))
                      continue
                  
               user= member_dn[userfield]
               if isinstance(user,(bytearray,list,tuple)):
                   user = user[0]
               if isinstance(user,(bytes)):
                   user = user.decode()
               logging.debug("   User={}".format(user))
               if user not in ldap_dict:
                        ldap_dict[user] = []
               ldap_dict[user].append(group)
            result_count += 1
        logging.info('Processed %d LDAP groups into dictionary, which contains info for %d users', result_count,len(ldap_dict))
        if "LDAPBYUSER" in self.args.reports or "ALL" in self.args.reports:
           self.Print_Report("LDAP BY USER",ldap_dict)

        return ldap_dict

        
    @classmethod
    def compute_changes(cls, ldap_dict, db_role_dict):
        """
        Routine to compute the changes between two dictionaries.
        Uses DeepDiff and Delta from deepdiff.
        :Param ldap_dict -- the incoming dictionary or fresh dictionary from LDAP
        :Param db_role_dict -- the current dictionary or dictionary from the DB
        :Return dictionary with the computed changes. If equivalent it will be empty.
        """
        logging.info('Compute changes phase...')
        xdiff = DeepDiff(db_role_dict, ldap_dict, ignore_order=True, report_repetition=True)
        deltadiff = Delta(xdiff, serializer=json.dumps)
        diff_library = json.loads(deltadiff.dumps())
        return diff_library

    @classmethod
    def process_changes(cls, diff_library, target_api, owned_counts, db_nologin_role, member_map):
        """
        Routine to process the computed changes back to YCQL.
        :Param diff_library -- the compute changes in dictionary form accessed by
        dictionary_item_add, dictionary_item_removed, iterable_items_added_at_indexes,
        iterable_items_removed_at_indexes
        :Return list of statements to execute against YCQL/YSQL
        """
        # Process new records - dictionary_item_added
        stmt_list = []
        mmap_stmt_list = []
        stmt_type = {'AddRole':0, 'DropRole':0, 'GrantRole':0, 'Revoke':0}
        logging.debug("Change Dictionary for {}:{}".format(target_api,diff_library))
        logging.debug("No Login roles for {}:{}".format(target_api, db_nologin_role))

        # Process member map items
        if 'dictionary_item_added' in diff_library: # In LDAP, NOT in DB
            # first, build out member map for YSQL
            if target_api == 'YSQL':
                if member_map is not None:
                    for m in member_map:
                        if m[1] not in db_nologin_role:
                            logging.debug("Creating (non-login) member mapped role {} in DB".format(m[1]))
                            mmap_stmt_list.append(YSQL_CREATE_NOLOGIN_ROLE.format(m[1]))
                            stmt_type['AddRole'] +=1
                            db_nologin_role[m[1]] = []
                        for k, v in diff_library['dictionary_item_added'].items():
                            regex = m[0]
                            role = m[1]
                            usr = get_uid_from_ddiff(k)
                            if re.search(regex, usr) is not None:
                                grant_roles = ','.join(['{0}'.format(role) for role in v])
                                if role not in grant_roles:
                                    logging.debug("Adding user {} to member mapped role {} in DB".format(get_uid_from_ddiff(k), role))
                                    mmap_stmt_list.append(YSQL_GRANT_ROLE.format(role, usr))
                                    stmt_type['GrantRole'] += 1

            for key, value in diff_library['dictionary_item_added'].items():
                logging.debug("Adding DB role for {}".format(key))
                stmt_type['AddRole'] +=1
                role_to_create = get_uid_from_ddiff(key)
                if target_api == 'YCQL':
                    stmt_list.append(YCQL_CREATE_ROLE.format(role_to_create, generate_random_password()))
                    for grant_role in value:
                        if not grant_role in db_nologin_role:
                           logging.debug("CREATing (non-login) role {} in DB, to assign to {}".format(grant_role,key))
                           stmt_list.append(YCQL_CREATE_NOLOGIN_ROLE.format(grant_role))
                           db_nologin_role[grant_role] = 1
                        stmt_list.append(YCQL_GRANT_ROLE.format(grant_role, role_to_create))
                        stmt_type['GrantRole'] +=1
                else:
                    grant_roles = '"' + '","'.join(['{0}'.format(role) for role in value]) + '"'
                    for grant_role in value:
                        if not grant_role in db_nologin_role:
                           logging.debug("CREATing (non-login) role {} in DB, to assign to {}".format(grant_role, key))
                           if grant_role == grant_roles.strip('"'):
                               stmt_list.append(YSQL_CREATE_NOLOGIN_ROLE.format(grant_role))
                           else:
                            stmt_list.append(YSQL_CREATE_NOLOGIN_ROLE_IN_ROLES.format(grant_role, grant_roles))
                            stmt_list.append(YSQL_GRANT_ROLE.format(grant_role, grant_roles.strip('"')))
                            stmt_type['GrantRole'] += 1
                           db_nologin_role[grant_role] = 1

                    if (role_to_create == grant_roles.strip('"')):
                        stmt_list.append(YSQL_CREATE_ROLE.format(role_to_create,
                                                            generate_random_password()))
                    else:
                        stmt_list.append(YSQL_CREATE_ROLE_IN_ROLES.format(role_to_create,
                                                            generate_random_password(),
                                                            grant_roles))

        # Process dropped records - dictionary_item_removed
        if 'dictionary_item_removed' in diff_library: # in DB, Not in LDAP 
            for key, value in diff_library['dictionary_item_removed'].items():
                role_to_drop = get_uid_from_ddiff(key)
                if role_to_drop in db_nologin_role:
                    continue # Don't drop non-login roles ..
                logging.debug("Dropping DB role for {}".format(key))
                stmt_type['DropRole'] +=1
                if target_api == 'YCQL':
                    stmt_list.append(YCQL_DROP_ROLE.format(role_to_drop))
                elif role_to_drop in owned_counts and owned_counts[role_to_drop] > 0:
                    logging.error("ERROR: Could not drop ROLE '{}' because it owns {} objects".format(role_to_drop,owned_counts[role_to_drop]))
                    stmt_list.append("--ERROR: Role {} could not be dropped because it owns {} objects".format(role_to_drop, owned_counts[role_to_drop]))
                else:
                    stmt_list.append(YSQL_DROP_ROLE.format(role_to_drop))
                    
        # Process changed records - new attribute - iterable_item_added
        if 'iterable_items_added_at_indexes' in diff_library: # Permission exists in LDAP, not in DB 
            for key, value in diff_library['iterable_items_added_at_indexes'].items():
                role_to_modify = get_uid_from_ddiff(key)
                logging.debug("GRANTing DB role {} to users:{}".format(key,value.values()))
                grant_role_list = []
                for grant_role in value.values():
                    stmt_type['GrantRole'] +=1
                    if target_api == 'YCQL':
                        if not grant_role in db_nologin_role:
                            logging.debug("CREATing (non-login) role {} in DB, to assign to {}".format(grant_role, key))
                            stmt_list.append(YCQL_CREATE_NOLOGIN_ROLE.format(grant_role))
                            db_nologin_role[grant_role] = 1
                        stmt_list.append(YCQL_GRANT_ROLE.format(grant_role, role_to_modify))
                    else:
                        grant_role_list.append(grant_role)
                        grant_roles = ','.join(['{0}'.format(role) for role in grant_role_list])
                        if not grant_role in db_nologin_role:
                            logging.debug("CREATing (non-login) role {} in DB, to assign to {}".format(grant_role, key))
                            stmt_list.append(YSQL_CREATE_NOLOGIN_ROLE.format(grant_role, grant_role))
                            db_nologin_role[grant_role] = 1
                        if grant_roles != role_to_modify:
                            stmt_list.append(YSQL_GRANT_ROLE.format(grant_roles, role_to_modify))

        # Process changed records - delete attribute - iterable_item_removed
        if 'iterable_items_removed_at_indexes' in diff_library: # Permission is in DB, not in LDAP
            mmap_roles = []

            if member_map is not None:
                for m in member_map:
                   mmap_roles.append(m[1])
            for key, value in diff_library['iterable_items_removed_at_indexes'].items():
                logging.debug("Revoking DB role for {}".format(key))
                role_to_modify = get_uid_from_ddiff(key)
                revoke_role_list = []
                for revoke_role in value.values():
                    if revoke_role in mmap_roles:
                        continue  # Do not revoke member mapped roles
                    stmt_type['Revoke'] +=1
                    if target_api == 'YCQL':
                        stmt_list.append(YCQL_REVOKE_ROLE.format(revoke_role, role_to_modify))
                    else:
                        revoke_role_list.append(revoke_role)
                if target_api == 'YSQL' and len(revoke_role_list) > 0:
                    revoke_roles = ','.join(['{0}'.format(role) for role in revoke_role_list])
                    stmt_list.append(YSQL_REVOKE_ROLE.format(revoke_roles, role_to_modify))
        # add member map to end of statements
        stmt_list.extend(mmap_stmt_list)
        print ('Prepared {} database update statements: {}'.format(len(stmt_list), stmt_type))
        return stmt_list

    @classmethod
    def apply_changes_ysql(cls, session, stmt_list):
        """
        Routine to apply changes to YSQL
        :Param session - YSQL session object
        :Param stmt_list - List of statements to execute
        """
        exec_cursor = session.cursor()
        for stmt in stmt_list:
            if stmt.startswith('CREATE ROLE'):
                logging.info('Creating new user: %s.',stmt.split(" ")[2])
            elif stmt.startswith('--'):
                logging.info(stmt) # It is a SQL comment
                continue           # Do not execute it 
            else:
                logging.info('Applying statement: %s', stmt)
            exec_cursor.execute(stmt)
        exec_cursor.close()

    @classmethod
    def apply_changes_ycql(cls, session, stmt_list):
        """
        Routine to apply changes to YCQL
        :Param session - YCQL session object
        :Param stmt_list - List of statements to execute
        """
        for stmt in stmt_list:
            if stmt.startswith('CREATE ROLE'):
                user_name = re.search('CREATE ROLE\s+(?:IF NOT EXISTS\s*)?[\'"]?([\w\-]+)',stmt,re.IGNORECASE)
                if user_name:
                   user_name = user_name.group(1) 
                else:
                   user_name = "*Unable to extract username from:" + stmt
                logging.info('Creating new user: %s.',user_name) # after 'IF NOT EXISTS'
            else:
                logging.info('Applying statement: %s', stmt)
            session.execute(stmt)

    def apply_changes(self, process_diff, universe, owned_counts, db_nologin_role, member_map=None):
        """
        Routine to apply changes to the given universe
        :Param process_diff - dictionary of changes that will be used to generate a list
        :Param universe - universe dictionary connection details
        """
        stmt_list = None
        db_certificate = universe['db_certificate']
        stmt_list = self.process_changes(process_diff, self.args.target_api, owned_counts, db_nologin_role, member_map)
        if "DBUPDATES" in self.args.reports or "ALL" in self.args.reports:
           self.Print_Report("DB UPDATES",stmt_list,0)
        if self.args.dryrun:
            print("--- Dry Run -- {} statements created. (No changes will be made) ---".format(len(stmt_list)))
            for stmt in stmt_list:
                print("  {}".format(stmt))
        elif self.args.target_api == 'YCQL':
            if not self.ycql_session:
                self.ycql_session = self.connect_to_ycql(universe,
                                                         self.args.dbuser,
                                                         self.args.dbpass,
                                                         db_certificate)
            self.apply_changes_ycql(self.ycql_session, stmt_list)
        else:
            if not self.ysql_session:
                self.ysql_session = self.connect_to_ysql(universe,
                                                         self.args.dbuser,
                                                         self.args.dbpass,
                                                         db_certificate)
            self.apply_changes_ysql(self.ysql_session, stmt_list)

    def query_db_state(self, universe):
        """
        Routine to query the database and extract the current state of users/groups
        :Param universe - the universe to connect to
        :Return dictionary that has the current state
        """
        db_certificate = universe['db_certificate']
        owned_counts={}
        db_role_dict={}
        db_nologin_role={}
        if self.args.target_api == 'YCQL':
            if not self.ycql_session:
                self.ycql_session = self.connect_to_ycql(universe,
                                                         self.args.dbuser,
                                                         self.args.dbpass,
                                                         db_certificate)
            db_role_dict,db_nologin_role = self.ycql_auth_to_dict(self.ycql_session, self.args.allow_drop_superuser)
        else:
            if not self.ysql_session:
                self.ysql_session = self.connect_to_ysql(universe,
                                                         self.args.dbuser,
                                                         self.args.dbpass,
                                                         db_certificate)
            (db_role_dict, db_nologin_role, owned_counts) = self.ysql_auth_to_dict(self.ysql_session, self.args.allow_drop_superuser)
        logging.info("Loaded {} DB Users (allow_drop_superuser={}).".format(len(db_role_dict),self.args.allow_drop_superuser))
        logging.debug(" DB Users:{}; NO-Login:{}".format(db_role_dict, db_nologin_role))

        if "DBROLE" in self.args.reports or "ALL" in self.args.reports:
           self.Print_Report("DB ROLE (allow_drop_superuser={})".format(self.args.allow_drop_superuser),db_role_dict)
        return db_role_dict,owned_counts,db_nologin_role

    def setup_yb_tls(self, universe, api_token, customeruuid):
        """
        Routine to setup TLS for connection to YugabyteDB.
        :Param universe -- the universe dictionary
        :Param api_token - YW API token
        :Param customeruuid - customer uuid
        :Return dictionary of TLS data
        """
        db_certificate = {}
        db_certificate['root_certificate_path'] = None
        db_certificate['sslmode'] = None
        db_certificate['user_certificate_path'] = None
        db_certificate['user_certificate_key'] = None
        if not universe['rootCA'] is None:
            logging.info('Downloading root certificate for universe...')
            db_certificate['root_certificate_path'] = self.download_root_certificate(
                                                                api_token,
                                                                customeruuid,
                                                                universe['rootCA'])
            logging.info('Downloaded root certificate to %s',
                         db_certificate['root_certificate_path'])
        if self.args.target_api == 'YSQL':
            db_certificate['sslmode'] = self.args.db_sslmode
            db_certificate['user_certificate_path'] = self.args.db_certpath
            db_certificate['user_certificate_key'] = self.args.db_certkey
        return db_certificate

    def post_process_args(self):
        """ Routine to post process arguments and do validation. """
        if self.args.ldapserver.startswith('ldap:') or self.args.ldapserver.startswith('ldaps:'):
            pass
        else:
            raise LDAPSyncException("Incorrect format for ldap server specification")
        if self.args.db_certpath and not os.access(self.args.db_certpath, os.R_OK):
            raise LDAPSyncException("DB cert {} not found".format(self.args.db_certpath))
        if self.args.db_certkey and not os.access(self.args.db_certkey, os.R_OK):
            raise LDAPSyncException("DB key {} not found".format(self.args.db_certkey))
        if self.args.ldap_certificate and not os.access(self.args.ldap_certificate, os.R_OK):
            raise LDAPSyncException("LDAP cert {} not found".format(self.args.ldap_certificate))
        if self.args.use_https:
            self.http_type = 'https'
        if self.args.apitoken or self.args.apiuser:
            pass
        else:
            raise LDAPSyncException("ERROR: Either --apitoken  OR (--apiuser + --apipassword) MUST BE SPECIFIED")
        if self.args.verbose:
            logging.getLogger().setLevel(logging.INFO)
            logging.info("Verbose logging enabled (logging.INFO)")
        if self.args.debug:
            logging.getLogger().setLevel(logging.DEBUG)
            # Enabling debugging at http.client level (requests->urllib3->http.client)
            # you will see the REQUEST, including HEADERS and DATA, and RESPONSE with HEADERS but without DATA.
            # the only thing missing will be the response.body which is not logged.
            import http.client
            http_client_logger = logging.getLogger("http.client")
            #the http.client library does not use logging to output debug messages; it will always use print()
            def print_to_log(*args):
                http_client_logger.debug(" ".join(args)) 
            # monkey-patch a `print` global into the http.client module; all calls to
            # print() in that module will then use our print_to_log implementation
            http.client.print = print_to_log
            try: # for Python 3
                from http.client import HTTPConnection
            except ImportError:
                # Python 2
                from httplib import HTTPConnection
            HTTPConnection.debuglevel = 1
            requests_log = logging.getLogger("requests.packages.urllib3")
            requests_log.setLevel(logging.DEBUG)
            requests_log.propagate = True
        logging.debug("DEBUG logging enabled")

        
    @classmethod
    def parse_arguments(cls):
        """
        Routine to parse arguments from the command line.
        """
        parser = argparse.ArgumentParser(description='YB LDAP sync script, Version {}'.format(VERSION))
        parser.add_argument('--debug', required=False,  action='store_true', default=os.getenv("DEBUG",False),
                            help="Enable debug logging (including http request logging) for this script")        
        parser.add_argument('--verbose', required=False,  action='store_true', default=os.getenv("VERBOSE",False),
                            help="Enable verbose logging for each action taken in this script")                                    
        parser.add_argument('--apihost', required=False, default=os.getenv("APIHOST"),
                            help="YBA/YW API Hostname or IP (Defaults to localhost)")
        parser.add_argument('--apitoken', required=False, default=os.getenv("APITOKEN"),
                            help="YW API TOKEN - Preferable to use this instead of apiuser+apipassword")
        parser.add_argument('--use_https', required=False,  action='store_true', default=os.getenv("USE_HTTPS",False),
                            help="YW API http type : Set for https (default is false (http))")
        parser.add_argument('--apiport', required=False, default=os.getenv("APIPORT",9000), type=int,
                            help="YW API PORT: Defaults to 9000, which is valid if running inside docker. For external, use 80 or 443" )
        parser.add_argument('--apiuser', required=False, default=os.getenv("APIUSER"),
                            help="YW API Username")
        parser.add_argument('--apipassword', required=False, default=os.getenv("APIPASSWORD"),
                            help="YW API Password")
        parser.add_argument('--ipv6', action='store_false', default=os.getenv("IPV6",False),
                            help="Is system ipv6 based")
        parser.add_argument('--target_api', default=os.getenv("TARGET_API","YCQL"), metavar="YCQL|YSQL",
                            choices=['YCQL', 'YSQL'],
                            type=str.upper,
                            help="Target API: YCQL or YSQL")
        parser.add_argument('--universe_name', required=True,default=os.getenv("UNIVERSE_NAME"),
                            help="Universe name")
        parser.add_argument('--dbhost', required=False, default=os.getenv("DBHOST"),
                            help="Database hostname of IP. Uses a random YB node if not specified.")
        parser.add_argument('--dbuser', required=True,default=os.getenv("DBUSER"),
                            help="Database user to connect as")
        parser.add_argument('--dbpass', required=True, default=os.getenv("DBPASS"),
                            help="Password for dbuser")
        parser.add_argument('--dbname', default=os.getenv("DBNAME",'yugabyte'),
                            type=str.lower,
                            help="YSQL database name to connect to")
        parser.add_argument('--db_sslmode', default=os.getenv("DB_SSL_MODE"),
                            choices=['disable',
                                     'allow',
                                     'prefer',
                                     'require',
                                     'verify-ca',
                                     'verify-full'],
                            help="SSL mode for YSQL TLS")
        parser.add_argument('--db_certpath', default=os.getenv("DB_CERTPATH"),
                            help="SSL certificate path for YSQL TLS")
        parser.add_argument('--db_certkey', default=os.getenv("DB_CERTKEY"),
                            help="SSL key path for YSQL TLS")
        parser.add_argument('--ldapserver', required=True, default=os.getenv("LDAPSERVER"),
                            help="LDAP server address. Should be prefaced with ldap://hostname")
        parser.add_argument('--ldapuser', required=True, default=os.getenv("LDAPUSER"),
                            help="LDAP Bind DN for retrieving directory information")
        parser.add_argument('--ldap_password', required=True, default=os.getenv("LDAP_PASSWORD"),
                            help="LDAP Bind DN password")
        parser.add_argument('--ldap_search_filter', required=True, default=os.getenv("LDAP_SEARCH_FILTER"),
                            help="LDAP Search filter, like  '(&(objectclass=group)(|(samaccountname=grp1)...))'")
        parser.add_argument('--ldap_basedn', required=True, metavar="dc=dept,dc=corp..", default=os.getenv("LDAP_BASEDN"),
                            help="LDAP BaseDN to search")
        parser.add_argument('--ldap_userfield', required=True, default=os.getenv("LDAP_USERFIELD"),
                            help="LDAP field to determine user's id to create")
        parser.add_argument('--ldap_groupfield', required=True, default=os.getenv("LDAP_GROUPFIELD"),
                            help="LDAP field to grab group information (e.g. cn)")
        parser.add_argument('--ldap_certificate',  default=os.getenv("LDAP_CERTIFICATE"),
                            help="File location that points to LDAP certificate")
        parser.add_argument('--ldap_tls', action='store_false', default=os.getenv("LDAP_TLS",False),
                            help="LDAP Use TLS")
        parser.add_argument('--dryrun', action='store_true', default=os.getenv("DRYRUN",False),
                            help="Show list of potential DB role changes, but DO NOT apply them")
        parser.add_argument('--reports', required=False, type=str.upper,metavar="COMMA,SEP,RPT...", default=os.getenv("REPORTS",""),
                            help="One or a comma separated list of 'tree' reports. Eg: LDAPRAW,LDAPBYUSER,LDAPBYGROUP,DBROLE,DBUPDATES or ALL")
        parser.add_argument('--allow_drop_superuser', action='store_true', default=os.getenv("ALLOW_DROP_SUPERUSER",False),
                            help="Allow this code to DROP a superuser role if absent in LDAP")
        parser.add_argument('--ldap_testvalue', required=False, help=argparse.SUPPRESS,default=os.getenv("LDAP_TESTVALUE"))
                   # This is a HIDDEN arg for TESTING, containing a stringified LDAP search result (if you dont have an LDAP test srv)
        parser.add_argument('--member_map', dest='mmap', nargs='+', action='append',
                            help="Additional YSQL roles to add users to - in the form of [<user regex> <rolename>]")

        return parser.parse_args()

    def run(self):
        """
        Main run routine.
        """
        try:
            self.post_process_args()
            print("[{}] YB LDAP sync script, Version {} for universe {}, {}.".format(time.strftime("%Y-%m-%d %H:%M:%S UTC", gmtime()),
                           VERSION,self.args.universe_name, self.args.target_api))
            logging.info("YW Contact point: %s",  self.host_ipaddr)
            # We do not verify API endpoint certs, so we need to suppress the warning:
            #    "InsecureRequestWarning: Unverified HTTPS request is being made"
            requests.packages.urllib3.disable_warnings(
                                requests.packages.urllib3.exceptions.InsecureRequestWarning)
            if self.args.apitoken is None:
              (auth_token, customeruuid) = self.get_auth_token(self.args.apiuser,
                                                               self.args.apipassword)
              logging.info('Received authtoken %s for customer %s', auth_token, customeruuid)
              api_token = self.get_api_token(auth_token, customeruuid)
              logging.info('Received apitoken %s for customer %s', api_token, customeruuid)
            else:
              api_token = self.args.apitoken
              customeruuid = self.get_customeruuid(api_token)
              logging.info('Customer UUID: %s',customeruuid)
              
            universe = self.get_universe_details(api_token,
                                                 customeruuid,
                                                 self.args.universe_name)
            logging.info('Target API: %s', self.args.target_api)
            logging.info('Node list: %s', universe['node_list'])
            logging.info('Universe uuid: %s', universe['universeuuid'])
            universe['db_certificate'] = self.setup_yb_tls(universe,
                                                           api_token,
                                                           customeruuid)
            old_ldap_data = self.load_previous_ldap_data(customeruuid, universe['universeuuid'])
            self.ldap_connection = self.yb_init_ldap_conn(self.args.ldapserver,
                                             self.args.ldapuser,
                                             self.args.ldap_password,
                                             self.args.ldap_tls,
                                             self.args.ldap_certificate)
            if self.ldap_connection:
                ldap_raw_data = self.query_ldap(self.ldap_connection,
                                                self.args.ldap_basedn,
                                                self.args.ldap_search_filter)
                if ldap_raw_data:
                   new_ldap_data = None
                   if 'objectclass=group' in self.args.ldap_search_filter:
                      new_ldap_data = self.process_ldap_group_list(ldap_raw_data,
                                                             self.args.ldap_userfield,
                                                             self.args.ldap_groupfield)
                   else:
                      new_ldap_data = self.process_ldap_user_list(ldap_raw_data,
                                                             self.args.ldap_userfield,
                                                             self.args.ldap_groupfield)
            process_diff = self.compute_changes(new_ldap_data, old_ldap_data)
            if process_diff:
                logging.info('Detected {} changes from previously saved ldap data'.format(len(process_diff)))
                #Cannot apply yet - need to check for DB user existance## self.apply_changes(process_diff, universe)
            # good idea to save the directory to disk now
            self.save_ldap_data(new_ldap_data, customeruuid, universe['universeuuid'])
            # query database and get current state, compare and process any lingering change
            logging.info('Querying the database for its state of users/groups')
            (db_role_dict,owned_counts,db_nologin_role) = self.query_db_state(universe)
            process_db_diff = self.compute_changes(new_ldap_data, db_role_dict)
            if process_db_diff:
                self.apply_changes(process_db_diff, universe, owned_counts, db_nologin_role, self.args.mmap)
            else:
                print("No DB changes. LDAP and {} are in sync.".format(self.args.target_api))
            if self.ycql_session:
                self.ycql_session.shutdown()
            if self.ysql_session:
                self.ysql_session.close()
            logging.info('Run complete.')
        except LDAPSyncException as ex:
            print(json.dumps({"error": "LDAP Sync exception: {}".format(str(ex))}))
            sys.exit(2)
        except Exception as ex:
            print(json.dumps({"error": "Exception: {}".format(str(ex))}))
            traceback.print_exc()
            traceback.print_stack()
            sys.exit(3)


if __name__ == "__main__":
    logging.basicConfig(level=logging.WARNING, format="%(asctime)s %(levelname)s: %(message)s")
    YBLDAPSync().run()
