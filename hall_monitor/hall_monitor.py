#!python3
# This program is going to get Gflags for an universe.
version = "0.09"
import requests
import urllib3
import json
import argparse
import re
import sys

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# curl -k --silent --request GET --url https://10.231.0.124/api/v1/customers/0c81e8ae-2e1d-44c9-b907-d580da260ff6/universes/ec6b18f3-69b9-43cc-b986-5f60bb0970b5/
# --header 'Accept: application/json' --header 'Content-Type: application/json' --header 'X-AUTH-YW-API-TOKEN: 7f743b97-defc-4e29-8e87-90a12b5d5cf3'

#YBA_HOST = 'https://10.231.0.124'
#CustomerUUID = '0c81e8ae-2e1d-44c9-b907-d580da260ff6'
#UniverseUUID = 'ec6b18f3-69b9-43cc-b986-5f60bb0970b5'
#AUTHTOKEN = '998f5bca-a8fb-464e-bbe5-a72f9e9890b0'
args = None ### This will contain command line Arguments

######################################################################################################################
class yba_api():
    def __init__(self):
        if args.customer_uuid is None:
            self.__set_customeruuid()

        self.baseurl = args.yba_url.strip("/") + '/api/v1/customers/' + args.customer_uuid + '/universes'
        self.raw_response = None

    def __set_customeruuid(self):
        url = args.yba_url.strip("/") + '/api/v1/customers'
        customer_list = self.Get(url, False)
        if len(customer_list) > 1:
            raise SystemExit("ERROR: Multiple Customers Found, Must Specify which Customer UUID to Use!")
        args.customer_uuid = customer_list[0]['uuid']

    def Get(self, targeturl="", use_base=True):
        if use_base:
            url = self.baseurl + targeturl
        else:
            url = targeturl
        self.raw_response = requests.get(url, headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': args.auth_token},
                                verify=False)
        self.raw_response.raise_for_status()
        # Convert response data to dict
        return json.loads(self.raw_response.text)
######################################################################################################################
class Namespace():
    def __init__(self, id,name,type):
        self.tables = []  # Since Namespace has tables
        self.id = id
        self.name = name
        self.type = type

    def Print(self):
        print ("Namespace " + self.name + " (" + self.id + ")" + " (" + self.type + ")" )
######################################################################################################################
class Table():
    def __init__(self, id, name, state, namespace): # "namespace" is the namespace object
        self.tablets = []  # Since Table has tablets
        self.id = id
        self.name = name
        self.state = state
        self.namespace = namespace

    def Print(self):
        print ("Table " + self.name + " (" + self.id + "," + self.state + ")" )
######################################################################################################################
class Tablet():
    def __init__(self, id, table, state, replicas, leader):
        self.id = id
        self.table = table
        self.state = state
        self.replicas = replicas # array of tserver Node objects
        self.leader = leader #Tserver Node object
        self.is_system_tablet = False

    def Print(self):
        leader_txt = "No Leader";
        if self.leader is not None:
            leader_txt = self.leader.name + "("+self.leader.IP + " Node#"+str(self.leader.IDX) +")"
        replica_text = "No Replicas";
        if self.replicas is not None:
            replica_text = "Replicas:"
            for r in self.replicas:
                replica_text= replica_text + " ["+r.name + "("+r.IP + " Node#"+str(r.IDX) +")]" 

        print ("Tablet " + self.id + " (" + self.state + ", leader : " + leader_txt + " "+replica_text+")" )
######################################################################################################################
class Node():
    def __init__(self, node_json, universe_object):
        self.name = node_json['nodeName']
        self.UUID = node_json['nodeUuid']
        self.IP = node_json['cloudInfo']['private_ip']
        self.IDX = node_json['nodeIdx']
        self.azuuid = node_json['azUuid']
        self.ismaster= node_json['isMaster']
        self.istserver = node_json['isTserver']
        self.tserver_uuid = None
        self.master_uuid = None
        self.region = None
        self.az = None
        self.tablets = []
        self.leadercount = 0
        self.universe= universe_object

    def nodeidx(self):
        return self.IDX

    def Print(self, Indent=0):
        print(" "*Indent, "Node: " + str(self.IDX) + "["+ self.UUID + "] " + self.name + " has IP: " + self.IP + \
              " is in Zone " + self.az.name + \
              " [" + (" TServer " if self.istserver else "" ) + (" Master " if self.ismaster else "") + " ]")
