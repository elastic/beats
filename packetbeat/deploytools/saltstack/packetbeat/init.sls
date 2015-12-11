{% set elasticsearch           = salt['pillar.get']('packetbeat:elasticsearch') -%}
{% set ignore_outgoing           = salt['pillar.get']('packetbeat:ignore_outgoing', 'false') -%}
{% set procs          = salt['pillar.get']('packetbeat:procs', 'mysqld,apache2,nginx,php-fpm,node') -%}
packetbeat-deps:
  pkg:
    - latest
    - names:
{% if grains['os_family'] == 'RedHat' %}
      - libpcap
{% elif grains['os_family'] == 'Debian' %}
      - libpcap0.8
{% endif %}
    - require_in: packetbeat
packetbeat:
  pkg:
    - installed
    - sources:
{% if grains['os_family'] == 'RedHat' %}
      - packetbeat: https://download.elastic.co/beats/packetbeat/packetbeat-1.0.1-x86_64.rpm
{% elif grains['os_family'] == 'Debian' %}
      - packetbeat: https://download.elastic.co/beats/packetbeat/packetbeat_1.0.1_amd64.deb
{% endif %}
  service:
    - running
    - watch:
      - file: /etc/packetbeat/packetbeat.yml
/etc/packetbeat/packetbeat.yml:
  file:
    - managed
    - source: salt://packetbeat/packetbeat.yml
    - user: root
    - group: root
    - mode: 755
    - template: jinja
    - context:
      elasticsearch: {{ elasticsearch }}
      procs: {{ procs }}
      ignore_outgoing: {{ ignore_outgoing }}
    - require:
      - pkg: packetbeat
