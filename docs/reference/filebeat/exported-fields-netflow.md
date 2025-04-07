---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-netflow.html
---

# NetFlow fields [exported-fields-netflow]

Fields from NetFlow and IPFIX flows.


## netflow [_netflow]

Fields from NetFlow and IPFIX.

**`netflow.type`**
:   The type of NetFlow record described by this event.

type: keyword



## exporter [_exporter]

Metadata related to the exporter device that generated this record.

**`netflow.exporter.address`**
:   Exporter’s network address in IP:port format.

type: keyword


**`netflow.exporter.source_id`**
:   Observation domain ID to which this record belongs.

type: long


**`netflow.exporter.timestamp`**
:   Time and date of export.

type: date


**`netflow.exporter.uptime_millis`**
:   How long the exporter process has been running, in milliseconds.

type: long


**`netflow.exporter.version`**
:   NetFlow version used.

type: integer


**`netflow.absolute_error`**
:   type: double


**`netflow.address_pool_high_threshold`**
:   type: long


**`netflow.address_pool_low_threshold`**
:   type: long


**`netflow.address_port_mapping_high_threshold`**
:   type: long


**`netflow.address_port_mapping_low_threshold`**
:   type: long


**`netflow.address_port_mapping_per_user_high_threshold`**
:   type: long


**`netflow.afc_protocol`**
:   type: integer


**`netflow.afc_protocol_name`**
:   type: keyword


**`netflow.anonymization_flags`**
:   type: integer


**`netflow.anonymization_technique`**
:   type: integer


**`netflow.application_business-relevance`**
:   type: long


**`netflow.application_category_name`**
:   type: keyword


**`netflow.application_description`**
:   type: keyword


**`netflow.application_group_name`**
:   type: keyword


**`netflow.application_http_uri_statistics`**
:   type: short


**`netflow.application_http_user-agent`**
:   type: short


**`netflow.application_id`**
:   type: short


**`netflow.application_name`**
:   type: keyword


**`netflow.application_sub_category_name`**
:   type: keyword


**`netflow.application_traffic-class`**
:   type: long


**`netflow.art_client_network_time_maximum`**
:   type: long


**`netflow.art_client_network_time_minimum`**
:   type: long


**`netflow.art_client_network_time_sum`**
:   type: long


**`netflow.art_clientpackets`**
:   type: long


**`netflow.art_count_late_responses`**
:   type: long


**`netflow.art_count_new_connections`**
:   type: long


**`netflow.art_count_responses`**
:   type: long


**`netflow.art_count_responses_histogram_bucket1`**
:   type: long


**`netflow.art_count_responses_histogram_bucket2`**
:   type: long


**`netflow.art_count_responses_histogram_bucket3`**
:   type: long


**`netflow.art_count_responses_histogram_bucket4`**
:   type: long


**`netflow.art_count_responses_histogram_bucket5`**
:   type: long


**`netflow.art_count_responses_histogram_bucket6`**
:   type: long


**`netflow.art_count_responses_histogram_bucket7`**
:   type: long


**`netflow.art_count_retransmissions`**
:   type: long


**`netflow.art_count_transactions`**
:   type: long


**`netflow.art_network_time_maximum`**
:   type: long


**`netflow.art_network_time_minimum`**
:   type: long


**`netflow.art_network_time_sum`**
:   type: long


**`netflow.art_response_time_maximum`**
:   type: long


**`netflow.art_response_time_minimum`**
:   type: long


**`netflow.art_response_time_sum`**
:   type: long


**`netflow.art_server_network_time_maximum`**
:   type: long


**`netflow.art_server_network_time_minimum`**
:   type: long


**`netflow.art_server_network_time_sum`**
:   type: long


**`netflow.art_server_response_time_maximum`**
:   type: long


**`netflow.art_server_response_time_minimum`**
:   type: long


**`netflow.art_server_response_time_sum`**
:   type: long


**`netflow.art_serverpackets`**
:   type: long


**`netflow.art_total_response_time_maximum`**
:   type: long


**`netflow.art_total_response_time_minimum`**
:   type: long


**`netflow.art_total_response_time_sum`**
:   type: long


**`netflow.art_total_transaction_time_maximum`**
:   type: long


**`netflow.art_total_transaction_time_minimum`**
:   type: long


**`netflow.art_total_transaction_time_sum`**
:   type: long


**`netflow.assembled_fragment_count`**
:   type: long


**`netflow.audit_counter`**
:   type: long


**`netflow.average_interarrival_time`**
:   type: long


**`netflow.bgp_destination_as_number`**
:   type: long


**`netflow.bgp_next_adjacent_as_number`**
:   type: long


**`netflow.bgp_next_hop_ipv4_address`**
:   type: ip


**`netflow.bgp_next_hop_ipv6_address`**
:   type: ip


**`netflow.bgp_prev_adjacent_as_number`**
:   type: long


**`netflow.bgp_source_as_number`**
:   type: long


**`netflow.bgp_validity_state`**
:   type: short


**`netflow.biflow_direction`**
:   type: short


**`netflow.bind_ipv4_address`**
:   type: ip


**`netflow.bind_transport_port`**
:   type: integer


**`netflow.class_id`**
:   type: long


**`netflow.class_name`**
:   type: keyword


**`netflow.classification_engine_id`**
:   type: short


**`netflow.collection_time_milliseconds`**
:   type: date


**`netflow.collector_certificate`**
:   type: short


**`netflow.collector_ipv4_address`**
:   type: ip


**`netflow.collector_ipv6_address`**
:   type: ip


**`netflow.collector_transport_port`**
:   type: integer


**`netflow.common_properties_id`**
:   type: long


**`netflow.confidence_level`**
:   type: double


**`netflow.conn_ipv4_address`**
:   type: ip


**`netflow.conn_transport_port`**
:   type: integer


**`netflow.connection_sum_duration_seconds`**
:   type: long


**`netflow.connection_transaction_id`**
:   type: long


**`netflow.conntrack_id`**
:   type: long


**`netflow.data_byte_count`**
:   type: long


**`netflow.data_link_frame_section`**
:   type: short


**`netflow.data_link_frame_size`**
:   type: integer


**`netflow.data_link_frame_type`**
:   type: integer


**`netflow.data_records_reliability`**
:   type: boolean


**`netflow.delta_flow_count`**
:   type: long


**`netflow.destination_ipv4_address`**
:   type: ip


**`netflow.destination_ipv4_prefix`**
:   type: ip


**`netflow.destination_ipv4_prefix_length`**
:   type: short


**`netflow.destination_ipv6_address`**
:   type: ip


**`netflow.destination_ipv6_prefix`**
:   type: ip


**`netflow.destination_ipv6_prefix_length`**
:   type: short


**`netflow.destination_mac_address`**
:   type: keyword


**`netflow.destination_transport_port`**
:   type: integer


**`netflow.digest_hash_value`**
:   type: long


**`netflow.distinct_count_of_destination_ip_address`**
:   type: long


**`netflow.distinct_count_of_destination_ipv4_address`**
:   type: long


**`netflow.distinct_count_of_destination_ipv6_address`**
:   type: long


**`netflow.distinct_count_of_source_ip_address`**
:   type: long


**`netflow.distinct_count_of_source_ipv4_address`**
:   type: long


**`netflow.distinct_count_of_source_ipv6_address`**
:   type: long


**`netflow.dns_authoritative`**
:   type: short


**`netflow.dns_cname`**
:   type: keyword


**`netflow.dns_id`**
:   type: integer


**`netflow.dns_mx_exchange`**
:   type: keyword


**`netflow.dns_mx_preference`**
:   type: integer


**`netflow.dns_nsd_name`**
:   type: keyword


**`netflow.dns_nx_domain`**
:   type: short


**`netflow.dns_ptrd_name`**
:   type: keyword


**`netflow.dns_qname`**
:   type: keyword


**`netflow.dns_qr_type`**
:   type: integer


**`netflow.dns_query_response`**
:   type: short


**`netflow.dns_rr_section`**
:   type: short


**`netflow.dns_soa_expire`**
:   type: long


**`netflow.dns_soa_minimum`**
:   type: long


**`netflow.dns_soa_refresh`**
:   type: long


**`netflow.dns_soa_retry`**
:   type: long


**`netflow.dns_soa_serial`**
:   type: long


**`netflow.dns_soam_name`**
:   type: keyword


**`netflow.dns_soar_name`**
:   type: keyword


**`netflow.dns_srv_port`**
:   type: integer


**`netflow.dns_srv_priority`**
:   type: integer


**`netflow.dns_srv_target`**
:   type: integer


**`netflow.dns_srv_weight`**
:   type: integer


**`netflow.dns_ttl`**
:   type: long


**`netflow.dns_txt_data`**
:   type: keyword


**`netflow.dot1q_customer_dei`**
:   type: boolean


**`netflow.dot1q_customer_destination_mac_address`**
:   type: keyword


**`netflow.dot1q_customer_priority`**
:   type: short


**`netflow.dot1q_customer_source_mac_address`**
:   type: keyword


**`netflow.dot1q_customer_vlan_id`**
:   type: integer


**`netflow.dot1q_dei`**
:   type: boolean


**`netflow.dot1q_priority`**
:   type: short


**`netflow.dot1q_service_instance_id`**
:   type: long


**`netflow.dot1q_service_instance_priority`**
:   type: short


**`netflow.dot1q_service_instance_tag`**
:   type: short


**`netflow.dot1q_vlan_id`**
:   type: integer


**`netflow.dropped_layer2_octet_delta_count`**
:   type: long


**`netflow.dropped_layer2_octet_total_count`**
:   type: long


**`netflow.dropped_octet_delta_count`**
:   type: long


**`netflow.dropped_octet_total_count`**
:   type: long


**`netflow.dropped_packet_delta_count`**
:   type: long


**`netflow.dropped_packet_total_count`**
:   type: long


**`netflow.dst_traffic_index`**
:   type: long


**`netflow.egress_broadcast_packet_total_count`**
:   type: long


**`netflow.egress_interface`**
:   type: long


**`netflow.egress_interface_type`**
:   type: long


**`netflow.egress_physical_interface`**
:   type: long


**`netflow.egress_unicast_packet_total_count`**
:   type: long


**`netflow.egress_vrfid`**
:   type: long


**`netflow.encrypted_technology`**
:   type: keyword


**`netflow.engine_id`**
:   type: short


**`netflow.engine_type`**
:   type: short


**`netflow.ethernet_header_length`**
:   type: short


**`netflow.ethernet_payload_length`**
:   type: integer


**`netflow.ethernet_total_length`**
:   type: integer


**`netflow.ethernet_type`**
:   type: integer