######################################################################################################################
class MasterLeader(Node): #Inherited from Node
    def __init__(self,node_object):
        self.__dict__ = node_object.__dict__.copy() # clone the node object

    def get_overview(self):
        url = 'https://' + self.IP + ':' + str(self.universe.masterHttpPort) + '/?raw'
        self.raw_response = requests.get(url,
                                         headers={'Content-Type': 'text/html'},
                                         verify=False)
        self.raw_response.raise_for_status()
        print("==== Master Leader Info ===")
        #print(self.raw_response.content)
        p = re.compile("Load Balancer Enabled.+?=..(\w+)..>.+Is Load Balanced.+?=..(\w+)..>")
        result = p.search(str(self.raw_response.content))
        print ("Load Balancer Enabled: " + result.group(1) + "  Load is Balanced: " + result.group(2))

    def get_tservers(self):
        url = 'https://' + self.IP + ':' + str(self.universe.masterHttpPort) + '/tablet-servers?raw'
        self.raw_response = requests.get(url,
                                         headers={'Content-Type': 'text/html'},
                                         verify=False)
        self.raw_response.raise_for_status()
        p = re.compile(r'.+>([\d\.]+):\d+<.+?(\w{32})<') #regex to search for IP Address and a string after > and looks for a continuous string of 32 characters
        for line in self.raw_response.content.splitlines():
            result = p.search(str(line))
            if result is None:
                continue
            ip = result.group(1)
            uuid = result.group(2)
            print ("TServer IP Address: " + ip + " With TServer UUID: " + uuid)
            for t in self.universe.nodelist:
                if not t.istserver:
                    continue
                if t.IP == ip:
                    t.tserver_uuid = uuid
                    self.universe.tserver_by_uuid[uuid] = t
                    break

    def get_entities(self):
        url = 'https://' + self.IP + ':' + str(self.universe.masterHttpPort) + '/dump-entities'
        self.raw_response = requests.get(url,
                                         headers={'Content-Type': 'text/html'},
                                         verify=False)
        self.raw_response.raise_for_status()
        print("==== Entity Info ===")
        #print(str(self.raw_response.content)[0:200])
        self.entity_json = json.loads(self.raw_response.text)
        #Handle Keyspaces/Namespaces
        for keyspace in self.entity_json['keyspaces']:
            ns = Namespace(keyspace['keyspace_id'],keyspace['keyspace_name'], keyspace['keyspace_type'])
            self.universe.namespace_by_id[ns.id] = ns
        #    print("keyspace:" + keyspace['keyspace_name'] + " " + keyspace['keyspace_type'] )
            ns.Print()

        #Handle Tables (relationship establised to Universe as well as Namespace)
        count = 0
        for table in self.entity_json['tables']:
            #print("table:" + table['table_name'] + " " + table['table_id'] )
            t = Table(table['table_id'], table['table_name'], table['state'], self.universe.namespace_by_id[table['keyspace_id']] )
            self.universe.table_by_id[t.id] = t
            count = count + 1
            if count < 6:
                t.Print() # Print first 5 tables

        #Handle Tablets (Table, Node)
        count = 0
        unknown_tservers = {}
        for tablet in self.entity_json['tablets']:
            #print("tablet:" + table['table_name'] + " " + table['table_id'])
            replica_nodes=[]
            my_table = self.universe.table_by_id[tablet['table_id']] # Table object
            if tablet.get('replicas') is not None:
                for r in tablet.get('replicas'):
                    if self.universe.tserver_by_uuid.get(r['server_uuid']) is None:
                        if unknown_tservers.get(r['server_uuid']) is None:
                            unknown_tservers[r['server_uuid']] = 0
                        unknown_tservers[r['server_uuid']] += 1
                    else:
                        replica_nodes.append(self.universe.tserver_by_uuid[r["server_uuid"]])
            leader = None  # Node object that is this tablet's leader
            if tablet.get('leader') is not None:
                leader = self.universe.tserver_by_uuid.get(tablet.get('leader'))
                if leader is not None:
                    leader.leadercount += 1

            t = Tablet(tablet['tablet_id'], my_table, tablet["state"],
                       replica_nodes, leader) # this needs to be worked further
            for r in t.replicas:    #  attach t (tablet object) to its table
                r.tablets.append(t) # So that Node object has what tablets are resident
            my_table.tablets.append(t)
            if my_table.namespace.name[0:6] == "system":
                t.is_system_tablet = True
            count += 1
            if count < 6:
                t.Print()

        for t,count in unknown_tservers.items():
            print("WARNING: Unknown Tserver with UUID: " + t + " has " + str(count) + " Tablet replicas")
