# Yugabyte wireshark plugins

## ybrpc.lua
`plugins/ybrpc.lua` is a wireshark dissector for Yugabyte RPC traffic

![wireshark_dissection](./images/ybrpc_dissection.png)

### Usage

Place the `ybrpc.lua` script in either your `Personal Lua Plugins`, or `Global Lua Plugins` Directory. These paths can be found in the ` Help -> About wireshark -> Folders` menu.

![plugins directory](./images/wireshark_about.png)

Enable the following settings in the `Edit -> Preferences -> Protocols -> ProtoBuf` dialog:

![protobuf_settings](./images/protobuf_settings.png)

In `Edit -> Preferences -> Protocols -> ProtoBuf -> Protobuf search paths` point your `Protobuf source directory` to the protobuf directory in this repository.

![protobuf_searchpath](./images/protobuf_searchpath.png)

Open a wireshark capture of the Yugabyte database to see the protocol dissection