**`netflow.expired_fragment_count`**
:   type: long


**`netflow.export_interface`**
:   type: long


**`netflow.export_protocol_version`**
:   type: short


**`netflow.export_sctp_stream_id`**
:   type: integer


**`netflow.export_transport_protocol`**
:   type: short


**`netflow.exported_flow_record_total_count`**
:   type: long


**`netflow.exported_message_total_count`**
:   type: long


**`netflow.exported_octet_total_count`**
:   type: long


**`netflow.exporter_certificate`**
:   type: short


**`netflow.exporter_ipv4_address`**
:   type: ip


**`netflow.exporter_ipv6_address`**
:   type: ip


**`netflow.exporter_transport_port`**
:   type: integer


**`netflow.exporting_process_id`**
:   type: long


**`netflow.external_address_realm`**
:   type: short


**`netflow.firewall_event`**
:   type: short


**`netflow.first_eight_non_empty_packet_directions`**
:   type: short


**`netflow.first_non_empty_packet_size`**
:   type: integer


**`netflow.first_packet_banner`**
:   type: keyword


**`netflow.flags_and_sampler_id`**
:   type: long


**`netflow.flow_active_timeout`**
:   type: integer


**`netflow.flow_attributes`**
:   type: integer


**`netflow.flow_direction`**
:   type: short


**`netflow.flow_duration_microseconds`**
:   type: long


**`netflow.flow_duration_milliseconds`**
:   type: long


**`netflow.flow_end_delta_microseconds`**
:   type: long


**`netflow.flow_end_microseconds`**
:   type: date


**`netflow.flow_end_milliseconds`**
:   type: date


**`netflow.flow_end_nanoseconds`**
:   type: date


**`netflow.flow_end_reason`**
:   type: short


**`netflow.flow_end_seconds`**
:   type: date


**`netflow.flow_end_sys_up_time`**
:   type: long


**`netflow.flow_id`**
:   type: long


**`netflow.flow_idle_timeout`**
:   type: integer


**`netflow.flow_key_indicator`**
:   type: long


**`netflow.flow_label_ipv6`**
:   type: long


**`netflow.flow_sampling_time_interval`**
:   type: long


**`netflow.flow_sampling_time_spacing`**
:   type: long


**`netflow.flow_selected_flow_delta_count`**
:   type: long


**`netflow.flow_selected_octet_delta_count`**
:   type: long


**`netflow.flow_selected_packet_delta_count`**
:   type: long


**`netflow.flow_selector_algorithm`**
:   type: integer


**`netflow.flow_start_delta_microseconds`**
:   type: long


**`netflow.flow_start_microseconds`**
:   type: date


**`netflow.flow_start_milliseconds`**
:   type: date


**`netflow.flow_start_nanoseconds`**
:   type: date


**`netflow.flow_start_seconds`**
:   type: date


**`netflow.flow_start_sys_up_time`**
:   type: long


**`netflow.flow_table_flush_event_count`**
:   type: long


**`netflow.flow_table_peak_count`**
:   type: long


**`netflow.forwarding_status`**
:   type: short


**`netflow.fragment_flags`**
:   type: short


**`netflow.fragment_identification`**
:   type: long


**`netflow.fragment_offset`**
:   type: integer


**`netflow.fw_blackout_secs`**
:   type: long


**`netflow.fw_configured_value`**
:   type: long


**`netflow.fw_cts_src_sgt`**
:   type: long


**`netflow.fw_event_level`**
:   type: long


**`netflow.fw_event_level_id`**
:   type: long


**`netflow.fw_ext_event`**
:   type: integer


**`netflow.fw_ext_event_alt`**
:   type: long


**`netflow.fw_ext_event_desc`**
:   type: keyword


**`netflow.fw_half_open_count`**
:   type: long


**`netflow.fw_half_open_high`**
:   type: long


**`netflow.fw_half_open_rate`**
:   type: long


**`netflow.fw_max_sessions`**
:   type: long


**`netflow.fw_rule`**
:   type: keyword


**`netflow.fw_summary_pkt_count`**
:   type: long


**`netflow.fw_zone_pair_id`**
:   type: long


**`netflow.fw_zone_pair_name`**
:   type: long


**`netflow.global_address_mapping_high_threshold`**
:   type: long


**`netflow.gre_key`**
:   type: long


**`netflow.hash_digest_output`**
:   type: boolean


**`netflow.hash_flow_domain`**
:   type: integer


**`netflow.hash_initialiser_value`**
:   type: long


**`netflow.hash_ip_payload_offset`**
:   type: long


**`netflow.hash_ip_payload_size`**
:   type: long


**`netflow.hash_output_range_max`**
:   type: long


**`netflow.hash_output_range_min`**
:   type: long


**`netflow.hash_selected_range_max`**
:   type: long


**`netflow.hash_selected_range_min`**
:   type: long


**`netflow.http_content_type`**
:   type: keyword


**`netflow.http_message_version`**
:   type: keyword


**`netflow.http_reason_phrase`**
:   type: keyword


**`netflow.http_request_host`**
:   type: keyword


**`netflow.http_request_method`**
:   type: keyword


**`netflow.http_request_target`**
:   type: keyword


**`netflow.http_status_code`**
:   type: integer


**`netflow.http_user_agent`**
:   type: keyword


**`netflow.icmp_code_ipv4`**
:   type: short


**`netflow.icmp_code_ipv6`**
:   type: short


**`netflow.icmp_type_code_ipv4`**
:   type: integer


**`netflow.icmp_type_code_ipv6`**
:   type: integer


**`netflow.icmp_type_ipv4`**
:   type: short


**`netflow.icmp_type_ipv6`**
:   type: short


**`netflow.igmp_type`**
:   type: short


**`netflow.ignored_data_record_total_count`**
:   type: long


**`netflow.ignored_layer2_frame_total_count`**
:   type: long


**`netflow.ignored_layer2_octet_total_count`**
:   type: long


**`netflow.ignored_octet_total_count`**
:   type: long


**`netflow.ignored_packet_total_count`**
:   type: long


**`netflow.information_element_data_type`**
:   type: short


**`netflow.information_element_description`**
:   type: keyword


**`netflow.information_element_id`**
:   type: integer


**`netflow.information_element_index`**
:   type: integer


**`netflow.information_element_name`**
:   type: keyword


**`netflow.information_element_range_begin`**
:   type: long


**`netflow.information_element_range_end`**
:   type: long


**`netflow.information_element_semantics`**
:   type: short


**`netflow.information_element_units`**
:   type: integer


**`netflow.ingress_broadcast_packet_total_count`**
:   type: long


**`netflow.ingress_interface`**
:   type: long


**`netflow.ingress_interface_type`**
:   type: long


**`netflow.ingress_multicast_packet_total_count`**
:   type: long


**`netflow.ingress_physical_interface`**
:   type: long


**`netflow.ingress_unicast_packet_total_count`**
:   type: long


**`netflow.ingress_vrfid`**
:   type: long


**`netflow.initial_tcp_flags`**
:   type: short


**`netflow.initiator_octets`**
:   type: long


**`netflow.initiator_packets`**
:   type: long


**`netflow.interface_description`**
:   type: keyword


**`netflow.interface_name`**
:   type: keyword


**`netflow.intermediate_process_id`**
:   type: long


**`netflow.internal_address_realm`**
:   type: short


**`netflow.ip_class_of_service`**
:   type: short


**`netflow.ip_diff_serv_code_point`**
:   type: short


**`netflow.ip_header_length`**
:   type: short


**`netflow.ip_header_packet_section`**
:   type: short


**`netflow.ip_next_hop_ipv4_address`**
:   type: ip


**`netflow.ip_next_hop_ipv6_address`**
:   type: ip


**`netflow.ip_payload_length`**
:   type: long


**`netflow.ip_payload_packet_section`**
:   type: short


**`netflow.ip_precedence`**
:   type: short


**`netflow.ip_sec_spi`**
:   type: long


**`netflow.ip_total_length`**
:   type: long


**`netflow.ip_ttl`**
:   type: short


**`netflow.ip_version`**
:   type: short


**`netflow.ipv4_ihl`**
:   type: short


**`netflow.ipv4_options`**
:   type: long


**`netflow.ipv4_router_sc`**
:   type: ip


**`netflow.ipv6_extension_headers`**
:   type: long


**`netflow.is_multicast`**
:   type: short


**`netflow.ixia_browser_id`**
:   type: short


**`netflow.ixia_browser_name`**
:   type: keyword


**`netflow.ixia_device_id`**
:   type: short


**`netflow.ixia_device_name`**
:   type: keyword


**`netflow.ixia_dns_answer`**
:   type: keyword


**`netflow.ixia_dns_classes`**
:   type: keyword


**`netflow.ixia_dns_query`**
:   type: keyword


**`netflow.ixia_dns_record_txt`**
:   type: keyword


**`netflow.ixia_dst_as_name`**
:   type: keyword


**`netflow.ixia_dst_city_name`**
:   type: keyword


**`netflow.ixia_dst_country_code`**
:   type: keyword


**`netflow.ixia_dst_country_name`**
:   type: keyword


**`netflow.ixia_dst_latitude`**
:   type: float


**`netflow.ixia_dst_longitude`**
:   type: float


**`netflow.ixia_dst_region_code`**
:   type: keyword


**`netflow.ixia_dst_region_node`**
:   type: keyword


**`netflow.ixia_encrypt_cipher`**
:   type: keyword


**`netflow.ixia_encrypt_key_length`**
:   type: integer


**`netflow.ixia_encrypt_type`**
:   type: keyword


**`netflow.ixia_http_host_name`**
:   type: keyword


**`netflow.ixia_http_uri`**
:   type: keyword


**`netflow.ixia_http_user_agent`**
:   type: keyword


**`netflow.ixia_imsi_subscriber`**
:   type: keyword


**`netflow.ixia_l7_app_id`**
:   type: long


**`netflow.ixia_l7_app_name`**
:   type: keyword


**`netflow.ixia_latency`**
:   type: long


**`netflow.ixia_rev_octet_delta_count`**
:   type: long


**`netflow.ixia_rev_packet_delta_count`**
:   type: long


**`netflow.ixia_src_as_name`**
:   type: keyword


**`netflow.ixia_src_city_name`**
:   type: keyword


**`netflow.ixia_src_country_code`**
:   type: keyword


**`netflow.ixia_src_country_name`**
:   type: keyword


**`netflow.ixia_src_latitude`**
:   type: float


**`netflow.ixia_src_longitude`**
:   type: float


**`netflow.ixia_src_region_code`**
:   type: keyword


