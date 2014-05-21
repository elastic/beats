Summary:	Packetbeat network agent
Name:		packetbeat
Version:	0.2.0
Release:	1%{?dist}
Source:		%{name}.tar.gz

Group:		Network
License:	GPLv2
URL:		http://packetbeat.com

Requires:	 libpcap, daemonize
Requires(post):  chkconfig
Requires(preun): chkconfig
Requires(preun): initscripts

%description
Packetbeat agent.

%prep
%setup -n %{name}

%build
make

%install
make install DESTDIR=%{buildroot}
install -D rpm/packetbeat.init %{buildroot}/etc/rc.d/init.d/packetbeat

%files
/usr/bin/*
/etc/packetbeat/*
/etc/rc.d/init.d/packetbeat

%post
# This adds the proper /etc/rc*.d links for the script
/sbin/chkconfig --add packetbeat

%preun
/etc/init.d/packetbeat stop
/sbin/chkconfig --del packetbeat

%doc

%changelog

