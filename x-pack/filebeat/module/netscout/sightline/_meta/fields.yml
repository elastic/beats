- name: network.interface.name
  type: keyword
  default_field: false
  description: >
    Name of the network interface where the traffic has been observed.
- name: rsa
  type: group
  default_field: false
  fields:
  - name: internal
    type: group
    fields:
    - name: msg
      type: keyword
      description: This key is used to capture the raw message that comes into the
        Log Decoder
    - name: messageid
      type: keyword
    - name: event_desc
      type: keyword
    - name: message
      type: keyword
      description: This key captures the contents of instant messages
    - name: time
      type: date
      description: This is the time at which a session hits a NetWitness Decoder.
        This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness.
    - name: level
      type: long
      description: Deprecated key defined only in table map.
    - name: msg_id
      type: keyword
      description: This is the Message ID1 value that identifies the exact log parser
        definition which parses a particular log session. This key should never be
        used to parse Meta data from a session (Logs/Packets) Directly, this is a
        Reserved key in NetWitness
    - name: msg_vid
      type: keyword
      description: This is the Message ID2 value that identifies the exact log parser
        definition which parses a particular log session. This key should never be
        used to parse Meta data from a session (Logs/Packets) Directly, this is a
        Reserved key in NetWitness
    - name: data
      type: keyword
      description: Deprecated key defined only in table map.
    - name: obj_server
      type: keyword
      description: Deprecated key defined only in table map.
    - name: obj_val
      type: keyword
      description: Deprecated key defined only in table map.
    - name: resource
      type: keyword
      description: Deprecated key defined only in table map.
    - name: obj_id
      type: keyword
      description: Deprecated key defined only in table map.
    - name: statement
      type: keyword
      description: Deprecated key defined only in table map.
    - name: audit_class
      type: keyword
      description: Deprecated key defined only in table map.
    - name: entry
      type: keyword
      description: Deprecated key defined only in table map.
    - name: hcode
      type: keyword
      description: Deprecated key defined only in table map.
    - name: inode
      type: long
      description: Deprecated key defined only in table map.
    - name: resource_class
      type: keyword
      description: Deprecated key defined only in table map.
    - name: dead
      type: long
      description: Deprecated key defined only in table map.
    - name: feed_desc
      type: keyword
      description: This is used to capture the description of the feed. This key should
        never be used to parse Meta data from a session (Logs/Packets) Directly, this
        is a Reserved key in NetWitness
    - name: feed_name
      type: keyword
      description: This is used to capture the name of the feed. This key should never
        be used to parse Meta data from a session (Logs/Packets) Directly, this is
        a Reserved key in NetWitness
    - name: cid
      type: keyword
      description: This is the unique identifier used to identify a NetWitness Concentrator.
        This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness
    - name: device_class
      type: keyword
      description: This is the Classification of the Log Event Source under a predefined
        fixed set of Event Source Classifications. This key should never be used to
        parse Meta data from a session (Logs/Packets) Directly, this is a Reserved
        key in NetWitness
    - name: device_group
      type: keyword
      description: This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: device_host
      type: keyword
      description: This is the Hostname of the log Event Source sending the logs to
        NetWitness. This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: device_ip
      type: ip
      description: This is the IPv4 address of the Log Event Source sending the logs
        to NetWitness. This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: device_ipv6
      type: ip
      description: This is the IPv6 address of the Log Event Source sending the logs
        to NetWitness. This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: device_type
      type: keyword
      description: This is the name of the log parser which parsed a given session.
        This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness
    - name: device_type_id
      type: long
      description: Deprecated key defined only in table map.
    - name: did
      type: keyword
      description: This is the unique identifier used to identify a NetWitness Decoder.
        This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness
    - name: entropy_req
      type: long
      description: This key is only used by the Entropy Parser, the Meta Type can
        be either UInt16 or Float32 based on the configuration
    - name: entropy_res
      type: long
      description: This key is only used by the Entropy Parser, the Meta Type can
        be either UInt16 or Float32 based on the configuration
    - name: event_name
      type: keyword
      description: Deprecated key defined only in table map.
    - name: feed_category
      type: keyword
      description: This is used to capture the category of the feed. This key should
        never be used to parse Meta data from a session (Logs/Packets) Directly, this
        is a Reserved key in NetWitness
    - name: forward_ip
      type: ip
      description: This key should be used to capture the IPV4 address of a relay
        system which forwarded the events from the original system to NetWitness.
    - name: forward_ipv6
      type: ip
      description: This key is used to capture the IPV6 address of a relay system
        which forwarded the events from the original system to NetWitness. This key
        should never be used to parse Meta data from a session (Logs/Packets) Directly,
        this is a Reserved key in NetWitness
    - name: header_id
      type: keyword
      description: This is the Header ID value that identifies the exact log parser
        header definition that parses a particular log session. This key should never
        be used to parse Meta data from a session (Logs/Packets) Directly, this is
        a Reserved key in NetWitness
    - name: lc_cid
      type: keyword
      description: This is a unique Identifier of a Log Collector. This key should
        never be used to parse Meta data from a session (Logs/Packets) Directly, this
        is a Reserved key in NetWitness
    - name: lc_ctime
      type: date
      description: This is the time at which a log is collected in a NetWitness Log
        Collector. This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: mcb_req
      type: long
      description: This key is only used by the Entropy Parser, the most common byte
        request is simply which byte for each side (0 thru 255) was seen the most
    - name: mcb_res
      type: long
      description: This key is only used by the Entropy Parser, the most common byte
        response is simply which byte for each side (0 thru 255) was seen the most
    - name: mcbc_req
      type: long
      description: This key is only used by the Entropy Parser, the most common byte
        count is the number of times the most common byte (above) was seen in the
        session streams
    - name: mcbc_res
      type: long
      description: This key is only used by the Entropy Parser, the most common byte
        count is the number of times the most common byte (above) was seen in the
        session streams
    - name: medium
      type: long
      description: "This key is used to identify if it\u2019s a log/packet session\
        \ or Layer 2 Encapsulation Type. This key should never be used to parse Meta\
        \ data from a session (Logs/Packets) Directly, this is a Reserved key in NetWitness.\
        \ 32 = log, 33 = correlation session, &lt; 32 is packet session"
    - name: node_name
      type: keyword
      description: Deprecated key defined only in table map.
    - name: nwe_callback_id
      type: keyword
      description: This key denotes that event is endpoint related
    - name: parse_error
      type: keyword
      description: This is a special key that stores any Meta key validation error
        found while parsing a log session. This key should never be used to parse
        Meta data from a session (Logs/Packets) Directly, this is a Reserved key in
        NetWitness
    - name: payload_req
      type: long
      description: This key is only used by the Entropy Parser, the payload size metrics
        are the payload sizes of each session side at the time of parsing. However,
        in order to keep
    - name: payload_res
      type: long
      description: This key is only used by the Entropy Parser, the payload size metrics
        are the payload sizes of each session side at the time of parsing. However,
        in order to keep
    - name: process_vid_dst
      type: keyword
      description: Endpoint generates and uses a unique virtual ID to identify any
        similar group of process. This ID represents the target process.
    - name: process_vid_src
      type: keyword
      description: Endpoint generates and uses a unique virtual ID to identify any
        similar group of process. This ID represents the source process.
    - name: rid
      type: long
      description: This is a special ID of the Remote Session created by NetWitness
        Decoder. This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness
    - name: session_split
      type: keyword
      description: This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: site
      type: keyword
      description: Deprecated key defined only in table map.
    - name: size
      type: long
      description: This is the size of the session as seen by the NetWitness Decoder.
        This key should never be used to parse Meta data from a session (Logs/Packets)
        Directly, this is a Reserved key in NetWitness
    - name: sourcefile
      type: keyword
      description: This is the name of the log file or PCAPs that can be imported
        into NetWitness. This key should never be used to parse Meta data from a session
        (Logs/Packets) Directly, this is a Reserved key in NetWitness
    - name: ubc_req
      type: long
      description: This key is only used by the Entropy Parser, Unique byte count
        is the number of unique bytes seen in each stream. 256 would mean all byte
        values of 0 thru 255 were seen at least once
    - name: ubc_res
      type: long
      description: This key is only used by the Entropy Parser, Unique byte count
        is the number of unique bytes seen in each stream. 256 would mean all byte
        values of 0 thru 255 were seen at least once
    - name: word
      type: keyword
      description: This is used by the Word Parsing technology to capture the first
        5 character of every word in an unparsed log
  - name: time
    type: group
    fields:
    - name: event_time
      type: date
      description: This key is used to capture the time mentioned in a raw session
        that represents the actual time an event occured in a standard normalized
        form
    - name: duration_time
      type: double
      description: This key is used to capture the normalized duration/lifetime in
        seconds.
    - name: event_time_str
      type: keyword
      description: This key is used to capture the incomplete time mentioned in a
        session as a string
    - name: starttime
      type: date
      description: This key is used to capture the Start time mentioned in a session
        in a standard form
    - name: month
      type: keyword
    - name: day
      type: keyword
    - name: endtime
      type: date
      description: This key is used to capture the End time mentioned in a session
        in a standard form
    - name: timezone
      type: keyword
      description: This key is used to capture the timezone of the Event Time
    - name: duration_str
      type: keyword
      description: A text string version of the duration
    - name: date
      type: keyword
    - name: year
      type: keyword
    - name: recorded_time
      type: date
      description: The event time as recorded by the system the event is collected
        from. The usage scenario is a multi-tier application where the management
        layer of the system records it's own timestamp at the time of collection from
        its child nodes. Must be in timestamp format.
    - name: datetime
      type: keyword
    - name: effective_time
      type: date
      description: This key is the effective time referenced by an individual event
        in a Standard Timestamp format
    - name: expire_time
      type: date
      description: This key is the timestamp that explicitly refers to an expiration.
    - name: process_time
      type: keyword
      description: Deprecated, use duration.time
    - name: hour
      type: keyword
    - name: min
      type: keyword
    - name: timestamp
      type: keyword
    - name: event_queue_time
      type: date
      description: This key is the Time that the event was queued.
    - name: p_time1
      type: keyword
    - name: tzone
      type: keyword
    - name: eventtime
      type: keyword
    - name: gmtdate
      type: keyword
    - name: gmttime
      type: keyword
    - name: p_date
      type: keyword
    - name: p_month
      type: keyword
    - name: p_time
      type: keyword
    - name: p_time2
      type: keyword
    - name: p_year
      type: keyword
    - name: expire_time_str
      type: keyword
      description: This key is used to capture incomplete timestamp that explicitly
        refers to an expiration.
    - name: stamp
      type: date
      description: Deprecated key defined only in table map.
  - name: misc
    type: group
    fields:
    - name: action
      type: keyword
    - name: result
      type: keyword
      description: This key is used to capture the outcome/result string value of
        an action in a session.
    - name: severity
      type: keyword
      description: This key is used to capture the severity given the session
    - name: event_type
      type: keyword
      description: This key captures the event category type as specified by the event
        source.
    - name: reference_id
      type: keyword
      description: This key is used to capture an event id from the session directly
    - name: version
      type: keyword
      description: This key captures Version of the application or OS which is generating
        the event.
    - name: disposition
      type: keyword
      description: This key captures the The end state of an action.
    - name: result_code
      type: keyword
      description: This key is used to capture the outcome/result numeric value of
        an action in a session
    - name: category
      type: keyword
      description: This key is used to capture the category of an event given by the
        vendor in the session
    - name: obj_name
      type: keyword
      description: This is used to capture name of object
    - name: obj_type
      type: keyword
      description: This is used to capture type of object
    - name: event_source
      type: keyword
      description: "This key captures Source of the event that\u2019s not a hostname"
    - name: log_session_id
      type: keyword
      description: This key is used to capture a sessionid from the session directly
    - name: group
      type: keyword
      description: This key captures the Group Name value
    - name: policy_name
      type: keyword
      description: This key is used to capture the Policy Name only.
    - name: rule_name
      type: keyword
      description: This key captures the Rule Name
    - name: context
      type: keyword
      description: This key captures Information which adds additional context to
        the event.
    - name: change_new
      type: keyword
      description: "This key is used to capture the new values of the attribute that\u2019\
        s changing in a session"
    - name: space
      type: keyword
    - name: client
      type: keyword
      description: This key is used to capture only the name of the client application
        requesting resources of the server. See the user.agent meta key for capture
        of the specific user agent identifier or browser identification string.
    - name: msgIdPart1
      type: keyword
    - name: msgIdPart2
      type: keyword
    - name: change_old
      type: keyword
      description: "This key is used to capture the old value of the attribute that\u2019\
        s changing in a session"
    - name: operation_id
      type: keyword
      description: An alert number or operation number. The values should be unique
        and non-repeating.
    - name: event_state
      type: keyword
      description: This key captures the current state of the object/item referenced
        within the event. Describing an on-going event.
    - name: group_object
      type: keyword
      description: This key captures a collection/grouping of entities. Specific usage
    - name: node
      type: keyword
      description: Common use case is the node name within a cluster. The cluster
        name is reflected by the host name.
    - name: rule
      type: keyword
      description: This key captures the Rule number
    - name: device_name
      type: keyword
      description: 'This is used to capture name of the Device associated with the
        node Like: a physical disk, printer, etc'
    - name: param
      type: keyword
      description: This key is the parameters passed as part of a command or application,
        etc.
    - name: change_attrib
      type: keyword
      description: "This key is used to capture the name of the attribute that\u2019\
        s changing in a session"
    - name: event_computer
      type: keyword
      description: This key is a windows only concept, where this key is used to capture
        fully qualified domain name in a windows log.
    - name: reference_id1
      type: keyword
      description: This key is for Linked ID to be used as an addition to "reference.id"
    - name: event_log
      type: keyword
      description: This key captures the Name of the event log
    - name: OS
      type: keyword
      description: This key captures the Name of the Operating System
    - name: terminal
      type: keyword
      description: This key captures the Terminal Names only
    - name: msgIdPart3
      type: keyword
    - name: filter
      type: keyword
      description: This key captures Filter used to reduce result set
    - name: serial_number
      type: keyword
      description: This key is the Serial number associated with a physical asset.
    - name: checksum
      type: keyword
      description: This key is used to capture the checksum or hash of the entity
        such as a file or process. Checksum should be used over checksum.src or checksum.dst
        when it is unclear whether the entity is a source or target of an action.
    - name: event_user
      type: keyword
      description: This key is a windows only concept, where this key is used to capture
        combination of domain name and username in a windows log.
    - name: virusname
      type: keyword
      description: This key captures the name of the virus
    - name: content_type
      type: keyword
      description: This key is used to capture Content Type only.
    - name: group_id
      type: keyword
      description: This key captures Group ID Number (related to the group name)
    - name: policy_id
      type: keyword
      description: This key is used to capture the Policy ID only, this should be
        a numeric value, use policy.name otherwise
    - name: vsys
      type: keyword
      description: This key captures Virtual System Name
    - name: connection_id
      type: keyword
      description: This key captures the Connection ID
    - name: reference_id2
      type: keyword
      description: This key is for the 2nd Linked ID. Can be either linked to "reference.id"
        or "reference.id1" value but should not be used unless the other two variables
        are in play.
    - name: sensor
      type: keyword
      description: This key captures Name of the sensor. Typically used in IDS/IPS
        based devices
    - name: sig_id
      type: long
      description: This key captures IDS/IPS Int Signature ID
    - name: port_name
      type: keyword
      description: 'This key is used for Physical or logical port connection but does
        NOT include a network port. (Example: Printer port name).'
    - name: rule_group
      type: keyword
      description: This key captures the Rule group name
    - name: risk_num
      type: double
      description: This key captures a Numeric Risk value
    - name: trigger_val
      type: keyword
      description: This key captures the Value of the trigger or threshold condition.
    - name: log_session_id1
      type: keyword
      description: This key is used to capture a Linked (Related) Session ID from
        the session directly
    - name: comp_version
      type: keyword
      description: This key captures the Version level of a sub-component of a product.
    - name: content_version
      type: keyword
      description: This key captures Version level of a signature or database content.
    - name: hardware_id
      type: keyword
      description: This key is used to capture unique identifier for a device or system
        (NOT a Mac address)
    - name: risk
      type: keyword
      description: This key captures the non-numeric risk value
    - name: event_id
      type: keyword
    - name: reason
      type: keyword
    - name: status
      type: keyword
    - name: mail_id
      type: keyword
      description: This key is used to capture the mailbox id/name
    - name: rule_uid
      type: keyword
      description: This key is the Unique Identifier for a rule.
    - name: trigger_desc
      type: keyword
      description: This key captures the Description of the trigger or threshold condition.
    - name: inout
      type: keyword
    - name: p_msgid
      type: keyword
    - name: data_type
      type: keyword
    - name: msgIdPart4
      type: keyword
    - name: error
      type: keyword
      description: This key captures All non successful Error codes or responses
    - name: index
      type: keyword
    - name: listnum
      type: keyword
      description: This key is used to capture listname or listnumber, primarily for
        collecting access-list
    - name: ntype
      type: keyword
    - name: observed_val
      type: keyword
      description: This key captures the Value observed (from the perspective of the
        device generating the log).
    - name: policy_value
      type: keyword
      description: This key captures the contents of the policy. This contains details
        about the policy
    - name: pool_name
      type: keyword
      description: This key captures the name of a resource pool
    - name: rule_template
      type: keyword
      description: A default set of parameters which are overlayed onto a rule (or
        rulename) which efffectively constitutes a template
    - name: count
      type: keyword
    - name: number
      type: keyword
    - name: sigcat
      type: keyword
    - name: type
      type: keyword
    - name: comments
      type: keyword
      description: Comment information provided in the log message
    - name: doc_number
      type: long
      description: This key captures File Identification number
    - name: expected_val
      type: keyword
      description: This key captures the Value expected (from the perspective of the
        device generating the log).
    - name: job_num
      type: keyword
      description: This key captures the Job Number
    - name: spi_dst
      type: keyword
      description: Destination SPI Index
    - name: spi_src
      type: keyword
      description: Source SPI Index
    - name: code
      type: keyword
    - name: agent_id
      type: keyword
      description: This key is used to capture agent id
    - name: message_body
      type: keyword
      description: This key captures the The contents of the message body.
    - name: phone
      type: keyword
    - name: sig_id_str
      type: keyword
      description: This key captures a string object of the sigid variable.
    - name: cmd
      type: keyword
    - name: misc
      type: keyword
    - name: name
      type: keyword
    - name: cpu
      type: long
      description: This key is the CPU time used in the execution of the event being
        recorded.
    - name: event_desc
      type: keyword
      description: This key is used to capture a description of an event available
        directly or inferred
    - name: sig_id1
      type: long
      description: This key captures IDS/IPS Int Signature ID. This must be linked
        to the sig.id
    - name: im_buddyid
      type: keyword
    - name: im_client
      type: keyword
    - name: im_userid
      type: keyword
    - name: pid
      type: keyword
    - name: priority
      type: keyword
    - name: context_subject
      type: keyword
      description: This key is to be used in an audit context where the subject is
        the object being identified
    - name: context_target
      type: keyword
    - name: cve
      type: keyword
      description: This key captures CVE (Common Vulnerabilities and Exposures) -
        an identifier for known information security vulnerabilities.
    - name: fcatnum
      type: keyword
      description: This key captures Filter Category Number. Legacy Usage
    - name: library
      type: keyword
      description: This key is used to capture library information in mainframe devices
    - name: parent_node
      type: keyword
      description: This key captures the Parent Node Name. Must be related to node
        variable.
    - name: risk_info
      type: keyword
      description: Deprecated, use New Hunting Model (inv.*, ioc, boc, eoc, analysis.*)
    - name: tcp_flags
      type: long
      description: This key is captures the TCP flags set in any packet of session
    - name: tos
      type: long
      description: This key describes the type of service
    - name: vm_target
      type: keyword
      description: VMWare Target **VMWARE** only varaible.
    - name: workspace
      type: keyword
      description: This key captures Workspace Description
    - name: command
      type: keyword
    - name: event_category
      type: keyword
    - name: facilityname
      type: keyword
    - name: forensic_info
      type: keyword
    - name: jobname
      type: keyword
    - name: mode
      type: keyword
    - name: policy
      type: keyword
    - name: policy_waiver
      type: keyword
    - name: second
      type: keyword
    - name: space1
      type: keyword
    - name: subcategory
      type: keyword
    - name: tbdstr2
      type: keyword
    - name: alert_id
      type: keyword
      description: Deprecated, New Hunting Model (inv.*, ioc, boc, eoc, analysis.*)
    - name: checksum_dst
      type: keyword
      description: This key is used to capture the checksum or hash of the the target
        entity such as a process or file.
    - name: checksum_src
      type: keyword
      description: This key is used to capture the checksum or hash of the source
        entity such as a file or process.
    - name: fresult
      type: long
      description: This key captures the Filter Result
    - name: payload_dst
      type: keyword
      description: This key is used to capture destination payload
    - name: payload_src
      type: keyword
      description: This key is used to capture source payload
    - name: pool_id
      type: keyword
      description: This key captures the identifier (typically numeric field) of a
        resource pool
    - name: process_id_val
      type: keyword
      description: This key is a failure key for Process ID when it is not an integer
        value
    - name: risk_num_comm
      type: double
      description: This key captures Risk Number Community
    - name: risk_num_next
      type: double
      description: This key captures Risk Number NextGen
    - name: risk_num_sand
      type: double
      description: This key captures Risk Number SandBox
    - name: risk_num_static
      type: double
      description: This key captures Risk Number Static
    - name: risk_suspicious
      type: keyword
      description: Deprecated, use New Hunting Model (inv.*, ioc, boc, eoc, analysis.*)
    - name: risk_warning
      type: keyword
      description: Deprecated, use New Hunting Model (inv.*, ioc, boc, eoc, analysis.*)
    - name: snmp_oid
      type: keyword
      description: SNMP Object Identifier
    - name: sql
      type: keyword
      description: This key captures the SQL query
    - name: vuln_ref
      type: keyword
      description: This key captures the Vulnerability Reference details
    - name: acl_id
      type: keyword
    - name: acl_op
      type: keyword
    - name: acl_pos
      type: keyword
    - name: acl_table
      type: keyword
    - name: admin
      type: keyword
    - name: alarm_id
      type: keyword
    - name: alarmname
      type: keyword
    - name: app_id
      type: keyword
    - name: audit
      type: keyword
    - name: audit_object
      type: keyword
    - name: auditdata
      type: keyword
    - name: benchmark
      type: keyword
    - name: bypass
      type: keyword
    - name: cache
      type: keyword
    - name: cache_hit
      type: keyword
    - name: cefversion
      type: keyword
    - name: cfg_attr
      type: keyword
    - name: cfg_obj
      type: keyword
    - name: cfg_path
      type: keyword
    - name: changes
      type: keyword
    - name: client_ip
      type: keyword
    - name: clustermembers
      type: keyword
    - name: cn_acttimeout
      type: keyword
    - name: cn_asn_src
      type: keyword
    - name: cn_bgpv4nxthop
      type: keyword
    - name: cn_ctr_dst_code
      type: keyword
    - name: cn_dst_tos
      type: keyword
    - name: cn_dst_vlan
      type: keyword
    - name: cn_engine_id
      type: keyword
    - name: cn_engine_type
      type: keyword
    - name: cn_f_switch
      type: keyword
    - name: cn_flowsampid
      type: keyword
    - name: cn_flowsampintv
      type: keyword
    - name: cn_flowsampmode
      type: keyword
    - name: cn_inacttimeout
      type: keyword
    - name: cn_inpermbyts
      type: keyword
    - name: cn_inpermpckts
      type: keyword
    - name: cn_invalid
      type: keyword
    - name: cn_ip_proto_ver
      type: keyword
    - name: cn_ipv4_ident
      type: keyword
    - name: cn_l_switch
      type: keyword
    - name: cn_log_did
      type: keyword
    - name: cn_log_rid
      type: keyword
    - name: cn_max_ttl
      type: keyword
    - name: cn_maxpcktlen
      type: keyword
    - name: cn_min_ttl
      type: keyword
    - name: cn_minpcktlen
      type: keyword
    - name: cn_mpls_lbl_1
      type: keyword
    - name: cn_mpls_lbl_10
      type: keyword
    - name: cn_mpls_lbl_2
      type: keyword
    - name: cn_mpls_lbl_3
      type: keyword
    - name: cn_mpls_lbl_4
      type: keyword
    - name: cn_mpls_lbl_5
      type: keyword
    - name: cn_mpls_lbl_6
      type: keyword
    - name: cn_mpls_lbl_7
      type: keyword
    - name: cn_mpls_lbl_8
      type: keyword
    - name: cn_mpls_lbl_9
      type: keyword
    - name: cn_mplstoplabel
      type: keyword
    - name: cn_mplstoplabip
      type: keyword
    - name: cn_mul_dst_byt
      type: keyword
    - name: cn_mul_dst_pks
      type: keyword
    - name: cn_muligmptype
      type: keyword
    - name: cn_sampalgo
      type: keyword
    - name: cn_sampint
      type: keyword
    - name: cn_seqctr
      type: keyword
    - name: cn_spackets
      type: keyword
    - name: cn_src_tos
      type: keyword
    - name: cn_src_vlan
      type: keyword
    - name: cn_sysuptime
      type: keyword
    - name: cn_template_id
      type: keyword
    - name: cn_totbytsexp
      type: keyword
    - name: cn_totflowexp
      type: keyword
    - name: cn_totpcktsexp
      type: keyword
    - name: cn_unixnanosecs
      type: keyword
    - name: cn_v6flowlabel
      type: keyword
    - name: cn_v6optheaders
      type: keyword
    - name: comp_class
      type: keyword
    - name: comp_name
      type: keyword
    - name: comp_rbytes
      type: keyword
    - name: comp_sbytes
      type: keyword
    - name: cpu_data
      type: keyword
    - name: criticality
      type: keyword
    - name: cs_agency_dst
      type: keyword
    - name: cs_analyzedby
      type: keyword
    - name: cs_av_other
      type: keyword
    - name: cs_av_primary
      type: keyword
    - name: cs_av_secondary
      type: keyword
    - name: cs_bgpv6nxthop
      type: keyword
    - name: cs_bit9status
      type: keyword
    - name: cs_context
      type: keyword
    - name: cs_control
      type: keyword
    - name: cs_data
      type: keyword
    - name: cs_datecret
      type: keyword
    - name: cs_dst_tld
      type: keyword
    - name: cs_eth_dst_ven
      type: keyword
    - name: cs_eth_src_ven
      type: keyword
    - name: cs_event_uuid
      type: keyword
    - name: cs_filetype
      type: keyword
    - name: cs_fld
      type: keyword
    - name: cs_if_desc
      type: keyword
    - name: cs_if_name
      type: keyword
    - name: cs_ip_next_hop
      type: keyword
    - name: cs_ipv4dstpre
      type: keyword
    - name: cs_ipv4srcpre
      type: keyword
    - name: cs_lifetime
      type: keyword
    - name: cs_log_medium
      type: keyword
    - name: cs_loginname
      type: keyword
    - name: cs_modulescore
      type: keyword
    - name: cs_modulesign
      type: keyword
    - name: cs_opswatresult
      type: keyword
    - name: cs_payload
      type: keyword
    - name: cs_registrant
      type: keyword
    - name: cs_registrar
      type: keyword
    - name: cs_represult
      type: keyword
    - name: cs_rpayload
      type: keyword
    - name: cs_sampler_name
      type: keyword
    - name: cs_sourcemodule
      type: keyword
    - name: cs_streams
      type: keyword
    - name: cs_targetmodule
      type: keyword
    - name: cs_v6nxthop
      type: keyword
    - name: cs_whois_server
      type: keyword
    - name: cs_yararesult
      type: keyword
    - name: description
      type: keyword
    - name: devvendor
      type: keyword
    - name: distance
      type: keyword
    - name: dstburb
      type: keyword
    - name: edomain
      type: keyword
    - name: edomaub
      type: keyword
    - name: euid
      type: keyword
    - name: facility
      type: keyword
    - name: finterface
      type: keyword
    - name: flags
      type: keyword
    - name: gaddr
      type: keyword
    - name: id3
      type: keyword
    - name: im_buddyname
      type: keyword
    - name: im_croomid
      type: keyword
    - name: im_croomtype
      type: keyword
    - name: im_members
      type: keyword
    - name: im_username
      type: keyword
    - name: ipkt
      type: keyword
    - name: ipscat
      type: keyword
    - name: ipspri
      type: keyword
    - name: latitude
      type: keyword
    - name: linenum
      type: keyword
    - name: list_name
      type: keyword
    - name: load_data
      type: keyword
    - name: location_floor
      type: keyword
    - name: location_mark
      type: keyword
    - name: log_id
      type: keyword
    - name: log_type
      type: keyword
    - name: logid
      type: keyword
    - name: logip
      type: keyword
    - name: logname
      type: keyword
    - name: longitude
      type: keyword
    - name: lport
      type: keyword
    - name: mbug_data
      type: keyword
    - name: misc_name
      type: keyword
    - name: msg_type
      type: keyword
    - name: msgid
      type: keyword
    - name: netsessid
      type: keyword
    - name: num
      type: keyword
    - name: number1
      type: keyword
    - name: number2
      type: keyword
    - name: nwwn
      type: keyword
    - name: object
      type: keyword
    - name: operation
      type: keyword
    - name: opkt
      type: keyword
    - name: orig_from
      type: keyword
    - name: owner_id
      type: keyword
    - name: p_action
      type: keyword
    - name: p_filter
      type: keyword
    - name: p_group_object
      type: keyword
    - name: p_id
      type: keyword
    - name: p_msgid1
      type: keyword
    - name: p_msgid2
      type: keyword
    - name: p_result1
      type: keyword
    - name: password_chg
      type: keyword
    - name: password_expire
      type: keyword
    - name: permgranted
      type: keyword
    - name: permwanted
      type: keyword
    - name: pgid
      type: keyword
    - name: policyUUID
      type: keyword
    - name: prog_asp_num
      type: keyword
    - name: program
      type: keyword
    - name: real_data
      type: keyword
    - name: rec_asp_device
      type: keyword
    - name: rec_asp_num
      type: keyword
    - name: rec_library
      type: keyword
    - name: recordnum
      type: keyword
    - name: ruid
      type: keyword
    - name: sburb
      type: keyword
    - name: sdomain_fld
      type: keyword
    - name: sec
      type: keyword
    - name: sensorname
      type: keyword
    - name: seqnum
      type: keyword
    - name: session
      type: keyword
    - name: sessiontype
      type: keyword
    - name: sigUUID
      type: keyword
    - name: spi
      type: keyword
    - name: srcburb
      type: keyword
    - name: srcdom
      type: keyword
    - name: srcservice
      type: keyword
    - name: state
      type: keyword
    - name: status1
      type: keyword
    - name: svcno
      type: keyword
    - name: system
      type: keyword
    - name: tbdstr1
      type: keyword
    - name: tgtdom
      type: keyword
    - name: tgtdomain
      type: keyword
    - name: threshold
      type: keyword
    - name: type1
      type: keyword
    - name: udb_class
      type: keyword
    - name: url_fld
      type: keyword
    - name: user_div
      type: keyword
    - name: userid
      type: keyword
    - name: username_fld
      type: keyword
    - name: utcstamp
      type: keyword
    - name: v_instafname
      type: keyword
    - name: virt_data
      type: keyword
    - name: vpnid
      type: keyword
    - name: autorun_type
      type: keyword
      description: This is used to capture Auto Run type
    - name: cc_number
      type: long
      description: Valid Credit Card Numbers only
    - name: content
      type: keyword
      description: This key captures the content type from protocol headers
    - name: ein_number
      type: long
      description: Employee Identification Numbers only
    - name: found
      type: keyword
      description: This is used to capture the results of regex match
    - name: language
      type: keyword
      description: This is used to capture list of languages the client support and
        what it prefers
    - name: lifetime
      type: long
      description: This key is used to capture the session lifetime in seconds.
    - name: link
      type: keyword
      description: This key is used to link the sessions together. This key should
        never be used to parse Meta data from a session (Logs/Packets) Directly, this
        is a Reserved key in NetWitness
    - name: match
      type: keyword
      description: This key is for regex match name from search.ini
    - name: param_dst
      type: keyword
      description: This key captures the command line/launch argument of the target
        process or file
    - name: param_src
      type: keyword
      description: This key captures source parameter
    - name: search_text
      type: keyword
      description: This key captures the Search Text used
    - name: sig_name
      type: keyword
      description: This key is used to capture the Signature Name only.
    - name: snmp_value
      type: keyword
      description: SNMP set request value
    - name: streams
      type: long
      description: This key captures number of streams in session
  - name: db
    type: group
    fields:
    - name: index
      type: keyword
      description: This key captures IndexID of the index.
    - name: instance
      type: keyword
      description: This key is used to capture the database server instance name
    - name: database
      type: keyword
      description: This key is used to capture the name of a database or an instance
        as seen in a session
    - name: transact_id
      type: keyword
      description: This key captures the SQL transantion ID of the current session
    - name: permissions
      type: keyword
      description: This key captures permission or privilege level assigned to a resource.
    - name: table_name
      type: keyword
      description: This key is used to capture the table name
    - name: db_id
      type: keyword
      description: This key is used to capture the unique identifier for a database
    - name: db_pid
      type: long
      description: This key captures the process id of a connection with database
        server
    - name: lread
      type: long
      description: This key is used for the number of logical reads
    - name: lwrite
      type: long
      description: This key is used for the number of logical writes
    - name: pread
      type: long
      description: This key is used for the number of physical writes
  - name: network
    type: group
    fields:
    - name: alias_host
      type: keyword
      description: This key should be used when the source or destination context
        of a hostname is not clear.Also it captures the Device Hostname. Any Hostname
        that isnt ad.computer.
    - name: domain
      type: keyword
    - name: host_dst
      type: keyword
      description: "This key should only be used when it\u2019s a Destination Hostname"
    - name: network_service
      type: keyword
      description: This is used to capture layer 7 protocols/service names
    - name: interface
      type: keyword
      description: This key should be used when the source or destination context
        of an interface is not clear
    - name: network_port
      type: long
      description: 'Deprecated, use port. NOTE: There is a type discrepancy as currently
        used, TM: Int32, INDEX: UInt64 (why neither chose the correct UInt16?!)'
    - name: eth_host
      type: keyword
      description: Deprecated, use alias.mac
    - name: sinterface
      type: keyword
      description: "This key should only be used when it\u2019s a Source Interface"
    - name: dinterface
      type: keyword
      description: "This key should only be used when it\u2019s a Destination Interface"
    - name: vlan
      type: long
      description: This key should only be used to capture the ID of the Virtual LAN
    - name: zone_src
      type: keyword
      description: "This key should only be used when it\u2019s a Source Zone."
    - name: zone
      type: keyword
      description: This key should be used when the source or destination context
        of a Zone is not clear
    - name: zone_dst
      type: keyword
      description: "This key should only be used when it\u2019s a Destination Zone."
    - name: gateway
      type: keyword
      description: This key is used to capture the IP Address of the gateway
    - name: icmp_type
      type: long
      description: This key is used to capture the ICMP type only
    - name: mask
      type: keyword
      description: This key is used to capture the device network IPmask.
    - name: icmp_code
      type: long
      description: This key is used to capture the ICMP code only
    - name: protocol_detail
      type: keyword
      description: This key should be used to capture additional protocol information
    - name: dmask
      type: keyword
      description: This key is used for Destionation Device network mask
    - name: port
      type: long
      description: This key should only be used to capture a Network Port when the
        directionality is not clear
    - name: smask
      type: keyword
      description: This key is used for capturing source Network Mask
    - name: netname
      type: keyword
      description: This key is used to capture the network name associated with an
        IP range. This is configured by the end user.
    - name: paddr
      type: ip
      description: Deprecated
    - name: faddr
      type: keyword
    - name: lhost
      type: keyword
    - name: origin
      type: keyword
    - name: remote_domain_id
      type: keyword
    - name: addr
      type: keyword
    - name: dns_a_record
      type: keyword
    - name: dns_ptr_record
      type: keyword
    - name: fhost
      type: keyword
    - name: fport
      type: keyword
    - name: laddr
      type: keyword
    - name: linterface
      type: keyword
    - name: phost
      type: keyword
    - name: ad_computer_dst
      type: keyword
      description: Deprecated, use host.dst
    - name: eth_type
      type: long
      description: This key is used to capture Ethernet Type, Used for Layer 3 Protocols
        Only
    - name: ip_proto
      type: long
      description: This key should be used to capture the Protocol number, all the
        protocol nubers are converted into string in UI
    - name: dns_cname_record
      type: keyword
    - name: dns_id
      type: keyword
    - name: dns_opcode
      type: keyword
    - name: dns_resp
      type: keyword
    - name: dns_type
      type: keyword
    - name: domain1
      type: keyword
    - name: host_type
      type: keyword
    - name: packet_length
      type: keyword
    - name: host_orig
      type: keyword
      description: This is used to capture the original hostname in case of a Forwarding
        Agent or a Proxy in between.
    - name: rpayload
      type: keyword
      description: This key is used to capture the total number of payload bytes seen
        in the retransmitted packets.
    - name: vlan_name
      type: keyword
      description: This key should only be used to capture the name of the Virtual
        LAN
  - name: investigations
    type: group
    fields:
    - name: ec_activity
      type: keyword
      description: This key captures the particular event activity(Ex:Logoff)
    - name: ec_theme
      type: keyword
      description: This key captures the Theme of a particular Event(Ex:Authentication)
    - name: ec_subject
      type: keyword
      description: This key captures the Subject of a particular Event(Ex:User)
    - name: ec_outcome
      type: keyword
      description: This key captures the outcome of a particular Event(Ex:Success)
    - name: event_cat
      type: long
      description: This key captures the Event category number
    - name: event_cat_name
      type: keyword
      description: This key captures the event category name corresponding to the
        event cat code
    - name: event_vcat
      type: keyword
      description: This is a vendor supplied category. This should be used in situations
        where the vendor has adopted their own event_category taxonomy.
    - name: analysis_file
      type: keyword
      description: This is used to capture all indicators used in a File Analysis.
        This key should be used to capture an analysis of a file
    - name: analysis_service
      type: keyword
      description: This is used to capture all indicators used in a Service Analysis.
        This key should be used to capture an analysis of a service
    - name: analysis_session
      type: keyword
      description: This is used to capture all indicators used for a Session Analysis.
        This key should be used to capture an analysis of a session
    - name: boc
      type: keyword
      description: This is used to capture behaviour of compromise
    - name: eoc
      type: keyword
      description: This is used to capture Enablers of Compromise
    - name: inv_category
      type: keyword
      description: This used to capture investigation category
    - name: inv_context
      type: keyword
      description: This used to capture investigation context
    - name: ioc
      type: keyword
      description: This is key capture indicator of compromise
  - name: counters
    type: group
    fields:
    - name: dclass_c1
      type: long
      description: This is a generic counter key that should be used with the label
        dclass.c1.str only
    - name: dclass_c2
      type: long
      description: This is a generic counter key that should be used with the label
        dclass.c2.str only
    - name: event_counter
      type: long
      description: This is used to capture the number of times an event repeated
    - name: dclass_r1
      type: keyword
      description: This is a generic ratio key that should be used with the label
        dclass.r1.str only
    - name: dclass_c3
      type: long
      description: This is a generic counter key that should be used with the label
        dclass.c3.str only
    - name: dclass_c1_str
      type: keyword
      description: This is a generic counter string key that should be used with the
        label dclass.c1 only
    - name: dclass_c2_str
      type: keyword
      description: This is a generic counter string key that should be used with the
        label dclass.c2 only
    - name: dclass_r1_str
      type: keyword
      description: This is a generic ratio string key that should be used with the
        label dclass.r1 only
    - name: dclass_r2
      type: keyword
      description: This is a generic ratio key that should be used with the label
        dclass.r2.str only
    - name: dclass_c3_str
      type: keyword
      description: This is a generic counter string key that should be used with the
        label dclass.c3 only
    - name: dclass_r3
      type: keyword
      description: This is a generic ratio key that should be used with the label
        dclass.r3.str only
    - name: dclass_r2_str
      type: keyword
      description: This is a generic ratio string key that should be used with the
        label dclass.r2 only
    - name: dclass_r3_str
      type: keyword
      description: This is a generic ratio string key that should be used with the
        label dclass.r3 only
  - name: identity
    type: group
    fields:
    - name: auth_method
      type: keyword
      description: This key is used to capture authentication methods used only
    - name: user_role
      type: keyword
      description: This key is used to capture the Role of a user only
    - name: dn
      type: keyword
      description: X.500 (LDAP) Distinguished Name
    - name: logon_type
      type: keyword
      description: This key is used to capture the type of logon method used.
    - name: profile
      type: keyword
      description: This key is used to capture the user profile
    - name: accesses
      type: keyword
      description: This key is used to capture actual privileges used in accessing
        an object
    - name: realm
      type: keyword
      description: Radius realm or similar grouping of accounts
    - name: user_sid_dst
      type: keyword
      description: This key captures Destination User Session ID
    - name: dn_src
      type: keyword
      description: An X.500 (LDAP) Distinguished name that is used in a context that
        indicates a Source dn
    - name: org
      type: keyword
      description: This key captures the User organization
    - name: dn_dst
      type: keyword
      description: An X.500 (LDAP) Distinguished name that used in a context that
        indicates a Destination dn
    - name: firstname
      type: keyword
      description: This key is for First Names only, this is used for Healthcare predominantly
        to capture Patients information
    - name: lastname
      type: keyword
      description: This key is for Last Names only, this is used for Healthcare predominantly
        to capture Patients information
    - name: user_dept
      type: keyword
      description: User's Department Names only
    - name: user_sid_src
      type: keyword
      description: This key captures Source User Session ID
    - name: federated_sp
      type: keyword
      description: This key is the Federated Service Provider. This is the application
        requesting authentication.
    - name: federated_idp
      type: keyword
      description: This key is the federated Identity Provider. This is the server
        providing the authentication.
    - name: logon_type_desc
      type: keyword
      description: This key is used to capture the textual description of an integer
        logon type as stored in the meta key 'logon.type'.
    - name: middlename
      type: keyword
      description: This key is for Middle Names only, this is used for Healthcare
        predominantly to capture Patients information
    - name: password
      type: keyword
      description: This key is for Passwords seen in any session, plain text or encrypted
    - name: host_role
      type: keyword
      description: This key should only be used to capture the role of a Host Machine
    - name: ldap
      type: keyword
      description: "This key is for Uninterpreted LDAP values. Ldap Values that don\u2019\
        t have a clear query or response context"
    - name: ldap_query
      type: keyword
      description: This key is the Search criteria from an LDAP search
    - name: ldap_response
      type: keyword
      description: This key is to capture Results from an LDAP search
    - name: owner
      type: keyword
      description: This is used to capture username the process or service is running
        as, the author of the task
    - name: service_account
      type: keyword
      description: This key is a windows specific key, used for capturing name of
        the account a service (referenced in the event) is running under. Legacy Usage
  - name: email
    type: group
    fields:
    - name: email_dst
      type: keyword
      description: This key is used to capture the Destination email address only,
        when the destination context is not clear use email
    - name: email_src
      type: keyword
      description: This key is used to capture the source email address only, when
        the source context is not clear use email
    - name: subject
      type: keyword
      description: This key is used to capture the subject string from an Email only.
    - name: email
      type: keyword
      description: This key is used to capture a generic email address where the source
        or destination context is not clear
    - name: trans_from
      type: keyword
      description: Deprecated key defined only in table map.
    - name: trans_to
      type: keyword
      description: Deprecated key defined only in table map.
  - name: file
    type: group
    fields:
    - name: privilege
      type: keyword
      description: Deprecated, use permissions
    - name: attachment
      type: keyword
      description: This key captures the attachment file name
    - name: filesystem
      type: keyword
    - name: binary
      type: keyword
      description: Deprecated key defined only in table map.
    - name: filename_dst
      type: keyword
      description: This is used to capture name of the file targeted by the action
    - name: filename_src
      type: keyword
      description: This is used to capture name of the parent filename, the file which
        performed the action
    - name: filename_tmp
      type: keyword
    - name: directory_dst
      type: keyword
      description: <span>This key is used to capture the directory of the target process
        or file</span>
    - name: directory_src
      type: keyword
      description: This key is used to capture the directory of the source process
        or file
    - name: file_entropy
      type: double
      description: This is used to capture entropy vale of a file
    - name: file_vendor
      type: keyword
      description: This is used to capture Company name of file located in version_info
    - name: task_name
      type: keyword
      description: This is used to capture name of the task
  - name: web
    type: group
    fields:
    - name: fqdn
      type: keyword
      description: Fully Qualified Domain Names
    - name: web_cookie
      type: keyword
      description: This key is used to capture the Web cookies specifically.
    - name: alias_host
      type: keyword
    - name: reputation_num
      type: double
      description: Reputation Number of an entity. Typically used for Web Domains
    - name: web_ref_domain
      type: keyword
      description: Web referer's domain
    - name: web_ref_query
      type: keyword
      description: This key captures Web referer's query portion of the URL
    - name: remote_domain
      type: keyword
    - name: web_ref_page
      type: keyword
      description: This key captures Web referer's page information
    - name: web_ref_root
      type: keyword
      description: Web referer's root URL path
    - name: cn_asn_dst
      type: keyword
    - name: cn_rpackets
      type: keyword
    - name: urlpage
      type: keyword
    - name: urlroot
      type: keyword
    - name: p_url
      type: keyword
    - name: p_user_agent
      type: keyword
    - name: p_web_cookie
      type: keyword
    - name: p_web_method
      type: keyword
    - name: p_web_referer
      type: keyword
    - name: web_extension_tmp
      type: keyword
    - name: web_page
      type: keyword
  - name: threat
    type: group
    fields:
    - name: threat_category
      type: keyword
      description: This key captures Threat Name/Threat Category/Categorization of
        alert
    - name: threat_desc
      type: keyword
      description: This key is used to capture the threat description from the session
        directly or inferred
    - name: alert
      type: keyword
      description: This key is used to capture name of the alert
    - name: threat_source
      type: keyword
      description: This key is used to capture source of the threat
  - name: crypto
    type: group
    fields:
    - name: crypto
      type: keyword
      description: This key is used to capture the Encryption Type or Encryption Key
        only
    - name: cipher_src
      type: keyword
      description: This key is for Source (Client) Cipher
    - name: cert_subject
      type: keyword
      description: This key is used to capture the Certificate organization only
    - name: peer
      type: keyword
      description: This key is for Encryption peer's IP Address
    - name: cipher_size_src
      type: long
      description: This key captures Source (Client) Cipher Size
    - name: ike
      type: keyword
      description: IKE negotiation phase.
    - name: scheme
      type: keyword
      description: This key captures the Encryption scheme used
    - name: peer_id
      type: keyword
      description: "This key is for Encryption peer\u2019s identity"
    - name: sig_type
      type: keyword
      description: This key captures the Signature Type
    - name: cert_issuer
      type: keyword
    - name: cert_host_name
      type: keyword
      description: Deprecated key defined only in table map.
    - name: cert_error
      type: keyword
      description: This key captures the Certificate Error String
    - name: cipher_dst
      type: keyword
      description: This key is for Destination (Server) Cipher
    - name: cipher_size_dst
      type: long
      description: This key captures Destination (Server) Cipher Size
    - name: ssl_ver_src
      type: keyword
      description: Deprecated, use version
    - name: d_certauth
      type: keyword
    - name: s_certauth
      type: keyword
    - name: ike_cookie1
      type: keyword
      description: "ID of the negotiation \u2014 sent for ISAKMP Phase One"
    - name: ike_cookie2
      type: keyword
      description: "ID of the negotiation \u2014 sent for ISAKMP Phase Two"
    - name: cert_checksum
      type: keyword
    - name: cert_host_cat
      type: keyword
      description: This key is used for the hostname category value of a certificate
    - name: cert_serial
      type: keyword
      description: This key is used to capture the Certificate serial number only
    - name: cert_status
      type: keyword
      description: This key captures Certificate validation status
    - name: ssl_ver_dst
      type: keyword
      description: Deprecated, use version
    - name: cert_keysize
      type: keyword
    - name: cert_username
      type: keyword
    - name: https_insact
      type: keyword
    - name: https_valid
      type: keyword
    - name: cert_ca
      type: keyword
      description: This key is used to capture the Certificate signing authority only
    - name: cert_common
      type: keyword
      description: This key is used to capture the Certificate common name only
  - name: wireless
    type: group
    fields:
    - name: wlan_ssid
      type: keyword
      description: This key is used to capture the ssid of a Wireless Session
    - name: access_point
      type: keyword
      description: This key is used to capture the access point name.
    - name: wlan_channel
      type: long
      description: This is used to capture the channel names
    - name: wlan_name
      type: keyword
      description: This key captures either WLAN number/name
  - name: storage
    type: group
    fields:
    - name: disk_volume
      type: keyword
      description: A unique name assigned to logical units (volumes) within a physical
        disk
    - name: lun
      type: keyword
      description: Logical Unit Number.This key is a very useful concept in Storage.
    - name: pwwn
      type: keyword
      description: This uniquely identifies a port on a HBA.
  - name: physical
    type: group
    fields:
    - name: org_dst
      type: keyword
      description: This is used to capture the destination organization based on the
        GEOPIP Maxmind database.
    - name: org_src
      type: keyword
      description: This is used to capture the source organization based on the GEOPIP
        Maxmind database.
  - name: healthcare
    type: group
    fields:
    - name: patient_fname
      type: keyword
      description: This key is for First Names only, this is used for Healthcare predominantly
        to capture Patients information
    - name: patient_id
      type: keyword
      description: This key captures the unique ID for a patient
    - name: patient_lname
      type: keyword
      description: This key is for Last Names only, this is used for Healthcare predominantly
        to capture Patients information
    - name: patient_mname
      type: keyword
      description: This key is for Middle Names only, this is used for Healthcare
        predominantly to capture Patients information
  - name: endpoint
    type: group
    fields:
    - name: host_state
      type: keyword
      description: This key is used to capture the current state of the machine, such
        as <strong>blacklisted</strong>, <strong>infected</strong>, <strong>firewall
        disabled</strong> and so on
    - name: registry_key
      type: keyword
      description: This key captures the path to the registry key
    - name: registry_value
      type: keyword
      description: This key captures values or decorators used within a registry entry