**`netflow.ixia_src_region_name`**
:   type: keyword


**`netflow.ixia_threat_ipv4`**
:   type: ip


**`netflow.ixia_threat_ipv6`**
:   type: ip


**`netflow.ixia_threat_type`**
:   type: keyword


**`netflow.large_packet_count`**
:   type: long


**`netflow.layer2_frame_delta_count`**
:   type: long


**`netflow.layer2_frame_total_count`**
:   type: long


**`netflow.layer2_octet_delta_count`**
:   type: long


**`netflow.layer2_octet_delta_sum_of_squares`**
:   type: long


**`netflow.layer2_octet_total_count`**
:   type: long


**`netflow.layer2_octet_total_sum_of_squares`**
:   type: long


**`netflow.layer2_segment_id`**
:   type: long


**`netflow.layer2packet_section_data`**
:   type: short


**`netflow.layer2packet_section_offset`**
:   type: integer


**`netflow.layer2packet_section_size`**
:   type: integer


**`netflow.line_card_id`**
:   type: long


**`netflow.log_op`**
:   type: short


**`netflow.lower_ci_limit`**
:   type: double


**`netflow.mark`**
:   type: long


**`netflow.max_bib_entries`**
:   type: long


**`netflow.max_entries_per_user`**
:   type: long


**`netflow.max_export_seconds`**
:   type: date


**`netflow.max_flow_end_microseconds`**
:   type: date


**`netflow.max_flow_end_milliseconds`**
:   type: date


**`netflow.max_flow_end_nanoseconds`**
:   type: date


**`netflow.max_flow_end_seconds`**
:   type: date


**`netflow.max_fragments_pending_reassembly`**
:   type: long


**`netflow.max_packet_size`**
:   type: integer


**`netflow.max_session_entries`**
:   type: long


**`netflow.max_subscribers`**
:   type: long


**`netflow.maximum_ip_total_length`**
:   type: long


**`netflow.maximum_layer2_total_length`**
:   type: long


**`netflow.maximum_ttl`**
:   type: short


**`netflow.mean_flow_rate`**
:   type: long


**`netflow.mean_packet_rate`**
:   type: long


**`netflow.message_md5_checksum`**
:   type: short


**`netflow.message_scope`**
:   type: short


**`netflow.metering_process_id`**
:   type: long


**`netflow.metro_evc_id`**
:   type: keyword


**`netflow.metro_evc_type`**
:   type: short


**`netflow.mib_capture_time_semantics`**
:   type: short


**`netflow.mib_context_engine_id`**
:   type: short


**`netflow.mib_context_name`**
:   type: keyword


**`netflow.mib_index_indicator`**
:   type: long


**`netflow.mib_module_name`**
:   type: keyword


**`netflow.mib_object_description`**
:   type: keyword


**`netflow.mib_object_identifier`**
:   type: short


**`netflow.mib_object_name`**
:   type: keyword


**`netflow.mib_object_syntax`**
:   type: keyword


**`netflow.mib_object_value_bits`**
:   type: short


**`netflow.mib_object_value_counter`**
:   type: long


**`netflow.mib_object_value_gauge`**
:   type: long


**`netflow.mib_object_value_integer`**
:   type: integer


**`netflow.mib_object_value_ip_address`**
:   type: ip


**`netflow.mib_object_value_octet_string`**
:   type: short


**`netflow.mib_object_value_oid`**
:   type: short


**`netflow.mib_object_value_time_ticks`**
:   type: long


**`netflow.mib_object_value_unsigned`**
:   type: long


**`netflow.mib_sub_identifier`**
:   type: long


**`netflow.min_export_seconds`**
:   type: date


**`netflow.min_flow_start_microseconds`**
:   type: date


**`netflow.min_flow_start_milliseconds`**
:   type: date


**`netflow.min_flow_start_nanoseconds`**
:   type: date


**`netflow.min_flow_start_seconds`**
:   type: date


**`netflow.minimum_ip_total_length`**
:   type: long


**`netflow.minimum_layer2_total_length`**
:   type: long


**`netflow.minimum_ttl`**
:   type: short


**`netflow.mobile_imsi`**
:   type: keyword


**`netflow.mobile_msisdn`**
:   type: keyword


**`netflow.monitoring_interval_end_milli_seconds`**
:   type: date


**`netflow.monitoring_interval_start_milli_seconds`**
:   type: date


**`netflow.mpls_label_stack_depth`**
:   type: long


**`netflow.mpls_label_stack_length`**
:   type: long


**`netflow.mpls_label_stack_section`**
:   type: short


**`netflow.mpls_label_stack_section10`**
:   type: short


**`netflow.mpls_label_stack_section2`**
:   type: short


**`netflow.mpls_label_stack_section3`**
:   type: short


**`netflow.mpls_label_stack_section4`**
:   type: short


**`netflow.mpls_label_stack_section5`**
:   type: short


**`netflow.mpls_label_stack_section6`**
:   type: short


**`netflow.mpls_label_stack_section7`**
:   type: short


**`netflow.mpls_label_stack_section8`**
:   type: short


**`netflow.mpls_label_stack_section9`**
:   type: short


**`netflow.mpls_payload_length`**
:   type: long


**`netflow.mpls_payload_packet_section`**
:   type: short


**`netflow.mpls_top_label_exp`**
:   type: short


**`netflow.mpls_top_label_ipv4_address`**
:   type: ip


**`netflow.mpls_top_label_ipv6_address`**
:   type: ip


**`netflow.mpls_top_label_prefix_length`**
:   type: short


**`netflow.mpls_top_label_stack_section`**
:   type: short


**`netflow.mpls_top_label_ttl`**
:   type: short


**`netflow.mpls_top_label_type`**
:   type: short


**`netflow.mpls_vpn_route_distinguisher`**
:   type: short


**`netflow.mptcp_address_id`**
:   type: short


**`netflow.mptcp_flags`**
:   type: short


**`netflow.mptcp_initial_data_sequence_number`**
:   type: long


**`netflow.mptcp_maximum_segment_size`**
:   type: integer


**`netflow.mptcp_receiver_token`**
:   type: long


**`netflow.multicast_replication_factor`**
:   type: long


**`netflow.nat_event`**
:   type: short


**`netflow.nat_inside_svcid`**
:   type: integer


**`netflow.nat_instance_id`**
:   type: long


**`netflow.nat_originating_address_realm`**
:   type: short


**`netflow.nat_outside_svcid`**
:   type: integer


**`netflow.nat_pool_id`**
:   type: long


**`netflow.nat_pool_name`**
:   type: keyword


**`netflow.nat_quota_exceeded_event`**
:   type: long


**`netflow.nat_sub_string`**
:   type: keyword


**`netflow.nat_threshold_event`**
:   type: long


**`netflow.nat_type`**
:   type: short


**`netflow.netscale_ica_client_version`**
:   type: keyword


**`netflow.netscaler_aaa_username`**
:   type: keyword


**`netflow.netscaler_app_name`**
:   type: keyword


**`netflow.netscaler_app_name_app_id`**
:   type: long


**`netflow.netscaler_app_name_incarnation_number`**
:   type: long


**`netflow.netscaler_app_template_name`**
:   type: keyword


**`netflow.netscaler_app_unit_name_app_id`**
:   type: long


**`netflow.netscaler_application_startup_duration`**
:   type: long


**`netflow.netscaler_application_startup_time`**
:   type: long


**`netflow.netscaler_cache_redir_client_connection_core_id`**
:   type: long


**`netflow.netscaler_cache_redir_client_connection_transaction_id`**
:   type: long


**`netflow.netscaler_client_rtt`**
:   type: long


**`netflow.netscaler_connection_chain_hop_count`**
:   type: long


**`netflow.netscaler_connection_chain_id`**
:   type: short


**`netflow.netscaler_connection_id`**
:   type: long


**`netflow.netscaler_current_license_consumed`**
:   type: long


**`netflow.netscaler_db_clt_host_name`**
:   type: keyword


**`netflow.netscaler_db_database_name`**
:   type: keyword


**`netflow.netscaler_db_login_flags`**
:   type: long


**`netflow.netscaler_db_protocol_name`**
:   type: short


**`netflow.netscaler_db_req_string`**
:   type: keyword


**`netflow.netscaler_db_req_type`**
:   type: short


**`netflow.netscaler_db_resp_length`**
:   type: long


**`netflow.netscaler_db_resp_status`**
:   type: long


**`netflow.netscaler_db_resp_status_string`**
:   type: keyword


**`netflow.netscaler_db_user_name`**
:   type: keyword


**`netflow.netscaler_flow_flags`**
:   type: long


**`netflow.netscaler_http_client_interaction_end_time`**
:   type: keyword


**`netflow.netscaler_http_client_interaction_start_time`**
:   type: keyword


**`netflow.netscaler_http_client_render_end_time`**
:   type: keyword


**`netflow.netscaler_http_client_render_start_time`**
:   type: keyword


**`netflow.netscaler_http_content_type`**
:   type: keyword


**`netflow.netscaler_http_domain_name`**
:   type: keyword


**`netflow.netscaler_http_req_authorization`**
:   type: keyword


**`netflow.netscaler_http_req_cookie`**
:   type: keyword


**`netflow.netscaler_http_req_forw_fb`**
:   type: long


**`netflow.netscaler_http_req_forw_lb`**
:   type: long


**`netflow.netscaler_http_req_host`**
:   type: keyword


**`netflow.netscaler_http_req_method`**
:   type: keyword


**`netflow.netscaler_http_req_rcv_fb`**
:   type: long


**`netflow.netscaler_http_req_rcv_lb`**
:   type: long


**`netflow.netscaler_http_req_referer`**
:   type: keyword


**`netflow.netscaler_http_req_url`**
:   type: keyword


**`netflow.netscaler_http_req_user_agent`**
:   type: keyword


**`netflow.netscaler_http_req_via`**
:   type: keyword


**`netflow.netscaler_http_req_xforwarded_for`**
:   type: keyword


**`netflow.netscaler_http_res_forw_fb`**
:   type: long


**`netflow.netscaler_http_res_forw_lb`**
:   type: long


**`netflow.netscaler_http_res_location`**
:   type: keyword


**`netflow.netscaler_http_res_rcv_fb`**
:   type: long


**`netflow.netscaler_http_res_rcv_lb`**
:   type: long


**`netflow.netscaler_http_res_set_cookie`**
:   type: keyword


**`netflow.netscaler_http_res_set_cookie2`**
:   type: keyword


**`netflow.netscaler_http_rsp_len`**
:   type: long


**`netflow.netscaler_http_rsp_status`**
:   type: integer


