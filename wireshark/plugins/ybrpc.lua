do
    local protobuf_dissector = Dissector.get("protobuf")

    -- Get the value of a varint
    -- https://developers.google.com/protocol-buffers/docs/encoding#varints
    -- @param tvb - The packet buffer
    -- @param offset - The offset in the packet buffer
    local function get_varint(tvb, offset, maxlen)
        local value = 0
        local cur_byte

        local varint_maxlen = 10

        if maxlen < varint_maxlen then
            varint_maxlen = maxlen
        end

        local i
        for i = 0, varint_maxlen, 1
        do
            cur_byte = tvb(offset+i, 1):uint()
            value = bit.bor(value, bit.lshift(bit.band(cur_byte, 0x7f), (i * 7)))

            if cur_byte < 0x80 then
                i = i+1
                return i, value
            end
        end

        return 0, 0
    end

    -- Get the byte length of a varint32
    -- @param tvb - The packet buffer
    -- @param offset - The offset in the packet buffer
    local function varint32_size(tvb, offset)
        local length, value = get_varint(tvb, offset, 4)
        return length
    end

    -- Get the value of a varint32
    -- @param tvb - The packet buffer
    -- @param offset - The offset in the packet buffer
    local function varint32_value(tvb, offset)
        local length, value = get_varint(tvb, offset, 4)
        return value
    end

    -- Decode the ybrpc call_id from the header
    --   This is a hack, instead of actually parsing the protobuf format, just do a basic
    --   validation and get the varint. I needed to do it manually because the protobuf_field
    --   dissector doesn't get called on varint fields.
    -- @param tvb - The packet buffer
    -- @param offset - The offset in the packet buffer
    local function get_call_id(tvb, offset)
        -- validate that this is the first field of the message, and the type is varint32 (0x08)
        -- https://developers.google.com/protocol-buffers/docs/encoding#structure
        local field_value = tvb(offset, 1):uint()
        if field_value ~= 0x08 then
            error("unexpected value for protobuf message header")
        end
        return varint32_size(tvb, offset+1)+1, varint32_value(tvb, offset+1)
    end

    -- Create a dissector for the yugabyte rpc protocol
    --   Loosely based on the example in this wireshark wiki:
    --   https://gitlab.com/wireshark/wireshark/-/wikis/Protobuf#write-your-own-protobuf-udp-or-tcp-dissectors
    -- @param name - The dissector name
    -- @param desc - The dissector description
    local function create_yugabyte_dissector(name, desc)
        local proto = Proto(name, desc)
        local f_call_id = ProtoField.uint32(name .. ".call_id", "Call ID", base.DEC)
        local f_message_service = ProtoField.string(name .. ".message_service", "Message Service")
        local f_message_method = ProtoField.string(name .. ".message_method", "Message Method")
        local f_length = ProtoField.uint32(name .. ".length", "Message Length", base.DEC)
        local f_protobuf_header_length = ProtoField.uint32(name .. ".protobuf_header_length", "Protobuf Header Length", base.DEC)
        local f_protobuf_message_length = ProtoField.uint32(name .. ".protobuf_message_length", "Protobuf Body Length", base.DEC)
        proto.fields = { f_call_id, f_message_service, f_message_method, f_length, f_protobuf_header_length, f_protobuf_message_length }

        -- Get the value of the tcp.stream in order to record response method
        local f_tcp_stream = Field.new("tcp.stream")
        --local f_tcp_seq = Field.new("tcp.seq")
        --local f_tcp_srcport = Field.new("tcp.srcport")

        -- Track the method for the RPC response
        --   TODO: While this seems to kind-of work, because every packet is dissected when you click a new packet
        --         sometimes this data isn't available for the response dissection... There needs to be a better
        --         way to accomplish this. Maybe a post dissector?
        local function register_call_with_stream(call_id, pinfo)
            if not f_tcp_stream() then return end

            streamno = f_tcp_stream().value

            if tcp_streams[streamno] == nil then
                tcp_streams[streamno] = {}
            end

            if pinfo.private["yb_service_string"] ~= "" and pinfo.private["yb_method_string"] ~= "" then
                if tcp_streams[streamno][call_id] == nil then
                    tcp_streams[streamno][call_id] = {}
                end
               tcp_streams[streamno][call_id]["yb_service_string"] = pinfo.private["yb_service_string"]
               tcp_streams[streamno][call_id]["yb_method_string"] = pinfo.private["yb_method_string"]
            end
        end

        -- Get the method/service for the RPC request

        local function get_request_rpc_method(pinfo)
            return pinfo.private["yb_service_string"], pinfo.private["yb_method_string"]
        end

        -- Get the method for the RPC response
        local function get_response_rpc_method(call_id)
            if tcp_streams[streamno] ~= nil then
                if tcp_streams[streamno][call_id] ~= nil then
                    return tcp_streams[streamno][call_id]["yb_service_string"], tcp_streams[streamno][call_id]["yb_method_string"]
                end
            end
            -- We don't know the message type, so just leave it blank
            return "", ""
        end

        proto.init = function()
            tcp_streams = {}
        end

        proto.dissector = function(tvb, pinfo, tree)
            local subtree = tree:add(proto, tvb())
            local offset = 0
            local remaining_len = tvb:len()
            local reported_len = tvb:reported_len()

            -- can't dissect because we don't have all the data
            if remaining_len ~= reported_len then
                return 0
            end

            -- TODO: there has to be some better way of figuring this out
            local is_response = true

            if pinfo.dst_port == 7100 then
                is_response = false
            elseif pinfo.dst_port == 9100 then
                is_response = false
            end

            if remaining_len < 3 then -- head not enough
                pinfo.desegment_offset = offset
                pinfo.desegment_len = DESEGMENT_ONE_MORE_SEGMENT
                return -1
            end

            -- Check for packet hello TODO: This should print something to the packet dump
            local hello = tvb(offset, 3):string()
            if hello == "YB\1" then
                offset = offset + 3
                remaining_len = remaining_len - 3
            end

            -- OK! Start decoding RPC messages
            while remaining_len > 0 do
                ::continue::
                -- Reset these values, as they are going to be set by subdissectors called by the protobuf_field dissector (below)
                pinfo.private["yb_service_string"] = ""
                pinfo.private["yb_method_string"] = ""

                if remaining_len < 4 then -- head not enough
                    pinfo.desegment_offset = offset
                    pinfo.desegment_len = DESEGMENT_ONE_MORE_SEGMENT
                    return -1
                end

                -- message + header length
                local data_len = tvb(offset, 4):uint()

                if remaining_len - 4 < data_len then -- data not enough
                    pinfo.desegment_offset = offset
                    pinfo.desegment_len = data_len - (remaining_len - 4)
                    return -1
                end

                message_tree = subtree:add(tvb(offset, data_len + 4), "Message")

                -- YBRPC total message length
                message_tree:add(f_length, tvb(offset, 4))
                offset = offset + 4

                -- TODO: This is to skip zero length messages... Why do we have empty messages sometimes? Keepalives?
                if data_len == 0 then
                    remaining_len = remaining_len - 4
                    offset = reported_len - remaining_len
                    goto continue
                end

                -- PacketHeader Length
                local pheader_length = varint32_value(tvb, offset)

                message_header = message_tree:add(tvb(offset, pheader_length + varint32_size(tvb, offset)), "Header")
                message_header:add(f_protobuf_header_length, tvb(offset, varint32_size(tvb,offset)), varint32_value(tvb, offset))
                offset = offset + varint32_size(tvb,offset)

                -- Set message type for the protobuf dissector
                if is_response then
                    pinfo.private["pb_msg_type"] = "message," .. "yb.rpc.ResponseHeader"
                else
                    -- TODO: get the sidecar addresses for the response packets
                    pinfo.private["pb_msg_type"] = "message," .. "yb.rpc.RequestHeader"
                end

                -- Get the call_id
                local call_id_len, call_id = get_call_id(tvb, offset)
                message_header:add(f_call_id, tvb(offset, call_id_len), call_id)
                message_tree:append_text(" [ID=" .. call_id .. "]")

                -- Dissect the header
                -- TODO: why can't I just use a regular tvb here, why does it require an intermediate bytearray
                --       in order to avoid a C stack overflow?
                local protobuf_header_bytearray = tvb:bytes(offset, pheader_length)
                pcall(Dissector.call, protobuf_dissector,
                        protobuf_header_bytearray:tvb(), pinfo, message_header)

                -- Try to record the message type for the response packet. This needs to be done after dissecting
                -- the header, because it uses a `protobuf_field` subdissector to intercept the method values from 
                -- the protobuf dissector
                if is_response == false then
                    register_call_with_stream(call_id, pinfo)
                end

                offset = offset + pheader_length

                -- Now we have everything we need to determine the message type
                if is_response then
                    message_type = "response"
                    message_service, message_method = get_response_rpc_method(call_id)
                else
                    message_service, message_method = get_request_rpc_method(pinfo)
                    message_type = "request"
                end

                if message_service ~= "" and message_method ~= "" then
                    message_header:add(f_message_service, message_service):set_generated()
                    message_header:add(f_message_method, message_method):set_generated()

                    -- Append message type to top of message tree
                    message_tree:append_text(" [" ..message_service .. "/" .. message_method .. "] [" .. string.upper(message_type) .. "]")

                    -- Set message type for the protobuf dissector
                    pinfo.private["pb_msg_type"] = "application/ybrpc," .. message_service .. "/" .. message_method .. "," .. message_type
                else
                    pinfo.private["pb_msg_type"] = "message,"
                end

                local varint_length, pb_message_length = get_varint(tvb, offset, 4)

                local message_body = message_tree:add(tvb(offset, varint_length + pb_message_length), "Message Body")

                message_body:add(f_protobuf_message_length, tvb(offset, varint_length), pb_message_length)
                offset = offset + varint_length

                local protobuf_message_bytearray = tvb:bytes(offset, pb_message_length)

                local message_service, message_method, message_type


                pcall(Dissector.call, protobuf_dissector,
                        protobuf_message_bytearray:tvb(), pinfo, message_body)

                offset = offset + pb_message_length
                remaining_len = reported_len - offset

            end
            pinfo.columns.protocol:set(name)
        end

        DissectorTable.get("tcp.port"):add(0, proto)
        return proto
    end

    create_yugabyte_dissector("YBRPC", "Yugabyte RPC")

    -- Create a new subdissector - these will be called by the protobuf dissector, and will 'slurp' the field
    --   into a pinfo.private[] variable and made available to the dissector above.
    -- @param name - The dissector name
    -- @param desc - The dissector description
    local function create_protobuf_subdissector(name, desc, field_name, private_field, as_hex)
        local proto = Proto(name, desc)
        proto.dissector = function(tvb, pinfo, tree)
            -- TODO: To make this general purpose, select a function to execute via an argument to this function
            pinfo.private[private_field] = tvb:bytes(0, tvb:len()):raw()
        end
        local protobuf_field_table = DissectorTable.get("protobuf_field")
        protobuf_field_table:add(field_name, proto)
        return proto
    end

    create_protobuf_subdissector("ybservicesubdissector", "Yugabyte Service Subdissector", "yb.rpc.RemoteMethodPB.service_name", "yb_service_string")
    create_protobuf_subdissector("ybmethodsubdissector", "Yugabyte Method Subdissector", "yb.rpc.RemoteMethodPB.method_name", "yb_method_string")

end