######################################################################################################################

class Zone():
    def __init__(self, zone_json):
        self.name = zone_json['name']
        self.uuid = zone_json['uuid']
        self.ispreferred =  zone_json['isAffinitized']
        self.nodelist = []

    def Print (self, Indent=5):
        print (" "*Indent, "Zone is " + self.name + (' [preferred] ' if self.ispreferred  else " ") + " with UUID " + self.uuid)
        for n in self.nodelist:
            print (" "*(Indent+2), " Node " + n.name)

######################################################################################################################
class Region():
    def __init__(self,region_json):
        self.name = region_json['name']
        self.code = region_json['code']
        self.azlist = []
        for az in region_json['azList']:
            self.azlist.append(Zone(az))

    def Print (self,Indent=5):
        print(" "*Indent + "Region is " + self.name + " and Region code is " + self.code )
        for az in self.azlist:
            az.Print(Indent+2)

######################################################################################################################
class Universe():
    def __init__(self, ujson, api):
        self.json = ujson
        self.yba_api = api
        self.universeDetails = ujson['universeDetails']
        self.name =ujson['name']
        self.UUID = ujson['universeUUID']
        self.paused = self.universeDetails['universePaused']
        self.providertype = self.universeDetails["clusters"][0]["userIntent"]["providerType"]
        self.regionlist = []
        self.masterHttpPort = self.universeDetails["communicationPorts"]["masterHttpPort"]
        self.tserverHttpPort = self.universeDetails["communicationPorts"]["tserverHttpPort"]
        self.tserver_by_uuid = {}
        self.namespace_by_id = {}
        self.table_by_id = {}
        self.namespaces = []
        self.replicationfactor = self.universeDetails["clusters"][0]["userIntent"]["replicationFactor"]


        for r_raw in self.universeDetails["clusters"][0]["placementInfo"]["cloudList"][0]["regionList"]:
            self.regionlist.append(Region(r_raw))
        self.nodelist = []
        for n_raw in self.universeDetails['nodeDetailsSet']:
            self.nodelist.append(Node(n_raw, self))
        self.nodelist.sort(key=Node.nodeidx)   ## This is a reference to a Function (Lamda Concept in Python)

        for r in self.regionlist:
            for a in r.azlist:
                for n in self.nodelist:
                    if a.uuid == n.azuuid:
                        n.az = a
                        n.region = r
                        a.nodelist.append(n)
                        break
        self.master_leader = self.get_master_leader()

    def get_master_leader(self):
        if self.paused:
            return None
        leader = self.yba_api.Get("/" + self.UUID + "/leader")
        # print ("Master Leader = " + str(leader))
        for n in self.nodelist:
            if leader['privateIP'] == n.IP:
                return MasterLeader(n)
        return None

    def Print_underreplicated_tablets(self):
        underreplicated_tablets = 0
        overeplicated_tablets = 0
        total_tablets = 0
        table_count = 0
        for table in self.table_by_id.values():
            table_count += 1
            for tablet in table.tablets:
                if tablet.is_system_tablet:
                    continue
                replicas = len(tablet.replicas)
                total_tablets += 1
                if replicas < self.replicationfactor:
                    print("Underelicated tablet:" + tablet.id + " in table " + table.name + " Found " + str(replicas) + " replicas")
                    underreplicated_tablets += 1
                if replicas > self.replicationfactor:
                    print("Overrelicated tablet:" + tablet.id + " in table " + table.name + " Found " + str(replicas) + " replicas")
                    overeplicated_tablets += 1
        print ("Total Tablets: " +str(total_tablets)+ ", Under Replicated tablets: " + str(underreplicated_tablets) + ", Over replicated tablets: " +str(overeplicated_tablets)+ ", Total tables " + str(table_count))





    def Print(self):
        """
         This prints the Universe and all its sub-objects
        """
        print('===== Universe ===== ' + self.name + ' UUID = ' + self.UUID + " Provider is " + self.providertype \
            + (" Paused " if self.paused else "")) ### name and UUID
        if self.master_leader is not None:
            print("     Master Leader is " + self.master_leader.name + " and Replication Factor is " + str(self.replicationfactor) )
        self.print_gflags(5)### Print the glags for the universe.
        self.print_nodes(5)
        self.print_regions(5)



    def print_gflags(self,Indent=0):
    #    print("======Universe Name is ====== " + self.name)

        print(" "*Indent + "Master Non-Default Gflags: ")
        for flag, value in self.universeDetails['clusters'][0]['userIntent']['masterGFlags'].items():
            print(" "*Indent*2 , flag, ' ', value)
        print(" "*Indent + "TServer Non-Default Gflags: ")
        for flag, value in self.universeDetails['clusters'][0]['userIntent']['tserverGFlags'].items():
            print(" "*Indent*2 , flag, ' ', value)
       # for n in self.nodelist:
       #     print ("Node Name:" + n.name)

    def print_nodes(self, Indent=0):
        print(" "*Indent + "Nodes in the Universe:" + self.name)
        if self.nodelist is None or len(self.nodelist) == 0:
            print (" "*Indent + "No nodes in the Universe")
            return
        for n in self.nodelist:
            n.Print(Indent*2)

    def print_regions(self, Indent=0):
        print(" "*Indent + "Regions in the Universe:" + self.name)
        if self.regionlist is None or len(self.regionlist) == 0:
            print (" "*Indent + "No regions in the Universe")
            return
        for n in self.regionlist:
            n.Print(Indent*2)