**`netflow.netscaler_ica_app_module_path`**
:   type: keyword


**`netflow.netscaler_ica_app_process_id`**
:   type: long


**`netflow.netscaler_ica_application_name`**
:   type: keyword


**`netflow.netscaler_ica_application_termination_time`**
:   type: long


**`netflow.netscaler_ica_application_termination_type`**
:   type: integer


**`netflow.netscaler_ica_channel_id1`**
:   type: long


**`netflow.netscaler_ica_channel_id1_bytes`**
:   type: long


**`netflow.netscaler_ica_channel_id2`**
:   type: long


**`netflow.netscaler_ica_channel_id2_bytes`**
:   type: long


**`netflow.netscaler_ica_channel_id3`**
:   type: long


**`netflow.netscaler_ica_channel_id3_bytes`**
:   type: long


**`netflow.netscaler_ica_channel_id4`**
:   type: long


**`netflow.netscaler_ica_channel_id4_bytes`**
:   type: long


**`netflow.netscaler_ica_channel_id5`**
:   type: long


**`netflow.netscaler_ica_channel_id5_bytes`**
:   type: long


**`netflow.netscaler_ica_client_host_name`**
:   type: keyword


**`netflow.netscaler_ica_client_ip`**
:   type: ip


**`netflow.netscaler_ica_client_launcher`**
:   type: integer


**`netflow.netscaler_ica_client_side_rto_count`**
:   type: integer


**`netflow.netscaler_ica_client_side_window_size`**
:   type: integer


**`netflow.netscaler_ica_client_type`**
:   type: integer


**`netflow.netscaler_ica_clientside_delay`**
:   type: long


**`netflow.netscaler_ica_clientside_jitter`**
:   type: long


**`netflow.netscaler_ica_clientside_packets_retransmit`**
:   type: integer


**`netflow.netscaler_ica_clientside_rtt`**
:   type: long


**`netflow.netscaler_ica_clientside_rx_bytes`**
:   type: long


**`netflow.netscaler_ica_clientside_srtt`**
:   type: long


**`netflow.netscaler_ica_clientside_tx_bytes`**
:   type: long


**`netflow.netscaler_ica_connection_priority`**
:   type: integer


**`netflow.netscaler_ica_device_serial_no`**
:   type: long


**`netflow.netscaler_ica_domain_name`**
:   type: keyword


**`netflow.netscaler_ica_flags`**
:   type: long


**`netflow.netscaler_ica_host_delay`**
:   type: long


**`netflow.netscaler_ica_l7_client_latency`**
:   type: long


**`netflow.netscaler_ica_l7_server_latency`**
:   type: long


**`netflow.netscaler_ica_launch_mechanism`**
:   type: integer


**`netflow.netscaler_ica_network_update_end_time`**
:   type: long


**`netflow.netscaler_ica_network_update_start_time`**
:   type: long


**`netflow.netscaler_ica_rtt`**
:   type: long


**`netflow.netscaler_ica_server_name`**
:   type: keyword


**`netflow.netscaler_ica_server_side_rto_count`**
:   type: integer


**`netflow.netscaler_ica_server_side_window_size`**
:   type: integer


**`netflow.netscaler_ica_serverside_delay`**
:   type: long


**`netflow.netscaler_ica_serverside_jitter`**
:   type: long


**`netflow.netscaler_ica_serverside_packets_retransmit`**
:   type: integer


**`netflow.netscaler_ica_serverside_rtt`**
:   type: long


**`netflow.netscaler_ica_serverside_srtt`**
:   type: long


**`netflow.netscaler_ica_session_end_time`**
:   type: long


**`netflow.netscaler_ica_session_guid`**
:   type: short


**`netflow.netscaler_ica_session_reconnects`**
:   type: short


**`netflow.netscaler_ica_session_setup_time`**
:   type: long


**`netflow.netscaler_ica_session_update_begin_sec`**
:   type: long


**`netflow.netscaler_ica_session_update_end_sec`**
:   type: long


**`netflow.netscaler_ica_username`**
:   type: keyword


**`netflow.netscaler_license_type`**
:   type: short


**`netflow.netscaler_main_page_core_id`**
:   type: long


**`netflow.netscaler_main_page_id`**
:   type: long


**`netflow.netscaler_max_license_count`**
:   type: long


**`netflow.netscaler_msi_client_cookie`**
:   type: short


**`netflow.netscaler_round_trip_time`**
:   type: long


**`netflow.netscaler_server_ttfb`**
:   type: long


**`netflow.netscaler_server_ttlb`**
:   type: long


**`netflow.netscaler_syslog_message`**
:   type: keyword


**`netflow.netscaler_syslog_priority`**
:   type: short


**`netflow.netscaler_syslog_timestamp`**
:   type: long


**`netflow.netscaler_transaction_id`**
:   type: long


**`netflow.netscaler_unknown270`**
:   type: long


**`netflow.netscaler_unknown271`**
:   type: long


**`netflow.netscaler_unknown272`**
:   type: long


**`netflow.netscaler_unknown273`**
:   type: long


**`netflow.netscaler_unknown274`**
:   type: long


**`netflow.netscaler_unknown275`**
:   type: long


**`netflow.netscaler_unknown276`**
:   type: long


**`netflow.netscaler_unknown277`**
:   type: long


**`netflow.netscaler_unknown278`**
:   type: long


**`netflow.netscaler_unknown279`**
:   type: long


**`netflow.netscaler_unknown280`**
:   type: long


**`netflow.netscaler_unknown281`**
:   type: long


**`netflow.netscaler_unknown282`**
:   type: long


**`netflow.netscaler_unknown283`**
:   type: long


**`netflow.netscaler_unknown284`**
:   type: long


**`netflow.netscaler_unknown285`**
:   type: long


**`netflow.netscaler_unknown286`**
:   type: long


**`netflow.netscaler_unknown287`**
:   type: long


**`netflow.netscaler_unknown288`**
:   type: long


**`netflow.netscaler_unknown289`**
:   type: long


**`netflow.netscaler_unknown290`**
:   type: long


**`netflow.netscaler_unknown291`**
:   type: long


**`netflow.netscaler_unknown292`**
:   type: long


**`netflow.netscaler_unknown293`**
:   type: long


**`netflow.netscaler_unknown294`**
:   type: long


**`netflow.netscaler_unknown295`**
:   type: long


**`netflow.netscaler_unknown296`**
:   type: long


**`netflow.netscaler_unknown297`**
:   type: long


**`netflow.netscaler_unknown298`**
:   type: long


**`netflow.netscaler_unknown299`**
:   type: long


**`netflow.netscaler_unknown300`**
:   type: long


**`netflow.netscaler_unknown301`**
:   type: long


**`netflow.netscaler_unknown302`**
:   type: long


**`netflow.netscaler_unknown303`**
:   type: long


**`netflow.netscaler_unknown304`**
:   type: long


**`netflow.netscaler_unknown305`**
:   type: long


**`netflow.netscaler_unknown306`**
:   type: long


**`netflow.netscaler_unknown307`**
:   type: long


**`netflow.netscaler_unknown308`**
:   type: long


**`netflow.netscaler_unknown309`**
:   type: long


**`netflow.netscaler_unknown310`**
:   type: long


**`netflow.netscaler_unknown311`**
:   type: long


**`netflow.netscaler_unknown312`**
:   type: long


**`netflow.netscaler_unknown313`**
:   type: long


**`netflow.netscaler_unknown314`**
:   type: long


**`netflow.netscaler_unknown315`**
:   type: long


**`netflow.netscaler_unknown316`**
:   type: keyword


**`netflow.netscaler_unknown317`**
:   type: long


**`netflow.netscaler_unknown318`**
:   type: long


**`netflow.netscaler_unknown319`**
:   type: keyword


**`netflow.netscaler_unknown320`**
:   type: integer


**`netflow.netscaler_unknown321`**
:   type: long


**`netflow.netscaler_unknown322`**
:   type: long


**`netflow.netscaler_unknown323`**
:   type: integer


**`netflow.netscaler_unknown324`**
:   type: integer


**`netflow.netscaler_unknown325`**
:   type: integer


**`netflow.netscaler_unknown326`**
:   type: integer


**`netflow.netscaler_unknown327`**
:   type: long


**`netflow.netscaler_unknown328`**
:   type: integer


**`netflow.netscaler_unknown329`**
:   type: integer


**`netflow.netscaler_unknown330`**
:   type: integer


**`netflow.netscaler_unknown331`**
:   type: integer


**`netflow.netscaler_unknown332`**
:   type: long


**`netflow.netscaler_unknown333`**
:   type: keyword


**`netflow.netscaler_unknown334`**
:   type: keyword


**`netflow.netscaler_unknown335`**
:   type: long


**`netflow.netscaler_unknown336`**
:   type: long


**`netflow.netscaler_unknown337`**
:   type: long


**`netflow.netscaler_unknown338`**
:   type: long


**`netflow.netscaler_unknown339`**
:   type: long


**`netflow.netscaler_unknown340`**
:   type: long


**`netflow.netscaler_unknown341`**
:   type: long


**`netflow.netscaler_unknown342`**
:   type: long


**`netflow.netscaler_unknown343`**
:   type: long


**`netflow.netscaler_unknown344`**
:   type: long


**`netflow.netscaler_unknown345`**
:   type: long


**`netflow.netscaler_unknown346`**
:   type: long


**`netflow.netscaler_unknown347`**
:   type: long


**`netflow.netscaler_unknown348`**
:   type: integer


**`netflow.netscaler_unknown349`**
:   type: keyword


**`netflow.netscaler_unknown350`**
:   type: keyword


**`netflow.netscaler_unknown351`**
:   type: keyword


**`netflow.netscaler_unknown352`**
:   type: integer


**`netflow.netscaler_unknown353`**
:   type: long


**`netflow.netscaler_unknown354`**
:   type: long


**`netflow.netscaler_unknown355`**
:   type: long


**`netflow.netscaler_unknown356`**
:   type: long


**`netflow.netscaler_unknown357`**
:   type: long


**`netflow.netscaler_unknown363`**
:   type: short


**`netflow.netscaler_unknown383`**
:   type: short


**`netflow.netscaler_unknown391`**
:   type: long


**`netflow.netscaler_unknown398`**
:   type: long


**`netflow.netscaler_unknown404`**
:   type: long


**`netflow.netscaler_unknown405`**
:   type: long


**`netflow.netscaler_unknown427`**
:   type: long


**`netflow.netscaler_unknown429`**
:   type: short


**`netflow.netscaler_unknown432`**
:   type: short


**`netflow.netscaler_unknown433`**
:   type: short


**`netflow.netscaler_unknown453`**
:   type: long


