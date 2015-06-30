Summary:	Packetbeat network agent
Name:		packetbeat
Version:	1.0.0~beta2
Release:	1%{dist}
Source:		%{name}.tar.gz
BuildRoot: %{_tmppath}/%{name}

Group:		Network
License:	GPLv2
URL:		http://packetbeat.com

Requires:	 libpcap
Requires(post):  chkconfig
Requires(preun): chkconfig
Requires(preun): initscripts

%description
Packetbeat agent.

%prep
%define _build_name_fmt %%{ARCH}/%%{NAME}-%%{VERSION}-%%{ARCH}.rpm
%setup -n %{name}

%build
make

%install
make install DESTDIR=%{buildroot}
install -D rpm/packetbeat.init %{buildroot}/etc/rc.d/init.d/packetbeat

%files
/usr/bin/*
/etc/rc.d/init.d/packetbeat
/etc/packetbeat/packetbeat.template.json
%config /etc/packetbeat/packetbeat.yml

%doc debian/copyright

%post
# This adds the proper /etc/rc*.d links for the script
/sbin/chkconfig --add packetbeat

%preun
/etc/init.d/packetbeat stop
/sbin/chkconfig --del packetbeat


%changelog

