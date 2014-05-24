packetbeat-deps:
  pkg:
    - latest
    - names:
{% if grains['os'] == 'RedHat' %}
      - libpcap
      - daemonize
{% elif grains['os'] == 'Ubuntu' %}
      - libpcap0.8
{% endif %}
    - require_in: packetbeat
packetbeat:
  pkg:
    - installed
    - sources:
{% if grains['os_family'] == 'RedHat' %}
      - packetbeat: https://github.com/packetbeat/packetbeat/releases/download/v0.2.0/packetbeat-0.2.0-1.el6.x86_64.rpm
{% elif grains['os_family'] == 'Debian' %}
      - packetbeat: https://github.com/packetbeat/packetbeat/releases/download/v0.2.0/packetbeat_0.2.0-1_amd64.deb
{% endif %}
  service:
      - running
      - watch:
          - file: /etc/packetbeat/packetbeat.conf
/etc/packetbeat/packetbeat.conf:
  file:
    - managed
    - source: salt://packetbeat/packetbeat.conf
    - user: root
    - group: root
    - mode: 755
    - template: jinja
    - context:
      elasticsearch: 192.168.100.136
      procs: [ mysqld,apache2,nginx,php-fpm,node ]
    - require:
      - pkg:  packetbeat