**`netflow.netscaler_unknown465`**
:   type: long


**`netflow.new_connection_delta_count`**
:   type: long


**`netflow.next_header_ipv6`**
:   type: short


**`netflow.non_empty_packet_count`**
:   type: long


**`netflow.not_sent_flow_total_count`**
:   type: long


**`netflow.not_sent_layer2_octet_total_count`**
:   type: long


**`netflow.not_sent_octet_total_count`**
:   type: long


**`netflow.not_sent_packet_total_count`**
:   type: long


**`netflow.observation_domain_id`**
:   type: long


**`netflow.observation_domain_name`**
:   type: keyword


**`netflow.observation_point_id`**
:   type: long


**`netflow.observation_point_type`**
:   type: short


**`netflow.observation_time_microseconds`**
:   type: date


**`netflow.observation_time_milliseconds`**
:   type: date


**`netflow.observation_time_nanoseconds`**
:   type: date


**`netflow.observation_time_seconds`**
:   type: date


**`netflow.observed_flow_total_count`**
:   type: long


**`netflow.octet_delta_count`**
:   type: long


**`netflow.octet_delta_sum_of_squares`**
:   type: long


**`netflow.octet_total_count`**
:   type: long


**`netflow.octet_total_sum_of_squares`**
:   type: long


**`netflow.opaque_octets`**
:   type: short


**`netflow.original_exporter_ipv4_address`**
:   type: ip


**`netflow.original_exporter_ipv6_address`**
:   type: ip


**`netflow.original_flows_completed`**
:   type: long


**`netflow.original_flows_initiated`**
:   type: long


**`netflow.original_flows_present`**
:   type: long


**`netflow.original_observation_domain_id`**
:   type: long


**`netflow.os_finger_print`**
:   type: keyword


**`netflow.os_name`**
:   type: keyword


**`netflow.os_version`**
:   type: keyword


**`netflow.p2p_technology`**
:   type: keyword


**`netflow.packet_delta_count`**
:   type: long


**`netflow.packet_total_count`**
:   type: long


**`netflow.padding_octets`**
:   type: short


**`netflow.payload`**
:   type: keyword


**`netflow.payload_entropy`**
:   type: short


**`netflow.payload_length_ipv6`**
:   type: integer


**`netflow.policy_qos_classification_hierarchy`**
:   type: long


**`netflow.policy_qos_queue_index`**
:   type: long


**`netflow.policy_qos_queuedrops`**
:   type: long


**`netflow.policy_qos_queueindex`**
:   type: long


**`netflow.port_id`**
:   type: long


**`netflow.port_range_end`**
:   type: integer


**`netflow.port_range_num_ports`**
:   type: integer


**`netflow.port_range_start`**
:   type: integer


**`netflow.port_range_step_size`**
:   type: integer


**`netflow.post_destination_mac_address`**
:   type: keyword


**`netflow.post_dot1q_customer_vlan_id`**
:   type: integer


**`netflow.post_dot1q_vlan_id`**
:   type: integer


**`netflow.post_ip_class_of_service`**
:   type: short


**`netflow.post_ip_diff_serv_code_point`**
:   type: short


**`netflow.post_ip_precedence`**
:   type: short


**`netflow.post_layer2_octet_delta_count`**
:   type: long


**`netflow.post_layer2_octet_total_count`**
:   type: long


**`netflow.post_mcast_layer2_octet_delta_count`**
:   type: long


**`netflow.post_mcast_layer2_octet_total_count`**
:   type: long


**`netflow.post_mcast_octet_delta_count`**
:   type: long


**`netflow.post_mcast_octet_total_count`**
:   type: long


**`netflow.post_mcast_packet_delta_count`**
:   type: long


**`netflow.post_mcast_packet_total_count`**
:   type: long


**`netflow.post_mpls_top_label_exp`**
:   type: short


**`netflow.post_napt_destination_transport_port`**
:   type: integer


**`netflow.post_napt_source_transport_port`**
:   type: integer


**`netflow.post_nat_destination_ipv4_address`**
:   type: ip


**`netflow.post_nat_destination_ipv6_address`**
:   type: ip


**`netflow.post_nat_source_ipv4_address`**
:   type: ip


**`netflow.post_nat_source_ipv6_address`**
:   type: ip


**`netflow.post_octet_delta_count`**
:   type: long


**`netflow.post_octet_total_count`**
:   type: long


**`netflow.post_packet_delta_count`**
:   type: long


**`netflow.post_packet_total_count`**
:   type: long


**`netflow.post_source_mac_address`**
:   type: keyword


**`netflow.post_vlan_id`**
:   type: integer


**`netflow.private_enterprise_number`**
:   type: long


**`netflow.procera_apn`**
:   type: keyword


**`netflow.procera_base_service`**
:   type: keyword


**`netflow.procera_content_categories`**
:   type: keyword


**`netflow.procera_device_id`**
:   type: long


**`netflow.procera_external_rtt`**
:   type: integer


**`netflow.procera_flow_behavior`**
:   type: keyword


**`netflow.procera_ggsn`**
:   type: keyword


**`netflow.procera_http_content_type`**
:   type: keyword


**`netflow.procera_http_file_length`**
:   type: long


**`netflow.procera_http_language`**
:   type: keyword


**`netflow.procera_http_location`**
:   type: keyword


**`netflow.procera_http_referer`**
:   type: keyword


**`netflow.procera_http_request_method`**
:   type: keyword


**`netflow.procera_http_request_version`**
:   type: keyword


**`netflow.procera_http_response_status`**
:   type: integer


**`netflow.procera_http_url`**
:   type: keyword


**`netflow.procera_http_user_agent`**
:   type: keyword


**`netflow.procera_imsi`**
:   type: long


**`netflow.procera_incoming_octets`**
:   type: long


**`netflow.procera_incoming_packets`**
:   type: long


**`netflow.procera_incoming_shaping_drops`**
:   type: long


**`netflow.procera_incoming_shaping_latency`**
:   type: integer


**`netflow.procera_internal_rtt`**
:   type: integer


**`netflow.procera_local_ipv4_host`**
:   type: ip


**`netflow.procera_local_ipv6_host`**
:   type: ip


**`netflow.procera_msisdn`**
:   type: long


**`netflow.procera_outgoing_octets`**
:   type: long


**`netflow.procera_outgoing_packets`**
:   type: long


**`netflow.procera_outgoing_shaping_drops`**
:   type: long


**`netflow.procera_outgoing_shaping_latency`**
:   type: integer


**`netflow.procera_property`**
:   type: keyword


**`netflow.procera_qoe_incoming_external`**
:   type: float


**`netflow.procera_qoe_incoming_internal`**
:   type: float


**`netflow.procera_qoe_outgoing_external`**
:   type: float


**`netflow.procera_qoe_outgoing_internal`**
:   type: float


**`netflow.procera_rat`**
:   type: keyword


**`netflow.procera_remote_ipv4_host`**
:   type: ip


**`netflow.procera_remote_ipv6_host`**
:   type: ip


**`netflow.procera_rnc`**
:   type: integer


**`netflow.procera_server_hostname`**
:   type: keyword


**`netflow.procera_service`**
:   type: keyword


**`netflow.procera_sgsn`**
:   type: keyword


**`netflow.procera_subscriber_identifier`**
:   type: keyword


**`netflow.procera_template_name`**
:   type: keyword


**`netflow.procera_user_location_information`**
:   type: keyword


**`netflow.protocol_identifier`**
:   type: short


**`netflow.pseudo_wire_control_word`**
:   type: long


**`netflow.pseudo_wire_destination_ipv4_address`**
:   type: ip


**`netflow.pseudo_wire_id`**
:   type: long


**`netflow.pseudo_wire_type`**
:   type: integer


**`netflow.reason`**
:   type: long


**`netflow.reason_text`**
:   type: keyword


**`netflow.relative_error`**
:   type: double


**`netflow.responder_octets`**
:   type: long


**`netflow.responder_packets`**
:   type: long


**`netflow.reverse_absolute_error`**
:   type: double


**`netflow.reverse_anonymization_flags`**
:   type: integer


**`netflow.reverse_anonymization_technique`**
:   type: integer


**`netflow.reverse_application_category_name`**
:   type: keyword


**`netflow.reverse_application_description`**
:   type: keyword


**`netflow.reverse_application_group_name`**
:   type: keyword


**`netflow.reverse_application_id`**
:   type: keyword


**`netflow.reverse_application_name`**
:   type: keyword


**`netflow.reverse_application_sub_category_name`**
:   type: keyword


**`netflow.reverse_average_interarrival_time`**
:   type: long


**`netflow.reverse_bgp_destination_as_number`**
:   type: long


**`netflow.reverse_bgp_next_adjacent_as_number`**
:   type: long


**`netflow.reverse_bgp_next_hop_ipv4_address`**
:   type: ip


**`netflow.reverse_bgp_next_hop_ipv6_address`**
:   type: ip


**`netflow.reverse_bgp_prev_adjacent_as_number`**
:   type: long


**`netflow.reverse_bgp_source_as_number`**
:   type: long


**`netflow.reverse_bgp_validity_state`**
:   type: short


**`netflow.reverse_class_id`**
:   type: short


**`netflow.reverse_class_name`**
:   type: keyword


**`netflow.reverse_classification_engine_id`**
:   type: short


**`netflow.reverse_collection_time_milliseconds`**
:   type: long


**`netflow.reverse_collector_certificate`**
:   type: keyword


**`netflow.reverse_confidence_level`**
:   type: double


**`netflow.reverse_connection_sum_duration_seconds`**
:   type: long


**`netflow.reverse_connection_transaction_id`**
:   type: long


**`netflow.reverse_data_byte_count`**
:   type: long


**`netflow.reverse_data_link_frame_section`**
:   type: keyword


**`netflow.reverse_data_link_frame_size`**
:   type: integer


**`netflow.reverse_data_link_frame_type`**
:   type: integer


**`netflow.reverse_data_records_reliability`**
:   type: short


**`netflow.reverse_delta_flow_count`**
:   type: long


**`netflow.reverse_destination_ipv4_address`**
:   type: ip


**`netflow.reverse_destination_ipv4_prefix`**
:   type: ip


**`netflow.reverse_destination_ipv4_prefix_length`**
:   type: short


**`netflow.reverse_destination_ipv6_address`**
:   type: ip


**`netflow.reverse_destination_ipv6_prefix`**
:   type: ip


**`netflow.reverse_destination_ipv6_prefix_length`**
:   type: short


**`netflow.reverse_destination_mac_address`**
:   type: keyword


**`netflow.reverse_destination_transport_port`**
:   type: integer


