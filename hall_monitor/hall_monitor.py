#!python3
# This program is going to get Gflags for an universe.
version = "0.04"
import requests
import urllib3
import json
import argparse
import re
import sys

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# curl -k --silent --request GET --url https://10.231.0.124/api/v1/customers/0c81e8ae-2e1d-44c9-b907-d580da260ff6/universes/ec6b18f3-69b9-43cc-b986-5f60bb0970b5/
# --header 'Accept: application/json' --header 'Content-Type: application/json' --header 'X-AUTH-YW-API-TOKEN: 7f743b97-defc-4e29-8e87-90a12b5d5cf3'

YBA_HOST = 'https://10.231.0.124'
CustomerUUID = '0c81e8ae-2e1d-44c9-b907-d580da260ff6'
UniverseUUID = 'ec6b18f3-69b9-43cc-b986-5f60bb0970b5'
AUTHTOKEN = '998f5bca-a8fb-464e-bbe5-a72f9e9890b0'
args = None ### This will contain command line Arguments


class yba_api():
    def __init__(self):
        self.baseurl = YBA_HOST + '/api/v1/customers/' + CustomerUUID + '/universes'
        self.raw_response = None

    def Get(self, targeturl=""):
        url = self.baseurl + targeturl
        self.raw_response = requests.get(url, headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': AUTHTOKEN},
                                verify=False)
        self.raw_response.raise_for_status()
        # Convert response data to dict
        return json.loads(self.raw_response.text)
class Node():
    def __init__(self, node_json, universe_object):
        self.name = node_json['nodeName']
        self.UUID = node_json['nodeUuid']
        self.IP = node_json['cloudInfo']['private_ip']
        self.IDX = node_json['nodeIdx']
        self.azuuid = node_json['azUuid']
        self.ismaster= node_json['isMaster']
        self.istserver = node_json['isTserver']
        self.region = None
        self.az = None
        self.universe= universe_object

    def nodeidx(self):
        return self.IDX

    def Print(self, Indent=0):
        print(" "*Indent, "Node: " + str(self.IDX) + "["+ self.UUID + "] " + self.name + " has IP: " + self.IP + \
              " is in Zone " + self.az.name + \
              " [" + (" TServer " if self.istserver else "" ) + (" Master " if self.ismaster else "") + " ]")

class MasterLeader(Node): #Inherited from Node
    def __init__(self,node_object):
        self.__dict__ = node_object.__dict__.copy()


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
        print ("Load Balancer Enabled = " + result.group(1) + "  Load is Balanced = " + result.group(2))




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



    def Print(self):
        """
         This prints the Universe and all its sub-objects
        """
        print('===== Universe ===== ' + self.name + ' UUID = ' + self.UUID + " Provider is " + self.providertype \
            + (" Paused " if self.paused else "")) ### name and UUID
        if self.master_leader is not None:
            print("     Master Leader is " + self.master_leader.name)
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
    for u_raw in api.Get():
        u = Universe(u_raw, api) ### This creates a Universe Object named "u"
        if args.universe == None:
            pass
        elif args.universe == u.name:
            pass
        else:
            continue

        u.Print()
        if args.tablets:
            ml = u.get_master_leader()
            ml.get_overview()


        print()
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

    args = parser.parse_args()
    if args.debug:
        print("Debugging Enabled")
    return args


# Main Program starts here

args = parse_arguments()
discover_universes()



###
# Print Region for the Universe. (May need two Classes, AZ and Region)
# Print the List of AZ for the Universe
# Which AZ got how many nodes
###
# All tablet details - https://10.231.0.124/universes/ec6b18f3-69b9-43cc-b986-5f60bb0970b5/proxy/10.231.0.10:7000/dump-entities

###