##################################################################################
#### Outer block starts here ####
##################################################################################
def discover_universes():
    '''
    url = YBA_HOST + '/api/v1/customers/' + CustomerUUID + '/universes'
    response = requests.get(url, headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': AUTHTOKEN},
                            verify=False)
    response.raise_for_status()
    # Convert response data to dict
    universe_list = json.loads(response.text)
    '''
    api = yba_api()
    found = False

    for u_raw in api.Get():
        u = Universe(u_raw, api) ### This creates a Universe Object named "u"
        if args.universe == None:
            found = True
        elif args.universe == u.name:
            found = True
        elif args.universe == u.UUID:
            found = True
        else:
            continue

        u.Print()
        if args.tablets:
            ml = u.get_master_leader()
            if ml is None:
                print ("Could not find Master leader for " + u.name)
                print ()
                continue
            ml.get_overview()
            ml.get_tservers()
            ml.get_entities()
            for n in u.tserver_by_uuid.values():
                system_tablet_count = 0
                for t in n.tablets:
                    if t.is_system_tablet:
                        system_tablet_count +=1
                print("Node " + n.name + " Has " + str(len(n.tablets)-system_tablet_count) + " Tablets and " + str(n.leadercount) + " Leaders" )
        u.Print_underreplicated_tablets()

        print()

    if not found:
        raise SystemExit("ERROR: Did not find the specified " + args.universe + " Universe!")

#        print_universe_gflags(u)

def parse_arguments():
    parser = argparse.ArgumentParser(
        prog='get_gflags.py',
        description='This Program gets Universe details including tablet information',
        epilog='End of help',
        add_help=True )
    parser.add_argument('-d', '--debug', action='store_true')
    parser.add_argument('-u', '--universe', help='Universe Name for which information is needed')
    parser.add_argument('-t', '--tablets', action='store_true', help='Get tablet details')
    parser.add_argument('-y', '--yba_url', required=True, help='YBAnywhere URL')
    parser.add_argument('-c', '--customer_uuid', help='Customer UUID')
    parser.add_argument('-a', '--auth_token', required=True, help='YBAnywhere Auth Token')


    args = parser.parse_args()
    if args.debug:
        print("Debugging Enabled")
    return args

######################################################################################################################
# Main Program starts here
######################################################################################################################
args = parse_arguments()
discover_universes()



###
# Print Region for the Universe. (May need two Classes, AZ and Region)
# Print the List of AZ for the Universe
# Which AZ got how many nodes
###
# All tablet details - https://10.231.0.124/universes/ec6b18f3-69b9-43cc-b986-5f60bb0970b5/proxy/10.231.0.10:7000/dump-entities

###