**`netflow.reverse_digest_hash_value`**
:   type: long


**`netflow.reverse_distinct_count_of_destination_ip_address`**
:   type: long


**`netflow.reverse_distinct_count_of_destination_ipv4_address`**
:   type: long


**`netflow.reverse_distinct_count_of_destination_ipv6_address`**
:   type: long


**`netflow.reverse_distinct_count_of_source_ip_address`**
:   type: long


**`netflow.reverse_distinct_count_of_source_ipv4_address`**
:   type: long


**`netflow.reverse_distinct_count_of_source_ipv6_address`**
:   type: long


**`netflow.reverse_dot1q_customer_dei`**
:   type: short


**`netflow.reverse_dot1q_customer_destination_mac_address`**
:   type: keyword


**`netflow.reverse_dot1q_customer_priority`**
:   type: short


**`netflow.reverse_dot1q_customer_source_mac_address`**
:   type: keyword


**`netflow.reverse_dot1q_customer_vlan_id`**
:   type: integer


**`netflow.reverse_dot1q_dei`**
:   type: short


**`netflow.reverse_dot1q_priority`**
:   type: short


**`netflow.reverse_dot1q_service_instance_id`**
:   type: long


**`netflow.reverse_dot1q_service_instance_priority`**
:   type: short


**`netflow.reverse_dot1q_service_instance_tag`**
:   type: keyword


**`netflow.reverse_dot1q_vlan_id`**
:   type: integer


**`netflow.reverse_dropped_layer2_octet_delta_count`**
:   type: long


**`netflow.reverse_dropped_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_dropped_octet_delta_count`**
:   type: long


**`netflow.reverse_dropped_octet_total_count`**
:   type: long


**`netflow.reverse_dropped_packet_delta_count`**
:   type: long


**`netflow.reverse_dropped_packet_total_count`**
:   type: long


**`netflow.reverse_dst_traffic_index`**
:   type: long


**`netflow.reverse_egress_broadcast_packet_total_count`**
:   type: long


**`netflow.reverse_egress_interface`**
:   type: long


**`netflow.reverse_egress_interface_type`**
:   type: long


**`netflow.reverse_egress_physical_interface`**
:   type: long


**`netflow.reverse_egress_unicast_packet_total_count`**
:   type: long


**`netflow.reverse_egress_vrfid`**
:   type: long


**`netflow.reverse_encrypted_technology`**
:   type: keyword


**`netflow.reverse_engine_id`**
:   type: short


**`netflow.reverse_engine_type`**
:   type: short


**`netflow.reverse_ethernet_header_length`**
:   type: short


**`netflow.reverse_ethernet_payload_length`**
:   type: integer


**`netflow.reverse_ethernet_total_length`**
:   type: integer


**`netflow.reverse_ethernet_type`**
:   type: integer


**`netflow.reverse_export_sctp_stream_id`**
:   type: integer


**`netflow.reverse_exporter_certificate`**
:   type: keyword


**`netflow.reverse_exporting_process_id`**
:   type: long


**`netflow.reverse_firewall_event`**
:   type: short


**`netflow.reverse_first_non_empty_packet_size`**
:   type: integer


**`netflow.reverse_first_packet_banner`**
:   type: keyword


**`netflow.reverse_flags_and_sampler_id`**
:   type: long


**`netflow.reverse_flow_active_timeout`**
:   type: integer


**`netflow.reverse_flow_attributes`**
:   type: integer


**`netflow.reverse_flow_delta_milliseconds`**
:   type: long


**`netflow.reverse_flow_direction`**
:   type: short


**`netflow.reverse_flow_duration_microseconds`**
:   type: long


**`netflow.reverse_flow_duration_milliseconds`**
:   type: long


**`netflow.reverse_flow_end_delta_microseconds`**
:   type: long


**`netflow.reverse_flow_end_microseconds`**
:   type: long


**`netflow.reverse_flow_end_milliseconds`**
:   type: long


**`netflow.reverse_flow_end_nanoseconds`**
:   type: long


**`netflow.reverse_flow_end_reason`**
:   type: short


**`netflow.reverse_flow_end_seconds`**
:   type: long


**`netflow.reverse_flow_end_sys_up_time`**
:   type: long


**`netflow.reverse_flow_idle_timeout`**
:   type: integer


**`netflow.reverse_flow_label_ipv6`**
:   type: long


**`netflow.reverse_flow_sampling_time_interval`**
:   type: long


**`netflow.reverse_flow_sampling_time_spacing`**
:   type: long


**`netflow.reverse_flow_selected_flow_delta_count`**
:   type: long


**`netflow.reverse_flow_selected_octet_delta_count`**
:   type: long


**`netflow.reverse_flow_selected_packet_delta_count`**
:   type: long


**`netflow.reverse_flow_selector_algorithm`**
:   type: integer


**`netflow.reverse_flow_start_delta_microseconds`**
:   type: long


**`netflow.reverse_flow_start_microseconds`**
:   type: long


**`netflow.reverse_flow_start_milliseconds`**
:   type: long


**`netflow.reverse_flow_start_nanoseconds`**
:   type: long


**`netflow.reverse_flow_start_seconds`**
:   type: long


**`netflow.reverse_flow_start_sys_up_time`**
:   type: long


**`netflow.reverse_forwarding_status`**
:   type: long


**`netflow.reverse_fragment_flags`**
:   type: short


**`netflow.reverse_fragment_identification`**
:   type: long


**`netflow.reverse_fragment_offset`**
:   type: integer


**`netflow.reverse_gre_key`**
:   type: long


**`netflow.reverse_hash_digest_output`**
:   type: short


**`netflow.reverse_hash_flow_domain`**
:   type: integer


**`netflow.reverse_hash_initialiser_value`**
:   type: long


**`netflow.reverse_hash_ip_payload_offset`**
:   type: long


**`netflow.reverse_hash_ip_payload_size`**
:   type: long


**`netflow.reverse_hash_output_range_max`**
:   type: long


**`netflow.reverse_hash_output_range_min`**
:   type: long


**`netflow.reverse_hash_selected_range_max`**
:   type: long


**`netflow.reverse_hash_selected_range_min`**
:   type: long


**`netflow.reverse_icmp_code_ipv4`**
:   type: short


**`netflow.reverse_icmp_code_ipv6`**
:   type: short


**`netflow.reverse_icmp_type_code_ipv4`**
:   type: integer


**`netflow.reverse_icmp_type_code_ipv6`**
:   type: integer


**`netflow.reverse_icmp_type_ipv4`**
:   type: short


**`netflow.reverse_icmp_type_ipv6`**
:   type: short


**`netflow.reverse_igmp_type`**
:   type: short


**`netflow.reverse_ignored_data_record_total_count`**
:   type: long


**`netflow.reverse_ignored_layer2_frame_total_count`**
:   type: long


**`netflow.reverse_ignored_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_information_element_data_type`**
:   type: short


**`netflow.reverse_information_element_description`**
:   type: keyword


**`netflow.reverse_information_element_id`**
:   type: integer


**`netflow.reverse_information_element_index`**
:   type: integer


**`netflow.reverse_information_element_name`**
:   type: keyword


**`netflow.reverse_information_element_range_begin`**
:   type: long


**`netflow.reverse_information_element_range_end`**
:   type: long


**`netflow.reverse_information_element_semantics`**
:   type: short


**`netflow.reverse_information_element_units`**
:   type: integer


**`netflow.reverse_ingress_broadcast_packet_total_count`**
:   type: long


**`netflow.reverse_ingress_interface`**
:   type: long


**`netflow.reverse_ingress_interface_type`**
:   type: long


**`netflow.reverse_ingress_multicast_packet_total_count`**
:   type: long


**`netflow.reverse_ingress_physical_interface`**
:   type: long


**`netflow.reverse_ingress_unicast_packet_total_count`**
:   type: long


**`netflow.reverse_ingress_vrfid`**
:   type: long


**`netflow.reverse_initial_tcp_flags`**
:   type: short


**`netflow.reverse_initiator_octets`**
:   type: long


**`netflow.reverse_initiator_packets`**
:   type: long


**`netflow.reverse_interface_description`**
:   type: keyword


**`netflow.reverse_interface_name`**
:   type: keyword


**`netflow.reverse_intermediate_process_id`**
:   type: long


**`netflow.reverse_ip_class_of_service`**
:   type: short


**`netflow.reverse_ip_diff_serv_code_point`**
:   type: short


**`netflow.reverse_ip_header_length`**
:   type: short


**`netflow.reverse_ip_header_packet_section`**
:   type: keyword


**`netflow.reverse_ip_next_hop_ipv4_address`**
:   type: ip


**`netflow.reverse_ip_next_hop_ipv6_address`**
:   type: ip


**`netflow.reverse_ip_payload_length`**
:   type: long


**`netflow.reverse_ip_payload_packet_section`**
:   type: keyword


**`netflow.reverse_ip_precedence`**
:   type: short


**`netflow.reverse_ip_sec_spi`**
:   type: long


**`netflow.reverse_ip_total_length`**
:   type: long


**`netflow.reverse_ip_ttl`**
:   type: short


**`netflow.reverse_ip_version`**
:   type: short


**`netflow.reverse_ipv4_ihl`**
:   type: short


**`netflow.reverse_ipv4_options`**
:   type: long


**`netflow.reverse_ipv4_router_sc`**
:   type: ip


**`netflow.reverse_ipv6_extension_headers`**
:   type: long


**`netflow.reverse_is_multicast`**
:   type: short


**`netflow.reverse_large_packet_count`**
:   type: long


**`netflow.reverse_layer2_frame_delta_count`**
:   type: long


**`netflow.reverse_layer2_frame_total_count`**
:   type: long


**`netflow.reverse_layer2_octet_delta_count`**
:   type: long


**`netflow.reverse_layer2_octet_delta_sum_of_squares`**
:   type: long


**`netflow.reverse_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_layer2_octet_total_sum_of_squares`**
:   type: long


**`netflow.reverse_layer2_segment_id`**
:   type: long


**`netflow.reverse_layer2packet_section_data`**
:   type: keyword


**`netflow.reverse_layer2packet_section_offset`**
:   type: integer


**`netflow.reverse_layer2packet_section_size`**
:   type: integer


**`netflow.reverse_line_card_id`**
:   type: long


**`netflow.reverse_lower_ci_limit`**
:   type: double


**`netflow.reverse_max_export_seconds`**
:   type: long


**`netflow.reverse_max_flow_end_microseconds`**
:   type: long


**`netflow.reverse_max_flow_end_milliseconds`**
:   type: long


**`netflow.reverse_max_flow_end_nanoseconds`**
:   type: long


**`netflow.reverse_max_flow_end_seconds`**
:   type: long


**`netflow.reverse_max_packet_size`**
:   type: integer


**`netflow.reverse_maximum_ip_total_length`**
:   type: long


**`netflow.reverse_maximum_layer2_total_length`**
:   type: long


**`netflow.reverse_maximum_ttl`**
:   type: short


**`netflow.reverse_message_md5_checksum`**
:   type: keyword


**`netflow.reverse_message_scope`**
:   type: short


**`netflow.reverse_metering_process_id`**
:   type: long


**`netflow.reverse_metro_evc_id`**
:   type: keyword


**`netflow.reverse_metro_evc_type`**
:   type: short


**`netflow.reverse_min_export_seconds`**
:   type: long


**`netflow.reverse_min_flow_start_microseconds`**
:   type: long


**`netflow.reverse_min_flow_start_milliseconds`**
:   type: long


**`netflow.reverse_min_flow_start_nanoseconds`**
:   type: long


**`netflow.reverse_min_flow_start_seconds`**
:   type: long


**`netflow.reverse_minimum_ip_total_length`**
:   type: long


**`netflow.reverse_minimum_layer2_total_length`**
:   type: long


**`netflow.reverse_minimum_ttl`**
:   type: short


**`netflow.reverse_monitoring_interval_end_milli_seconds`**
:   type: long


**`netflow.reverse_monitoring_interval_start_milli_seconds`**
:   type: long


**`netflow.reverse_mpls_label_stack_depth`**
:   type: long


**`netflow.reverse_mpls_label_stack_length`**
:   type: long


**`netflow.reverse_mpls_label_stack_section`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section10`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section2`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section3`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section4`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section5`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section6`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section7`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section8`**
:   type: keyword


**`netflow.reverse_mpls_label_stack_section9`**
:   type: keyword


**`netflow.reverse_mpls_payload_length`**
:   type: long


**`netflow.reverse_mpls_payload_packet_section`**
:   type: keyword


**`netflow.reverse_mpls_top_label_exp`**
:   type: short


**`netflow.reverse_mpls_top_label_ipv4_address`**
:   type: ip


**`netflow.reverse_mpls_top_label_ipv6_address`**
:   type: ip


**`netflow.reverse_mpls_top_label_prefix_length`**
:   type: short


**`netflow.reverse_mpls_top_label_stack_section`**
:   type: keyword


**`netflow.reverse_mpls_top_label_ttl`**
:   type: short


**`netflow.reverse_mpls_top_label_type`**
:   type: short


**`netflow.reverse_mpls_vpn_route_distinguisher`**
:   type: keyword


**`netflow.reverse_multicast_replication_factor`**
:   type: long


**`netflow.reverse_nat_event`**
:   type: short


**`netflow.reverse_nat_originating_address_realm`**
:   type: short


**`netflow.reverse_nat_pool_id`**
:   type: long


**`netflow.reverse_nat_pool_name`**
:   type: keyword


**`netflow.reverse_nat_type`**
:   type: short


**`netflow.reverse_new_connection_delta_count`**
:   type: long


**`netflow.reverse_next_header_ipv6`**
:   type: short


**`netflow.reverse_non_empty_packet_count`**
:   type: long


**`netflow.reverse_not_sent_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_observation_domain_name`**
:   type: keyword


**`netflow.reverse_observation_point_id`**
:   type: long


**`netflow.reverse_observation_point_type`**
:   type: short


**`netflow.reverse_observation_time_microseconds`**
:   type: long


**`netflow.reverse_observation_time_milliseconds`**
:   type: long


**`netflow.reverse_observation_time_nanoseconds`**
:   type: long


**`netflow.reverse_observation_time_seconds`**
:   type: long


**`netflow.reverse_octet_delta_count`**
:   type: long


**`netflow.reverse_octet_delta_sum_of_squares`**
:   type: long


**`netflow.reverse_octet_total_count`**
:   type: long


**`netflow.reverse_octet_total_sum_of_squares`**
:   type: long


**`netflow.reverse_opaque_octets`**
:   type: keyword


**`netflow.reverse_original_exporter_ipv4_address`**
:   type: ip


**`netflow.reverse_original_exporter_ipv6_address`**
:   type: ip


**`netflow.reverse_original_flows_completed`**
:   type: long


**`netflow.reverse_original_flows_initiated`**
:   type: long


**`netflow.reverse_original_flows_present`**
:   type: long


**`netflow.reverse_original_observation_domain_id`**
:   type: long


**`netflow.reverse_os_finger_print`**
:   type: keyword


**`netflow.reverse_os_name`**
:   type: keyword


**`netflow.reverse_os_version`**
:   type: keyword


**`netflow.reverse_p2p_technology`**
:   type: keyword


**`netflow.reverse_packet_delta_count`**
:   type: long


**`netflow.reverse_packet_total_count`**
:   type: long


**`netflow.reverse_payload`**
:   type: keyword


**`netflow.reverse_payload_entropy`**
:   type: short


**`netflow.reverse_payload_length_ipv6`**
:   type: integer


**`netflow.reverse_port_id`**
:   type: long


**`netflow.reverse_port_range_end`**
:   type: integer


**`netflow.reverse_port_range_num_ports`**
:   type: integer


**`netflow.reverse_port_range_start`**
:   type: integer


**`netflow.reverse_port_range_step_size`**
:   type: integer


**`netflow.reverse_post_destination_mac_address`**
:   type: keyword


**`netflow.reverse_post_dot1q_customer_vlan_id`**
:   type: integer


**`netflow.reverse_post_dot1q_vlan_id`**
:   type: integer


**`netflow.reverse_post_ip_class_of_service`**
:   type: short


**`netflow.reverse_post_ip_diff_serv_code_point`**
:   type: short


**`netflow.reverse_post_ip_precedence`**
:   type: short


**`netflow.reverse_post_layer2_octet_delta_count`**
:   type: long


**`netflow.reverse_post_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_post_mcast_layer2_octet_delta_count`**
:   type: long


**`netflow.reverse_post_mcast_layer2_octet_total_count`**
:   type: long


**`netflow.reverse_post_mcast_octet_delta_count`**
:   type: long


**`netflow.reverse_post_mcast_octet_total_count`**
:   type: long


**`netflow.reverse_post_mcast_packet_delta_count`**
:   type: long


**`netflow.reverse_post_mcast_packet_total_count`**
:   type: long


**`netflow.reverse_post_mpls_top_label_exp`**
:   type: short


**`netflow.reverse_post_napt_destination_transport_port`**
:   type: integer


**`netflow.reverse_post_napt_source_transport_port`**
:   type: integer


**`netflow.reverse_post_nat_destination_ipv4_address`**
:   type: ip


**`netflow.reverse_post_nat_destination_ipv6_address`**
:   type: ip


**`netflow.reverse_post_nat_source_ipv4_address`**
:   type: ip


**`netflow.reverse_post_nat_source_ipv6_address`**
:   type: ip


**`netflow.reverse_post_octet_delta_count`**
:   type: long


**`netflow.reverse_post_octet_total_count`**
:   type: long


**`netflow.reverse_post_packet_delta_count`**
:   type: long


**`netflow.reverse_post_packet_total_count`**
:   type: long


**`netflow.reverse_post_source_mac_address`**
:   type: keyword


**`netflow.reverse_post_vlan_id`**
:   type: integer


**`netflow.reverse_private_enterprise_number`**
:   type: long


**`netflow.reverse_protocol_identifier`**
:   type: short


**`netflow.reverse_pseudo_wire_control_word`**
:   type: long


**`netflow.reverse_pseudo_wire_destination_ipv4_address`**
:   type: ip


**`netflow.reverse_pseudo_wire_id`**
:   type: long


**`netflow.reverse_pseudo_wire_type`**
:   type: integer


**`netflow.reverse_relative_error`**
:   type: double


**`netflow.reverse_responder_octets`**
:   type: long


**`netflow.reverse_responder_packets`**
:   type: long


**`netflow.reverse_rfc3550_jitter_microseconds`**
:   type: long


**`netflow.reverse_rfc3550_jitter_milliseconds`**
:   type: long


**`netflow.reverse_rfc3550_jitter_nanoseconds`**
:   type: long


**`netflow.reverse_rtp_payload_type`**
:   type: short


**`netflow.reverse_rtp_sequence_number`**
:   type: integer


**`netflow.reverse_sampler_id`**
:   type: short


**`netflow.reverse_sampler_mode`**
:   type: short


**`netflow.reverse_sampler_name`**
:   type: keyword


**`netflow.reverse_sampler_random_interval`**
:   type: long


**`netflow.reverse_sampling_algorithm`**
:   type: short


**`netflow.reverse_sampling_flow_interval`**
:   type: long


**`netflow.reverse_sampling_flow_spacing`**
:   type: long


**`netflow.reverse_sampling_interval`**
:   type: long


**`netflow.reverse_sampling_packet_interval`**
:   type: long


**`netflow.reverse_sampling_packet_space`**
:   type: long


**`netflow.reverse_sampling_population`**
:   type: long


**`netflow.reverse_sampling_probability`**
:   type: double


**`netflow.reverse_sampling_size`**
:   type: long


**`netflow.reverse_sampling_time_interval`**
:   type: long


**`netflow.reverse_sampling_time_space`**
:   type: long


**`netflow.reverse_second_packet_banner`**
:   type: keyword


**`netflow.reverse_section_exported_octets`**
:   type: integer


**`netflow.reverse_section_offset`**
:   type: integer


**`netflow.reverse_selection_sequence_id`**
:   type: long


**`netflow.reverse_selector_algorithm`**
:   type: integer


**`netflow.reverse_selector_id`**
:   type: long


**`netflow.reverse_selector_id_total_flows_observed`**
:   type: long


**`netflow.reverse_selector_id_total_flows_selected`**
:   type: long


**`netflow.reverse_selector_id_total_pkts_observed`**
:   type: long


**`netflow.reverse_selector_id_total_pkts_selected`**
:   type: long


**`netflow.reverse_selector_name`**
:   type: keyword


**`netflow.reverse_session_scope`**
:   type: short


**`netflow.reverse_small_packet_count`**
:   type: long


**`netflow.reverse_source_ipv4_address`**
:   type: ip


**`netflow.reverse_source_ipv4_prefix`**
:   type: ip


**`netflow.reverse_source_ipv4_prefix_length`**
:   type: short


**`netflow.reverse_source_ipv6_address`**
:   type: ip


**`netflow.reverse_source_ipv6_prefix`**
:   type: ip


**`netflow.reverse_source_ipv6_prefix_length`**
:   type: short


**`netflow.reverse_source_mac_address`**
:   type: keyword


**`netflow.reverse_source_transport_port`**
:   type: integer


**`netflow.reverse_src_traffic_index`**
:   type: long


**`netflow.reverse_sta_ipv4_address`**
:   type: ip


**`netflow.reverse_sta_mac_address`**
:   type: keyword


**`netflow.reverse_standard_deviation_interarrival_time`**
:   type: long


**`netflow.reverse_standard_deviation_payload_length`**
:   type: integer


**`netflow.reverse_system_init_time_milliseconds`**
:   type: long


**`netflow.reverse_tcp_ack_total_count`**
:   type: long


**`netflow.reverse_tcp_acknowledgement_number`**
:   type: long


**`netflow.reverse_tcp_control_bits`**
:   type: integer


**`netflow.reverse_tcp_destination_port`**
:   type: integer


**`netflow.reverse_tcp_fin_total_count`**
:   type: long


**`netflow.reverse_tcp_header_length`**
:   type: short


**`netflow.reverse_tcp_options`**
:   type: long


**`netflow.reverse_tcp_psh_total_count`**
:   type: long


**`netflow.reverse_tcp_rst_total_count`**
:   type: long


**`netflow.reverse_tcp_sequence_number`**
:   type: long


**`netflow.reverse_tcp_source_port`**
:   type: integer


**`netflow.reverse_tcp_syn_total_count`**
:   type: long


**`netflow.reverse_tcp_urg_total_count`**
:   type: long


**`netflow.reverse_tcp_urgent_pointer`**
:   type: integer


**`netflow.reverse_tcp_window_scale`**
:   type: integer


**`netflow.reverse_tcp_window_size`**
:   type: integer


**`netflow.reverse_total_length_ipv4`**
:   type: integer


**`netflow.reverse_transport_octet_delta_count`**
:   type: long


**`netflow.reverse_transport_packet_delta_count`**
:   type: long


**`netflow.reverse_tunnel_technology`**
:   type: keyword


**`netflow.reverse_udp_destination_port`**
:   type: integer


**`netflow.reverse_udp_message_length`**
:   type: integer


**`netflow.reverse_udp_source_port`**
:   type: integer


**`netflow.reverse_union_tcp_flags`**
:   type: short


**`netflow.reverse_upper_ci_limit`**
:   type: double


**`netflow.reverse_user_name`**
:   type: keyword


**`netflow.reverse_value_distribution_method`**
:   type: short


**`netflow.reverse_virtual_station_interface_id`**
:   type: keyword


**`netflow.reverse_virtual_station_interface_name`**
:   type: keyword


**`netflow.reverse_virtual_station_name`**
:   type: keyword


**`netflow.reverse_virtual_station_uuid`**
:   type: keyword


**`netflow.reverse_vlan_id`**
:   type: integer


**`netflow.reverse_vr_fname`**
:   type: keyword


**`netflow.reverse_wlan_channel_id`**
:   type: short


**`netflow.reverse_wlan_ssid`**
:   type: keyword


**`netflow.reverse_wtp_mac_address`**
:   type: keyword


**`netflow.rfc3550_jitter_microseconds`**
:   type: long


**`netflow.rfc3550_jitter_milliseconds`**
:   type: long


**`netflow.rfc3550_jitter_nanoseconds`**
:   type: long


**`netflow.rtp_payload_type`**
:   type: short


**`netflow.rtp_sequence_number`**
:   type: integer


**`netflow.sampler_id`**
:   type: short


**`netflow.sampler_mode`**
:   type: short


**`netflow.sampler_name`**
:   type: keyword


**`netflow.sampler_random_interval`**
:   type: long


**`netflow.sampling_algorithm`**
:   type: short


**`netflow.sampling_flow_interval`**
:   type: long


**`netflow.sampling_flow_spacing`**
:   type: long


**`netflow.sampling_interval`**
:   type: long


**`netflow.sampling_packet_interval`**
:   type: long


**`netflow.sampling_packet_space`**
:   type: long


**`netflow.sampling_population`**
:   type: long


**`netflow.sampling_probability`**
:   type: double


**`netflow.sampling_size`**
:   type: long


**`netflow.sampling_time_interval`**
:   type: long


**`netflow.sampling_time_space`**
:   type: long


**`netflow.second_packet_banner`**
:   type: keyword


**`netflow.section_exported_octets`**
:   type: integer


**`netflow.section_offset`**
:   type: integer


**`netflow.selection_sequence_id`**
:   type: long


**`netflow.selector_algorithm`**
:   type: integer


**`netflow.selector_id`**
:   type: long


**`netflow.selector_id_total_flows_observed`**
:   type: long


**`netflow.selector_id_total_flows_selected`**
:   type: long


**`netflow.selector_id_total_pkts_observed`**
:   type: long


**`netflow.selector_id_total_pkts_selected`**
:   type: long


**`netflow.selector_name`**
:   type: keyword


**`netflow.service_name`**
:   type: keyword


**`netflow.session_scope`**
:   type: short


**`netflow.silk_app_label`**
:   type: integer


**`netflow.small_packet_count`**
:   type: long


**`netflow.source_ipv4_address`**
:   type: ip


**`netflow.source_ipv4_prefix`**
:   type: ip


**`netflow.source_ipv4_prefix_length`**
:   type: short


**`netflow.source_ipv6_address`**
:   type: ip


**`netflow.source_ipv6_prefix`**
:   type: ip


**`netflow.source_ipv6_prefix_length`**
:   type: short


**`netflow.source_mac_address`**
:   type: keyword


**`netflow.source_transport_port`**
:   type: integer


**`netflow.source_transport_ports_limit`**
:   type: integer


**`netflow.src_traffic_index`**
:   type: long


**`netflow.ssl_cert_serial_number`**
:   type: keyword


**`netflow.ssl_cert_signature`**
:   type: keyword


**`netflow.ssl_cert_validity_not_after`**
:   type: keyword


**`netflow.ssl_cert_validity_not_before`**
:   type: keyword


**`netflow.ssl_cert_version`**
:   type: short


**`netflow.ssl_certificate_hash`**
:   type: keyword


**`netflow.ssl_cipher`**
:   type: keyword


**`netflow.ssl_client_version`**
:   type: short


**`netflow.ssl_compression_method`**
:   type: short


**`netflow.ssl_object_type`**
:   type: keyword


**`netflow.ssl_object_value`**
:   type: keyword


**`netflow.ssl_public_key_algorithm`**
:   type: keyword


**`netflow.ssl_public_key_length`**
:   type: keyword


**`netflow.ssl_server_cipher`**
:   type: long


**`netflow.ssl_server_name`**
:   type: keyword


**`netflow.sta_ipv4_address`**
:   type: ip


**`netflow.sta_mac_address`**
:   type: keyword


**`netflow.standard_deviation_interarrival_time`**
:   type: long


**`netflow.standard_deviation_payload_length`**
:   type: short


**`netflow.system_init_time_milliseconds`**
:   type: date


**`netflow.tcp_ack_total_count`**
:   type: long


**`netflow.tcp_acknowledgement_number`**
:   type: long


**`netflow.tcp_control_bits`**
:   type: integer


**`netflow.tcp_destination_port`**
:   type: integer


**`netflow.tcp_fin_total_count`**
:   type: long


**`netflow.tcp_header_length`**
:   type: short


**`netflow.tcp_options`**
:   type: long


**`netflow.tcp_psh_total_count`**
:   type: long


**`netflow.tcp_rst_total_count`**
:   type: long


**`netflow.tcp_sequence_number`**
:   type: long


**`netflow.tcp_source_port`**
:   type: integer


**`netflow.tcp_syn_total_count`**
:   type: long


**`netflow.tcp_urg_total_count`**
:   type: long


**`netflow.tcp_urgent_pointer`**
:   type: integer


**`netflow.tcp_window_scale`**
:   type: integer


**`netflow.tcp_window_size`**
:   type: integer


**`netflow.template_id`**
:   type: integer


**`netflow.tftp_filename`**
:   type: keyword


**`netflow.tftp_mode`**
:   type: keyword


**`netflow.timestamp`**
:   type: long


**`netflow.timestamp_absolute_monitoring-interval`**
:   type: long


**`netflow.total_length_ipv4`**
:   type: integer


**`netflow.traffic_type`**
:   type: short


**`netflow.transport_octet_delta_count`**
:   type: long


**`netflow.transport_packet_delta_count`**
:   type: long


**`netflow.tunnel_technology`**
:   type: keyword


**`netflow.udp_destination_port`**
:   type: integer


**`netflow.udp_message_length`**
:   type: integer


**`netflow.udp_source_port`**
:   type: integer


**`netflow.union_tcp_flags`**
:   type: short


**`netflow.upper_ci_limit`**
:   type: double


**`netflow.user_name`**
:   type: keyword


**`netflow.username`**
:   type: keyword


**`netflow.value_distribution_method`**
:   type: short


**`netflow.viptela_vpn_id`**
:   type: long


**`netflow.virtual_station_interface_id`**
:   type: short


**`netflow.virtual_station_interface_name`**
:   type: keyword


**`netflow.virtual_station_name`**
:   type: keyword


**`netflow.virtual_station_uuid`**
:   type: short


**`netflow.vlan_id`**
:   type: integer


**`netflow.vmware_egress_interface_attr`**
:   type: integer


**`netflow.vmware_ingress_interface_attr`**
:   type: integer


**`netflow.vmware_tenant_dest_ipv4`**
:   type: ip


**`netflow.vmware_tenant_dest_ipv6`**
:   type: ip


**`netflow.vmware_tenant_dest_port`**
:   type: integer


**`netflow.vmware_tenant_protocol`**
:   type: short


**`netflow.vmware_tenant_source_ipv4`**
:   type: ip


**`netflow.vmware_tenant_source_ipv6`**
:   type: ip


**`netflow.vmware_tenant_source_port`**
:   type: integer


**`netflow.vmware_vxlan_export_role`**
:   type: short


**`netflow.vpn_identifier`**
:   type: short


**`netflow.vr_fname`**
:   type: keyword


**`netflow.waasoptimization_segment`**
:   type: short


**`netflow.wlan_channel_id`**
:   type: short


**`netflow.wlan_ssid`**
:   type: keyword


**`netflow.wtp_mac_address`**
:   type: keyword


**`netflow.xlate_destination_address_ip_v4`**
:   type: ip


**`netflow.xlate_destination_port`**
:   type: integer


**`netflow.xlate_source_address_ip_v4`**
:   type: ip


**`netflow.xlate_source_port`**
:   type: integer


